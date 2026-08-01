package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type PlayerIdentity struct {
	ent.Schema
}

func (PlayerIdentity) Fields() []ent.Field {
	return []ent.Field{
		field.String("uid").
			MaxLen(32).
			Immutable(),
		field.Int32("login_type").
			Immutable(),
		field.String("app_id").
			MaxLen(255).
			Immutable(),
		field.String("open_id").
			MaxLen(255).
			Immutable(),
		field.String("union_id").
			MaxLen(255).
			Optional().
			Immutable(),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}

func (PlayerIdentity) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("login_type", "app_id", "open_id").Unique(),
		index.Fields("uid", "login_type", "app_id").Unique(),
	}
}
