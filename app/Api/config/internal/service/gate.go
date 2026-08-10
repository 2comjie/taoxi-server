package service

import (
	"context"

	nodeDeploy "github.com/2comjie/taoxi-server/internal/deploy/node"
	"github.com/2comjie/taoxi-server/pkg/stderr"
	"github.com/2comjie/wali/core/endpoint"
	"github.com/2comjie/wali/logx"
)

func RdGateAddress(ctx context.Context, uid uint64) (endpoint.ServiceInstance, *stderr.Error) {
	logCtx := logx.WithField("uid", uid).WithField("action", "获取网关地址")
	serviceList, err := nodeDeploy.App().ListServices(ctx)
	if err != nil {
		logCtx.Error("获取网关地址失败 %v", err)
		return endpoint.ServiceInstance{}, stderr.InternalServerError("获取网关地址失败")
	}
	// 根据 service id 求映射
	// 把这个 uid 映射到 最近的 service id 上
	gateList := make([]endpoint.ServiceInstance, 0, len(serviceList))
	for _, service := range serviceList {
		if service.Status != endpoint.Working {
			continue
		}
		gateList = append(gateList, service)
	}
	if len(gateList) == 0 {
		logCtx.Error("获取网关地址失败 没有网关节点")
		return endpoint.ServiceInstance{}, stderr.InternalServerError("获取网关地址失败")
	}
	index := uid % uint64(len(gateList))
	logCtx.Infof("获取网关地址成功 %v", gateList[index])
	return gateList[index], nil
}
