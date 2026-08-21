package router

import (
	"strconv"

	"github.com/2comjie/nova/app/gate"
)

func Init() *gate.Router {
	root := gate.NewRouter()
	if err := root.RegisterActorKeyResolver("player", func(ctx *gate.Context) string {
		return strconv.FormatUint(ctx.Uid, 10)
	}); err != nil {
		panic(err)
	}
	return root
}
