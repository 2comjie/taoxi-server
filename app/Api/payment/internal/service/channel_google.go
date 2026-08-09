package paymentService

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	paymentStore "github.com/2comjie/taoxi-server/app/Api/payment/internal/store"
	paymentent "github.com/2comjie/taoxi-server/app/Api/payment/internal/store/ent"
	paymentTypes "github.com/2comjie/taoxi-server/app/Api/payment/types"
	googleplayIAP "github.com/2comjie/taoxi-server/pkg/googleplay/iap"
	"github.com/2comjie/wali/etc"
	"github.com/2comjie/wali/logx"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/androidpublisher/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/pubsub/v1"
)

const (
	googlePlayPackageNameEnv     = "GOOGLE_PLAY_PACKAGE_NAME"
	googlePlayCredentialsJSONEnv = "GOOGLE_PLAY_CREDENTIALS_JSON"
	googlePlayPubSubEnv          = "GOOGLE_PLAY_PUBSUB_SUBSCRIPTION"
)

type GoogleChannel struct {
	packageName  string
	service      *androidpublisher.Service
	pubsub       *pubsub.Service
	subscription string
	payloadKey   []byte
}

// InitGoogleChannel 初始化 Google Play 支付渠道。
// 配置：GOOGLE_PLAY_PACKAGE_NAME、GOOGLE_PLAY_CREDENTIALS_JSON、GOOGLE_PLAY_PUBSUB_SUBSCRIPTION。
// Google API：https://developers.google.com/android-publisher/getting_started
// RTDN：https://developer.android.com/google/play/billing/getting-ready#configure-rtdn
func InitGoogleChannel(ctx context.Context) error {
	packageName := etc.String(googlePlayPackageNameEnv)
	if packageName == "" {
		logx.Infof("未配置%s，Google Play支付渠道未启用", googlePlayPackageNameEnv)
		return nil
	}

	config, err := google.JWTConfigFromJSON(
		[]byte(etc.String(googlePlayCredentialsJSONEnv)),
		androidpublisher.AndroidpublisherScope,
		pubsub.PubsubScope,
	)
	if err != nil {
		return fmt.Errorf("payment: 解析Google服务账号失败: %w", err)
	}
	service, err := androidpublisher.NewService(ctx, option.WithTokenSource(config.TokenSource(ctx)))
	if err != nil {
		return fmt.Errorf("payment: 初始化Google Android Publisher服务失败: %w", err)
	}
	pubsubService, err := pubsub.NewService(ctx, option.WithTokenSource(config.TokenSource(ctx)))
	if err != nil {
		return fmt.Errorf("payment: 初始化Google Pub/Sub服务失败: %w", err)
	}
	payloadKey := sha256.Sum256(config.PrivateKey)

	RegisterChannel(&GoogleChannel{
		packageName:  packageName,
		service:      service,
		pubsub:       pubsubService,
		subscription: etc.String(googlePlayPubSubEnv),
		payloadKey:   payloadKey[:16],
	})
	logx.Infof("Google Play支付渠道初始化成功 package_name=%s", packageName)
	return nil
}

func (g *GoogleChannel) Type() paymentTypes.PaymentType {
	return paymentTypes.PaymentTypeGoogle
}

func (g *GoogleChannel) BuildCreateOrderExtra(_ context.Context, params *paymentTypes.BuildCreateOrderParams) (*paymentTypes.CreateOrderExtra, error) {
	accountId, err := googleplayIAP.EncryptUid(params.Uid, g.payloadKey)
	if err != nil {
		return nil, fmt.Errorf("生成Google ObfuscatedAccountID失败: %w", err)
	}

	profileID, err := googleplayIAP.EncryptProfile(&googleplayIAP.Profile{
		OrderID:   params.OrderId,
		ProductID: params.ProductId,
	}, g.payloadKey)
	if err != nil {
		return nil, fmt.Errorf("生成Google ObfuscatedProfileID失败: %w", err)
	}
	return &paymentTypes.CreateOrderExtra{
		ObfuscatedAccountID: accountId,
		ObfuscatedProfileID: profileID,
	}, nil
}

func (g *GoogleChannel) ParseCallback(_ context.Context, _ *paymentTypes.CallbackRequest) (*paymentTypes.CallbackEvent, error) {
	return nil, errors.New("Google Play RTDN使用Pub/Sub Pull，不走支付HTTP回调")
}

