package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// PolicyRun holds the schema definition for the PolicyRun entity.
type PolicyRun struct {
	ent.Schema
}

// Fields of the PolicyRun.
func (PolicyRun) Fields() []ent.Field {
	return []ent.Field{
		field.Int("policy_id"),
		field.String("status").Default("running"),
		field.JSON("audit_run_ids", []int{}).Optional(),
		field.Time("started_at").Default(time.Now),
		field.Time("completed_at").Optional().Nillable(),
	}
}

// Edges of the PolicyRun.
func (PolicyRun) Edges() []ent.Edge {
	return nil
}
