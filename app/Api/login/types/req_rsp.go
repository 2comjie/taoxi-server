package types

type LoginType int32

const (
	LoginTypeGuest  LoginType = 1
	LoginTypeGoogle LoginType = 2
	LoginTypeApple  LoginType = 3
	LoginTypeWeChat LoginType = 4
)

type LoginReq struct {
	LoginType LoginType `json:"login_type" form:"login_type" binding:"required"`

	GuestID       string `json:"guest_id" form:"guest_id"`
	Code          string `json:"code" form:"code"`
	IdentityToken string `json:"identity_token" form:"identity_token"`
	AccessToken   string `json:"access_token" form:"access_token"`
	Nonce         string `json:"nonce" form:"nonce"`
}

type LoginRsp struct {
	UID         string `json:"uid"`
	OpenID      string `json:"openid"`
	IsRegister  bool   `json:"is_register"`
	GateToken   string `json:"gate_token"`
	AccessToken string `json:"access_token"`
}
