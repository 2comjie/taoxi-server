package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	itemTypes "github.com/2comjie/taoxi-server/app/Api/items/types"
	"github.com/2comjie/taoxi-server/pkg/timex"
)

type ItemNonce struct {
	ent.Schema
}

func (ItemNonce) Fields() []ent.Field {
	return []ent.Field{
		field.Uint64("id").
			Immutable().
			Comment("幂等记录自增ID"),

		field.Uint64("uid").
			Immutable().
			Comment("玩家UID"),

		field.String("nonce").
			MaxLen(128).
			Immutable().
			Comment("业务幂等标识"),

		field.Int32("op_type").
			GoType(itemTypes.OperationType(0)).
			Immutable().
			Comment("操作类型"),

		field.Int64("create_at_unix").
			DefaultFunc(timex.NowUnix).
			Immutable().
			Comment("创建时间，Unix秒"),
	}
}

func (ItemNonce) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("uid", "nonce", "op_type").Unique(),
	}
}
