package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Issue holds the schema definition for the Issue entity.
type Issue struct {
	ent.Schema
}

// Fields of the Issue.
func (Issue) Fields() []ent.Field {
	return []ent.Field{
		field.Int("audit_id"),
		field.Int("audit_run_id"),
		field.String("query_name").Optional(),
		field.String("expected_value").NotEmpty(),
		field.String("actual_value").NotEmpty(),
		field.String("description").NotEmpty(),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Edges of the Issue.
func (Issue) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("audit", Audit.Type).Ref("issues").Field("audit_id").Required().Unique(),
		edge.From("audit_run", AuditRun.Type).Ref("issues").Field("audit_run_id").Required().Unique(),
	}
}
