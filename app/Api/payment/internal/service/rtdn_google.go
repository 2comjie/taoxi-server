package paymentService

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	redisLock "github.com/2comjie/nova/lock/redis"
	"github.com/2comjie/nova/logx"
	"github.com/2comjie/nova/logx/logdef"
	paymentTypes "github.com/2comjie/taoxi-server/app/Api/payment/types"
	paymentRedisKey "github.com/2comjie/taoxi-server/internal/redis_key/payment"
	"google.golang.org/api/pubsub/v1"
)

// Google Play RTDN DeveloperNotification：
// https://developer.android.com/google/play/billing/rtdn-reference#developernotification
type googleDeveloperNotification struct {
	PackageName                string                            `json:"packageName"`
	OneTimeProductNotification *googleOneTimeProductNotification `json:"oneTimeProductNotification"`
	VoidedPurchaseNotification *googleVoidedPurchaseNotification `json:"voidedPurchaseNotification"`
	TestNotification           json.RawMessage                   `json:"testNotification"`
}

type googleOneTimeProductNotification struct {
	PurchaseToken string `json:"purchaseToken"`
}

type googleVoidedPurchaseNotification struct {
	PurchaseToken string `json:"purchaseToken"`
	ProductType   int32  `json:"productType"`
}

func CheckGoogleRTDN(ctx context.Context) error {
	registered, exists := GetChannel(paymentTypes.PaymentTypeGoogle)
	if !exists {
		return nil
	}
	channel := registered.(*GoogleChannel)
	if channel.subscription == "" {
		return nil
	}

	response, err := channel.pubsub.Projects.Subscriptions.Pull(
		channel.subscription,
		&pubsub.PullRequest{MaxMessages: 100, ReturnImmediately: true},
	).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("拉取Google Play RTDN失败: %w", err)
	}

	ackIDs := make([]string, 0, len(response.ReceivedMessages))
	for _, received := range response.ReceivedMessages {
		message := received.Message
		logCtx := logx.WithField("action", "处理Google Play RTDN").WithField("message_id", message.MessageId)

		data, err := base64.StdEncoding.DecodeString(message.Data)
		if err != nil {
			logCtx.Errorf("解码RTDN失败，丢弃消息 err=%v", err)
			ackIDs = append(ackIDs, received.AckId)
			continue
		}

		var notification googleDeveloperNotification
		if err = json.Unmarshal(data, &notification); err != nil {
			logCtx.Errorf("解析RTDN失败，丢弃消息 err=%v", err)
			ackIDs = append(ackIDs, received.AckId)
			continue
		}
		if err = handleGoogleRTDN(ctx, channel, logCtx, &notification); err != nil {
			logCtx.Errorf("处理RTDN失败，等待重新投递 err=%v", err)
			continue
		}
		ackIDs = append(ackIDs, received.AckId)
	}

	if len(ackIDs) == 0 {
		return nil
	}
	_, err = channel.pubsub.Projects.Subscriptions.Acknowledge(
		channel.subscription,
		&pubsub.AcknowledgeRequest{AckIds: ackIDs},
	).Context(ctx).Do()
	return err
}

func handleGoogleRTDN(ctx context.Context, channel *GoogleChannel, logCtx logdef.ILogger, notification *googleDeveloperNotification) error {
	if notification.PackageName != channel.packageName {
		logCtx.Errorf("RTDN包名不匹配 expected=%s actual=%s", channel.packageName, notification.PackageName)
		return nil
	}

	var purchaseToken string
	switch {
	case len(notification.TestNotification) > 0:
		logCtx.Infof("收到Google Play RTDN测试消息")
		return nil
	case notification.OneTimeProductNotification != nil:
		purchaseToken = notification.OneTimeProductNotification.PurchaseToken
	case notification.VoidedPurchaseNotification != nil:
		if notification.VoidedPurchaseNotification.ProductType != 2 {
			return nil
		}
		purchaseToken = notification.VoidedPurchaseNotification.PurchaseToken
	default:
		return nil
	}

	return processGoogleRTDNCredential(ctx, channel, purchaseToken)
}

func processGoogleRTDNCredential(ctx context.Context, channel *GoogleChannel, purchaseToken string) error {
	thirdPartyOrder, err := channel.QueryOrder(ctx, purchaseToken)
	if err != nil {
		return err
	}

	logCtx := logx.WithField("action", "处理Google Play RTDN订单").
		WithField("order_id", thirdPartyOrder.InternalOrderId).
		WithField("uid", thirdPartyOrder.Uid)
	lease, locked, err := redisLock.TryLockDefault(redisClient, ctx, paymentRedisKey.UserLock(thirdPartyOrder.Uid))
	if err != nil {
		return err
	}
	if !locked {
		return errors.New("玩家支付锁正被占用")
	}
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), unlockTimeout)
		defer cancel()
		if err := lease.Unlock(unlockCtx); err != nil {
			logCtx.Errorf("释放玩家支付锁失败 err=%v", err)
		}
	}()

	order, err := channel.FindOrBind(ctx, thirdPartyOrder.Uid, purchaseToken, thirdPartyOrder)
	if err != nil {
		return err
	}
	if thirdPartyOrder.ProductId != order.ThirdPartyProductID {
		return fmt.Errorf("Google Play商品不匹配 expected=%s actual=%s", order.ThirdPartyProductID, thirdPartyOrder.ProductId)
	}

	result, stdErr := ProcessOrderWithoutLock(logCtx, ctx, channel, order, thirdPartyOrder)
	if stdErr != nil {
		return errors.New(stdErr.Msg)
	}
	if result == paymentTypes.OrderProcessWaiting {
		return errors.New("Google Play订单仍未完成")
	}
	return nil
}
