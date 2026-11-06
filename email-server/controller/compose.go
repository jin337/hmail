package controller

import (
	"email-server/apperr"
	"email-server/service"
	"email-server/utils"
	"fmt"
	"regexp"

	"github.com/gin-gonic/gin"
)

// dataHrefPattern 匹配前端详情页插入的 data-href 属性，发送前需要移除。
var dataHrefPattern = regexp.MustCompile(`\s+data-href=["'][^"']*["']`)

// parseCompose 从 multipart 表单解析"撰写邮件"参数，并按需加载原邮件资源。
func (h *Handler) parseCompose(c *gin.Context, stripDataHref bool) (service.Compose, error) {
	var comp service.Compose

	comp.From = c.PostForm("from")
	comp.To = utils.SplitAddressList(c.PostForm("to"))
	comp.Cc = utils.SplitAddressList(c.PostForm("cc"))
	comp.Subject = c.PostForm("subject")
	comp.Body = c.PostForm("content")
	comp.InReplyTo = c.PostForm("in_reply_to")
	comp.References = c.PostForm("references")
	comp.Schedule = c.PostForm("x-schedule-send")
	comp.Folder = c.PostForm("folder")
	comp.PartIDs = c.PostForm("part_ids")

	if comp.From == "" {
		comp.From, _ = currentUser(c)
	}

	if stripDataHref && comp.Body != "" {
		comp.Body = dataHrefPattern.ReplaceAllString(comp.Body, "")
	}

	// 上传附件：只有 multipart 请求才有文件集合。
	if c.Request.MultipartForm != nil {
		comp.Files = c.Request.MultipartForm.File["files"]
	}

	uidStr := c.PostForm("uid")
	if uidStr != "" {
		var uid int64
		if _, err := fmt.Sscanf(uidStr, "%d", &uid); err != nil {
			return comp, apperr.BadRequest("无效的 UID 格式")
		}
		comp.SourceUID = uid
	}

	// 编辑草稿 / 回复转发时，复用原邮件中的内联图片与附件。
	if comp.SourceUID > 0 && comp.Folder != "" {
		email, pwd := currentUser(c)
		inlines, attaches, err := h.svc.MailResources(email, pwd, comp.Folder, comp.SourceUID, comp.PartIDs)
		if err != nil {
			return comp, apperr.Internal("获取旧邮件资源失败", err)
		}
		comp.OriginInlines = inlines
		comp.OriginAttaches = attaches
	}

	return comp, nil
}

// SaveDraft 保存草稿。
func (h *Handler) SaveDraft(c *gin.Context) {
	email, pwd := currentUser(c)

	comp, err := h.parseCompose(c, false)
	if err != nil {
		fail(c, apperr.CodeInternal, err.Error())
		return
	}

	uid, msg, err := h.svc.SaveDraft(email, pwd, comp)
	if err != nil {
		failError(c, "保存草稿失败", err)
		return
	}

	body := gin.H{"code": apperr.CodeOk, "msg": msg}
	if uid > 0 {
		body["uid"] = uid
	}
	c.JSON(200, body)
}

// SendEmail 发送邮件。
func (h *Handler) SendEmail(c *gin.Context) {
	email, pwd := currentUser(c)

	comp, err := h.parseCompose(c, true)
	if err != nil {
		fail(c, apperr.CodeInternal, err.Error())
		return
	}
	if len(comp.To) == 0 {
		fail(c, apperr.CodeBadRequest, "收件人不能为空")
		return
	}

	msg, err := h.svc.SendEmail(email, pwd, comp)
	if err != nil {
		failError(c, "发送失败", err)
		return
	}
	okMsg(c, msg)
}
