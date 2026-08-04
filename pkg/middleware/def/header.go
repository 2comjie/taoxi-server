package midef

import (
	"context"
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

type RequestFormat int64

const (
	AllInForm                       RequestFormat = 0
	HeaderFromHeaderAndBodyFromBody RequestFormat = 1
)

var ErrInvalidRequestFormat = errors.New("midef: 不支持的请求格式")

func ParseRequestFormat(value string) (RequestFormat, error) {
	if value == "" {
		return AllInForm, nil
	}
	formatValue := cast.ToInt(value)
	format := RequestFormat(formatValue)
	if !format.Valid() {
		return 0, fmt.Errorf("%w: %q", ErrInvalidRequestFormat, value)
	}
	return format, nil
}

func (f RequestFormat) Valid() bool {
	return f == AllInForm || f == HeaderFromHeaderAndBodyFromBody
}

type RawHeader struct {
	Token      string `json:"token" form:"token" header:"token" bson:"token"` // 登陆的token
	Lang       string `json:"lang" form:"lang" header:"lang" bson:"lang"`     // 多语言
	Nonce      string `json:"nonce" form:"nonce" header:"nonce" bson:"nonce"`
	Sign       string `json:"sign" form:"sign" header:"sign" bson:"sign"` // 签名
	DistinctID string `json:"distinct_id" form:"distinct_id" header:"distinct_id" bson:"distinct_id"`

	AppVersion  string `json:"app_version" form:"app_version" header:"app_version" bson:"app_version"`
	Platform    string `json:"platform" form:"platform" header:"platform" bson:"platform"` //unity 内部的 platform 打点用
	DeviceID    string `json:"device_id" form:"device_id" header:"device_id" bson:"device_id"`
	AppChannel  string `json:"app_channel" form:"app_channel" header:"app_channel" bson:"app_channel"`     //应用渠道
	AppPlatform string `json:"app_platform" form:"app_platform" header:"app_platform" bson:"app_platform"` // App/小程序/小游戏
	DeviceOS    int    `json:"device_os" form:"device_os" header:"device_os" bson:"device_os"`             //归因服务需要的平台/os

	IP string `json:"-" form:"-" header:"-" bson:"ip"`
}

type Header struct {
	RawHeader
	Uid           uint64        `json:"-" form:"-" header:"-" bson:"-"`
	Body          string        `json:"body" form:"body" header:"-" bson:"-"`
	RequestFormat RequestFormat `json:"-" form:"-" header:"-" bson:"request_format"`
	ctx           context.Context
}

const ctxHTTPHeaderKey = "taoxi-client-request-header"

func (h *Header) Context() context.Context {
	if h != nil && h.ctx != nil {
		return h.ctx
	}
	return context.Background()
}

func NewHeader(ctx context.Context) *Header {
	if ctx == nil {
		ctx = context.Background()
	}
	return &Header{ctx: ctx}
}

func SetClientRequestHeader(c *gin.Context, header *Header) {
	header.ctx = c.Request.Context()
	c.Set(ctxHTTPHeaderKey, header)
}

func GetClientRequestHeader(c *gin.Context) (*Header, bool) {
	value, exists := c.Get(ctxHTTPHeaderKey)
	if !exists {
		return nil, false
	}
	header, ok := value.(*Header)
	if !ok || header == nil {
		return nil, false
	}
	header.ctx = c.Request.Context()
	return header, true
}
