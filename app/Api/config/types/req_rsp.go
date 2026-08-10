package configTypes

type GetGateAddressReq struct {
}
type GetGateAddressRsp struct {
	WsAddress  string `json:"ws_address"`
	KcpAddress string `json:"kcp_address"`
	TcpAddress string `json:"tcp_address"`
}
