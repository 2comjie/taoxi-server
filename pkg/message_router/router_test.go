package message_router

import (
	"errors"
	"testing"

	"github.com/2comjie/nova/app/node"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestReg(t *testing.T) {
	router := node.NewRouter()
	calls := 0
	Reg(router, 1, func(ctx *node.Context, req *wrapperspb.StringValue, rsp *wrapperspb.StringValue) error {
		calls++
		if calls == 1 {
			rsp.Value = "hello " + req.Value
		}
		return nil
	})
	if err := router.Freeze(); err != nil {
		t.Fatal(err)
	}

	firstBody, err := proto.Marshal(wrapperspb.String("taoxi"))
	if err != nil {
		t.Fatal(err)
	}
	firstCtx := &node.Context{Request: &node.Request{Route: 1, Body: firstBody, NeedReply: true}}
	if err = router.Dispatch(firstCtx); err != nil {
		t.Fatal(err)
	}
	firstRsp := &wrapperspb.StringValue{}
	if err = proto.Unmarshal(firstCtx.ResponseBody(), firstRsp); err != nil {
		t.Fatal(err)
	}
	if firstRsp.Value != "hello taoxi" {
		t.Fatalf("response=%q", firstRsp.Value)
	}

	secondCtx := &node.Context{Request: &node.Request{Route: 1, NeedReply: true}}
	if err = router.Dispatch(secondCtx); err != nil {
		t.Fatal(err)
	}
	secondRsp := &wrapperspb.StringValue{}
	if err = proto.Unmarshal(secondCtx.ResponseBody(), secondRsp); err != nil {
		t.Fatal(err)
	}
	if secondRsp.Value != "" {
		t.Fatalf("pooled response was not reset: %q", secondRsp.Value)
	}
}

func TestRegTellDoesNotReply(t *testing.T) {
	router := node.NewRouter()
	Reg(router, 1, func(ctx *node.Context, req *wrapperspb.StringValue, rsp *wrapperspb.StringValue) error {
		rsp.Value = req.Value
		return nil
	})
	if err := router.Freeze(); err != nil {
		t.Fatal(err)
	}

	body, err := proto.Marshal(wrapperspb.String("taoxi"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := &node.Context{Request: &node.Request{Route: 1, Body: body}}
	if err = router.Dispatch(ctx); err != nil {
		t.Fatal(err)
	}
	if ctx.ResponseBody() != nil {
		t.Fatal("Tell request must not have a response")
	}
}

func TestRegReturnsDecodeAndHandlerErrors(t *testing.T) {
	handlerErr := errors.New("handler failed")
	router := node.NewRouter()
	called := false
	Reg(router, 1, func(ctx *node.Context, req *wrapperspb.StringValue, rsp *wrapperspb.StringValue) error {
		called = true
		return handlerErr
	})
	if err := router.Freeze(); err != nil {
		t.Fatal(err)
	}

	decodeErr := router.Dispatch(&node.Context{Request: &node.Request{Route: 1, Body: []byte{0xff}, NeedReply: true}})
	if decodeErr == nil {
		t.Fatal("expected protobuf decode error")
	}
	if called {
		t.Fatal("handler ran after protobuf decode failed")
	}

	err := router.Dispatch(&node.Context{Request: &node.Request{Route: 1, NeedReply: true}})
	if !errors.Is(err, handlerErr) {
		t.Fatalf("dispatch error=%v", err)
	}
	if !called {
		t.Fatal("handler did not run")
	}
}
