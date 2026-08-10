package router

import (
	"github.com/2comjie/taoxi-server/app/Api/items/internal/service"
	itemTypes "github.com/2comjie/taoxi-server/app/Api/items/types"
	midef "github.com/2comjie/taoxi-server/pkg/middleware/def"
	"github.com/2comjie/taoxi-server/pkg/middleware/inout"
	"github.com/2comjie/taoxi-server/pkg/modules"
	"github.com/2comjie/taoxi-server/pkg/stderr"
)

func Init(args modules.Modules) {
	itemGroup := args.ClientGroup.Group("items")
	itemGroup.POST("list", inout.UidHandler[itemTypes.ListItemReq, itemTypes.ListItemRsp](handleListItem))
}

func handleListItem(ctx *midef.Header, req *itemTypes.ListItemReq) (*itemTypes.ListItemRsp, *stderr.Error) {
	itemList, err := service.GetItemWithCache(ctx.Context(), ctx.Uid)
	if err != nil {
		return nil, err
	}
	return &itemTypes.ListItemRsp{List: itemList}, nil
}
