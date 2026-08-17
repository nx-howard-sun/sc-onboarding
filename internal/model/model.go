package model

import "time"

type ExpectedResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type QueryRule struct {
	Name           string         `json:"name"`
	SQLQuery       string         `json:"sql_query"`
	ExpectedResult ExpectedResult `json:"expected_result"`
}

type Audit struct {
	ID        int         `json:"id"`
	Name      string      `json:"name"`
	SQLQuery  []QueryRule `json:"queries"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

type AuditRun struct {
	ID          int        `json:"id"`
	AuditID     int        `json:"audit_id"`
	Status      string     `json:"status"`
	ActualValue *string    `json:"actual_value,omitempty"`
	Error       *string    `json:"error,omitempty"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

type Issue struct {
	ID            int       `json:"id"`
	AuditID       int       `json:"audit_id"`
	AuditRunID    int       `json:"audit_run_id"`
	QueryName     string    `json:"query_name,omitempty"`
	ExpectedValue string    `json:"expected_value"`
	ActualValue   string    `json:"actual_value"`
	Description   string    `json:"description"`
	CreatedAt     time.Time `json:"created_at"`
}

// Policy represents a collection of grouped audits
type Policy struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	AuditIDs  []int     `json:"audit_ids"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PolicyRun tracks the overall execution status of a policy
type PolicyRun struct {
	ID          int        `json:"id"`
	PolicyID    int        `json:"policy_id"`
	Status      string     `json:"status"` // running, passed, failed, error
	AuditRunIDs []int      `json:"audit_run_ids,omitempty"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// PolicyRunDetail combines a PolicyRun with the full status of all its child audit runs
type PolicyRunDetail struct {
	PolicyRun
	AuditRuns []AuditRun `json:"audit_runs"`
}
