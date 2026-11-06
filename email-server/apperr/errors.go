// Package apperr 定义业务错误及其对应的接口返回码。
//
// HTTP 层统一返回 200（除鉴权失败由中间件返回 401），
// 具体业务结果通过响应体中的 code 字段区分。
package apperr

import (
	"errors"
	"fmt"
)

// 业务返回码，与前端约定的 code 字段保持一致。
const (
	CodeOk           = 200
	CodeBadRequest   = 400
	CodeUnauthorized = 401
	CodeInternal     = 500
)

// Error 是可携带返回码与底层原因的业务错误。
type Error struct {
	Code int
	Msg  string
	Err  error
}

// Error 实现 error 接口，保留底层原因便于日志排查。
func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Msg, e.Err)
	}
	return e.Msg
}

// Unwrap 支持 errors.Is/As 追溯底层错误。
func (e *Error) Unwrap() error { return e.Err }

// BadRequest 构造参数/业务校验类错误。
func BadRequest(msg string) *Error {
	return &Error{Code: CodeBadRequest, Msg: msg}
}

// BadRequestf 构造带格式化参数的参数错误。
func BadRequestf(format string, args ...any) *Error {
	return BadRequest(fmt.Sprintf(format, args...))
}

// Internal 构造服务端错误。
func Internal(msg string, err error) *Error {
	return &Error{Code: CodeInternal, Msg: msg, Err: err}
}

// InternalWrap 在底层错误上附加上下文，保留原始错误链。
func InternalWrap(msg string, err error) *Error {
	if err == nil {
		return Internal(msg, nil)
	}
	return &Error{Code: CodeInternal, Msg: msg, Err: err}
}

// From 把任意错误归一化为业务错误：已是 *Error 则原样返回，
// 否则使用 fallbackMsg 构造 500 错误。
func From(err error, fallbackMsg string) *Error {
	if err == nil {
		return Internal(fallbackMsg, nil)
	}
	var be *Error
	if errors.As(err, &be) {
		return be
	}
	return &Error{Code: CodeInternal, Msg: fallbackMsg, Err: err}
}
