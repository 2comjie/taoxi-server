package mockClient

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/2comjie/nova/logx"
	"github.com/2comjie/nova/logx/logdef"
	"github.com/2comjie/nova/network"
	netWs "github.com/2comjie/nova/network/transport/ws"
	configTypes "github.com/2comjie/taoxi-server/app/Api/config/types"
	loginTypes "github.com/2comjie/taoxi-server/app/Api/login/types"
	"github.com/2comjie/taoxi-server/internal/fsm"
	"github.com/go-resty/resty/v2"
	"github.com/spf13/cast"
)

type MockClient struct {
	logdef.ILogger
	uid uint64
	ctx context.Context

	gateClient *network.Client
	httpClient *resty.Client
	baseUrl    string

	// 所有的请求头信息1
	accessToken string
	lang        string
	appVersion  string
	appChannel  string
	appPlatform string
	deviceOS    int

	fsm *fsm.FSM[struct{}, string]

	gateToken string
}

func NewMockClient(uid uint64, baseUrl string, ctx context.Context) *MockClient {
	c := &MockClient{
		ILogger:    logx.WithField("uid", uid),
		uid:        uid,
		ctx:        ctx,
		httpClient: resty.New(),
		baseUrl:    baseUrl,
		lang:       "Zh",
	}

	c.httpClient.SetRetryCount(3)
	c.httpClient.SetRetryWaitTime(5 * time.Second)
	c.httpClient.SetRetryMaxWaitTime(30 * time.Second)

	c.initFsm()

	c.fsm.Switch(struct{}{}, Login)
	return c
}

const (
	Undefine = "Undefine"
	Login    = "Login"
	BindGate = "BindGate"
	Playing  = "Playing"
)

func (c *MockClient) initFsm() {
	c.fsm = fsm.NewFsm[struct{}, string](Undefine)
	c.fsm.PostSwitchState(func(oldState string, newState string, arg struct{}) {
		c.Infof("切换状态 %s -> %s", oldState, newState)
	})

	c.fsm.Register(Login, &fsm.SimpleState[struct{}, string]{
		MPreEnter: func(lastState string, arg struct{}, isTimeOut bool) time.Duration {
			return 100 * time.Millisecond
		},
		MPostEnter: nil,
		MUpdate:    nil,
		MPreExist:  nil,
		MPostExist: nil,
		MGetNextStateOnTimeOut: func(arg struct{}) string {
			loginReq := &loginTypes.LoginReq{
				LoginType:     loginTypes.LoginTypeDebug,
				IdentityToken: cast.ToString(c.uid),
			}
			rsp, err := POST[loginTypes.LoginReq, loginTypes.LoginRsp](c, "api/login", loginReq)
			if err != nil {
				c.Errorf("登陆请求失败 %v", err)
				return Undefine
			}

			if rsp.Code != http.StatusOK {
				c.Errorf("登陆失败 %d %v", rsp.Code, rsp.Msg)
				return Undefine
			}
			c.accessToken = rsp.Result.AccessToken
			c.uid = rsp.Result.Uid
			c.gateToken = rsp.Result.GateToken
			c.Infof("登陆成功 %+v", rsp.Result)
			return BindGate
		},
		MOnEventTrigger: nil,
	})

	c.fsm.Register(BindGate, &fsm.SimpleState[struct{}, string]{
		MPreEnter: func(lastState string, arg struct{}, isTimeOut bool) time.Duration {
			return 100 * time.Millisecond
		},
		MPostEnter: nil,
		MUpdate:    nil,
		MPreExist:  nil,
		MPostExist: nil,
		MGetNextStateOnTimeOut: func(arg struct{}) string {
			getGateReq := &configTypes.GetGateAddressReq{}
			rsp, err := POST[configTypes.GetGateAddressReq, configTypes.GetGateAddressRsp](c, "api/config/gate_address", getGateReq)
			if err != nil {
				c.Errorf("获取网关地址失败 %v", err)
				return Undefine
			}
			if rsp.Code != http.StatusOK {
				c.Errorf("获取网关地址失败 %d %v", rsp.Code, rsp.Msg)
				return Undefine
			}

			wsAddr := rsp.Result.WsAddress
			dialer := netWs.NewDialer(wsAddr)

			c.gateClient, err = network.NewClient(network.WithDialer(dialer))
			if err != nil {
				c.Errorf("创建gate客户端失败 %v", err)
				return Undefine
			}
			err = c.gateClient.Dial(c.ctx)
			if err != nil {
				c.Errorf("连接网关失败 %v", err)
				return Undefine
			}

			err = c.gateClient.Bind(c.ctx, []byte(c.gateToken))
			if err != nil {
				c.Errorf("绑定网关失败 %v", err)
				return Undefine
			}

			c.Infof("网关绑定成功 %v", wsAddr)
			return Playing
		},
		MOnEventTrigger: nil,
	})

	c.fsm.Register(Playing, &fsm.SimpleState[struct{}, string]{
		MPreEnter: nil,
		MPostEnter: func(lastState string, arg struct{}, isTimeOut bool) {
			c.Infof("进入游戏")
		},
		MUpdate:                nil,
		MPreExist:              nil,
		MPostExist:             nil,
		MGetNextStateOnTimeOut: nil,
		MOnEventTrigger:        nil,
	})

	c.fsm.Register(Undefine, &fsm.SimpleState[struct{}, string]{
		MPreEnter: nil,
		MPostEnter: func(lastState string, arg struct{}, isTimeOut bool) {
			if c.gateClient != nil {
				_ = c.gateClient.Close()
			}
		},
		MUpdate:                nil,
		MPreExist:              nil,
		MPostExist:             nil,
		MGetNextStateOnTimeOut: nil,
		MOnEventTrigger:        nil,
	})
}

func (c *MockClient) path(p string) (string, error) {
	baseUrl := c.baseUrl
	if !strings.Contains(baseUrl, "://") {
		baseUrl = "http://" + baseUrl
	}
	return url.JoinPath(baseUrl, p)
}

func POST[Req any, Rsp any](c *MockClient, path string, req *Req) (ApiRsp[*Rsp], error) {
	result := ApiRsp[*Rsp]{}
	requestUrl, err := c.path(path)
	if err != nil {
		return result, err
	}
	httpReq := c.httpClient.R().
		SetHeaders(map[string]string{
			"Content-Type": "application/json",
			"format":       "1",
			"lang":         c.lang,
			"app_version":  c.appVersion,
			"app_channel":  c.appChannel,
			"app_platform": c.appPlatform,
			"device_os":    cast.ToString(c.deviceOS),
		}).
		SetBody(req).
		SetResult(&result)
	if c.accessToken != "" {
		httpReq.SetHeader("token", c.accessToken)
	}
	rsp, err := httpReq.Post(requestUrl)
	if err != nil {
		return result, err
	}
	if rsp.StatusCode() != http.StatusOK {
		return result, fmt.Errorf("HTTP请求失败 status=%d body=%s", rsp.StatusCode(), rsp.String())
	}
	return result, nil
}

func (c *MockClient) Update(dt time.Duration) {
	c.fsm.Update(struct{}{}, dt)
}
