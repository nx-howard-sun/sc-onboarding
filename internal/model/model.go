package model

import "time"

type ExpectedResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type Audit struct {
	ID             int            `json:"id"`
	Name           string         `json:"name"`
	SQLQuery       string         `json:"sql_query"`
	ExpectedResult ExpectedResult `json:"expected_result"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
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
	ExpectedValue string    `json:"expected_value"`
	ActualValue   string    `json:"actual_value"`
	Description   string    `json:"description"`
	CreatedAt     time.Time `json:"created_at"`
}
