package service

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/textproto"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"email-server/model"
	"email-server/utils"

	"github.com/google/uuid"
)

// Compose 是"撰写邮件"的输入。
type Compose struct {
	From       string
	To         []string
	Cc         []string
	Subject    string
	Body       string
	Files      []*multipart.FileHeader
	InReplyTo  string
	References string
	// Schedule 为本地时间字符串（2006-01-02 15:04:05），空表示立即发送/普通草稿。
	Schedule  string
	PartIDs   string
	SourceUID int64
	Folder    string

	// OriginInlines/OriginAttaches 是草稿续写、回复转发时从原邮件带过来的资源。
	OriginInlines  []model.MailInline
	OriginAttaches []model.MailOriginAttach
}

// BuildRawEmail 构建 MIME 原始邮件报文。
//
// 结构：multipart/mixed（正文 multipart/alternative: text/plain + text/html，
// 以及正文内 data-uri 图片、原邮件内联图片、原邮件附件、本次上传附件）。
func (s *Service) BuildRawEmail(req Compose) ([]byte, error) {
	buf := &bytes.Buffer{}
	mixedBoundary := newBoundary()
	altBoundary := newBoundary()

	writeMixedHeader(buf, s.headerValue(req.From), s.addressList(req.To), s.addressList(req.Cc), req, mixedBoundary)
	buf.WriteString("\r\nThis is a multi-part message in MIME format.\r\n")

	// ===== multipart/alternative 正文 =====
	altHeader := textproto.MIMEHeader{}
	altHeader.Set("Content-Type", fmt.Sprintf("multipart/alternative;\r\n\tboundary=\"%s\"", altBoundary))

	alt, err := writePart(buf, altHeader, mixedBoundary)
	if err != nil {
		return nil, err
	}

	plainHeader := textproto.MIMEHeader{}
	plainHeader.Set("Content-Type", "text/plain;\r\n\tcharset=\"utf-8\"")
	plainHeader.Set("Content-Transfer-Encoding", "base64")
	if plainPart, err := writePart(alt, plainHeader, altBoundary); err == nil {
		plainPart.Write([]byte(encodeBase64Lines([]byte(stripHTML(req.Body)))))
	}

	processedHTML, inlineImages := extractInlineImages(req.Body)

	htmlHeader := textproto.MIMEHeader{}
	htmlHeader.Set("Content-Type", "text/html;\r\n\tcharset=\"utf-8\"")
	htmlHeader.Set("Content-Transfer-Encoding", "base64")
	htmlPart, err := writePart(alt, htmlHeader, altBoundary)
	if err != nil {
		return nil, fmt.Errorf("创建html part失败: %w", err)
	}
	htmlPart.Write([]byte(encodeBase64Lines([]byte(processedHTML))))

	fmt.Fprintf(alt, "\r\n--%s--\r\n", altBoundary)

	// ===== 正文内 data-uri 图片 =====
	for _, img := range inlineImages {
		header := textproto.MIMEHeader{}
		header.Set("Content-Type", fmt.Sprintf("%s;\r\n\tname=\"%s\"", img.ContentType, encodeHeaderName(img.CID)))
		header.Set("Content-Transfer-Encoding", "base64")
		header.Set("Content-ID", fmt.Sprintf("<%s>", img.CID))
		header.Set("Content-Disposition", fmt.Sprintf("inline;\r\n\tfilename=\"%s\"", encodeHeaderName(img.CID)))

		part, err := writePart(buf, header, mixedBoundary)
		if err != nil {
			fmt.Printf("写入HTML提取的内联图片失败 cid=%s err=%v\n", img.CID, err)
			continue
		}
		part.Write([]byte(encodeBase64Lines(img.Content)))
	}

	// ===== 原邮件内联图片 =====
	for _, inline := range req.OriginInlines {
		header := textproto.MIMEHeader{}
		header.Set("Content-Type", fmt.Sprintf("%s;\r\n\tname=\"%s\"", inline.ContentType, encodeHeaderName(inline.FileName)))
		header.Set("Content-Transfer-Encoding", "base64")
		header.Set("Content-ID", fmt.Sprintf("<%s>", inline.CID))
		header.Set("Content-Disposition", fmt.Sprintf("inline;\r\n\tfilename=\"%s\"", encodeHeaderName(inline.FileName)))

		part, err := writePart(buf, header, mixedBoundary)
		if err != nil {
			fmt.Printf("写入内联图片失败 cid=%s err=%v\n", inline.CID, err)
			continue
		}
		part.Write([]byte(encodeBase64Lines(inline.Content)))
	}

	// ===== 原邮件附件 =====
	for _, att := range req.OriginAttaches {
		header := textproto.MIMEHeader{}
		header.Set("Content-Type", fmt.Sprintf("%s;\r\n\tname=\"%s\"", att.ContentType, encodeHeaderName(att.FileName)))
		header.Set("Content-Transfer-Encoding", "base64")
		header.Set("Content-Disposition", fmt.Sprintf("attachment;\r\n\tfilename=\"%s\"", encodeHeaderName(att.FileName)))

		part, err := writePart(buf, header, mixedBoundary)
		if err != nil {
			fmt.Printf("写入预加载附件失败 filename=%s err=%v\n", att.FileName, err)
			continue
		}
		part.Write([]byte(encodeBase64Lines(att.Content)))
	}

	// ===== 本次上传附件 =====
	for _, fileHeader := range req.Files {
		content, err := readUpload(fileHeader)
		if err != nil {
			fmt.Printf("读取上传附件失败: %s err=%v\n", fileHeader.Filename, err)
			continue
		}

		contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(fileHeader.Filename)))
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		header := textproto.MIMEHeader{}
		header.Set("Content-Type", fmt.Sprintf("%s;\r\n\tname=\"%s\"", contentType, encodeHeaderName(fileHeader.Filename)))
		header.Set("Content-Transfer-Encoding", "base64")
		header.Set("Content-Disposition", fmt.Sprintf("attachment;\r\n\tfilename=\"%s\"", encodeHeaderName(fileHeader.Filename)))

		part, err := writePart(buf, header, mixedBoundary)
		if err != nil {
			fmt.Printf("新建附件part失败: %s err=%v\n", fileHeader.Filename, err)
			continue
		}
		part.Write([]byte(encodeBase64Lines(content)))
	}

	fmt.Fprintf(buf, "\r\n--%s--\r\n", mixedBoundary)
	return buf.Bytes(), nil
}

