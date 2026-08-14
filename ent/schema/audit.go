package schema

import (
	"security-central/internal/model"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Audit holds the schema definition for the Audit entity.
type Audit struct {
	ent.Schema
}

// Fields of the Audit.
func (Audit) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").NotEmpty(),
		field.JSON("queries", []model.QueryRule{}).
			Comment("List of SQL query rules executed during this audit"),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Edges of the Audit.
func (Audit) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("runs", AuditRun.Type),
		edge.To("issues", Issue.Type),
	}
}
