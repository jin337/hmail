package controller

import (
	"email-server/config"
	"email-server/model"
	"email-server/service"
	"fmt"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

// MailList 获取邮件列表
func MailList(c *gin.Context) {
	email, _ := c.Get("userEmail")
	pwd, _ := c.Get("userPwd")

	var req model.MailListReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	// 验证必传
	if req.Folder == "" {
		c.JSON(200, gin.H{"code": 400, "msg": "folder 参数不能为空"})
		return
	}

	// 验证文件夹
	validFolders := []string{
		config.FolderInbox,
		config.FolderSent,
		config.FolderDrafts,
		config.FolderDeleted,
		config.FolderJunk,
	}
	isValidFolder := false
	for _, f := range validFolders {
		if f == req.Folder {
			isValidFolder = true
			break
		}
	}
	if !isValidFolder {
		c.JSON(200, gin.H{"code": 400, "msg": "folder 参数无效"})
		return
	}

	// 验证分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 10
	}

	list, total, err := service.MailList(email.(string), pwd.(string), req.Folder, req.Page, req.Size, req.Keyword)
	if err != nil {
		c.JSON(200, gin.H{"code": 500, "msg": "获取邮件列表失败: " + err.Error()})
		return
	}
	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "success",
		"data": gin.H{
			"list":  list,
			"total": total,
			"page":  req.Page,
			"size":  req.Size,
		},
	})
}

// MailDetail 获取邮件详情
func MailDetail(c *gin.Context) {
	email, _ := c.Get("userEmail")
	pwd, _ := c.Get("userPwd")

	var req model.MailDetailReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	// 验证必传
	if req.Uid <= 0 || req.Folder == "" {
		c.JSON(200, gin.H{"code": 400, "msg": "uid 或 folder 参数不能为空"})
		return
	}

	mailItem, err := service.MailDetail(email.(string), pwd.(string), req.Folder, req.Uid)
	if err != nil {
		c.JSON(200, gin.H{"code": 500, "msg": "获取邮件详情失败: " + err.Error()})
		return
	}
	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "success",
		"data": mailItem,
	})
}

// MarkRead 标记已读
func MarkRead(c *gin.Context) {
	email, _ := c.Get("userEmail")
	pwd, _ := c.Get("userPwd")

	var req model.UpdateMailStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	// 验证必传
	if req.Uid <= 0 || req.Folder == "" {
		c.JSON(200, gin.H{"code": 400, "msg": "uid 或 folder 参数不能为空"})
		return
	}

	err := service.UpdateMailStatus(email.(string), pwd.(string), req.Folder, req.Uid, req.Status)
	if err != nil {
		c.JSON(200, gin.H{"code": 500, "msg": "更新邮件状态失败: " + err.Error()})
		return
	}
	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "状态成功",
	})
}

// DownloadAttachment 下载附件
func DownloadAttachment(c *gin.Context) {
	email, _ := c.Get("userEmail")
	pwd, _ := c.Get("userPwd")

	var req model.DownloadAttachReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	// 验证必传
	if req.Uid <= 0 || req.Folder == "" || req.PartID == "" {
		c.JSON(200, gin.H{"code": 400, "msg": "uid 或 folder 或 part_id 参数不能为空"})
		return
	}

	fileName, fileBytes, err := service.DownloadAttachment(email.(string), pwd.(string), req.Folder, req.Uid, req.PartID)
	if err != nil {
		// 错误时也返回 Blob 格式，方便前端统一处理
		errorMsg := fmt.Sprintf("下载失败: %s", err.Error())
		c.Header("Content-Disposition", "attachment; filename=error.txt")
		c.Header("Content-Type", "text/plain; charset=utf-8")
		c.Data(200, "text/plain; charset=utf-8", []byte(errorMsg))
		return
	}
	// URL 编码文件名以支持中文和特殊字符
	encodedFileName := url.QueryEscape(fileName)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s; filename=%s", encodedFileName, encodedFileName))
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Length", fmt.Sprintf("%d", len(fileBytes)))
	c.Data(200, "application/octet-stream", fileBytes)
}

// MoveMail 移动邮件
func MoveMail(c *gin.Context) {
	email, _ := c.Get("userEmail")
	pwd, _ := c.Get("userPwd")

	var req model.MoveMailReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	// 验证必传
	if len(req.Uids) == 0 || req.FromFolder == "" || req.ToFolder == "" {
		c.JSON(200, gin.H{"code": 400, "msg": "uid 或 folder 或 new_folder 参数不能为空"})
		return
	}

	// 参数验证
	if len(req.Uids) == 0 {
		c.JSON(200, gin.H{"code": 400, "msg": "邮件UID列表不能为空"})
		return
	}
	if req.FromFolder == req.ToFolder {
		c.JSON(200, gin.H{"code": 400, "msg": "源文件夹和目标文件夹不能相同"})
		return
	}

	err := service.MoveMail(email.(string), pwd.(string), req.FromFolder, req.ToFolder, req.Uids)
	if err != nil {
		c.JSON(200, gin.H{"code": 500, "msg": "移动邮件失败: " + err.Error()})
		return
	}
	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "移动成功",
	})
}

