package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	logintypes "github.com/2comjie/taoxi-server/app/Api/login/types"
)

const defaultAppID = "taoxi"

type GuestProvider struct{}

func (GuestProvider) Type() logintypes.LoginType {
	return logintypes.LoginTypeGuest
}

func (GuestProvider) Authenticate(_ context.Context, req *logintypes.LoginReq) (*logintypes.Identity, error) {
	guestID := strings.TrimSpace(req.GuestID)
	if guestID == "" {
		value := make([]byte, 32)
		if _, err := rand.Read(value); err != nil {
			return nil, fmt.Errorf("login: 生成游客凭证失败: %w", err)
		}
		guestID = "guest_" + base64.RawURLEncoding.EncodeToString(value)
	}
	if len(guestID) > 255 {
		return nil, ErrInvalidCredential
	}
	return &logintypes.Identity{AppID: defaultAppID, OpenID: guestID}, nil
}