// writeMixedHeader 写入邮件头（顺序与旧实现一致）。
func writeMixedHeader(buf *bytes.Buffer, from string, to, cc []string, req Compose, boundary string) {
	if from != "" {
		fmt.Fprintf(buf, "From: %s\r\n", from)
	}
	fmt.Fprintf(buf, "To: %s\r\n", strings.Join(to, ", "))
	if len(cc) > 0 {
		fmt.Fprintf(buf, "Cc: %s\r\n", strings.Join(cc, ", "))
	}
	fmt.Fprintf(buf, "Subject: %s\r\n", mime.BEncoding.Encode("utf-8", req.Subject))
	fmt.Fprintf(buf, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(buf, "Content-Type: multipart/mixed;\r\n\tboundary=\"%s\"\r\n", boundary)
	fmt.Fprintf(buf, "Content-Transfer-Encoding: 8Bit\r\n")
	fmt.Fprintf(buf, "Date: %s\r\n", time.Now().UTC().Format(time.RFC1123))
	fmt.Fprintf(buf, "X-Priority: 3\r\n")
	fmt.Fprintf(buf, "Message-ID: <%s@%s>\r\n", uuid.NewString(), messageDomain(req.From))
	if req.InReplyTo != "" {
		fmt.Fprintf(buf, "In-Reply-To: %s\r\n", req.InReplyTo)
	}
	if req.References != "" {
		fmt.Fprintf(buf, "References: %s\r\n", req.References)
	}
	if req.Schedule != "" {
		if t, err := parseLocalSchedule(req.Schedule); err == nil {
			fmt.Fprintf(buf, "X-Schedule-Send: %s\r\n", t.UTC().Format(time.RFC1123))
		}
	}
	buf.WriteString("\r\n")
}

// writePart 写入 "\r\n--boundary\r\n" + 头部 + 空行，返回可写正文的 writer。
func writePart(w io.Writer, header textproto.MIMEHeader, boundary string) (io.Writer, error) {
	if _, err := fmt.Fprintf(w, "\r\n--%s\r\n", boundary); err != nil {
		return nil, err
	}
	for key, values := range header {
		for _, value := range values {
			if _, err := fmt.Fprintf(w, "%s: %s\r\n", key, value); err != nil {
				return nil, err
			}
		}
	}
	_, err := w.Write([]byte("\r\n"))
	return w, err
}

// 正文中的 data:image/...;base64 图片。
var dataURIUpper = regexp.MustCompile(`(?i)(src\s*=\s*["'])data:image/([a-zA-Z0-9]+);base64,([^"']+)["']`)

// extractInlineImages 把正文中的 data-uri 图片转换为 cid 引用并返回图片内容。
func extractInlineImages(body string) (string, []model.MailInline) {
	var images []model.MailInline

	newHTML := dataURIUpper.ReplaceAllStringFunc(body, func(match string) string {
		sub := dataURIUpper.FindStringSubmatch(match)
		if len(sub) < 4 {
			return match
		}
		prefix, imgType, b64Data := sub[1], strings.ToLower(sub[2]), sub[3]

		decoded, err := base64.StdEncoding.DecodeString(b64Data)
		if err != nil {
			return match
		}

		cid := uuid.NewString()
		images = append(images, model.MailInline{
			CID:         cid,
			ContentType: "image/" + imgType,
			Content:     decoded,
		})
		return fmt.Sprintf(`%scid:%s"`, prefix, cid)
	})

	return newHTML, images
}

func newBoundary() string {
	return "----=_NextPart_" + strings.ReplaceAll(uuid.NewString(), "-", "_")
}

// encodeBase64Lines 按 76 字符换行编码 base64。
func encodeBase64Lines(data []byte) string {
	raw := base64.StdEncoding.EncodeToString(data)
	var sb strings.Builder
	sb.Grow(len(raw) + len(raw)/76*2)
	for start := 0; start < len(raw); start += 76 {
		end := start + 76
		if end > len(raw) {
			end = len(raw)
		}
		sb.WriteString(raw[start:end])
		sb.WriteString("\r\n")
	}
	return sb.String()
}

// encodeHeaderName 对 MIME 头中的文件名做 RFC2047 编码。
func encodeHeaderName(name string) string {
	return mime.QEncoding.Encode("utf-8", name)
}

// messageDomain 取发件人域名作为 Message-ID 的后缀。
func messageDomain(from string) string {
	idx := strings.LastIndex(from, "@")
	if idx >= 0 && idx < len(from)-1 {
		return from[idx+1:]
	}
	return "localhost"
}

// readUpload 读取上传文件内容。
func readUpload(header *multipart.FileHeader) ([]byte, error) {
	f, err := header.Open()
	if err != nil {
		return nil, fmt.Errorf("打开上传附件失败: %w", err)
	}
	defer f.Close()

	content, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("读取附件内容失败: %w", err)
	}
	return content, nil
}

// parseLocalSchedule 解析前端传入的本地时间；同时兼容 RFC3339。
func parseLocalSchedule(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", value, time.Local); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, value)
}

// headerValue 生成 "姓名 <邮箱>" 形式的头部值；账号目录不可用时回退为原始地址。
func (s *Service) headerValue(email string) string {
	email = strings.TrimSpace(email)
	if email == "" {
		return ""
	}
	display, err := s.dir.DisplayName(email)
	if err != nil || display == "" {
		return email
	}
	return display
}

// addressList 把地址列表格式化为带显示名的头部件。
func (s *Service) addressList(addresses []string) []string {
	out := make([]string, 0, len(addresses))
	for _, addr := range addresses {
		if trimmed := strings.TrimSpace(addr); trimmed != "" {
			out = append(out, s.headerValue(trimmed))
		}
	}
	return out
}

// stripHTML 用于生成 text/plain 备选正文。
func stripHTML(html string) string {
	return utils.StripHTML(html)
}
