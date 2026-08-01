package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type Player struct {
	ent.Schema
}

func (Player) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			MaxLen(32).
			Immutable(),
		field.Int8("status").
			Default(1),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}
