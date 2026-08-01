package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	logintypes "github.com/2comjie/taoxi-server/app/Api/login/types"
	"golang.org/x/oauth2"
)

const googlePlayerURL = "https://games.googleapis.com/games/v1/players/me"

type GoogleConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
}

type GoogleProvider struct {
	oauth2Config *oauth2.Config
}

func NewGoogleProvider(config GoogleConfig) (*GoogleProvider, error) {
	config.ClientID = strings.TrimSpace(config.ClientID)
	config.ClientSecret = strings.TrimSpace(config.ClientSecret)
	if config.ClientID == "" || config.ClientSecret == "" {
		return nil, errors.New("login: Google client_id和client_secret不能为空")
	}
	return &GoogleProvider{oauth2Config: &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  strings.TrimSpace(config.RedirectURI),
		Endpoint: oauth2.Endpoint{
			AuthURL:   "https://accounts.google.com/o/oauth2/auth",
			TokenURL:  "https://oauth2.googleapis.com/token",
			AuthStyle: oauth2.AuthStyleInParams,
		},
		Scopes: []string{"https://www.googleapis.com/auth/games"},
	}}, nil
}

func (*GoogleProvider) Type() logintypes.LoginType {
	return logintypes.LoginTypeGoogle
}

func (p *GoogleProvider) Authenticate(ctx context.Context, req *logintypes.LoginReq) (*logintypes.Identity, error) {
	code := strings.TrimSpace(req.Code)
	if code == "" {
		return nil, ErrInvalidCredential
	}
	requestCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	oauthToken, err := p.oauth2Config.Exchange(requestCtx, code)
	if err != nil {
		var retrieveErr *oauth2.RetrieveError
		if errors.As(err, &retrieveErr) {
			return nil, ErrInvalidCredential
		}
		return nil, fmt.Errorf("login: Google授权码无效: %w", err)
	}
	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, googlePlayerURL, nil)
	if err != nil {
		return nil, fmt.Errorf("login: 创建Google玩家请求失败: %w", err)
	}
	response, err := p.oauth2Config.Client(requestCtx, oauthToken).Do(request)
	if err != nil {
		return nil, fmt.Errorf("login: 获取Google玩家失败: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4<<10))
		if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
			return nil, ErrInvalidCredential
		}
		return nil, fmt.Errorf("login: Google玩家接口返回状态%d", response.StatusCode)
	}
	var player struct {
		GamePlayerID string `json:"gamePlayerId"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&player); err != nil {
		return nil, fmt.Errorf("login: 解析Google玩家失败: %w", err)
	}
	if strings.TrimSpace(player.GamePlayerID) == "" {
		return nil, ErrInvalidCredential
	}
	return &logintypes.Identity{AppID: p.oauth2Config.ClientID, OpenID: player.GamePlayerID}, nil
}
