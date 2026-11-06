package controller

import (
	"email-server/apperr"
	"email-server/model"
	"email-server/utils"

	"github.com/gin-gonic/gin"
)

// Login 登录。
func (h *Handler) Login(c *gin.Context) {
	var req model.LoginReq
	if !bindJSON(c, &req) {
		return
	}
	if err := utils.CheckRequired(
		utils.Required("Email", req.Email),
		utils.Required("Password", req.Password),
	); err != nil {
		fail(c, apperr.CodeBadRequest, err.Error())
		return
	}

	user, err := h.svc.Login(req.Email, req.Password)
	if err != nil {
		fail(c, apperr.CodeInternal, err.Error())
		return
	}
	ok(c, user, "登录成功")
}

// ChangePassword 修改密码。
func (h *Handler) ChangePassword(c *gin.Context) {
	email, _ := currentUser(c)

	var req model.PasswordReq
	if !bindJSON(c, &req) {
		return
	}
	if err := utils.CheckRequired(
		utils.Required("OldPassword", req.OldPassword),
		utils.Required("NewPassword", req.NewPassword),
	); err != nil {
		fail(c, apperr.CodeBadRequest, err.Error())
		return
	}

	if err := h.svc.ChangePassword(email, req.OldPassword, req.NewPassword); err != nil {
		fail(c, apperr.CodeInternal, err.Error())
		return
	}
	okMsg(c, "修改密码成功")
}

// UserList 获取用户列表。
func (h *Handler) UserList(c *gin.Context) {
	email, _ := currentUser(c)

	list, total, err := h.svc.UserList(email)
	if err != nil {
		fail(c, apperr.CodeInternal, "获取用户列表失败: "+err.Error())
		return
	}
	ok(c, gin.H{"list": list, "total": total}, "success")
}

// CreateUser 创建用户。
func (h *Handler) CreateUser(c *gin.Context) {
	var req model.UserReq
	if !bindJSON(c, &req) {
		return
	}
	if err := utils.CheckRequired(
		utils.Required("Email", req.Email),
		utils.Required("Password", req.Password),
	); err != nil {
		fail(c, apperr.CodeBadRequest, err.Error())
		return
	}

	if err := h.svc.CreateUser(req.Email, req.Password, req.PersonFirstName, req.PersonLastName, req.IsAdmin); err != nil {
		fail(c, apperr.CodeInternal, err.Error())
		return
	}
	okMsg(c, "创建用户成功")
}

// DeleteUser 删除用户。
func (h *Handler) DeleteUser(c *gin.Context) {
	var req model.UserDeleteReq
	if !bindJSON(c, &req) {
		return
	}
	if err := utils.CheckRequired(utils.Required("Email", req.Email)); err != nil {
		fail(c, apperr.CodeBadRequest, err.Error())
		return
	}

	if err := h.svc.DeleteUser(req.Email); err != nil {
		fail(c, apperr.CodeInternal, err.Error())
		return
	}
	okMsg(c, "删除用户成功")
}

// UpdateUser 更新用户。
func (h *Handler) UpdateUser(c *gin.Context) {
	var req model.UserUpdateReq
	if !bindJSON(c, &req) {
		return
	}
	if err := utils.CheckRequired(utils.Required("Email", req.Email)); err != nil {
		fail(c, apperr.CodeBadRequest, err.Error())
		return
	}

	if err := h.svc.UpdateUser(req.Email, req.PersonFirstName, req.PersonLastName, req.IsAdmin); err != nil {
		fail(c, apperr.CodeInternal, err.Error())
		return
	}
	okMsg(c, "更新用户成功")
}

// ContactList 获取联系人列表。
func (h *Handler) ContactList(c *gin.Context) {
	email, _ := currentUser(c)

	var req model.ContactReq
	if !bindJSON(c, &req) {
		return
	}

	list, total, err := h.svc.ContactList(req.Prefix, email)
	if err != nil {
		fail(c, apperr.CodeInternal, "获取联系人列表失败: "+err.Error())
		return
	}
	ok(c, gin.H{"list": list, "total": total}, "success")
}

// SaveContact 保存联系人。
func (h *Handler) SaveContact(c *gin.Context) {
	email, _ := currentUser(c)

	var req model.ContactReq
	if !bindJSON(c, &req) {
		return
	}
	if err := utils.CheckRequired(
		utils.Required("Email", req.Email),
		utils.Required("Name", req.Name),
	); err != nil {
		fail(c, apperr.CodeBadRequest, err.Error())
		return
	}

	if err := h.svc.SaveContact(req.Prefix, email, req.Email, req.Name); err != nil {
		fail(c, apperr.CodeInternal, "保存联系人失败: "+err.Error())
		return
	}
	okMsg(c, "保存成功")
}

// DeleteContact 删除联系人。
func (h *Handler) DeleteContact(c *gin.Context) {
	email, _ := currentUser(c)

	var req model.ContactReq
	if !bindJSON(c, &req) {
		return
	}
	if err := utils.CheckRequired(utils.Required("Email", req.Email)); err != nil {
		fail(c, apperr.CodeBadRequest, err.Error())
		return
	}

	if err := h.svc.DeleteContact(req.Prefix, email, req.Email); err != nil {
		fail(c, apperr.CodeInternal, "删除联系人失败: "+err.Error())
		return
	}
	okMsg(c, "删除成功")
}

// ClearContact 清空联系人。
func (h *Handler) ClearContact(c *gin.Context) {
	email, _ := currentUser(c)

	var req model.ContactReq
	if !bindJSON(c, &req) {
		return
	}

	if err := h.svc.ClearContact(req.Prefix, email); err != nil {
		fail(c, apperr.CodeInternal, "清空联系人失败: "+err.Error())
		return
	}
	okMsg(c, "删除成功")
}

// UploadAvatar 上传头像。
func (h *Handler) UploadAvatar(c *gin.Context) {
	email, _ := currentUser(c)

	file, _ := c.FormFile("file")
	if err := h.svc.UploadAvatar(email, file); err != nil {
		fail(c, apperr.CodeInternal, err.Error())
		return
	}
	okMsg(c, "更新头像成功")
}
