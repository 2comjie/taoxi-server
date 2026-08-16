package message_router

import (
	"reflect"
	"sync"

	"github.com/2comjie/nova/actor"
	"github.com/2comjie/nova/actor/actorDef"
	"github.com/2comjie/nova/app/node"
	"google.golang.org/protobuf/proto"
)

func Reg[Req proto.Message, Rsp proto.Message](router *node.Router, route uint32, handler func(ctx *node.Context, req Req, rsp Rsp) error) {
	reqPool := newMessagePool[Req]()
	rspPool := newMessagePool[Rsp]()
	err := router.Handle(route, func(ctx *node.Context) error {
		return handle(ctx, reqPool, rspPool, func(req Req, rsp Rsp) error {
			return handler(ctx, req, rsp)
		})
	})
	if err != nil {
		panic(err)
	}
}

func RegActor[T actorDef.Actor, Req proto.Message, Rsp proto.Message](router *actor.RouteGroup[T], route uint32, handler func(actorValue T, pid actorDef.PID, ctx *node.Context, req Req, rsp Rsp) error) {
	reqPool := newMessagePool[Req]()
	rspPool := newMessagePool[Rsp]()
	router.Handle(route, func(actorValue T, pid actorDef.PID, ctx *node.Context) error {
		return handle(ctx, reqPool, rspPool, func(req Req, rsp Rsp) error {
			return handler(actorValue, pid, ctx, req, rsp)
		})
	})
}

func handle[Req proto.Message, Rsp proto.Message](ctx *node.Context, reqPool, rspPool *sync.Pool, handler func(req Req, rsp Rsp) error) error {
	req := reqPool.Get().(Req)
	defer release(reqPool, req)
	if err := proto.Unmarshal(ctx.Request.Body, req); err != nil {
		return err
	}

	rsp := rspPool.Get().(Rsp)
	defer release(rspPool, rsp)
	if err := handler(req, rsp); err != nil {
		return err
	}
	if !ctx.NeedReply() {
		return nil
	}

	body, err := proto.Marshal(rsp)
	if err != nil {
		return err
	}
	return ctx.Reply(body)
}

func newMessagePool[M proto.Message]() *sync.Pool {
	messageType := reflect.TypeFor[M]().Elem()
	return &sync.Pool{New: func() any {
		return reflect.New(messageType).Interface().(M)
	}}
}

func release[M proto.Message](pool *sync.Pool, message M) {
	proto.Reset(message)
	pool.Put(message)
}
