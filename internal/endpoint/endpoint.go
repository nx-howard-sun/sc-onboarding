package endpoint

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"security-central/internal/model"
	"security-central/internal/service"
)

type Endpoints struct {
	CreateAudit  endpoint.Endpoint
	GetAudit     endpoint.Endpoint
	RunAudit     endpoint.Endpoint
	GetRunStatus endpoint.Endpoint
	ListIssues   endpoint.Endpoint
	GetIssue     endpoint.Endpoint
}

type CreateAuditRequest struct {
	Name           string               `json:"name"`
	SQLQuery       string               `json:"sql_query"`
	ExpectedResult model.ExpectedResult `json:"expected_result"`
}

type GetAuditRequest struct {
	ID int
}

type RunAuditRequest struct {
	AuditID int
}

type GetRunStatusRequest struct {
	AuditID int
	RunID   int
}

type ListIssuesRequest struct {
	Page     int
	PageSize int
}

type GetIssueRequest struct {
	ID int
}

func New(svc service.Service) Endpoints {
	return Endpoints{
		CreateAudit:  makeCreateAuditEndpoint(svc),
		GetAudit:     makeGetAuditEndpoint(svc),
		RunAudit:     makeRunAuditEndpoint(svc),
		GetRunStatus: makeGetRunStatusEndpoint(svc),
		ListIssues:   makeListIssuesEndpoint(svc),
		GetIssue:     makeGetIssueEndpoint(svc),
	}
}

func makeCreateAuditEndpoint(svc service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(CreateAuditRequest)
		return svc.CreateAudit(ctx, service.CreateAuditRequest{
			Name:           req.Name,
			SQLQuery:       req.SQLQuery,
			ExpectedResult: req.ExpectedResult,
		})
	}
}

func makeGetAuditEndpoint(svc service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GetAuditRequest)
		return svc.GetAudit(ctx, req.ID)
	}
}

func makeRunAuditEndpoint(svc service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(RunAuditRequest)
		return svc.RunAudit(ctx, req.AuditID)
	}
}

func makeGetRunStatusEndpoint(svc service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GetRunStatusRequest)
		return svc.GetRunStatus(ctx, req.AuditID, req.RunID)
	}
}

func makeListIssuesEndpoint(svc service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ListIssuesRequest)
		return svc.ListIssues(ctx, req.Page, req.PageSize)
	}
}

func makeGetIssueEndpoint(svc service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GetIssueRequest)
		return svc.GetIssue(ctx, req.ID)
	}
}
