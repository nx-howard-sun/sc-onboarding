package endpoint

import (
	"context"

	"security-central/internal/model"
	"security-central/internal/service"

	"github.com/go-kit/kit/endpoint"
)

type Endpoints struct {
	CreateAudit        endpoint.Endpoint
	GetAudit           endpoint.Endpoint
	RunAudit           endpoint.Endpoint
	GetRunStatus       endpoint.Endpoint
	ListIssues         endpoint.Endpoint
	GetIssue           endpoint.Endpoint
	CreatePolicy       endpoint.Endpoint
	GetPolicy          endpoint.Endpoint
	RunPolicy          endpoint.Endpoint
	GetPolicyRunStatus endpoint.Endpoint
}

type CreateAuditRequest struct {
	Name    string            `json:"name"`
	Queries []model.QueryRule `json:"queries"`
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

type CreatePolicyRequest struct {
	Name     string `json:"name"`
	AuditIDs []int  `json:"audit_ids"`
}

type GetPolicyRequest struct {
	ID int
}

type RunPolicyRequest struct {
	PolicyID int
}

type GetPolicyRunStatusRequest struct {
	PolicyID int
	RunID    int
}

func New(svc service.Service) Endpoints {
	return Endpoints{
		CreateAudit:        makeCreateAuditEndpoint(svc),
		GetAudit:           makeGetAuditEndpoint(svc),
		RunAudit:           makeRunAuditEndpoint(svc),
		GetRunStatus:       makeGetRunStatusEndpoint(svc),
		ListIssues:         makeListIssuesEndpoint(svc),
		GetIssue:           makeGetIssueEndpoint(svc),
		CreatePolicy:       makeCreatePolicyEndpoint(svc),
		GetPolicy:          makeGetPolicyEndpoint(svc),
		RunPolicy:          makeRunPolicyEndpoint(svc),
		GetPolicyRunStatus: makeGetPolicyRunStatusEndpoint(svc),
	}
}

func makeCreateAuditEndpoint(svc service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(CreateAuditRequest)
		return svc.CreateAudit(ctx, service.CreateAuditRequest{
			Name:    req.Name,
			Queries: req.Queries,
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

func makeCreatePolicyEndpoint(svc service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(CreatePolicyRequest)
		return svc.CreatePolicy(ctx, service.CreatePolicyRequest{
			Name:     req.Name,
			AuditIDs: req.AuditIDs,
		})
	}
}

func makeGetPolicyEndpoint(svc service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GetPolicyRequest)
		return svc.GetPolicy(ctx, req.ID)
	}
}

func makeRunPolicyEndpoint(svc service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(RunPolicyRequest)
		return svc.RunPolicy(ctx, req.PolicyID)
	}
}

func makeGetPolicyRunStatusEndpoint(svc service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GetPolicyRunStatusRequest)
		return svc.GetPolicyRunStatus(ctx, req.PolicyID, req.RunID)
	}
}
