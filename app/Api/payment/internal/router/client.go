package router

import (
	paymentService "github.com/2comjie/taoxi-server/app/Api/payment/internal/service"
	paymentTypes "github.com/2comjie/taoxi-server/app/Api/payment/types"
	"github.com/2comjie/taoxi-server/internal/deploy/external"
	paymentRedisKey "github.com/2comjie/taoxi-server/internal/redis_key/payment"
	midef "github.com/2comjie/taoxi-server/pkg/middleware/def"
	"github.com/2comjie/taoxi-server/pkg/middleware/inout"
	"github.com/2comjie/taoxi-server/pkg/middleware/limit"
	"github.com/2comjie/taoxi-server/pkg/modules"
	"github.com/2comjie/taoxi-server/pkg/stderr"
	"github.com/2comjie/wali/logx"
)

func Init(args modules.Modules) {
	paymentGroup := args.ClientGroup.Group("payment")

	paymentGroup.POST("create_order",
		limit.Limiter("create_order", 1, 1),
		limit.LockUser(paymentRedisKey.UserLock, external.RedisPayment()),
		inout.UidHandler[paymentTypes.CreateOrderReq, paymentTypes.CreateOrderRsp](handleCreateOrder))

	paymentGroup.POST("upload_receipt_req",
		limit.LockUser(paymentRedisKey.UserLock, external.RedisPayment()),
		inout.UidHandler[paymentTypes.UploadReceiptReq, paymentTypes.UploadReceiptRsp](handleUploadReceipt))
}

func handleCreateOrder(ctx *midef.Header, req *paymentTypes.CreateOrderReq) (*paymentTypes.CreateOrderRsp, *stderr.Error) {
	logCtx := logx.WithField("action", "创建订单").WithField("uid", ctx.Uid).WithField("req", req)
	rsp, err := paymentService.CreateOrderWithoutLock(logCtx, ctx.Context(), ctx.Uid, req)
	return rsp, err
}

func handleUploadReceipt(ctx *midef.Header, req *paymentTypes.UploadReceiptReq) (*paymentTypes.UploadReceiptRsp, *stderr.Error) {
	logCtx := logx.WithField("action", "上传支付凭证").WithField("uid", ctx.Uid).WithField("req", req)
	rsp, err := paymentService.UploadReceiptWithoutLock(logCtx, ctx.Context(), ctx.Uid, req)
	return rsp, err
}
