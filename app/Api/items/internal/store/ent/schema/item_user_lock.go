package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type ItemUserLock struct {
	ent.Schema
}

func (ItemUserLock) Fields() []ent.Field {
	return []ent.Field{
		field.Uint64("id").
			StorageKey("uid").
			Immutable().
			Comment("玩家UID"),
	}
}
