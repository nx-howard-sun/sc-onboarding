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
	"security-central/internal/model"
)

type Repository interface {
	CreateAudit(ctx context.Context, name string, queries []model.QueryRule) (*model.Audit, error)
	GetAudit(ctx context.Context, id int) (*model.Audit, error)
	CreateAuditRun(ctx context.Context, auditID int) (*model.AuditRun, error)
	UpdateAuditRunResult(ctx context.Context, runID int, status string, actualValue *string, errMsg *string) (*model.AuditRun, error)
	GetAuditRun(ctx context.Context, auditID, runID int) (*model.AuditRun, error)
	CreateIssue(ctx context.Context, auditID, runID int, queryName, expectedValue, actualValue, description string) (*model.Issue, error)
	ListIssues(ctx context.Context, page, pageSize int) ([]model.Issue, error)
	GetIssue(ctx context.Context, id int) (*model.Issue, error)
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
