package stderr

import "net/http"

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

func BadRequest(msg string) *Error {
	return New(http.StatusBadRequest, msg)
}
func InternalServerError(msg string) *Error {
	return New(http.StatusInternalServerError, msg)
}
func NotFound(msg string) *Error {
	return New(http.StatusNotFound, msg)
}
func Unauthorized(msg string) *Error {
	return New(http.StatusUnauthorized, msg)
}
func Forbidden(msg string) *Error {
	return New(http.StatusForbidden, msg)
}
