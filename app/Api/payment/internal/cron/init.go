package cron

import (
	"context"
	"time"

	"github.com/2comjie/nova/core/help"
	"github.com/2comjie/nova/logx"
	paymentService "github.com/2comjie/taoxi-server/app/Api/payment/internal/service"
	robfigCron "github.com/robfig/cron/v3"
)

func Init(systemCron *robfigCron.Cron) {
	if systemCron == nil {
		panic("payment cron: Cron不能为空")
	}
	_, err := systemCron.AddFunc("@every 10s", checkExpiredPendingOrders)
	if err != nil {
		panic("payment cron: 注册超时订单任务失败: " + err.Error())
	}
	_, err = systemCron.AddFunc("@every 30s", checkGoogleRTDN)
	if err != nil {
		panic("payment cron: 注册Google RTDN任务失败: " + err.Error())
	}
}

func checkGoogleRTDN() {
	help.SafeRun(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()

		if err := paymentService.CheckGoogleRTDN(ctx); err != nil {
			logx.Errorf("payment cron: 处理Google RTDN失败 err=%v", err)
		}
	})
}

func checkExpiredPendingOrders() {
	help.SafeRun(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if stdErr := paymentService.CheckExpiredPendingOrders(ctx); stdErr != nil {
			logx.Errorf("payment cron: 处理超时订单失败 code=%d msg=%s", stdErr.Code, stdErr.Msg)
		}
	})
}
