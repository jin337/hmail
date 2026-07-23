package controller

import (
	"email-server/config"
	"email-server/constant"
	"email-server/model"
	"email-server/service"
	"email-server/utils"
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

	// 验证必传参数
	if err := utils.ValidateRequiredParams([]string{"Folder"}, req); err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	// 验证文件夹
	isValidFolder := false
	for _, f := range config.DefaultFolders {
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

	list, total, err := service.MailList(email.(string), pwd.(string), req.Folder, req.Page, req.Size, req.Keyword, req.Filter)
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

// StarMailList 获取星标邮件列表
func StarMailList(c *gin.Context) {
	email, _ := c.Get("userEmail")
	pwd, _ := c.Get("userPwd")

	var req model.MailStarListReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	list, total, err := service.StarMailList(email.(string), pwd.(string), req.Page, req.Size, req.Keyword, req.Filter)
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
		},
	})
}

// MailDetail 获取邮件详情
func MailDetail(c *gin.Context) {
	email, _ := c.Get("userEmail")
	pwd, _ := c.Get("userPwd")
	tokenStr := c.GetHeader("Authorization")

	var req model.MailDetailReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	// 验证必传参数
	if err := utils.ValidateRequiredParams([]string{"Uid", "Folder"}, req); err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": err.Error()})
		return
	}
	host := c.Request.Host
	mailItem, err := service.MailDetail(email.(string), pwd.(string), tokenStr, req.Folder, req.Uid, host)
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

