package controller

import (
	"email-server/config"
	"email-server/constant"
	"email-server/model"
	"email-server/service"
	"email-server/utils"

	"github.com/gin-gonic/gin"
)

// Login 登录
func Login(c *gin.Context) {
	// 验证参数
	var req model.LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	// 验证必传参数
	if err := utils.ValidateRequiredParams([]string{"Email", "Password"}, req); err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	// 获取用户信息
	user, err := service.Login(config.GetConfig(constant.AdminPassword), req.Email, req.Password, config.DefaultFolders)
	if err != nil {
		c.JSON(200, gin.H{"code": 500, "msg": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "登录成功",
		"data": user,
	})
}

// ChangePassword 修改密码
func ChangePassword(c *gin.Context) {
	email, _ := c.Get("userEmail")

	var req model.PasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": "参数错误"})
		return
	}
	// 验证必传参数
	if err := utils.ValidateRequiredParams([]string{"OldPassword", "NewPassword"}, req); err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	err := service.UpdatePassword(config.GetConfig(constant.AdminPassword), email.(string), req.OldPassword, req.NewPassword)
	if err != nil {
		c.JSON(200, gin.H{"code": 500, "msg": err.Error()})
		return
	}
	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "修改密码成功",
	})
}

// UserList 获取用户列表
func UserList(c *gin.Context) {
	email, _ := c.Get("userEmail")
	list, total, err := service.UserList(config.GetConfig(constant.AdminPassword), email.(string))
	if err != nil {
		c.JSON(200, gin.H{"code": 500, "msg": "获取用户列表失败: " + err.Error()})
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

// CreateUser 创建用户
func CreateUser(c *gin.Context) {
	var req model.UserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	// 验证必传参数
	if err := utils.ValidateRequiredParams([]string{"Email", "Password"}, req); err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	err := service.CreateUser(config.DefaultFolders, config.GetConfig(constant.AdminPassword), req.Email, req.Password, req.PersonFirstName, req.PersonLastName, req.IsAdmin)
	if err != nil {
		c.JSON(200, gin.H{"code": 500, "msg": err.Error()})
		return
	}
	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "创建用户成功",
	})
}

// DeleteUser 删除用户
func DeleteUser(c *gin.Context) {
	var req model.UserDeleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	// 验证必传参数
	if err := utils.ValidateRequiredParams([]string{"Email"}, req); err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	err := service.DeleteUser(config.GetConfig(constant.AdminPassword), req.Email)
	if err != nil {
		c.JSON(200, gin.H{"code": 500, "msg": err.Error()})
		return
	}
	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "删除用户成功",
	})
}

// UpdateUser 更新用户
func UpdateUser(c *gin.Context) {
	var req model.UserUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	// 验证必传参数
	if err := utils.ValidateRequiredParams([]string{"Email"}, req); err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	err := service.UpdateUser(config.GetConfig(constant.AdminPassword), req.Email, req.PersonFirstName, req.PersonLastName, req.IsAdmin)
	if err != nil {
		c.JSON(200, gin.H{"code": 500, "msg": err.Error()})
		return
	}
	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "更新用户成功",
	})
}

// ContactList 获取联系人列表
func ContactList(c *gin.Context) {
	email, _ := c.Get("userEmail")

	var req model.ContactReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	list, total, err := service.ContactList(req.Prefix, email.(string))
	if err != nil {
		c.JSON(200, gin.H{"code": 500, "msg": "获取联系人列表失败: " + err.Error()})
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

// SaveContact	保存联系人
func SaveContact(c *gin.Context) {
	email, _ := c.Get("userEmail")

	var req model.ContactReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	// 验证必传参数
	if err := utils.ValidateRequiredParams([]string{"Email", "Name"}, req); err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	err := service.SaveContact(req.Prefix, email.(string), req.Email, req.Name)
	if err != nil {
		c.JSON(200, gin.H{"code": 500, "msg": "保存联系人失败: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "保存成功",
	})
}

// DeleteContact 删除联系人
func DeleteContact(c *gin.Context) {
	email, _ := c.Get("userEmail")

	var req model.ContactReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	// 验证必传参数
	if err := utils.ValidateRequiredParams([]string{"Email"}, req); err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	err := service.DeleteContact(req.Prefix, email.(string), req.Email)
	if err != nil {
		c.JSON(200, gin.H{"code": 500, "msg": "删除联系人失败: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "删除成功",
	})
}

// ClearContact 清空联系人
func ClearContact(c *gin.Context) {
	email, _ := c.Get("userEmail")

	var req model.ContactReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	err := service.ClearContact(req.Prefix, email.(string))
	if err != nil {
		c.JSON(200, gin.H{"code": 500, "msg": "清空联系人失败: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "删除成功",
	})
}
