package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/2comjie/taoxi-server/pkg/timex"
)

type Identity struct {
	ent.Schema
}

func (Identity) Fields() []ent.Field {
	return []ent.Field{
		field.Uint64("id").
			Immutable().
			Comment("身份记录自增主键"),
		field.Uint64("uid").
			Immutable().
			Comment("玩家UID"),
		field.Int32("login_type").
			Immutable().
			Comment("登录类型"),
		field.String("app_id").
			MaxLen(255).
			Immutable().
			Comment("第三方应用ID"),
		field.String("open_id").
			MaxLen(255).
			Immutable().
			Comment("第三方账号ID"),
		field.String("union_id").
			MaxLen(255).
			Optional().
			Nillable().
			Immutable().
			Comment("第三方跨应用账号ID"),
		field.Int64("create_at_unix").
			DefaultFunc(timex.NowUnix).
			Immutable().
			Comment("创建时间，Unix秒"),
	}
}

func (Identity) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("login_type", "app_id", "open_id").Unique(),
		index.Fields("uid", "login_type", "app_id").Unique(),
	}
}

func (Identity) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "identity"},
	}
}