// DeleteMail 删除邮件
func DeleteMail(c *gin.Context) {
	email, _ := c.Get("userEmail")
	pwd, _ := c.Get("userPwd")

	var req model.DelMailReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	// 验证必传
	if len(req.Uids) == 0 || req.Folder == "" {
		c.JSON(200, gin.H{"code": 400, "msg": "uid 列表或 folder 参数不能为空"})
		return
	}

	err := service.DeleteMail(email.(string), pwd.(string), req.Folder, req.Uids)
	if err != nil {
		c.JSON(200, gin.H{"code": 500, "msg": "删除邮件失败: " + err.Error()})
		return
	}
	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "删除成功",
	})
}

// SaveDraft 保存草稿
func SaveDraft(c *gin.Context) {
	email, _ := c.Get("userEmail")
	pwd, _ := c.Get("userPwd")

	to := c.PostForm("to")
	cc := c.PostForm("cc")
	subject := c.PostForm("subject")
	content := c.PostForm("content")

	uidStr := c.PostForm("uid")
	partIds := c.PostForm("part_ids")

	files := c.Request.MultipartForm.File["files"]
	if len(files) == 0 {
		files = nil
	}

	// 解析 UID
	var uid uint32
	if _, parseErr := fmt.Sscanf(uidStr, "%d", &uid); parseErr != nil {
		c.JSON(200, gin.H{"code": 400, "msg": "无效的 UID 格式"})
		return
	}

	// 构建邮件
	toList := strings.Split(to, ",")
	ccList := strings.Split(cc, ",")
	raw, err := service.BuildRawEmail(email.(string), pwd.(string), config.FolderDrafts, uid, partIds, []string{email.(string)}, toList, ccList, subject, content, files)
	if err != nil {
		c.JSON(200, gin.H{"code": 500, "msg": "构建邮件失败", "err": err.Error()})
		return
	}

	// 更新草稿
	if uidStr != "" {
		err := service.UpdateDraft(email.(string), pwd.(string), config.FolderDrafts, raw, uid)
		if err != nil {
			c.JSON(200, gin.H{"code": 500, "msg": "更新草稿失败", "err": err.Error()})
			return
		}
		c.JSON(200, gin.H{"code": 200, "msg": "草稿更新成功", "uid": uid})
	} else {
		// 新建草稿
		err := service.SaveMailToFolder(email.(string), pwd.(string), config.FolderDrafts, raw)
		if err != nil {
			c.JSON(200, gin.H{"code": 500, "msg": "保存草稿失败", "err": err.Error()})
			return
		}
		c.JSON(200, gin.H{"code": 200, "msg": "草稿保存成功"})
	}
}

// SendEmail 发送邮件
func SendEmail(c *gin.Context) {
	email, _ := c.Get("userEmail")
	pwd, _ := c.Get("userPwd")

	to := c.PostForm("to")
	cc := c.PostForm("cc")
	subject := c.PostForm("subject")
	content := c.PostForm("content")

	uidStr := c.PostForm("uid")
	partIds := c.PostForm("part_ids")

	files := c.Request.MultipartForm.File["files"]
	if len(files) == 0 {
		files = nil
	}

	// 解析 UID
	var uid uint32
	if _, parseErr := fmt.Sscanf(uidStr, "%d", &uid); parseErr != nil {
		c.JSON(200, gin.H{"code": 400, "msg": "无效的 UID 格式"})
		return
	}

	// 构建邮件内容
	toList := strings.Split(to, ",")
	ccList := strings.Split(cc, ",")
	raw, err := service.BuildRawEmail(email.(string), pwd.(string), config.FolderDrafts, uid, partIds, []string{email.(string)}, toList, ccList, subject, content, files)
	if err != nil {
		c.JSON(200, gin.H{"code": 500, "msg": "构建邮件失败", "err": err.Error()})
		return
	}

	// 发送邮件
	if err := service.SmtpSendEmail(email.(string), pwd.(string), toList, ccList, raw); err != nil {
		c.JSON(200, gin.H{"code": 500, "msg": "发送失败", "err": err.Error()})
		return
	}

	// 存入已发送
	err = service.SaveMailToFolder(email.(string), pwd.(string), config.FolderSent, raw)
	if err != nil {
		c.JSON(200, gin.H{"code": 500, "msg": "发送成功，存入已发送失败", "err": err.Error()})
		return
	}

	// 删除草稿
	if uidStr != "" {
		err = service.DeleteMail(email.(string), pwd.(string), config.FolderDrafts, []uint32{uid})
		if err != nil {
			c.JSON(200, gin.H{"code": 500, "msg": "删除草稿失败: " + err.Error()})
			return
		}
	}

	c.JSON(200, gin.H{"code": 200, "msg": "发送成功"})
}
