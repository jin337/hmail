package controller

import (
	"email-server/config"
	"email-server/constant"
	"email-server/model"
	"email-server/service"
	"email-server/utils"
	"fmt"
	"net/url"
	"regexp"
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

	host := c.GetHeader("X-Client-Host")
	// 区分客户端请求源
	envValue := config.GetConfig(constant.GinMode)
	if envValue == "debug" {
		host = c.Request.Host
	}

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

// MarkFlag 标记状态
func MarkFlag(c *gin.Context) {
	email, _ := c.Get("userEmail")
	pwd, _ := c.Get("userPwd")

	var req model.UpdateMailFlagReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	// 验证必传参数
	if err := utils.ValidateRequiredParams([]string{"Uids", "Folder", "Status", "Type"}, req); err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	err := service.UpdateMailFlag(email.(string), pwd.(string), req.Folder, req.Uids, req.Type, req.Status)
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

// emailBuildResult 邮件构建结果，供 SendEmail 和 SaveDraft 共用
type emailBuildResult struct {
	raw           []byte
	toList        []string
	ccList        []string
	uid           int64
	uidStr        string
	folder        string
	xScheduleSend string
}

// buildEmailFromRequest 从请求中解析参数并构建 raw 邮件，供 SendEmail 和 SaveDraft 共用
func buildEmailFromRequest(c *gin.Context, email, pwd string, stripDataHref bool) (*emailBuildResult, error) {
	to := c.PostForm("to")
	cc := c.PostForm("cc")
	subject := c.PostForm("subject")
	content := c.PostForm("content")

	// 移除 data-href 属性
	var imgDataHrefRegex = regexp.MustCompile(`\s+data-href=["'][^"']*["']`)
	if stripDataHref {
		content = imgDataHrefRegex.ReplaceAllString(content, "")
	}

	files := c.Request.MultipartForm.File["files"]
	if len(files) == 0 {
		files = nil
	}

	inReplyTo := c.PostForm("in_reply_to")
	references := c.PostForm("references")
	xScheduleSend := c.PostForm("x-schedule-send")
	folder := c.PostForm("folder")
	partIds := c.PostForm("part_ids")
	uidStr := c.PostForm("uid")

	var uid int64
	if uidStr != "" {
		if _, parseErr := fmt.Sscanf(uidStr, "%d", &uid); parseErr != nil {
			return nil, fmt.Errorf("无效的 UID 格式")
		}
	}

	// 获取旧邮件资源
	var inlineList []model.MailInline
	var keepAttachList []model.MailOriginAttach
	if uidStr != "" && folder != "" {
		inline, attach, err := utils.GetMailResource(email, pwd, folder, uid, partIds)
		if err != nil {
			return nil, fmt.Errorf("获取旧邮件资源失败: %w", err)
		}
		inlineList = inline
		keepAttachList = attach
	}

	// 构建邮件内容
	toList := strings.Split(to, ",")
	var ccList []string
	if cc != "" {
		ccList = strings.Split(cc, ",")
	}
	extra := model.EmailExtra{
		InReplyTo:      inReplyTo,
		References:     references,
		XScheduleSend:  xScheduleSend,
		OriginInlines:  inlineList,
		OriginAttaches: keepAttachList,
	}
	raw, err := service.BuildRawEmail(email, pwd, []string{email}, toList, ccList, subject, content, files, extra)
	if err != nil {
		return nil, fmt.Errorf("构建邮件失败: %w", err)
	}

	return &emailBuildResult{
		raw:           raw,
		toList:        toList,
		ccList:        ccList,
		uid:           uid,
		uidStr:        uidStr,
		folder:        folder,
		xScheduleSend: xScheduleSend,
	}, nil
}

// saveContactsAsync 异步保存联系人
func saveContactsAsync(toList []string, userEmail string) {
	go func() {
		for _, to := range toList {
			name, mail, err := utils.GetMailName(config.GetConfig(constant.AdminPassword), to)
			if err != nil {
				fmt.Printf("获取联系人名称失败: %v，使用邮箱前缀作为默认名称\n", err)
			}
			_ = service.SaveContact("user_sent", userEmail, mail, name)
		}
	}()
}

// SaveDraft 保存草稿
func SaveDraft(c *gin.Context) {
	email, _ := c.Get("userEmail")
	pwd, _ := c.Get("userPwd")

	result, err := buildEmailFromRequest(c, email.(string), pwd.(string), false)
	if err != nil {
		c.JSON(200, gin.H{"code": 500, "msg": err.Error()})
		return
	}

	// 异步保存联系人
	saveContactsAsync(result.toList, email.(string))

	// 更新草稿
	if result.uidStr != "" {
		err := service.UpdateDraft(email.(string), pwd.(string), config.FolderDrafts, result.raw, result.uid)
		if err != nil {
			c.JSON(200, gin.H{"code": 500, "msg": "更新草稿失败", "err": err.Error()})
			return
		}
		c.JSON(200, gin.H{"code": 200, "msg": "草稿更新成功", "uid": result.uid})
	} else {
		// 新建草稿
		err := service.SaveMailToFolder(email.(string), pwd.(string), config.FolderDrafts, result.raw)
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

	result, err := buildEmailFromRequest(c, email.(string), pwd.(string), true)
	if err != nil {
		c.JSON(200, gin.H{"code": 500, "msg": err.Error()})
		return
	}

	// 参数校验：收件人不能为空
	if len(result.toList) == 0 || (len(result.toList) == 1 && strings.TrimSpace(result.toList[0]) == "") {
		c.JSON(200, gin.H{"code": 400, "msg": "收件人不能为空"})
		return
	}

	if result.xScheduleSend == "" {
		// 立即发送
		if err := service.ScheduleSendEmail(email.(string), pwd.(string), result.toList, result.ccList, result.raw); err != nil {
			c.JSON(200, gin.H{"code": 500, "msg": "发送失败", "err": err.Error()})
			return
		}

		var warnMsg string
		if result.uidStr != "" {
			// 从草稿发送：先移除 Draft 标记，再 MOVE 到已发送（原子操作，保留原始邮件）
			if err := service.UpdateMailFlag(email.(string), pwd.(string), config.FolderDrafts, []int64{result.uid}, 2, "Draft"); err != nil {
				fmt.Printf("移除 Draft 标记失败: %v\n", err)
			}
			if err := service.MoveMail(email.(string), pwd.(string), config.FolderDrafts, config.FolderSent, []int64{result.uid}); err != nil {
				warnMsg = "，但移动到已发送失败"
				fmt.Printf("移动到已发送失败: %v\n", err)
			}
		} else {
			// 新建邮件直接发送：存入已发送
			if err := service.SaveMailToFolder(email.(string), pwd.(string), config.FolderSent, result.raw); err != nil {
				warnMsg = "，但存入已发送失败"
				fmt.Printf("存入已发送失败: %v\n", err)
			}
		}

		c.JSON(200, gin.H{"code": 200, "msg": "发送成功" + warnMsg})
	} else {
		// 定时发送：先启动定时任务，再存草稿，最后打标签
		if err := service.ScheduleSendEmail(email.(string), pwd.(string), result.toList, result.ccList, result.raw); err != nil {
			c.JSON(200, gin.H{"code": 500, "msg": "定时发送设置失败", "err": err.Error()})
			return
		}

		// 存入草稿箱
		if err := service.SaveMailToFolder(email.(string), pwd.(string), config.FolderDrafts, result.raw); err != nil {
			c.JSON(200, gin.H{"code": 500, "msg": "定时发送已设置，但保存草稿失败", "err": err.Error()})
			return
		}

		// 给定时邮件打 Draft 标签，通过 Message-ID 反查 UID
		messageID := utils.GetExtractHeader(result.raw, "Message-ID")
		if messageID != "" {
			if scheduleUID, err := utils.GetUid(email.(string), pwd.(string), messageID, config.FolderDrafts); err == nil {
				if err = service.UpdateMailFlag(email.(string), pwd.(string), config.FolderDrafts, []int64{scheduleUID}, 1, "Draft"); err != nil {
					fmt.Printf("标记定时邮件失败: %v\n", err)
				}
			} else {
				fmt.Printf("定时邮件打标签失败：未找到对应草稿 UID: %v\n", err)
			}
		} else {
			fmt.Printf("未找到 Message-ID，跳过定时邮件标记\n")
		}

		c.JSON(200, gin.H{"code": 200, "msg": "定时发送已设置"})
	}

	// 异步保存联系人
	saveContactsAsync(result.toList, email.(string))
}

// UnScheduleEmail 取消定时发送
func UnScheduleEmail(c *gin.Context) {
	email, _ := c.Get("userEmail")
	pwd, _ := c.Get("userPwd")

	var req model.ScheduleMailFlagReq
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

// UnreadMailTotal 获取未读邮件数量
func UnreadMailTotal(c *gin.Context) {
	email, _ := c.Get("userEmail")
	pwd, _ := c.Get("userPwd")

	total, err := service.UnreadMailTotal(email.(string), pwd.(string))
	if err != nil {
		c.JSON(200, gin.H{"code": 500, "msg": "获取未读邮件数量失败: " + err.Error()})
		return
	}
	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "success",
		"data": total,
	})
}
