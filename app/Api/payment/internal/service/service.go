package paymentService

import (
	"context"
	"encoding/json"

	"github.com/2comjie/taoxi-server/pkg/asynqx"

	paymentStore "github.com/2comjie/taoxi-server/app/Api/payment/internal/store"
	paymentent "github.com/2comjie/taoxi-server/app/Api/payment/internal/store/ent"
	paymentTypes "github.com/2comjie/taoxi-server/app/Api/payment/types"
	paymentConfig "github.com/2comjie/taoxi-server/internal/config/payment"
	"github.com/2comjie/taoxi-server/pkg/stderr"
	"github.com/2comjie/wali/logx/logdef"
	"github.com/redis/go-redis/v9"
)

var redisClient redis.UniversalClient

const (
	defaultCurrency = "CNY"
)

func Init(client redis.UniversalClient, server *asynqx.Server) {
	if client == nil {
		panic("payment: Redis客户端不能为空")
	}
	redisClient = client
	RegisterTasks(server)
	if err := InitGoogleChannel(context.Background()); err != nil {
		panic(err)
	}
}

// CreateOrderWithoutLock 不获取玩家支付锁，调用方必须已经持有该玩家的支付锁
func CreateOrderWithoutLock(logCtx logdef.ILogger, ctx context.Context, uid uint64, req *paymentTypes.CreateOrderReq) (*paymentTypes.CreateOrderRsp, *stderr.Error) {
	channel, exists := GetChannel(req.PaymentType)
	if !exists {
		logCtx.Infof("不支持的支付渠道")
		return nil, stderr.BadRequest("不支持的支付渠道")
	}

	product := paymentConfig.GetProduct(req.ProductId)
	if product == nil {
		logCtx.Infof("购买的商品不存在")
		return nil, stderr.BadRequest("商品不存在")
	}

	thirdPartyProductId := product.ThirdPartyProductId[int32(req.PaymentType)]
	if thirdPartyProductId == "" {
		logCtx.Warnf("商品未配置当前支付渠道")
		return nil, stderr.BadRequest("商品不支持当前支付渠道")
	}
	price := product.PriceMap[defaultCurrency]
	if price == nil {
		logCtx.Errorf("商品未配置价格")
		return nil, stderr.BadRequest("商品未配置价格")
	}

	// 创建或者复用已有的订单
	order, stdErr := findOrCreateOrderWithoutLock(logCtx, ctx, uid, req, product, thirdPartyProductId, price)
	if stdErr != nil {
		return nil, stdErr
	}

	// 构建客户端参数
	extra, err := channel.BuildCreateOrderExtra(ctx, &paymentTypes.BuildCreateOrderParams{
		OrderId:             order.ID,
		Uid:                 uid,
		ProductId:           order.ProductID,
		ThirdPartyProductId: order.ThirdPartyProductID,
	})
	if err != nil {
		logCtx.Errorf("构建订单渠道参数失败 err %v", err)
		return nil, stderr.InternalServerError("创建订单失败")
	}

	return &paymentTypes.CreateOrderRsp{
		OrderId:             order.ID,
		ProductId:           order.ProductID,
		ThirdPartyProductId: order.ThirdPartyProductID,
		Extra:               extra,
	}, nil
}

// findOrCreateOrderWithoutLock 只查询或写入订单，不获取分布式锁
func findOrCreateOrderWithoutLock(logCtx logdef.ILogger, ctx context.Context, uid uint64, req *paymentTypes.CreateOrderReq, product *paymentConfig.Product, thirdPartyProductId string, price *paymentConfig.Price) (*paymentent.PaymentOrder, *stderr.Error) {
	order, found, err := paymentStore.FindPendingOrder(ctx, uid, req.PaymentType, req.ProductId)
	if err != nil {
		logCtx.Errorf("查询未完成订单失败 uid=%d product_id=%d err=%v", uid, req.ProductId, err)
		return nil, stderr.InternalServerError("创建订单失败")
	}
	if found {
		// 使用之前存在的 但是没有完成的订单
		return order, nil
	}

	var rewards json.RawMessage
	if product.Rewards != nil {
		rewards, err = json.Marshal(product.Rewards)
		if err != nil {
			logCtx.Errorf("序列化商品奖励失败 product_id=%d err=%v", req.ProductId, err)
			return nil, stderr.InternalServerError("商品配置错误")
		}
	}

	// 创建新的订单
	order, err = paymentStore.CreateOrder(ctx, &paymentTypes.CreateOrderParams{
		Uid:                 uid,
		ProductId:           req.ProductId,
		ThirdPartyProductId: thirdPartyProductId,
		PaymentType:         req.PaymentType,
		AmountUnit:          price.AmountUnit,
		AmountNanos:         price.AmountNanos,
		Currency:            defaultCurrency,
		Rewards:             rewards,
	})
	if err != nil {
		logCtx.Errorf("创建订单失败 uid=%d product_id=%d err=%v", uid, req.ProductId, err)
		return nil, stderr.InternalServerError("创建订单失败")
	}
	return order, nil
}

func UploadReceiptWithoutLock(logCtx logdef.ILogger, ctx context.Context, uid uint64, req *paymentTypes.UploadReceiptReq) (*paymentTypes.UploadReceiptRsp, *stderr.Error) {
	channel, found := GetChannel(req.PaymentType)
	if !found {
		logCtx.Infof("不支持的支付渠道")
		return nil, stderr.BadRequest("不支持的支付渠道")
	}
	// 根据第三方凭证 查询第三方订单信息
	thirdPartyOrder, err := channel.QueryOrder(ctx, req.Credential)
	if err != nil {
		logCtx.Errorf("查询内部订单失败 %v", err)
		return nil, stderr.InternalServerError("查询内部订单失败")
	}
	if thirdPartyOrder == nil {
		logCtx.Infof("未找到三方订单")
		return nil, stderr.BadRequest("未找到内部订单")
	}

	// 根据第三方信息查找或绑定内部Pending订单
	order, err := channel.FindOrBind(ctx, uid, req.Credential, thirdPartyOrder)
	if err != nil {
		logCtx.Errorf("查找或绑定支付订单失败 %v", err)
		return nil, stderr.InternalServerError("查找或绑定支付订单失败")
	}
	if order == nil {
		logCtx.Infof("没有找到对应的支付订单")
		return nil, stderr.BadRequest("没有找到对应的支付订单")
	}

	// 校验第三方商品
	logCtx = logCtx.WithFields(logdef.Fields{
		"order_id":             order.ID,
		"product_id":           order.ProductID,
		"third_party_order_id": thirdPartyOrder.OrderId,
		"internal_status":      order.Status,
		"third_party_status":   thirdPartyOrder.Status,
	})
	if thirdPartyOrder.ProductId == "" || thirdPartyOrder.ProductId != order.ThirdPartyProductID {
		logCtx.Errorf("第三方商品不匹配 expected=%s actual=%s", order.ThirdPartyProductID, thirdPartyOrder.ProductId)
		return nil, stderr.BadRequest("支付商品不匹配")
	}

	// 根据第三方订单的状态处理
	result, stdErr := ProcessOrderWithoutLock(logCtx, ctx, channel, order, thirdPartyOrder)
	if stdErr != nil {
		return nil, stdErr
	}
	if result == paymentTypes.OrderProcessWaiting {
		return nil, stderr.BadRequest("支付尚未完成")
	}
	return &paymentTypes.UploadReceiptRsp{}, nil
}
