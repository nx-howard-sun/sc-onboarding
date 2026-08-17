package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"security-central/ent"
	"security-central/ent/audit"
	"security-central/ent/auditrun"
	"security-central/ent/issue"
	"security-central/ent/policyaudit"
	"security-central/ent/policyrun"
	"security-central/ent/schedule"
	"security-central/ent/user"
	"security-central/internal/model"
)

type Repository interface {
	// Audit & Execution
	CreateAudit(ctx context.Context, name string, queries []model.QueryRule) (*model.Audit, error)
	GetAudit(ctx context.Context, id int) (*model.Audit, error)
	CreateAuditRun(ctx context.Context, auditID int) (*model.AuditRun, error)
	UpdateAuditRunResult(ctx context.Context, runID int, status string, actualValue *string, errMsg *string) (*model.AuditRun, error)
	GetAuditRun(ctx context.Context, auditID, runID int) (*model.AuditRun, error)
	GetAuditRunsByIDs(ctx context.Context, ids []int) ([]model.AuditRun, error)

	// Issues
	CreateIssue(ctx context.Context, auditID, runID int, queryName, expectedValue, actualValue, description string) (*model.Issue, error)
	ListIssues(ctx context.Context, page, pageSize int) ([]model.Issue, error)
	GetIssue(ctx context.Context, id int) (*model.Issue, error)

	// Policy Management
	CreatePolicy(ctx context.Context, name string, auditIDs []int) (*model.Policy, error)
	GetPolicy(ctx context.Context, id int) (*model.Policy, error)
	CreatePolicyRun(ctx context.Context, policyID int, auditRunIDs []int) (*model.PolicyRun, error)
	GetPolicyRun(ctx context.Context, runID int) (*model.PolicyRun, error)
	UpdatePolicyRunStatus(ctx context.Context, runID int, status string) (*model.PolicyRun, error)

	// User Management
	CreateUser(ctx context.Context, username, base64Password, role string) (*model.User, error)
	GetUserByUsername(ctx context.Context, username string) (*model.User, error)

	// Schedule Management
	CreateSchedule(ctx context.Context, targetType string, targetID, intervalSeconds int, nextRunAt time.Time) (*model.Schedule, error)
	GetDueSchedules(ctx context.Context, now time.Time) ([]model.Schedule, error)
	UpdateScheduleNextRun(ctx context.Context, id int, nextRunAt time.Time) error

	// SQL Runner
	RunScalarQuery(ctx context.Context, query string) (string, error)
}

type EntRepository struct {
	client *ent.Client
	sqlDB  *sql.DB
}

func NewEntRepository(client *ent.Client, sqlDB *sql.DB) *EntRepository {
	return &EntRepository{client: client, sqlDB: sqlDB}
}

func (r *EntRepository) CreateAudit(ctx context.Context, name string, queries []model.QueryRule) (*model.Audit, error) {
	row, err := r.client.Audit.Create().
		SetName(name).
		SetQueries(queries).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return toAudit(row), nil
}

func (r *EntRepository) GetAudit(ctx context.Context, id int) (*model.Audit, error) {
	row, err := r.client.Audit.Query().Where(audit.IDEQ(id)).Only(ctx)
	if err != nil {
		return nil, err
	}
	return toAudit(row), nil
}

func (r *EntRepository) CreateAuditRun(ctx context.Context, auditID int) (*model.AuditRun, error) {
	row, err := r.client.AuditRun.Create().
		SetAuditID(auditID).
		SetStatus("running").
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return toAuditRun(row), nil
}

func (r *EntRepository) UpdateAuditRunResult(ctx context.Context, runID int, status string, actualValue *string, errMsg *string) (*model.AuditRun, error) {
	upd := r.client.AuditRun.UpdateOneID(runID).SetStatus(status).SetCompletedAt(time.Now())
	if actualValue != nil {
		upd = upd.SetActualValue(*actualValue)
	}
	if errMsg != nil {
		upd = upd.SetErrorMessage(*errMsg)
	}
	row, err := upd.Save(ctx)
	if err != nil {
		return nil, err
	}
	return toAuditRun(row), nil
}

