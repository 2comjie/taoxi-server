package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/2comjie/taoxi-server/pkg/timex"
)

type Item struct {
	ent.Schema
}

func (Item) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").
			Immutable().
			Comment("道具记录自增ID"),

		field.Uint64("uid").
			Immutable().
			Comment("玩家UID"),

		field.Int32("item_id").
			Immutable().
			Comment("道具配置ID"),

		field.Int64("count").
			Min(0).
			Comment("道具总数量"),

		field.Int64("used_count").
			Default(0).
			Min(0).
			Comment("已使用数量"),

		field.Int64("expire_at_unix").
			Default(0).
			Comment("过期时间，Unix秒，0表示永不过期"),

		field.String("source").
			Default("").
			MaxLen(255).
			Immutable().
			Comment("道具来源"),

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

func (Item) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("expire_at_unix"),

		// 查询玩家指定道具的永久记录和各个有效期批次
		// 不设置Unique，同一来源可以多次发放相同过期时间的道具
		index.Fields("uid", "item_id", "expire_at_unix"),
	}
}
