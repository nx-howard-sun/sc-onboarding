package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-kit/kit/log"
	"security-central/internal/endpoint"
	"security-central/internal/model"
	"security-central/internal/service"
)

type fakeService struct{}

func (f *fakeService) CreateAudit(_ context.Context, req service.CreateAuditRequest) (*model.Audit, error) {
	return &model.Audit{
		ID:       1,
		Name:     req.Name,
		SQLQuery: req.SQLQuery,
		ExpectedResult: model.ExpectedResult{
			Type:  req.ExpectedResult.Type,
			Value: req.ExpectedResult.Value,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func (f *fakeService) GetAudit(_ context.Context, id int) (*model.Audit, error) {
	return &model.Audit{
		ID:       id,
		Name:     "Memory Compliance",
		SQLQuery: "SELECT count(*) FROM vm_inventory WHERE memory > 50",
		ExpectedResult: model.ExpectedResult{
			Type:  "int",
			Value: "0",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func (f *fakeService) RunAudit(_ context.Context, auditID int) (*service.RunAuditResponse, error) {
	return &service.RunAuditResponse{
		Run: &model.AuditRun{
			ID:        10,
			AuditID:   auditID,
			Status:    "passed",
			StartedAt: time.Now(),
		},
	}, nil
}

func (f *fakeService) GetRunStatus(_ context.Context, auditID, runID int) (*model.AuditRun, error) {
	return &model.AuditRun{
		ID:        runID,
		AuditID:   auditID,
		Status:    "failed",
		StartedAt: time.Now(),
	}, nil
}

func (f *fakeService) ListIssues(_ context.Context, _, _ int) ([]model.Issue, error) {
	return []model.Issue{
		{
			ID:            1,
			AuditID:       2,
			AuditRunID:    3,
			ExpectedValue: "0",
			ActualValue:   "1",
			Description:   "Mismatch",
			CreatedAt:     time.Now(),
		},
	}, nil
}

func (f *fakeService) GetIssue(_ context.Context, id int) (*model.Issue, error) {
	return &model.Issue{
		ID:            id,
		AuditID:       2,
		AuditRunID:    3,
		ExpectedValue: "0",
		ActualValue:   "1",
		Description:   "Mismatch",
		CreatedAt:     time.Now(),
	}, nil
}

func TestPOSTAudits_Returns200AndJSON(t *testing.T) {
	svc := &fakeService{}
	eps := endpoint.New(svc)
	h := NewHTTPHandler(eps, log.NewNopLogger())

	body := map[string]any{
		"name":      "Memory Compliance",
		"sql_query": "SELECT count(*) FROM vm_inventory WHERE memory > 50",
		"expected_result": map[string]any{
			"type":  "int",
			"value": "0",
		},
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/audits", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("expected application/json content type, got %s", rec.Header().Get("Content-Type"))
	}
}

func TestGETAudit_InvalidPathParam_Returns400(t *testing.T) {
	svc := &fakeService{}
	eps := endpoint.New(svc)
	h := NewHTTPHandler(eps, log.NewNopLogger())

	req := httptest.NewRequest(http.MethodGet, "/audits/not-an-int", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestGETIssuesList_InvalidPage_Returns400(t *testing.T) {
	svc := &fakeService{}
	eps := endpoint.New(svc)
	h := NewHTTPHandler(eps, log.NewNopLogger())

	req := httptest.NewRequest(http.MethodGet, "/issues/list?page=abc", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestGETRunStatus_ReturnsRunPayload(t *testing.T) {
	svc := &fakeService{}
	eps := endpoint.New(svc)
	h := NewHTTPHandler(eps, log.NewNopLogger())

	req := httptest.NewRequest(http.MethodGet, "/audits/11/run/99/status", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if out["status"] != "failed" {
		t.Fatalf("expected run status failed, got %v", out["status"])
	}
}
