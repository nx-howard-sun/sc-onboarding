package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// AuditRun holds the schema definition for the AuditRun entity.
type AuditRun struct {
	ent.Schema
}

// Fields of the AuditRun.
func (AuditRun) Fields() []ent.Field {
	return []ent.Field{
		field.Int("audit_id"),
		field.String("status").NotEmpty(),
		field.String("actual_value").Optional().Nillable(),
		field.String("error_message").Optional().Nillable(),
		field.Time("started_at").Default(time.Now),
		field.Time("completed_at").Optional().Nillable(),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Edges of the AuditRun.
func (AuditRun) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("audit", Audit.Type).Ref("runs").Field("audit_id").Required().Unique(),
		edge.To("issues", Issue.Type),
	}
}
