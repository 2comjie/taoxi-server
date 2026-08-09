package cron

import (
	"context"
	"time"

	paymentService "github.com/2comjie/taoxi-server/app/Api/payment/internal/service"
	"github.com/2comjie/wali/core/help"
	"github.com/2comjie/wali/logx"
	robfigCron "github.com/robfig/cron/v3"
)

const (
	cronProcessTimeout = 30 * time.Second
)

func Init(systemCron *robfigCron.Cron) {
	if systemCron == nil {
		panic("payment cron: Cron不能为空")
	}
	_, err := systemCron.AddFunc("@every 10s", checkExpiredPendingOrders)
	if err != nil {
		panic("payment cron: 注册超时订单任务失败: " + err.Error())
	}
}

func checkExpiredPendingOrders() {
	help.SafeRun(func() {
		ctx, cancel := context.WithTimeout(context.Background(), cronProcessTimeout)
		defer cancel()

		if stdErr := paymentService.CheckExpiredPendingOrders(ctx); stdErr != nil {
			logx.Errorf("payment cron: 处理超时订单失败 code=%d msg=%s", stdErr.Code, stdErr.Msg)
		}
	})
}
