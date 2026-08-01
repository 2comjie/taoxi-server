package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	logintypes "github.com/2comjie/taoxi-server/app/Api/login/types"
)

const weChatCode2SessionURL = "https://api.weixin.qq.com/sns/jscode2session"

type WeChatConfig struct {
	AppID     string
	AppSecret string
}

type WeChatProvider struct {
	appID     string
	appSecret string
	client    *http.Client
}

func NewWeChatProvider(config WeChatConfig) (*WeChatProvider, error) {
	config.AppID = strings.TrimSpace(config.AppID)
	config.AppSecret = strings.TrimSpace(config.AppSecret)
	if config.AppID == "" || config.AppSecret == "" {
		return nil, errors.New("login: 微信app_id和app_secret不能为空")
	}
	return &WeChatProvider{
		appID:     config.AppID,
		appSecret: config.AppSecret,
		client:    &http.Client{Timeout: 10 * time.Second},
	}, nil
}

func (*WeChatProvider) Type() logintypes.LoginType {
	return logintypes.LoginTypeWeChat
}

func (p *WeChatProvider) Authenticate(ctx context.Context, req *logintypes.LoginReq) (*logintypes.Identity, error) {
	code := strings.TrimSpace(req.Code)
	if code == "" {
		return nil, ErrInvalidCredential
	}
	query := url.Values{
		"appid":      {p.appID},
		"secret":     {p.appSecret},
		"js_code":    {code},
		"grant_type": {"authorization_code"},
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, weChatCode2SessionURL+"?"+query.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("login: 创建微信登录请求失败: %w", err)
	}
	response, err := p.client.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("login: 微信登录请求失败: %w", ctx.Err())
		}
		return nil, errors.New("login: 微信登录请求失败")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4<<10))
		return nil, fmt.Errorf("login: 微信登录接口返回状态%d", response.StatusCode)
	}
	var result struct {
		OpenID  string `json:"openid"`
		UnionID string `json:"unionid"`
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&result); err != nil {
		return nil, fmt.Errorf("login: 解析微信登录响应失败: %w", err)
	}
	if result.ErrCode != 0 || strings.TrimSpace(result.OpenID) == "" {
		return nil, ErrInvalidCredential
	}
	return &logintypes.Identity{AppID: p.appID, OpenID: result.OpenID, UnionID: result.UnionID}, nil
}
