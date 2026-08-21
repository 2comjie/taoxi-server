package push

import (
	"context"

	"github.com/2comjie/nova/deploy"
)

var app *deploy.NodeApp

func Init(nodeApp *deploy.NodeApp) {
	app = nodeApp
}

func Push(ctx context.Context, uid uint64, route uint32, body []byte) error {
	return app.Push(ctx, uid, route, body)
}

func Broadcast(ctx context.Context, route uint32, body []byte) (uint32, error) {
	return app.Broadcast(ctx, route, body)
}
