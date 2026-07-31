package stderr

// 业务错误

type Error struct {
	Code int
	Msg  string
}

func New(code int, msg string) *Error {
	return &Error{
		Code: code,
		Msg:  msg,
	}
}