// MarkFlag 标记已读
func MarkFlag(c *gin.Context) {
	email, _ := c.Get("userEmail")
	pwd, _ := c.Get("userPwd")

	var req model.UpdateMailFlagReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	// 验证必传参数
	if err := utils.ValidateRequiredParams([]string{"Uid", "Folder", "Status", "Type"}, req); err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	err := service.UpdateMailFlag(email.(string), pwd.(string), req.Folder, req.Uid, req.Type, req.Status)
	if err != nil {
		c.JSON(200, gin.H{"code": 500, "msg": "更新邮件状态失败: " + err.Error()})
		return
	}
	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "状态修改成功",
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

	// 验证必传参数
	if err := utils.ValidateRequiredParams([]string{"Uid", "Folder", "PartID"}, req); err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": err.Error()})
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

	// 验证必传参数
	if err := utils.ValidateRequiredParams([]string{"FromFolder", "ToFolder", "Uids"}, req); err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	// 参数验证
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

	// 验证必传参数
	if err := utils.ValidateRequiredParams([]string{"Uids", "Folder"}, req); err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": err.Error()})
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

	// in-reply-to
	inReplyTo := c.PostForm("in_reply_to")
	// references
	references := c.PostForm("references")

	files := c.Request.MultipartForm.File["files"]
	if len(files) == 0 {
		files = nil
	}

	// 解析 UID
	var uid uint32
	if uidStr != "" {
		if _, parseErr := fmt.Sscanf(uidStr, "%d", &uid); parseErr != nil {
			c.JSON(200, gin.H{"code": 400, "msg": "无效的 UID 格式"})
			return
		}
	}

	extra := model.EmailExtra{
		InReplyTo:  inReplyTo,
		References: references,
	}
	// 构建邮件
	toList := strings.Split(to, ",")
	ccList := strings.Split(cc, ",")
	raw, err := service.BuildRawEmail(email.(string), pwd.(string), config.FolderDrafts, int64(uid), partIds, []string{email.(string)}, toList, ccList, subject, content, files, extra)
	if err != nil {
		c.JSON(200, gin.H{"code": 500, "msg": "构建邮件失败", "err": err.Error()})
		return
	}

	// 保存联系人
	for _, to := range toList {
		name, mail, err := utils.GetMailName(config.GetConfig(constant.AdminPassword), to)
		if err != nil {
			fmt.Printf("获取联系人名称失败: %v，使用邮箱前缀作为默认名称\n", err)
		}
		_ = service.SaveContact("user_sent", email.(string), mail, name)
	}

	// 更新草稿
	if uidStr != "" {
		err := service.UpdateDraft(email.(string), pwd.(string), config.FolderDrafts, raw, int64(uid))
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

	// in-reply-to
	inReplyTo := c.PostForm("in_reply_to")
	// references
	references := c.PostForm("references")

	// 定时
	xScheduleSend := c.PostForm("x-schedule-send")

	files := c.Request.MultipartForm.File["files"]
	if len(files) == 0 {
		files = nil
	}

	// 解析 UID
	var uid int64
	if uidStr != "" {
		if _, parseErr := fmt.Sscanf(uidStr, "%d", &uid); parseErr != nil {
			c.JSON(200, gin.H{"code": 400, "msg": "无效的 UID 格式"})
			return
		}
	}

	extra := model.EmailExtra{
		InReplyTo:     inReplyTo,
		References:    references,
		XScheduleSend: xScheduleSend,
	}

	// 构建邮件内容
	toList := strings.Split(to, ",")
	ccList := strings.Split(cc, ",")
	raw, err := service.BuildRawEmail(email.(string), pwd.(string), config.FolderDrafts, int64(uid), partIds, []string{email.(string)}, toList, ccList, subject, content, files, extra)
	if err != nil {
		c.JSON(200, gin.H{"code": 500, "msg": "构建邮件失败", "err": err.Error()})
		return
	}

	// 发送邮件
	if err := service.ScheduleSendEmail(email.(string), pwd.(string), toList, ccList, raw); err != nil {
		c.JSON(200, gin.H{"code": 500, "msg": "发送失败", "err": err.Error()})
		return
	}

	// 如果是定时发送，不立即保存到已发送，而是保存到草稿箱
	if xScheduleSend == "" {
		// 立即发送：存入已发送
		err = service.SaveMailToFolder(email.(string), pwd.(string), config.FolderSent, raw)
		if err != nil {
			c.JSON(200, gin.H{"code": 500, "msg": "发送成功，存入已发送失败", "err": err.Error()})
			return
		}

		// 删除草稿
		if uidStr != "" {
			err = service.DeleteMail(email.(string), pwd.(string), config.FolderDrafts, []int64{uid})
			if err != nil {
				c.JSON(200, gin.H{"code": 500, "msg": "删除草稿失败: " + err.Error()})
				return
			}
		}
	} else {
		// 定时发送：存入草稿箱，等待定时发送后再移动到已发送
		err = service.SaveMailToFolder(email.(string), pwd.(string), config.FolderDrafts, raw)
		if err != nil {
			c.JSON(200, gin.H{"code": 500, "msg": "定时发送已设置，但保存草稿失败", "err": err.Error()})
			return
		}

		// 增加定时标签
		messageID := utils.GetExtractHeader(raw, "Message-ID")

		// 发送成功后，将邮件从草稿箱移动到已发送文件夹
		if messageID != "" {
			if uid, err := utils.GetUid(email.(string), pwd.(string), messageID, config.FolderDrafts); err == nil {
				// 添加重要标签
				if err = service.UpdateMailFlag(email.(string), pwd.(string), config.FolderDrafts, uid, 1, "Draft"); err != nil {
					fmt.Printf("标记邮件失败: %v\n", err)
				}
			}
		} else {
			fmt.Printf("未找到 Message-ID，跳过移动操作\n")
		}
	}

	// 保存联系人
	for _, to := range toList {
		name, mail, err := utils.GetMailName(config.GetConfig(constant.AdminPassword), to)
		if err != nil {
			fmt.Printf("获取联系人名称失败: %v，使用邮箱前缀作为默认名称\n", err)
		}
		_ = service.SaveContact("user_sent", email.(string), mail, name)
	}

	c.JSON(200, gin.H{"code": 200, "msg": "发送成功"})
}

// UnScheduleEmail 取消定时发送
func UnScheduleEmail(c *gin.Context) {
	email, _ := c.Get("userEmail")
	pwd, _ := c.Get("userPwd")

	var req model.UpdateMailFlagReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	// 验证必传参数
	if err := utils.ValidateRequiredParams([]string{"Uid", "Folder", "Status", "Type"}, req); err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	err := service.UnScheduleEmail(email.(string), pwd.(string), req.Folder, req.Uid, req.Type, req.Status)
	if err != nil {
		c.JSON(200, gin.H{"code": 500, "msg": "取消定时失败: " + err.Error()})
		return
	}
	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "修改成功",
	})
}
