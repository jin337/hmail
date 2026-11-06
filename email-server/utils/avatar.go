package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// AvatarDir 返回头像目录。
func AvatarDir(staticDir string) string {
	return filepath.Join(staticDir, "avatars")
}

// EnsureAvatar 保证头像目录存在，并为没有头像的账号复制默认头像。
func EnsureAvatar(staticDir string, accounts []string) error {
	dir := AvatarDir(staticDir)
	defaultPath := filepath.Join(dir, "default.webp")

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建头像目录失败: %w", err)
	}

	info, err := os.Stat(defaultPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("默认头像 default.webp 不存在，请检查文件")
		}
		return fmt.Errorf("读取默认头像状态失败: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("默认头像路径是文件夹，非法")
	}

	for _, account := range accounts {
		target := filepath.Join(dir, fmt.Sprintf("%s.webp", account))
		stat, err := os.Stat(target)
		if err != nil {
			if os.IsNotExist(err) {
				if err := copyFile(defaultPath, target); err != nil {
					return fmt.Errorf("账号[%s]复制默认头像失败: %w", account, err)
				}
				continue
			}
			return fmt.Errorf("检查账号[%s]头像状态失败: %w", account, err)
		}
		if stat.IsDir() {
			return fmt.Errorf("账号[%s]头像路径是文件夹，非法", account)
		}
	}

	return nil
}

// SaveAvatar 把头像内容写入指定目录，返回写入后的路径。
func SaveAvatar(staticDir, email string, content []byte) (string, error) {
	dir := AvatarDir(staticDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("创建静态目录失败: %v", err)
	}
	path := filepath.Join(dir, fmt.Sprintf("%s%s", email, ".webp"))
	if err := os.WriteFile(path, content, 0644); err != nil {
		return "", fmt.Errorf("保存头像文件失败: %v", err)
	}
	return path, nil
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}
