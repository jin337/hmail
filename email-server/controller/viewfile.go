package controller

import (
	"path"
	"strings"

	"email-server/apperr"

	"github.com/gin-gonic/gin"
)

// ViewFile 读取静态文件（头像、邮件内联图片）。
//
// 与旧实现的区别：只允许访问配置的静态目录内的文件，避免任意路径读取。
func (h *Handler) ViewFile(c *gin.Context) {
	raw := c.Query("url")
	if raw == "" {
		fail(c, apperr.CodeInternal, "参数缺失 ")
		return
	}

	// 去掉查询串（前端会带 ?v=时间戳 做缓存击穿）。
	if idx := strings.Index(raw, "?"); idx >= 0 {
		raw = raw[:idx]
	}

	rel := strings.TrimPrefix(strings.ReplaceAll(raw, "\\", "/"), "/")
	rel = path.Clean(rel)

	staticDir := strings.TrimPrefix(strings.ReplaceAll(h.cfg.StaticDir, "\\", "/"), "./")
	staticDir = strings.Trim(staticDir, "/")

	if rel == "." || !(rel == staticDir || strings.HasPrefix(rel, staticDir+"/")) {
		fail(c, apperr.CodeBadRequest, "非法文件路径")
		return
	}

	c.File(rel)
}
