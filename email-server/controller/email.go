package controller

import (
	"email-server/apperr"
	"email-server/config"
	"email-server/model"
	"email-server/utils"
	"fmt"
	"net/url"

	"github.com/gin-gonic/gin"
)

// MailList 获取邮件列表。
func (h *Handler) MailList(c *gin.Context) {
	email, pwd := currentUser(c)

	var req model.MailListReq
	if !bindJSON(c, &req) {
		return
	}
	if err := utils.CheckRequired(utils.Required("Folder", req.Folder)); err != nil {
		fail(c, apperr.CodeBadRequest, err.Error())
		return
	}
	if !isStandardFolder(req.Folder) {
		fail(c, apperr.CodeBadRequest, "folder 参数无效")
		return
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 10
	}

	list, total, err := h.svc.MailList(email, pwd, req.Folder, req.Page, req.Size, req.Keyword, req.Filter)
	if err != nil {
		fail(c, apperr.CodeInternal, "获取邮件列表失败: "+err.Error())
		return
	}
	ok(c, gin.H{"list": list, "total": total, "page": req.Page, "size": req.Size}, "success")
}

// StarMailList 获取星标邮件列表。
func (h *Handler) StarMailList(c *gin.Context) {
	email, pwd := currentUser(c)

	var req model.MailStarListReq
	if !bindJSON(c, &req) {
		return
	}

	list, total, err := h.svc.StarMailList(email, pwd, req.Keyword, req.Filter)
	if err != nil {
		fail(c, apperr.CodeInternal, "获取邮件列表失败: "+err.Error())
		return
	}
	ok(c, gin.H{"list": list, "total": total}, "success")
}

// MailDetail 获取邮件详情。
func (h *Handler) MailDetail(c *gin.Context) {
	email, pwd := currentUser(c)

	var req model.MailDetailReq
	if !bindJSON(c, &req) {
		return
	}
	if err := utils.CheckRequired(utils.Required("Folder", req.Folder)); err != nil {
		fail(c, apperr.CodeBadRequest, err.Error())
		return
	}

	// 调试模式直接用请求 Host；生产模式信任前端上报的客户端 Host。
	host := c.GetHeader("X-Client-Host")
	if h.cfg.Mode == "debug" {
		host = c.Request.Host
	}

	detail, err := h.svc.MailDetail(email, pwd, req.Folder, req.Uid, host)
	if err != nil {
		fail(c, apperr.CodeInternal, "获取邮件详情失败: "+err.Error())
		return
	}
	ok(c, detail, "success")
}

// MarkFlag 标记状态。
func (h *Handler) MarkFlag(c *gin.Context) {
	email, pwd := currentUser(c)

	var req model.UpdateMailFlagReq
	if !bindJSON(c, &req) {
		return
	}
	if err := utils.CheckRequired(
		utils.Required("Folder", req.Folder),
		utils.Required("Status", req.Status),
	); err != nil {
		fail(c, apperr.CodeBadRequest, err.Error())
		return
	}

	if err := h.svc.UpdateMailFlag(email, pwd, req.Folder, req.Uids, req.Type, req.Status); err != nil {
		fail(c, apperr.CodeInternal, "更新邮件状态失败: "+err.Error())
		return
	}
	okMsg(c, "状态修改成功")
}

// DownloadAttachment 下载附件。
func (h *Handler) DownloadAttachment(c *gin.Context) {
	email, pwd := currentUser(c)

	var req model.DownloadAttachReq
	if !bindJSON(c, &req) {
		return
	}
	if err := utils.CheckRequired(
		utils.Required("Folder", req.Folder),
		utils.Required("PartID", req.PartID),
	); err != nil {
		fail(c, apperr.CodeBadRequest, err.Error())
		return
	}

	fileName, data, err := h.svc.DownloadAttachment(email, pwd, req.Folder, req.Uid, req.PartID)
	if err != nil {
		// 失败也返回 Blob，前端统一按 blob 处理。
		c.Header("Content-Disposition", "attachment; filename=error.txt")
		c.Header("Content-Type", "text/plain; charset=utf-8")
		c.Data(200, "text/plain; charset=utf-8", []byte(fmt.Sprintf("下载失败: %s", err.Error())))
		return
	}

	encoded := url.QueryEscape(fileName)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s; filename=%s", encoded, encoded))
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Length", fmt.Sprintf("%d", len(data)))
	c.Data(200, "application/octet-stream", data)
}

// MoveMail 移动邮件。
func (h *Handler) MoveMail(c *gin.Context) {
	email, pwd := currentUser(c)

	var req model.MoveMailReq
	if !bindJSON(c, &req) {
		return
	}
	if err := utils.CheckRequired(
		utils.Required("FromFolder", req.FromFolder),
		utils.Required("ToFolder", req.ToFolder),
	); err != nil {
		fail(c, apperr.CodeBadRequest, err.Error())
		return
	}
	if req.FromFolder == req.ToFolder {
		fail(c, apperr.CodeBadRequest, "源文件夹和目标文件夹不能相同")
		return
	}

	if err := h.svc.MoveMail(email, pwd, req.FromFolder, req.ToFolder, req.Uids); err != nil {
		fail(c, apperr.CodeInternal, "移动邮件失败: "+err.Error())
		return
	}
	okMsg(c, "移动成功")
}

// DeleteMail 删除邮件。
func (h *Handler) DeleteMail(c *gin.Context) {
	email, pwd := currentUser(c)

	var req model.DelMailReq
	if !bindJSON(c, &req) {
		return
	}
	if err := utils.CheckRequired(utils.Required("Folder", req.Folder)); err != nil {
		fail(c, apperr.CodeBadRequest, err.Error())
		return
	}

	if err := h.svc.DeleteMail(email, pwd, req.Folder, req.Uids); err != nil {
		fail(c, apperr.CodeInternal, "删除邮件失败: "+err.Error())
		return
	}
	okMsg(c, "删除成功")
}

// UnScheduleEmail 取消定时发送。
func (h *Handler) UnScheduleEmail(c *gin.Context) {
	email, pwd := currentUser(c)

	var req model.ScheduleMailFlagReq
	if !bindJSON(c, &req) {
		return
	}
	if err := utils.CheckRequired(
		utils.Required("Folder", req.Folder),
		utils.Required("Status", req.Status),
	); err != nil {
		fail(c, apperr.CodeBadRequest, err.Error())
		return
	}

	if err := h.svc.UnScheduleEmail(email, pwd, req.Folder, req.Uid, req.Type, req.Status); err != nil {
		fail(c, apperr.CodeInternal, "取消定时失败: "+err.Error())
		return
	}
	okMsg(c, "修改成功")
}

// UnreadMailTotal 获取未读邮件数量。
func (h *Handler) UnreadMailTotal(c *gin.Context) {
	email, pwd := currentUser(c)

	total, err := h.svc.UnreadMailTotal(email, pwd)
	if err != nil {
		fail(c, apperr.CodeInternal, "获取未读邮件数量失败: "+err.Error())
		return
	}
	ok(c, total, "success")
}

// isStandardFolder 判断是否为标准文件夹（不含前端虚拟的 Star）。
func isStandardFolder(folder string) bool {
	for _, f := range config.DefaultFolders {
		if f == folder {
			return true
		}
	}
	return false
}
