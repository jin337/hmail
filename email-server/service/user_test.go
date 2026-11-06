package service

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"email-server/contacts"
	"email-server/directory"
	"email-server/internal/testutil"
	"email-server/model"

	"github.com/golang-jwt/jwt/v5"
)

func TestLoginSignsToken(t *testing.T) {
	svc, _, _, _ := newTestService(t)

	// 准备默认头像，登录时会被复制给账号
	avatarDir := filepath.Join(svc.Config().StaticDir, "avatars")
	if err := os.MkdirAll(avatarDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(avatarDir, "default.webp"), []byte("default"), 0644); err != nil {
		t.Fatal(err)
	}

	user, err := svc.Login("user@example.com", "pwd")
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}
	if user.Email != "user@example.com" || user.FullName != "张三" {
		t.Errorf("登录信息错误: %+v", user)
	}

	claims := &model.UserClaims{}
	parsed, err := jwt.ParseWithClaims(user.Token, claims, func(token *jwt.Token) (any, error) {
		return []byte(svc.Config().JWT.Secret), nil
	})
	if err != nil || !parsed.Valid {
		t.Fatalf("Token 校验失败: %v", err)
	}
	if claims.Email != "user@example.com" || claims.Password != "pwd" {
		t.Errorf("Token 载荷错误: %+v", claims)
	}

	if _, err := os.Stat(filepath.Join(avatarDir, "user@example.com.webp")); err != nil {
		t.Errorf("登录后应复制默认头像: %v", err)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	svc, _, _, _ := newTestService(t)
	if _, err := svc.Login("user@example.com", "bad"); err == nil {
		t.Fatal("错误密码应登录失败")
	}
}

func TestLoginDirectoryUnsupported(t *testing.T) {
	svc, _, _, dir := newTestService(t)
	dir.FailWith = directory.ErrUnsupported

	user, err := svc.Login("user@example.com", "pwd")
	if err != nil {
		t.Fatalf("账号目录不可用时应允许登录: %v", err)
	}
	if user.Email != "user@example.com" {
		t.Errorf("降级账号信息错误: %+v", user)
	}
}

func TestChangePassword(t *testing.T) {
	svc, _, _, _ := newTestService(t)

	if err := svc.ChangePassword("user@example.com", "wrong", "new"); err == nil {
		t.Error("旧密码错误应报错")
	}
	if err := svc.ChangePassword("user@example.com", "pwd", "new"); err != nil {
		t.Errorf("修改密码失败: %v", err)
	}
}

func TestUserListAndManage(t *testing.T) {
	svc, _, _, dir := newTestService(t)
	dir.Add("other@example.com", "李", "四", 1)
	dir.Add("foreign@other.com", "王", "五", 1)

	list, total, err := svc.UserList("user@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 {
		t.Errorf("同域名账号应为 2，实际 %d", total)
	}
	for _, u := range list {
		if u.Email == "foreign@other.com" {
			t.Error("不应包含其它域名账号")
		}
	}

	if err := svc.CreateUser("new@example.com", "pwd", "新", "人", 0); err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	if err := svc.UpdateUser("new@example.com", "改", "名", 1); err != nil {
		t.Fatalf("更新用户失败: %v", err)
	}
	if acc := dir.Accounts["new@example.com"]; acc.FirstName != "改" || acc.AdminLevel != 1 {
		t.Errorf("更新未生效: %+v", acc)
	}
	if err := svc.DeleteUser("new@example.com"); err != nil {
		t.Fatalf("删除用户失败: %v", err)
	}
	if _, ok := dir.Accounts["new@example.com"]; ok {
		t.Error("用户未被删除")
	}

	if err := svc.CreateUser("user@example.com", "x", "", "", 0); err == nil {
		t.Error("重复创建应报错")
	}
}

func TestContacts(t *testing.T) {
	svc, _, _, _ := newTestService(t)

	if err := svc.SaveContact(contacts.PrefixContact, "user@example.com", "a@example.com", "A"); err != nil {
		t.Fatal(err)
	}
	// 自己给自己不记录
	if err := svc.SaveContact(contacts.PrefixContact, "user@example.com", "user@example.com", "me"); err != nil {
		t.Fatal(err)
	}

	list, total, err := svc.ContactList(contacts.PrefixContact, "user@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || list[0].Email != "a@example.com" {
		t.Errorf("联系人列表错误: %+v", list)
	}

	if err := svc.DeleteContact(contacts.PrefixContact, "user@example.com", "a@example.com"); err != nil {
		t.Fatal(err)
	}
	list, total, _ = svc.ContactList(contacts.PrefixContact, "user@example.com")
	if total != 0 {
		t.Errorf("删除后应为空: %+v", list)
	}

	if err := svc.SaveContact(contacts.PrefixContact, "user@example.com", "b@example.com", "B"); err != nil {
		t.Fatal(err)
	}
	if err := svc.ClearContact(contacts.PrefixContact, "user@example.com"); err != nil {
		t.Fatal(err)
	}
	list, total, _ = svc.ContactList(contacts.PrefixContact, "user@example.com")
	if total != 0 {
		t.Errorf("清空后应为空: %+v", list)
	}
}

func TestUploadAvatar(t *testing.T) {
	svc, _, _, _ := newTestService(t)

	pngBytes := []byte("\x89PNG\r\n\x1a\n" + "fake-png-body")

	upload := func(name string, content []byte) *multipart.FileHeader {
		buf, writer := testutil.MultipartWriter()
		part, err := writer.CreateFormFile("file", name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(content); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest("POST", "/api/user/uploadavatar", bytes.NewReader(buf.Bytes()))
		req.Header.Set("Content-Type", writer.FormDataContentType())
		if err := req.ParseMultipartForm(32 << 20); err != nil {
			t.Fatal(err)
		}
		return req.MultipartForm.File["file"][0]
	}

	if err := svc.UploadAvatar("user@example.com", nil); err == nil {
		t.Error("缺少文件应报错")
	}

	if err := svc.UploadAvatar("user@example.com", upload("a.txt", []byte("not an image"))); err == nil {
		t.Error("非图片应报错")
	}

	if err := svc.UploadAvatar("user@example.com", upload("a.png", pngBytes)); err != nil {
		t.Fatalf("上传头像失败: %v", err)
	}
}
