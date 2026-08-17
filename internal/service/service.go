package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"security-central/internal/model"
	"security-central/internal/repository"
)

var selectOnlyRegex = regexp.MustCompile(`(?is)^\s*select\s+`)

type Service interface {
	CreateAudit(ctx context.Context, req CreateAuditRequest) (*model.Audit, error)
	GetAudit(ctx context.Context, id int) (*model.Audit, error)
	RunAudit(ctx context.Context, auditID int) (*RunAuditResponse, error)
	ExecuteAuditRun(ctx context.Context, auditID, runID int) (*model.AuditRun, error)
	GetRunStatus(ctx context.Context, auditID, runID int) (*model.AuditRun, error)
	ListIssues(ctx context.Context, page, pageSize int) ([]model.Issue, error)
	GetIssue(ctx context.Context, id int) (*model.Issue, error)
	CreatePolicy(ctx context.Context, req CreatePolicyRequest) (*model.Policy, error)
	GetPolicy(ctx context.Context, id int) (*model.Policy, error)
	RunPolicy(ctx context.Context, policyID int) (*RunPolicyResponse, error)
	GetPolicyRunStatus(ctx context.Context, policyID, runID int) (*model.PolicyRunDetail, error)
}

type CreateAuditRequest struct {
	Name    string
	Queries []model.QueryRule
}

type RunAuditResponse struct {
	Run      *model.AuditRun `json:"run"`
	Accepted bool            `json:"accepted"`
	Message  string          `json:"message,omitempty"`
}

type CreatePolicyRequest struct {
	Name     string `json:"name"`
	AuditIDs []int  `json:"audit_ids"`
}

type RunPolicyResponse struct {
	Run      *model.PolicyRun `json:"run"`
	Accepted bool             `json:"accepted"`
	Message  string           `json:"message,omitempty"`
}

type AuditExecutor interface {
	ExecuteAudit(ctx context.Context, auditID, runID int) error
}

type securityCentralService struct {
	repo     repository.Repository
	executor AuditExecutor
}

func New(repo repository.Repository, executor AuditExecutor) Service {
	return &securityCentralService{repo: repo, executor: executor}
}

func (s *securityCentralService) CreateAudit(ctx context.Context, req CreateAuditRequest) (*model.Audit, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, errors.New("name is required")
	}
	if len(req.Queries) == 0 {
		return nil, errors.New("queries must have at least one query rule")
	}
	for i, q := range req.Queries {
		if strings.TrimSpace(q.Name) == "" {
			return nil, fmt.Errorf("queries[%d].name is required", i)
		}
		if err := validateSQLQuery(q.SQLQuery); err != nil {
			return nil, fmt.Errorf("queries[%d].sql_query invalid: %w", i, err)
		}
		if err := validateExpectedType(q.ExpectedResult.Type); err != nil {
			return nil, fmt.Errorf("queries[%d].expected_result.type invalid: %w", i, err)
		}
		if strings.TrimSpace(q.ExpectedResult.Value) == "" {
			return nil, fmt.Errorf("queries[%d].expected_result.value is required", i)
		}
	}
	return s.repo.CreateAudit(ctx, req.Name, req.Queries)
}

func (s *securityCentralService) GetAudit(ctx context.Context, id int) (*model.Audit, error) {
	return s.repo.GetAudit(ctx, id)
}

func (s *securityCentralService) RunAudit(ctx context.Context, auditID int) (*RunAuditResponse, error) {
	if _, err := s.repo.GetAudit(ctx, auditID); err != nil {
		return nil, err
	}
	run, err := s.repo.CreateAuditRun(ctx, auditID)
	if err != nil {
		return nil, err
	}
	if s.executor == nil {
		msg := "audit executor is not configured"
		updatedRun, updateErr := s.repo.UpdateAuditRunResult(ctx, run.ID, "error", nil, &msg)
		if updateErr != nil {
			return nil, updateErr
		}
		return &RunAuditResponse{Run: updatedRun, Accepted: false, Message: msg}, nil
	}
	if err := s.executor.ExecuteAudit(ctx, auditID, run.ID); err != nil {
		msg := fmt.Sprintf("failed to dispatch audit run to worker: %v", err)
		updatedRun, updateErr := s.repo.UpdateAuditRunResult(ctx, run.ID, "error", nil, &msg)
		if updateErr != nil {
			return nil, updateErr
		}
		return &RunAuditResponse{Run: updatedRun, Accepted: false, Message: msg}, nil
	}
	return &RunAuditResponse{
		Run:      run,
		Accepted: true,
		Message:  "audit run accepted for asynchronous execution",
	}, nil
}

