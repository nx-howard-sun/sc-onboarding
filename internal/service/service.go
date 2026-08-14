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
	Name           string
	SQLQuery       string
	ExpectedResult model.ExpectedResult
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
	if err := validateSQLQuery(req.SQLQuery); err != nil {
		return nil, err
	}
	if err := validateExpectedType(req.ExpectedResult.Type); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.ExpectedResult.Value) == "" {
		return nil, errors.New("expected_result.value is required")
	}
	return s.repo.CreateAudit(ctx, req.Name, req.SQLQuery, req.ExpectedResult.Type, req.ExpectedResult.Value)
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

	actualValue, queryErr := s.repo.RunScalarQuery(ctx, a.SQLQuery)
	if queryErr != nil {
		msg := queryErr.Error()
		updatedRun, updateErr := s.repo.UpdateAuditRunResult(ctx, run.ID, "error", nil, &msg)
		if updateErr != nil {
			return nil, updateErr
		}
		return &RunAuditResponse{Run: updatedRun}, nil
	}

	status := "failed"
	if valuesEqualByType(a.ExpectedResult.Type, a.ExpectedResult.Value, actualValue) {
		status = "passed"
	}

	updatedRun, err := s.repo.UpdateAuditRunResult(ctx, run.ID, status, &actualValue, nil)
	if err != nil {
		return nil, err
	}

	resp := &RunAuditResponse{Run: updatedRun}
	if status == "failed" {
		description := fmt.Sprintf("Audit %q deviated: expected=%s actual=%s", a.Name, a.ExpectedResult.Value, actualValue)
		issue, issueErr := s.repo.CreateIssue(ctx, a.ID, run.ID, a.ExpectedResult.Value, actualValue, description)
		if issueErr != nil {
			return nil, issueErr
		}
		resp.Issue = issue
	}
	return resp, nil
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
