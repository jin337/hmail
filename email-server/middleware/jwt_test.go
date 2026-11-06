package middleware

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"email-server/model"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func token(t *testing.T, secret string, expires time.Time) string {
	t.Helper()
	claims := model.UserClaims{
		Email:    "user@example.com",
		Password: "pwd",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expires),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	return signed
}

func perform(secret string, header string) (*httptest.ResponseRecorder, string, string) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	email, pwd := "", ""
	r.GET("/protected", JWTAuth(secret), func(c *gin.Context) {
		rawEmail, _ := c.Get("userEmail")
		rawPwd, _ := c.Get("userPwd")
		email, _ = rawEmail.(string)
		pwd, _ = rawPwd.(string)
		c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	if header != "" {
		req.Header.Set("Authorization", header)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec, email, pwd
}

func TestJWTAuthWithoutToken(t *testing.T) {
	rec, email, pwd := perform("secret", "")
	if email != "" || pwd != "" {
		t.Fatal("无 Token 不应进入业务处理")
	}
	if rec.Code != 401 {
		t.Errorf("状态码错误: %d", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["msg"] != "未登录，请先登录获取Token" {
		t.Errorf("提示文案错误: %v", body["msg"])
	}
}

func TestJWTAuthValidToken(t *testing.T) {
	secret := "secret"
	tk := token(t, secret, time.Now().Add(time.Hour))

	for _, header := range []string{tk, "Bearer " + tk} {
		rec, email, pwd := perform(secret, header)
		if email == "" || pwd == "" {
			t.Fatalf("合法 Token 应放行（header=%q）", header)
		}
		if rec.Code != 200 {
			t.Errorf("状态码错误: %d", rec.Code)
		}
	}
}

func TestJWTAuthExpiredToken(t *testing.T) {
	secret := "secret"
	rec, email, _pwd := perform(secret, token(t, secret, time.Now().Add(-time.Hour)))
	if email != "" || _pwd != "" {
		t.Fatal("过期 Token 不应放行")
	}
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if !strings.Contains(body["msg"].(string), "已过期") {
		t.Errorf("过期提示错误: %v", body["msg"])
	}
}

func TestJWTAuthWrongSecret(t *testing.T) {
	rec, email, _pwd := perform("secret", token(t, "other", time.Now().Add(time.Hour)))
	if email != "" || _pwd != "" {
		t.Fatal("签名不匹配不应放行")
	}
	if rec.Code != 401 {
		t.Errorf("状态码错误: %d", rec.Code)
	}
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if !strings.HasPrefix(body["msg"].(string), "Token无效") {
		t.Errorf("提示文案错误: %v", body["msg"])
	}
}
