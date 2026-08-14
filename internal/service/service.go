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
	GetRunStatus(ctx context.Context, auditID, runID int) (*model.AuditRun, error)
	ListIssues(ctx context.Context, page, pageSize int) ([]model.Issue, error)
	GetIssue(ctx context.Context, id int) (*model.Issue, error)
}

type CreateAuditRequest struct {
	Name    string
	Queries []model.QueryRule
}

type RunAuditResponse struct {
	Run   *model.AuditRun `json:"run"`
	Issue *model.Issue    `json:"issue,omitempty"`
}

type securityCentralService struct {
	repo repository.Repository
}

func New(repo repository.Repository) Service {
	return &securityCentralService{repo: repo}
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
	a, err := s.repo.GetAudit(ctx, auditID)
	if err != nil {
		return nil, err
	}

	run, err := s.repo.CreateAuditRun(ctx, auditID)
	if err != nil {
		return nil, err
	}

	failCount := 0
	actualPairs := make([]string, 0, len(a.SQLQuery))
	var firstIssue *model.Issue

	for _, q := range a.SQLQuery {
		actualValue, queryErr := s.repo.RunScalarQuery(ctx, q.SQLQuery)
		if queryErr != nil {
			msg := fmt.Sprintf("query %q failed: %s", q.Name, queryErr.Error())
			updatedRun, updateErr := s.repo.UpdateAuditRunResult(ctx, run.ID, "error", nil, &msg)
			if updateErr != nil {
				return nil, updateErr
			}
			return &RunAuditResponse{Run: updatedRun}, nil
		}
		actualPairs = append(actualPairs, fmt.Sprintf("%s=%s", q.Name, actualValue))
		if !valuesEqualByType(q.ExpectedResult.Type, q.ExpectedResult.Value, actualValue) {
			failCount++
			description := fmt.Sprintf("Audit %q query %q deviated: expected=%s actual=%s", a.Name, q.Name, q.ExpectedResult.Value, actualValue)
			issue, issueErr := s.repo.CreateIssue(ctx, a.ID, run.ID, q.Name, q.ExpectedResult.Value, actualValue, description)
			if issueErr != nil {
				return nil, issueErr
			}
			if firstIssue == nil {
				firstIssue = issue
			}
		}
	}

	status := "passed"
	if failCount > 0 {
		status = "failed"
	}
	actualSummary := strings.Join(actualPairs, ", ")

	updatedRun, err := s.repo.UpdateAuditRunResult(ctx, run.ID, status, &actualSummary, nil)
	if err != nil {
		return nil, err
	}

	return &RunAuditResponse{Run: updatedRun, Issue: firstIssue}, nil
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