func (r *EntRepository) GetAuditRun(ctx context.Context, auditID, runID int) (*model.AuditRun, error) {
	row, err := r.client.AuditRun.Query().
		Where(auditrun.IDEQ(runID), auditrun.AuditIDEQ(auditID)).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return toAuditRun(row), nil
}

func (r *EntRepository) GetAuditRunsByIDs(ctx context.Context, ids []int) ([]model.AuditRun, error) {
	rows, err := r.client.AuditRun.Query().
		Where(auditrun.IDIn(ids...)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]model.AuditRun, 0, len(rows))
	for _, row := range rows {
		out = append(out, *toAuditRun(row))
	}
	return out, nil
}

func (r *EntRepository) CreateIssue(ctx context.Context, auditID, runID int, queryName, expectedValue, actualValue, description string) (*model.Issue, error) {
	row, err := r.client.Issue.Create().
		SetAuditID(auditID).
		SetAuditRunID(runID).
		SetQueryName(queryName).
		SetExpectedValue(expectedValue).
		SetActualValue(actualValue).
		SetDescription(description).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return toIssue(row), nil
}

func (r *EntRepository) ListIssues(ctx context.Context, page, pageSize int) ([]model.Issue, error) {
	offset := (page - 1) * pageSize
	rows, err := r.client.Issue.Query().
		Order(ent.Desc(issue.FieldCreatedAt)).
		Offset(offset).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]model.Issue, 0, len(rows))
	for _, i := range rows {
		out = append(out, *toIssue(i))
	}
	return out, nil
}

func (r *EntRepository) GetIssue(ctx context.Context, id int) (*model.Issue, error) {
	row, err := r.client.Issue.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return toIssue(row), nil
}

func (r *EntRepository) RunScalarQuery(ctx context.Context, query string) (string, error) {
	row := r.sqlDB.QueryRowContext(ctx, query)
	var value interface{}
	if err := row.Scan(&value); err != nil {
		return "", err
	}
	if value == nil {
		return "null", nil
	}
	return fmt.Sprint(value), nil
}

// ==========================================
// POLICIES
// ==========================================

func (r *EntRepository) CreatePolicy(ctx context.Context, name string, auditIDs []int) (*model.Policy, error) {
	row, err := r.client.PolicyAudit.Create().
		SetName(name).
		SetAuditIds(auditIDs).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return toPolicy(row), nil
}

func (r *EntRepository) GetPolicy(ctx context.Context, id int) (*model.Policy, error) {
	row, err := r.client.PolicyAudit.Query().
		Where(policyaudit.IDEQ(id)).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return toPolicy(row), nil
}

func (r *EntRepository) CreatePolicyRun(ctx context.Context, policyID int, auditRunIDs []int) (*model.PolicyRun, error) {
	row, err := r.client.PolicyRun.Create().
		SetPolicyID(policyID).
		SetAuditRunIds(auditRunIDs).
		SetStatus("running").
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return toPolicyRun(row), nil
}

func (r *EntRepository) GetPolicyRun(ctx context.Context, runID int) (*model.PolicyRun, error) {
	row, err := r.client.PolicyRun.Query().
		Where(policyrun.IDEQ(runID)).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return toPolicyRun(row), nil
}

func (r *EntRepository) UpdatePolicyRunStatus(ctx context.Context, runID int, status string) (*model.PolicyRun, error) {
	now := time.Now()
	row, err := r.client.PolicyRun.UpdateOneID(runID).
		SetStatus(status).
		SetCompletedAt(now).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return toPolicyRun(row), nil
}

