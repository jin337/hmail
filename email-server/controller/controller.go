// Package controller 提供 HTTP 处理器。
//
// 统一响应约定（与前端保持一致）：
//   - 成功：HTTP 200 + {"code":200,"msg":...,"data":...}
//   - 业务失败：HTTP 200 + {"code":400|500,"msg":...}
//   - 鉴权失败由 middleware 返回 HTTP 401。
package controller

import (
	"errors"
	"net/http"

	"email-server/apperr"
	"email-server/config"
	"email-server/service"

	"github.com/gin-gonic/gin"
)

// Handler 汇总业务服务与配置，供路由注册使用。
type Handler struct {
	svc *service.Service
	cfg config.Config
}

// New 构造 Handler。
func New(svc *service.Service, cfg config.Config) *Handler {
	return &Handler{svc: svc, cfg: cfg}
}

// EnsureScheduledRestored 供鉴权中间件调用：进程重启后首次见到该用户时恢复定时任务。
// 内部异步执行，不会阻塞请求。
func (h *Handler) EnsureScheduledRestored(email, pwd string) {
	h.svc.EnsureScheduledRestored(email, pwd)
}

// ok 返回成功响应。
func ok(c *gin.Context, data any, msg string) {
	c.JSON(http.StatusOK, gin.H{"code": apperr.CodeOk, "msg": msg, "data": data})
}

// okMsg 返回不带数据的成功响应。
func okMsg(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, gin.H{"code": apperr.CodeOk, "msg": msg})
}

// fail 返回业务失败响应。
func fail(c *gin.Context, code int, msg string) {
	c.JSON(http.StatusOK, gin.H{"code": code, "msg": msg})
}

// failWithDetail 返回带 err 明细的业务失败响应（保持旧接口的 err 字段）。
func failWithDetail(c *gin.Context, code int, msg, detail string) {
	body := gin.H{"code": code, "msg": msg}
	if detail != "" {
		body["err"] = detail
	}
	c.JSON(http.StatusOK, body)
}

// failError 把错误映射为响应：业务错误使用其自带 code/msg，其余按 fallbackMsg 处理。
func failError(c *gin.Context, fallbackMsg string, err error) {
	var be *apperr.Error
	if errors.As(err, &be) {
		if be.Err != nil {
			failWithDetail(c, be.Code, be.Msg, be.Err.Error())
			return
		}
		fail(c, be.Code, be.Msg)
		return
	}
	failWithDetail(c, apperr.CodeInternal, fallbackMsg, err.Error())
}

// currentUser 从鉴权上下文取出邮箱与密码。
func currentUser(c *gin.Context) (string, string) {
	email, _ := c.Get("userEmail")
	pwd, _ := c.Get("userPwd")
	e, _ := email.(string)
	p, _ := pwd.(string)
	return e, p
}

// bindJSON 绑定 JSON 请求体，失败时写入统一 400 响应。
func bindJSON(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		fail(c, apperr.CodeBadRequest, "参数错误")
		return false
	}
	return true
}
