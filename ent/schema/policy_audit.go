package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// PolicyAudit holds the schema definition for the PolicyAudit entity.
type PolicyAudit struct {
	ent.Schema
}

// Annotations overrides the DB table name so it maps to "policies" in Postgres
func (PolicyAudit) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "policies"},
	}
}

// Fields of the PolicyAudit.
func (PolicyAudit) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").NotEmpty(),
		field.JSON("audit_ids", []int{}),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Edges of the PolicyAudit.
func (PolicyAudit) Edges() []ent.Edge {
	return nil
}
