package schema

import (
	"time"

	"entgo.io/ent"
	entsql "entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// VMInventory holds the schema definition for the VM inventory entity.
type VMInventory struct {
	ent.Schema
}

// Fields of the VMInventory.
func (VMInventory) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").NotEmpty(),
		field.Int("cpu").NonNegative(),
		field.Int("memory").NonNegative(),
		field.String("power_state").Default("on"),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Annotations of the VMInventory.
func (VMInventory) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "vm_inventory"},
	}
}
