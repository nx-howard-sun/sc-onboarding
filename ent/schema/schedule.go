package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Schedule holds the schema definition for scheduled tasks.
type Schedule struct {
	ent.Schema
}

func (Schedule) Fields() []ent.Field {
	return []ent.Field{
		field.String("target_type"),               // "audit" or "policy"
		field.Int("target_id"),                    // Audit ID or Policy ID
		field.Int("interval_seconds").Default(60), // Interval in seconds
		field.Time("next_run_at"),
		field.Time("created_at").Default(time.Now),
	}
}
