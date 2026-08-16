package mockTypes

import "encoding/json"

type GateMockReq struct {
	Uid      uint64          `json:"uid" binding:"required"`
	Route    uint32          `json:"route" binding:"required"`
	JsonBody json.RawMessage `json:"json_body" binding:"required"`
}

type GateMockRsp struct {
	JsonRsp json.RawMessage `json:"json_rsp"`
}
