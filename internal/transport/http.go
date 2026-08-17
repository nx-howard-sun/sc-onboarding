package transport

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"security-central/internal/endpoint"
	"security-central/internal/service"

	"github.com/go-kit/kit/log"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
)

type errorResponse struct {
	Error string `json:"error"`
}

func NewHTTPHandler(e endpoint.Endpoints, logger log.Logger) http.Handler {
	_ = logger
	r := mux.NewRouter()

	options := []httptransport.ServerOption{
		httptransport.ServerErrorEncoder(encodeError),
	}

	// ==========================================
	// PUBLIC ROUTE
	// ==========================================
	r.Methods(http.MethodPost).Path("/login").Handler(httptransport.NewServer(
		e.Login,
		decodeLoginRequest,
		encodeResponse,
		options...,
	))

	// ==========================================
	// PROTECTED ROUTES (Requires Bearer JWT)
	// ==========================================
	protected := r.PathPrefix("/").Subrouter()
	protected.Use(authAndRBACMiddleware)

	// Audits
	protected.Methods(http.MethodPost).Path("/audits").Handler(httptransport.NewServer(
		e.CreateAudit,
		decodeCreateAuditRequest,
		encodeResponse,
		options...,
	))
	protected.Methods(http.MethodGet).Path("/audits/{id}").Handler(httptransport.NewServer(
		e.GetAudit,
		decodeGetAuditRequest,
		encodeResponse,
		options...,
	))
	protected.Methods(http.MethodPost).Path("/audits/{id}/run").Handler(httptransport.NewServer(
		e.RunAudit,
		decodeRunAuditRequest,
		encodeResponse,
		options...,
	))
	protected.Methods(http.MethodGet).Path("/audits/{id}/run/{run_id}/status").Handler(httptransport.NewServer(
		e.GetRunStatus,
		decodeGetRunStatusRequest,
		encodeResponse,
		options...,
	))

	// Issues
	protected.Methods(http.MethodGet).Path("/issues/list").Handler(httptransport.NewServer(
		e.ListIssues,
		decodeListIssuesRequest,
		encodeResponse,
		options...,
	))
	protected.Methods(http.MethodGet).Path("/issues/{id}").Handler(httptransport.NewServer(
		e.GetIssue,
		decodeGetIssueRequest,
		encodeResponse,
		options...,
	))

	// Policies
	protected.Methods(http.MethodPost).Path("/policies").Handler(httptransport.NewServer(
		e.CreatePolicy,
		decodeCreatePolicyRequest,
		encodeResponse,
		options...,
	))
	protected.Methods(http.MethodGet).Path("/policies/{id}").Handler(httptransport.NewServer(
		e.GetPolicy,
		decodeGetPolicyRequest,
		encodeResponse,
		options...,
	))
	protected.Methods(http.MethodPost).Path("/policies/{id}/run").Handler(httptransport.NewServer(
		e.RunPolicy,
		decodeRunPolicyRequest,
		encodeResponse,
		options...,
	))
	protected.Methods(http.MethodGet).Path("/policies/{id}/run/{run_id}/status").Handler(httptransport.NewServer(
		e.GetPolicyRunStatus,
		decodeGetPolicyRunStatusRequest,
		encodeResponse,
		options...,
	))

	return r
}

// ==========================================
// AUTH MIDDLEWARE
// ==========================================

func authAndRBACMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(errorResponse{Error: "missing or invalid authorization header"})
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims := &service.JWTClaims{}

		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			return service.JWTSecret, nil
		})

		if err != nil || !token.Valid {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(errorResponse{Error: "invalid or expired token"})
			return
		}

		// Enforce RBAC: Non-admin users cannot trigger mutation/execution requests (POST)
		if r.Method == http.MethodPost && claims.Role != "admin" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(errorResponse{Error: "forbidden: viewer role is restricted to read-only access"})
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ==========================================
// AUTH DECODERS
// ==========================================

func decodeLoginRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req endpoint.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return req, nil
}

// ==========================================
// AUDIT DECODERS
// ==========================================

func decodeCreateAuditRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req endpoint.CreateAuditRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return req, nil
}

func decodeGetAuditRequest(_ context.Context, r *http.Request) (interface{}, error) {
	id, err := getPathInt(r, "id")
	if err != nil {
		return nil, err
	}
	return endpoint.GetAuditRequest{ID: id}, nil
}

func decodeRunAuditRequest(_ context.Context, r *http.Request) (interface{}, error) {
	id, err := getPathInt(r, "id")
	if err != nil {
		return nil, err
	}
	return endpoint.RunAuditRequest{AuditID: id}, nil
}

func decodeGetRunStatusRequest(_ context.Context, r *http.Request) (interface{}, error) {
	auditID, err := getPathInt(r, "id")
	if err != nil {
		return nil, err
	}
	runID, err := getPathInt(r, "run_id")
	if err != nil {
		return nil, err
	}
	return endpoint.GetRunStatusRequest{AuditID: auditID, RunID: runID}, nil
}

// ==========================================
// ISSUE DECODERS
// ==========================================

func decodeListIssuesRequest(_ context.Context, r *http.Request) (interface{}, error) {
	page := 1
	pageSize := 20
	if raw := r.URL.Query().Get("page"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			return nil, errors.New("invalid page")
		}
		page = v
	}
	if raw := r.URL.Query().Get("page_size"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			return nil, errors.New("invalid page_size")
		}
		pageSize = v
	}
	return endpoint.ListIssuesRequest{Page: page, PageSize: pageSize}, nil
}

func decodeGetIssueRequest(_ context.Context, r *http.Request) (interface{}, error) {
	id, err := getPathInt(r, "id")
	if err != nil {
		return nil, err
	}
	return endpoint.GetIssueRequest{ID: id}, nil
}

// ==========================================
// POLICY DECODERS
// ==========================================

func decodeCreatePolicyRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req endpoint.CreatePolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return req, nil
}

func decodeGetPolicyRequest(_ context.Context, r *http.Request) (interface{}, error) {
	id, err := getPathInt(r, "id")
	if err != nil {
		return nil, err
	}
	return endpoint.GetPolicyRequest{ID: id}, nil
}

func decodeRunPolicyRequest(_ context.Context, r *http.Request) (interface{}, error) {
	id, err := getPathInt(r, "id")
	if err != nil {
		return nil, err
	}
	return endpoint.RunPolicyRequest{PolicyID: id}, nil
}

func decodeGetPolicyRunStatusRequest(_ context.Context, r *http.Request) (interface{}, error) {
	policyID, err := getPathInt(r, "id")
	if err != nil {
		return nil, err
	}
	runID, err := getPathInt(r, "run_id")
	if err != nil {
		return nil, err
	}
	return endpoint.GetPolicyRunStatusRequest{PolicyID: policyID, RunID: runID}, nil
}

// ==========================================
// ENCODERS & HELPERS
// ==========================================

func encodeResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(response)
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	code := http.StatusBadRequest
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(errorResponse{Error: err.Error()})
}

func getPathInt(r *http.Request, name string) (int, error) {
	vars := mux.Vars(r)
	raw := vars[name]
	if raw == "" {
		return 0, errors.New("missing path parameter: " + name)
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, errors.New("invalid path parameter: " + name)
	}
	return v, nil
}
