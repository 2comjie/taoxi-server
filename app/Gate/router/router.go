package router

import (
	"github.com/2comjie/nova/app/gate"
)

func Init() *gate.Router {
	root := gate.NewRouter()
	if err := root.RegisterActorKeyResolver("player", func(ctx *gate.Context) string {
		return ctx.Uid
	}); err != nil {
		panic(err)
	}
	return root
}
