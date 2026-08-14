package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"security-central/internal/model"
)

type mockRepo struct {
	getAuditFn            func(ctx context.Context, id int) (*model.Audit, error)
	createAuditRunFn      func(ctx context.Context, auditID int) (*model.AuditRun, error)
	runScalarQueryFn      func(ctx context.Context, query string) (string, error)
	updateAuditRunResult  func(ctx context.Context, runID int, status string, actualValue *string, errMsg *string) (*model.AuditRun, error)
	createIssueFn         func(ctx context.Context, auditID, runID int, expectedValue, actualValue, description string) (*model.Issue, error)
	issueCreateCallCount  int
	lastIssueExpected     string
	lastIssueActual       string
	lastUpdatedRunStatus  string
	lastUpdatedRunErrText *string
}

func (m *mockRepo) CreateAudit(context.Context, string, string, string, string) (*model.Audit, error) {
	panic("not implemented")
}
func (m *mockRepo) GetAudit(ctx context.Context, id int) (*model.Audit, error) {
	return m.getAuditFn(ctx, id)
}
func (m *mockRepo) CreateAuditRun(ctx context.Context, auditID int) (*model.AuditRun, error) {
	return m.createAuditRunFn(ctx, auditID)
}
func (m *mockRepo) UpdateAuditRunResult(ctx context.Context, runID int, status string, actualValue *string, errMsg *string) (*model.AuditRun, error) {
	m.lastUpdatedRunStatus = status
	m.lastUpdatedRunErrText = errMsg
	return m.updateAuditRunResult(ctx, runID, status, actualValue, errMsg)
}
func (m *mockRepo) GetAuditRun(context.Context, int, int) (*model.AuditRun, error) {
	panic("not implemented")
}
func (m *mockRepo) CreateIssue(ctx context.Context, auditID, runID int, expectedValue, actualValue, description string) (*model.Issue, error) {
	m.issueCreateCallCount++
	m.lastIssueExpected = expectedValue
	m.lastIssueActual = actualValue
	return m.createIssueFn(ctx, auditID, runID, expectedValue, actualValue, description)
}
func (m *mockRepo) ListIssues(context.Context, int, int) ([]model.Issue, error) { panic("not implemented") }
func (m *mockRepo) GetIssue(context.Context, int) (*model.Issue, error)          { panic("not implemented") }
func (m *mockRepo) RunScalarQuery(ctx context.Context, query string) (string, error) {
	return m.runScalarQueryFn(ctx, query)
}

func TestRunAudit_Passed_NoIssue(t *testing.T) {
	repo := &mockRepo{
		getAuditFn: func(context.Context, int) (*model.Audit, error) {
			return &model.Audit{
				ID:       10,
				Name:     "Memory Compliance",
				SQLQuery: "SELECT count(*) FROM vm_inventory WHERE memory > 50",
				ExpectedResult: model.ExpectedResult{
					Type:  "int",
					Value: "0",
				},
			}, nil
		},
		createAuditRunFn: func(context.Context, int) (*model.AuditRun, error) {
			return &model.AuditRun{ID: 99, AuditID: 10, Status: "running", StartedAt: time.Now()}, nil
		},
		runScalarQueryFn: func(context.Context, string) (string, error) {
			return "0", nil
		},
		updateAuditRunResult: func(context.Context, int, string, *string, *string) (*model.AuditRun, error) {
			return &model.AuditRun{ID: 99, AuditID: 10, Status: "passed", StartedAt: time.Now()}, nil
		},
		createIssueFn: func(context.Context, int, int, string, string, string) (*model.Issue, error) {
			return nil, errors.New("should not create issue on pass")
		},
	}

	svc := New(repo)
	resp, err := svc.RunAudit(context.Background(), 10)
	if err != nil {
		t.Fatalf("RunAudit returned error: %v", err)
	}
	if resp.Run.Status != "passed" {
		t.Fatalf("expected status passed, got %s", resp.Run.Status)
	}
	if resp.Issue != nil {
		t.Fatalf("expected no issue on pass, got %+v", resp.Issue)
	}
	if repo.issueCreateCallCount != 0 {
		t.Fatalf("expected no issue creation, got %d", repo.issueCreateCallCount)
	}
}