func (s *securityCentralService) ExecuteAuditRun(ctx context.Context, auditID, runID int) (*model.AuditRun, error) {
	a, err := s.repo.GetAudit(ctx, auditID)
	if err != nil {
		return nil, err
	}
	if _, err := s.repo.GetAuditRun(ctx, auditID, runID); err != nil {
		return nil, err
	}
	failCount := 0
	actualPairs := make([]string, 0, len(a.SQLQuery))

	for _, q := range a.SQLQuery {
		actualValue, queryErr := s.repo.RunScalarQuery(ctx, q.SQLQuery)
		if queryErr != nil {
			msg := fmt.Sprintf("query %q failed: %s", q.Name, queryErr.Error())
			updatedRun, updateErr := s.repo.UpdateAuditRunResult(ctx, runID, "error", nil, &msg)
			if updateErr != nil {
				return nil, updateErr
			}
			return updatedRun, nil
		}
		actualPairs = append(actualPairs, fmt.Sprintf("%s=%s", q.Name, actualValue))
		if !valuesEqualByType(q.ExpectedResult.Type, q.ExpectedResult.Value, actualValue) {
			failCount++
			description := fmt.Sprintf("Audit %q query %q deviated: expected=%s actual=%s", a.Name, q.Name, q.ExpectedResult.Value, actualValue)
			_, issueErr := s.repo.CreateIssue(ctx, a.ID, runID, q.Name, q.ExpectedResult.Value, actualValue, description)
			if issueErr != nil {
				return nil, issueErr
			}
		}
	}

	status := "passed"
	if failCount > 0 {
		status = "failed"
	}
	actualSummary := strings.Join(actualPairs, ", ")

	updatedRun, err := s.repo.UpdateAuditRunResult(ctx, runID, status, &actualSummary, nil)
	if err != nil {
		return nil, err
	}
	return updatedRun, nil
}

func (s *securityCentralService) GetRunStatus(ctx context.Context, auditID, runID int) (*model.AuditRun, error) {
	return s.repo.GetAuditRun(ctx, auditID, runID)
}

func (s *securityCentralService) ListIssues(ctx context.Context, page, pageSize int) ([]model.Issue, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return s.repo.ListIssues(ctx, page, pageSize)
}

func (s *securityCentralService) GetIssue(ctx context.Context, id int) (*model.Issue, error) {
	return s.repo.GetIssue(ctx, id)
}

// CreatePolicy validates and creates a new policy grouping multiple audit IDs.
func (s *securityCentralService) CreatePolicy(ctx context.Context, req CreatePolicyRequest) (*model.Policy, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, errors.New("policy name is required")
	}
	if len(req.AuditIDs) == 0 {
		return nil, errors.New("policy must contain at least one audit_id")
	}

	// Validate that all referenced audit IDs exist in database
	for _, auditID := range req.AuditIDs {
		if _, err := s.repo.GetAudit(ctx, auditID); err != nil {
			return nil, fmt.Errorf("audit id %d not found: %w", auditID, err)
		}
	}

	return s.repo.CreatePolicy(ctx, req.Name, req.AuditIDs)
}

// GetPolicy retrieves a policy definition by ID.
func (s *securityCentralService) GetPolicy(ctx context.Context, id int) (*model.Policy, error) {
	return s.repo.GetPolicy(ctx, id)
}

