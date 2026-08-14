package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kit/kit/log"
	"security-central/internal/service"
	auditpb "security-central/proto"
)

// Server implements the audit execution gRPC worker service.
type Server struct {
	auditpb.UnimplementedAuditExecutorServer
	svc    service.Service
	logger log.Logger
}

func NewServer(svc service.Service, logger log.Logger) *Server {
	return &Server{svc: svc, logger: logger}
}

func (s *Server) ExecuteAudit(ctx context.Context, req *auditpb.ExecuteAuditRequest) (*auditpb.ExecuteAuditResponse, error) {
	auditID := int(req.GetAuditId())
	runID := int(req.GetRunId())
	if auditID <= 0 || runID <= 0 {
		return &auditpb.ExecuteAuditResponse{
			Accepted: false,
			Message:  "audit_id and run_id must be positive",
		}, nil
	}

	go func() {
		// Detached worker execution keeps RPC quick and non-blocking for API calls.
		execCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		if _, err := s.svc.ExecuteAuditRun(execCtx, auditID, runID); err != nil {
			_ = s.logger.Log("component", "grpc-worker", "audit_id", auditID, "run_id", runID, "err", err.Error())
		}
	}()

	return &auditpb.ExecuteAuditResponse{
		Accepted: true,
		Message:  fmt.Sprintf("audit run %d accepted", runID),
	}, nil
}