func (g *GoogleChannel) QueryOrder(ctx context.Context, credential string) (*paymentTypes.ThirdPartyOrder, error) {
	purchase, err := g.service.Purchases.Productsv2.Getproductpurchasev2(g.packageName, credential).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("查询Google Play购买凭证失败: %w", err)
	}
	if len(purchase.ProductLineItem) != 1 {
		return nil, fmt.Errorf("当前只支持单商品购买，Google商品行数量=%d", len(purchase.ProductLineItem))
	}

	lineItem := purchase.ProductLineItem[0]
	if lineItem.ProductOfferDetails != nil && lineItem.ProductOfferDetails.Quantity > 1 {
		return nil, fmt.Errorf("当前不支持一次购买多个商品，quantity=%d", lineItem.ProductOfferDetails.Quantity)
	}

	uid, err := googleplayIAP.DecryptUid(purchase.ObfuscatedExternalAccountId, g.payloadKey)
	if err != nil {
		return nil, fmt.Errorf("校验Google ObfuscatedAccountID失败: %w", err)
	}
	profile, err := googleplayIAP.DecryptProfile(purchase.ObfuscatedExternalProfileId, g.payloadKey)
	if err != nil {
		return nil, fmt.Errorf("校验Google ObfuscatedProfileID失败: %w", err)
	}

	info := &paymentTypes.ThirdPartyOrder{
		InternalOrderId:   profile.OrderID,
		InternalProductId: profile.ProductID,
		Uid:               uid,
		OrderId:           purchase.OrderId,
		ProductId:         lineItem.ProductId,
		Credential:        credential,
		IsSandbox:         purchase.TestPurchaseContext != nil,
	}
	info.Consumed = googlePurchaseConsumed(purchase)

	// PurchaseState: Output only. The purchase state of the purchase.
	//
	// Possible values:
	//   "PURCHASE_STATE_UNSPECIFIED" - Purchase state unspecified. This value
	// should never be set.
	//   "PURCHASED" - Purchased successfully.
	//   "CANCELLED" - Purchase canceled.
	//   "PENDING" - The purchase is in a pending state and has not yet been
	// completed. For more information on handling pending purchases, see
	// https://developer.android.com/google/play/billing/integrate#pending.
	switch purchase.PurchaseStateContext.PurchaseState {
	case "PENDING":
		info.Status = paymentTypes.ThirdPartyOrderStatusPending
	case "PURCHASED":
		info.Status = paymentTypes.ThirdPartyOrderStatusPurchased
	case "CANCELLED":
		info.Status = paymentTypes.ThirdPartyOrderStatusCancelled
	default:
		return nil, fmt.Errorf("未知Google Play购买状态: %s", purchase.PurchaseStateContext.PurchaseState)
	}

	info.PayAtUnix = googleTimeUnix(purchase.PurchaseCompletionTime)

	if purchase.OrderId != "" {
		order, err := g.service.Orders.Get(g.packageName, purchase.OrderId).Context(ctx).Do()
		if err != nil {
			return nil, fmt.Errorf("查询Google Play订单详情失败 order_id=%s: %w", purchase.OrderId, err)
		}
		if order.Total != nil {
			info.AmountUnit = order.Total.Units
			info.AmountNanos = int32(order.Total.Nanos)
			info.Currency = order.Total.CurrencyCode
		}
		if history := order.OrderHistory; history != nil {
			if history.RefundEvent != nil {
				info.RefundReason = history.RefundEvent.RefundReason
				info.RefundAtUnix = googleTimeUnix(history.RefundEvent.EventTime)
			}
			if info.RefundAtUnix == 0 && history.CancellationEvent != nil {
				info.RefundAtUnix = googleTimeUnix(history.CancellationEvent.EventTime)
			}
		}
	}

	return info, nil
}

func (g *GoogleChannel) Complete(ctx context.Context, order *paymentTypes.ThirdPartyOrder) error {
	if order.Consumed {
		return nil
	}

	err := g.service.Purchases.Products.Consume(g.packageName, order.ProductId, order.Credential).Context(ctx).Do()
	if err == nil {
		return nil
	}

	// consume成功后如果进程在返回前中断，重试会收到错误；重新查询消费状态保证幂等
	purchase, queryErr := g.service.Purchases.Productsv2.Getproductpurchasev2(g.packageName, order.Credential).Context(ctx).Do()
	if queryErr == nil && googlePurchaseConsumed(purchase) {
		return nil
	}
	return fmt.Errorf("消费Google Play商品失败: %w", err)
}

func (g *GoogleChannel) FindOrBind(ctx context.Context, uid uint64, credential string, info *paymentTypes.ThirdPartyOrder) (*paymentent.PaymentOrder, error) {
	if info.Uid != uid {
		return nil, fmt.Errorf("Google Play订单用户不匹配 expected=%d actual=%d", uid, info.Uid)
	}
	return paymentStore.FindAndBindOrder(
		ctx,
		uid,
		paymentTypes.PaymentTypeGoogle,
		info.InternalOrderId,
		info.InternalProductId,
		credential,
		info.OrderId,
	)
}

func googlePurchaseConsumed(purchase *androidpublisher.ProductPurchaseV2) bool {
	return len(purchase.ProductLineItem) == 1 &&
		purchase.ProductLineItem[0].ProductOfferDetails != nil &&
		purchase.ProductLineItem[0].ProductOfferDetails.ConsumptionState == "CONSUMPTION_STATE_CONSUMED"
}

func googleTimeUnix(value string) int64 {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return 0
	}
	return parsed.Unix()
}
