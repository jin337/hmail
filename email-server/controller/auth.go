package controller

import (
	"time"

	"email-server/config"
	"email-server/model"
	"email-server/service"
	"email-server/utils"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Login 登录
func Login(c *gin.Context) {
	var req model.LoginReq

	// 验证参数
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	// 验证用户
	imapClient, err := utils.DialIMAPClient(req.Email, req.Password)
	if err != nil {
		c.JSON(200, gin.H{"code": 500, "msg": err.Error()})
		return
	}

	defer imapClient.Logout()

	// 生成token
	claims := model.UserClaims{
		Email:    req.Email,
		Password: req.Password,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * config.JwtExpireHour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(config.JwtSecretKey))
	if err != nil {
		c.JSON(200, gin.H{"code": 500, "msg": "生成Token失败"})
		return
	}

	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "登录成功",
		"data": gin.H{
			"email": req.Email,
			"token": tokenStr,
		}})
}

// ChangePassword 修改密码
func ChangePassword(c *gin.Context) {
	email, _ := c.Get("userEmail")

	var req model.PasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{"code": 400, "msg": "参数错误"})
		return
	}
	if req.OldPwd == "" || req.NewPwd == "" {
		c.JSON(200, gin.H{"code": 400, "msg": "旧密码或新密码参数不能为空"})
		return
	}

	err := service.UpdatePassword(config.AdminPwd, email.(string), req.OldPwd, req.NewPwd)
	if err != nil {
		c.JSON(200, gin.H{"code": 500, "msg": err.Error()})
		return
	}
	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "修改密码成功",
	})
}