// RunPolicy triggers execution for all audits grouped within a policy.
func (s *securityCentralService) RunPolicy(ctx context.Context, policyID int) (*RunPolicyResponse, error) {
	p, err := s.repo.GetPolicy(ctx, policyID)
	if err != nil {
		return nil, fmt.Errorf("policy not found: %w", err)
	}

	if s.executor == nil {
		return nil, errors.New("audit executor is not configured")
	}

	auditRunIDs := make([]int, 0, len(p.AuditIDs))
	var dispatchErrs []string

	// Loop over all audits in the policy and dispatch each to gRPC worker
	for _, auditID := range p.AuditIDs {
		run, err := s.repo.CreateAuditRun(ctx, auditID)
		if err != nil {
			dispatchErrs = append(dispatchErrs, fmt.Sprintf("audit %d run creation failed: %v", auditID, err))
			continue
		}
		auditRunIDs = append(auditRunIDs, run.ID)

		// Dispatch individual audit execution via existing AuditExecutor interface
		if err := s.executor.ExecuteAudit(ctx, auditID, run.ID); err != nil {
			msg := fmt.Sprintf("failed to dispatch audit run to worker: %v", err)
			s.repo.UpdateAuditRunResult(ctx, run.ID, "error", nil, &msg)
			dispatchErrs = append(dispatchErrs, fmt.Sprintf("audit %d dispatch failed: %v", auditID, err))
		}
	}

	policyRun, err := s.repo.CreatePolicyRun(ctx, policyID, auditRunIDs)
	if err != nil {
		return nil, err
	}

	msg := fmt.Sprintf("Policy execution started across %d child audits", len(auditRunIDs))
	if len(dispatchErrs) > 0 {
		msg += fmt.Sprintf(" (with errors: %s)", strings.Join(dispatchErrs, "; "))
	}

	return &RunPolicyResponse{
		Run:      policyRun,
		Accepted: true,
		Message:  msg,
	}, nil
}

// GetPolicyRunStatus retrieves current status of a policy run and evaluates aggregate status from child runs.
func (s *securityCentralService) GetPolicyRunStatus(ctx context.Context, policyID, runID int) (*model.PolicyRunDetail, error) {
	policyRun, err := s.repo.GetPolicyRun(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("policy run not found: %w", err)
	}
	if policyRun.PolicyID != policyID {
		return nil, errors.New("policy_id mismatch for requested run")
	}

	// Fetch status of all child audit runs
	auditRuns, err := s.repo.GetAuditRunsByIDs(ctx, policyRun.AuditRunIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch child audit runs: %w", err)
	}

	// Dynamic status evaluation: If policy run is currently "running", check if all child runs finished
	if policyRun.Status == "running" && len(auditRuns) > 0 {
		allCompleted := true
		hasError := false
		hasFailed := false

		for _, ar := range auditRuns {
			switch ar.Status {
			case "running":
				allCompleted = false
			case "error":
				hasError = true
			case "failed":
				hasFailed = true
			}
		}

		// Update PolicyRun status once all child audit runs finish
		if allCompleted {
			finalStatus := "passed"
			if hasError {
				finalStatus = "error"
			} else if hasFailed {
				finalStatus = "failed"
			}

			updatedRun, err := s.repo.UpdatePolicyRunStatus(ctx, runID, finalStatus)
			if err == nil {
				policyRun = updatedRun
			}
		}
	}

	return &model.PolicyRunDetail{
		PolicyRun: *policyRun,
		AuditRuns: auditRuns,
	}, nil
}

func validateSQLQuery(query string) error {
	q := strings.TrimSpace(query)
	if q == "" {
		return errors.New("sql_query is required")
	}
	if strings.Contains(q, ";") {
		return errors.New("sql_query must be a single SELECT statement")
	}
	if !selectOnlyRegex.MatchString(q) {
		return errors.New("only SELECT statements are allowed")
	}
	return nil
}

func validateExpectedType(t string) error {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "int", "string", "bool", "float":
		return nil
	default:
		return errors.New("expected_result.type must be one of: int, string, bool, float")
	}
}

func valuesEqualByType(t, expected, actual string) bool {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "int":
		e, err1 := strconv.ParseInt(strings.TrimSpace(expected), 10, 64)
		a, err2 := strconv.ParseInt(strings.TrimSpace(actual), 10, 64)
		return err1 == nil && err2 == nil && e == a
	case "float":
		e, err1 := strconv.ParseFloat(strings.TrimSpace(expected), 64)
		a, err2 := strconv.ParseFloat(strings.TrimSpace(actual), 64)
		return err1 == nil && err2 == nil && e == a
	case "bool":
		e, err1 := strconv.ParseBool(strings.TrimSpace(expected))
		a, err2 := strconv.ParseBool(strings.TrimSpace(actual))
		return err1 == nil && err2 == nil && e == a
	default:
		return strings.TrimSpace(expected) == strings.TrimSpace(actual)
	}
}