func TestRunAudit_Failed_CreatesIssue(t *testing.T) {
	repo := &mockRepo{
		getAuditFn: func(context.Context, int) (*model.Audit, error) {
			return &model.Audit{
				ID:       11,
				Name:     "Memory Compliance",
				SQLQuery: "SELECT count(*) FROM vm_inventory WHERE memory > 50",
				ExpectedResult: model.ExpectedResult{
					Type:  "int",
					Value: "0",
				},
			}, nil
		},
		createAuditRunFn: func(context.Context, int) (*model.AuditRun, error) {
			return &model.AuditRun{ID: 100, AuditID: 11, Status: "running", StartedAt: time.Now()}, nil
		},
		runScalarQueryFn: func(context.Context, string) (string, error) {
			return "3", nil
		},
		updateAuditRunResult: func(context.Context, int, string, *string, *string) (*model.AuditRun, error) {
			return &model.AuditRun{ID: 100, AuditID: 11, Status: "failed", StartedAt: time.Now()}, nil
		},
		createIssueFn: func(context.Context, int, int, string, string, string) (*model.Issue, error) {
			return &model.Issue{ID: 7, AuditID: 11, AuditRunID: 100, ExpectedValue: "0", ActualValue: "3"}, nil
		},
	}

	svc := New(repo)
	resp, err := svc.RunAudit(context.Background(), 11)
	if err != nil {
		t.Fatalf("RunAudit returned error: %v", err)
	}
	if resp.Run.Status != "failed" {
		t.Fatalf("expected status failed, got %s", resp.Run.Status)
	}
	if resp.Issue == nil {
		t.Fatal("expected issue on failed run, got nil")
	}
	if repo.issueCreateCallCount != 1 {
		t.Fatalf("expected one issue creation, got %d", repo.issueCreateCallCount)
	}
	if repo.lastIssueExpected != "0" || repo.lastIssueActual != "3" {
		t.Fatalf("unexpected issue payload expected=%s actual=%s", repo.lastIssueExpected, repo.lastIssueActual)
	}
}

func TestRunAudit_QueryError_MarksRunError(t *testing.T) {
	repo := &mockRepo{
		getAuditFn: func(context.Context, int) (*model.Audit, error) {
			return &model.Audit{
				ID:       12,
				Name:     "Broken Query",
				SQLQuery: "SELECT bad",
				ExpectedResult: model.ExpectedResult{
					Type:  "int",
					Value: "0",
				},
			}, nil
		},
		createAuditRunFn: func(context.Context, int) (*model.AuditRun, error) {
			return &model.AuditRun{ID: 101, AuditID: 12, Status: "running", StartedAt: time.Now()}, nil
		},
		runScalarQueryFn: func(context.Context, string) (string, error) {
			return "", errors.New("pq: relation does not exist")
		},
		updateAuditRunResult: func(_ context.Context, _ int, status string, _ *string, errMsg *string) (*model.AuditRun, error) {
			return &model.AuditRun{ID: 101, AuditID: 12, Status: status, Error: errMsg, StartedAt: time.Now()}, nil
		},
		createIssueFn: func(context.Context, int, int, string, string, string) (*model.Issue, error) {
			return nil, errors.New("should not create issue on query error")
		},
	}

	svc := New(repo)
	resp, err := svc.RunAudit(context.Background(), 12)
	if err != nil {
		t.Fatalf("RunAudit returned error: %v", err)
	}
	if resp.Run.Status != "error" {
		t.Fatalf("expected status error, got %s", resp.Run.Status)
	}
	if resp.Run.Error == nil || *resp.Run.Error == "" {
		t.Fatal("expected error message to be recorded on run")
	}
	if repo.issueCreateCallCount != 0 {
		t.Fatalf("expected no issue creation on query error, got %d", repo.issueCreateCallCount)
	}
}
