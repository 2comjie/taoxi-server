package service

import (
	"context"
	"fmt"

	mockTypes "github.com/2comjie/taoxi-server/app/Api/mock/types"
	"github.com/2comjie/taoxi-server/pkg/stderr"
	"github.com/2comjie/wali/deploy"
	"github.com/2comjie/wali/logx"
	"github.com/spf13/cast"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type messageBinding struct {
	request  protoreflect.MessageType
	response protoreflect.MessageType
}

var messageBindings = make(map[uint32]messageBinding)

func Register(route uint32, request proto.Message, response proto.Message) {
	if route == 0 {
		panic("mock: route必须大于0")
	}
	if _, exists := messageBindings[route]; exists {
		panic(fmt.Sprintf("mock: route已经注册 route=%d", route))
	}
	binding := messageBinding{request: request.ProtoReflect().Type()}
	if response != nil {
		binding.response = response.ProtoReflect().Type()
	}
	messageBindings[route] = binding
}

func GateMock(ctx context.Context, app *deploy.NodeApp, req *mockTypes.GateMockReq) (*mockTypes.GateMockRsp, *stderr.Error) {
	binding, exists := messageBindings[req.Route]
	if !exists {
		return nil, stderr.BadRequest("route协议未注册")
	}

	requestMessage := binding.request.New().Interface()
	if err := protojson.Unmarshal(req.JsonBody, requestMessage); err != nil {
		return nil, stderr.BadRequest("请求JSON与protobuf协议不匹配")
	}
	body, err := proto.Marshal(requestMessage)
	if err != nil {
		logx.Errorf("mock: 编码protobuf请求失败 uid=%d route=%d err=%v", req.Uid, req.Route, err)
		return nil, stderr.InternalServerError("编码请求失败")
	}

	responseBody, replied, err := app.MockGateCall(ctx, cast.ToString(req.Uid), req.Route, body)
	if err != nil {
		logx.Errorf("mock: Gate请求失败 uid=%d route=%d err=%v", req.Uid, req.Route, err)
		return nil, stderr.InternalServerError("Gate请求失败")
	}
	if !replied || binding.response == nil {
		return &mockTypes.GateMockRsp{}, nil
	}

	responseMessage := binding.response.New().Interface()
	if err = proto.Unmarshal(responseBody, responseMessage); err != nil {
		logx.Errorf("mock: 解码protobuf响应失败 uid=%d route=%d err=%v", req.Uid, req.Route, err)
		return nil, stderr.InternalServerError("解码响应失败")
	}
	jsonRsp, err := protojson.Marshal(responseMessage)
	if err != nil {
		logx.Errorf("mock: 编码JSON响应失败 uid=%d route=%d err=%v", req.Uid, req.Route, err)
		return nil, stderr.InternalServerError("编码响应失败")
	}
	return &mockTypes.GateMockRsp{JsonRsp: jsonRsp}, nil
}
