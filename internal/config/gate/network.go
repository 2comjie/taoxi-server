package gateConfig

import (
	"fmt"

	"github.com/2comjie/nova/config"
)

const networkConfigKey = "gate.network"

type Network struct {
	TCPListenAddress   string `json:"tcp_listen_address"`
	TCPExternalAddress string `json:"tcp_external_address"`
	KCPListenAddress   string `json:"kcp_listen_address"`
	KCPExternalAddress string `json:"kcp_external_address"`
	WSListenAddress    string `json:"ws_listen_address"`
	WSExternalAddress  string `json:"ws_external_address"`
}

type networkMap map[int]*Network

var networks config.WatchedValue[networkMap]

func Init(center config.Config) error {
	return networks.Init(center, networkConfigKey)
}

func GetNetwork(serviceIndex int) (*Network, error) {
	network := networks.Load()[serviceIndex]
	if network == nil {
		return nil, fmt.Errorf("gate config: 找不到service_index=%d的网络配置", serviceIndex)
	}
	return network, nil
}
