package mock

import (
	"github.com/2comjie/nova/deploy"
	"github.com/2comjie/taoxi-server/app/Api/mock/internal/router"
	"github.com/2comjie/taoxi-server/app/Api/mock/internal/service"
	pbPlayer "github.com/2comjie/taoxi-server/pb/player"
	"github.com/2comjie/taoxi-server/pkg/modules"
	"google.golang.org/protobuf/proto"
)

func Init(args modules.Modules, app *deploy.NodeApp) {
	Register(uint32(pbPlayer.ReqType_Hi), &pbPlayer.HiReq{}, &pbPlayer.HiRsp{})
	Register(uint32(pbPlayer.ReqType_Offload), &pbPlayer.OffloadReq{}, &pbPlayer.OffloadRsp{})

	router.Init(args, app)
}

func Register(route uint32, request proto.Message, response proto.Message) {
	service.Register(route, request, response)
}
