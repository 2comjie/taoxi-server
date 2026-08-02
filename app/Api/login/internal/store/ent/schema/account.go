package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"

	"github.com/2comjie/taoxi-server/pkg/timex"
)

type Account struct {
	ent.Schema
}

func (Account) Fields() []ent.Field {
	return []ent.Field{
		field.Uint64("id").
			Immutable().
			Comment("玩家UID，账号表主键"),
		field.Bool("is_deleted").
			Default(false).
			Comment("账号是否已删除"),
		field.Int64("create_at_unix").
			DefaultFunc(timex.NowUnix).
			Immutable().
			Comment("创建时间，Unix秒"),
		field.Int64("update_at_unix").
			DefaultFunc(timex.NowUnix).
			UpdateDefault(timex.NowUnix).
			Comment("更新时间，Unix秒"),
	}
}

func (Account) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "account"},
		entsql.IncrementStart(10_000_000),
	}
}