// ==========================================
// USER MANAGEMENT
// ==========================================
func (r *EntRepository) CreateUser(ctx context.Context, username, base64Password, role string) (*model.User, error) {
	row, err := r.client.User.Create().
		SetUsername(username).
		SetPassword(base64Password).
		SetRole(role).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return &model.User{ID: row.ID, Username: row.Username, Password: row.Password, Role: row.Role}, nil
}

func (r *EntRepository) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	row, err := r.client.User.Query().
		Where(user.UsernameEQ(username)).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return &model.User{ID: row.ID, Username: row.Username, Password: row.Password, Role: row.Role}, nil
}

// ==========================================
// SCHEDULE MANAGEMENT
// ==========================================

func (r *EntRepository) CreateSchedule(ctx context.Context, targetType string, targetID, intervalSeconds int, nextRunAt time.Time) (*model.Schedule, error) {
	row, err := r.client.Schedule.Create().
		SetTargetType(targetType).
		SetTargetID(targetID).
		SetIntervalSeconds(intervalSeconds).
		SetNextRunAt(nextRunAt).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return &model.Schedule{
		ID:              row.ID,
		TargetType:      row.TargetType,
		TargetID:        row.TargetID,
		IntervalSeconds: row.IntervalSeconds,
		NextRunAt:       row.NextRunAt,
		CreatedAt:       row.CreatedAt,
	}, nil
}

func (r *EntRepository) GetDueSchedules(ctx context.Context, now time.Time) ([]model.Schedule, error) {
	rows, err := r.client.Schedule.Query().
		Where(schedule.NextRunAtLTE(now)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	schedules := make([]model.Schedule, len(rows))
	for i, row := range rows {
		schedules[i] = model.Schedule{
			ID:              row.ID,
			TargetType:      row.TargetType,
			TargetID:        row.TargetID,
			IntervalSeconds: row.IntervalSeconds,
			NextRunAt:       row.NextRunAt,
			CreatedAt:       row.CreatedAt,
		}
	}
	return schedules, nil
}

func (r *EntRepository) UpdateScheduleNextRun(ctx context.Context, id int, nextRunAt time.Time) error {
	return r.client.Schedule.UpdateOneID(id).
		SetNextRunAt(nextRunAt).
		Exec(ctx)
}

func toAudit(a *ent.Audit) *model.Audit {
	return &model.Audit{
		ID:        a.ID,
		Name:      a.Name,
		SQLQuery:  a.Queries,
		CreatedAt: a.CreatedAt,
		UpdatedAt: a.UpdatedAt,
	}
}

func toAuditRun(r *ent.AuditRun) *model.AuditRun {
	return &model.AuditRun{
		ID:          r.ID,
		AuditID:     r.AuditID,
		Status:      r.Status,
		ActualValue: r.ActualValue,
		Error:       r.ErrorMessage,
		StartedAt:   r.StartedAt,
		CompletedAt: r.CompletedAt,
	}
}

func toIssue(i *ent.Issue) *model.Issue {
	return &model.Issue{
		ID:            i.ID,
		AuditID:       i.AuditID,
		AuditRunID:    i.AuditRunID,
		QueryName:     i.QueryName,
		ExpectedValue: i.ExpectedValue,
		ActualValue:   i.ActualValue,
		Description:   i.Description,
		CreatedAt:     i.CreatedAt,
	}
}

func toPolicy(p *ent.PolicyAudit) *model.Policy {
	return &model.Policy{
		ID:        p.ID,
		Name:      p.Name,
		AuditIDs:  p.AuditIds,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}

func toPolicyRun(pr *ent.PolicyRun) *model.PolicyRun {
	return &model.PolicyRun{
		ID:          pr.ID,
		PolicyID:    pr.PolicyID,
		Status:      pr.Status,
		AuditRunIDs: pr.AuditRunIds,
		StartedAt:   pr.StartedAt,
		CompletedAt: pr.CompletedAt,
	}
}
