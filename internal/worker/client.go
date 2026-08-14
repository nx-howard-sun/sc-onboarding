package worker

import (
	"context"
	"fmt"
	"time"

	auditpb "security-central/proto"
)

// Client dispatches audit execution requests to the gRPC worker.
type Client struct {
	grpcClient auditpb.AuditExecutorClient
	timeout    time.Duration
}

func NewClient(grpcClient auditpb.AuditExecutorClient) *Client {
	return &Client{
		grpcClient: grpcClient,
		timeout:    5 * time.Second,
	}
}

func (c *Client) ExecuteAudit(ctx context.Context, auditID, runID int) error {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.grpcClient.ExecuteAudit(callCtx, &auditpb.ExecuteAuditRequest{
		AuditId: int32(auditID),
		RunId:   int32(runID),
	})
	if err != nil {
		return err
	}
	if !resp.GetAccepted() {
		return fmt.Errorf("worker rejected audit execution: %s", resp.GetMessage())
	}
	return nil
}
