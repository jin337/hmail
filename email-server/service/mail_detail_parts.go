package service

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/quotedprintable"
	"path/filepath"
	"strings"

	"email-server/mailbox"
	"email-server/model"
	"email-server/utils"

	"github.com/jhillyerd/enmime"
)

// detailSmallSectionBytes 小于该编码尺寸的附件直接整段抓取，保证字节级精确。
const detailSmallSectionBytes = 4096

// detailHeadSampleBytes / detailTailSampleBytes 用于推断 base64 附件解码后大小。
// 尾部样本需覆盖一整行（base64 常按 76 列折行）及其前面的换行。
const (
	detailHeadSampleBytes = 200
	detailTailSampleBytes = 120
)

// mailDetailParts 用 BODYSTRUCTURE + 分段抓取组装详情。
// 返回 ok=false 时由调用方回退到整封解析，保证语义与兼容性。
func (s *Service) mailDetailParts(email string, uid int64, client mailbox.Client, host string) (*model.MailDetail, bool) {
	root, err := client.FetchStructure(uint32(uid))
	if err != nil || root == nil {
		return nil, false
	}
	plan, ok := buildDetailPlan(root)
	if !ok {
		return nil, false
	}

	bodies, err := client.FetchSections(uint32(uid), plan.sections)
	if err != nil {
		return nil, false
	}

	// 尺寸推断失败的 base64 附件再补抓整段，避免展示尺寸出现偏差。
	var needExact []*mailbox.Part
	for _, part := range plan.attachments {
		if part.Disposition == "inline" || !strings.EqualFold(part.Encoding, "base64") || part.Size <= detailSmallSectionBytes {
			continue
		}
		if _, fetched := bodies[part.Path]; fetched {
			continue
		}
		head := bodies[mailbox.Section{Path: part.Path, Offset: 0, Length: detailHeadSampleBytes}.Key()]
		tail := bodies[mailbox.Section{Path: part.Path, Offset: part.Size - detailTailSampleBytes, Length: detailTailSampleBytes}.Key()]
		if _, ok := base64DecodedSize(part.Size, head, tail); !ok {
			needExact = append(needExact, part)
		}
	}
	if len(needExact) > 0 {
		extra := make([]mailbox.Section, 0, len(needExact))
		for _, part := range needExact {
			extra = append(extra, mailbox.Section{Path: part.Path})
		}
		more, err := client.FetchSections(uint32(uid), extra)
		if err != nil {
			return nil, false
		}
		for key, value := range more {
			bodies[key] = value
		}
	}

	decoded := make(map[string]string)
	rawContents := make(map[string][]byte)
	for _, part := range plan.textBodies {
		raw, ok := bodies[part.Path]
		if !ok {
			return nil, false
		}
		var content []byte
		var err error
		// 单段正文与 enmime 的 parseTextOnlyBody 走同一套字符集处理（含 HTML meta 探测）。
		if (!root.IsMultipart() && part == root) || !detailCharsetSupported(part) {
			content, err = decodeDetailTextExact(part, raw)
		} else {
			content, err = decodeDetailBody(part, raw)
		}
		if err != nil {
			return nil, false
		}
		decoded[part.Path] = string(content)
		rawContents[part.Path] = content
	}
	for _, part := range plan.contentParts {
		raw, ok := bodies[part.Path]
		if !ok {
			return nil, false
		}
		var content []byte
		var err error
		if strings.EqualFold(part.Encoding, "quoted-printable") {
			content, err = decodeDetailPartExact(part, raw)
		} else {
			content, err = decodeDetailBody(part, raw)
		}
		if err != nil {
			return nil, false
		}
		rawContents[part.Path] = content
	}

	content := selectDetailContent(root, decoded)

	cidMap := make(map[string]string)
	attachments := make([]model.AttachmentInfo, 0, len(plan.attachments)+1)
	dir := filepath.Join(s.cfg.StaticDir, "images", safePathSegment(email), fmt.Sprint(uid))
	var totalSize int64

	// 单段邮件：enmime 把非正文根部件当作附件（PartID 为 0）或内联内容。
	if plan.rootAttachment != nil {
		part := plan.rootAttachment
		size, ok := detailAttachmentSize(part, bodies, rawContents)
		if !ok {
			return nil, false
		}
		totalSize += size
		attachments = append(attachments, model.AttachmentInfo{
			PartID:      "0",
			FileName:    part.FileName,
			ContentType: part.Type,
			FileType:    utils.FileExtension(part.FileName),
			Size:        utils.FormatUnitSize(size),
		})
	}
	if plan.rootInline != nil {
		part := plan.rootInline
		content := rawContents[part.Path]
		totalSize += int64(len(content))
		if part.ContentID != "" && len(content) > 0 {
			ext := filepath.Ext(part.FileName)
			if ext == "" {
				ext = extFromContentType(part.Type)
			}
			saveInline(dir, "0", ext, content, func(localPath, fileName string) {
				cidMap[part.ContentID] = s.staticURL(host, localPath, fileName)
			})
		}
	}

	for _, part := range plan.attachments {
		size, ok := detailAttachmentSize(part, bodies, rawContents)
		if !ok {
			return nil, false
		}
		totalSize += size

		if part.Disposition == "attachment" {
			attachments = append(attachments, model.AttachmentInfo{
				PartID:      part.Path,
				FileName:    part.FileName,
				ContentType: part.Type,
				FileType:    utils.FileExtension(part.FileName),
				Size:        utils.FormatUnitSize(size),
			})
			continue
		}

		// 非 attachment 且带 Content-ID 的部分按内联图片处理（与整封解析路径一致）。
		content := rawContents[part.Path]
		if part.ContentID == "" || len(content) == 0 {
			continue
		}
		saveInline(dir, part.Path, filepath.Ext(part.FileName), content, func(localPath, fileName string) {
			cidMap[part.ContentID] = s.staticURL(host, localPath, fileName)
		})
	}

	for _, part := range plan.inlines {
		content := rawContents[part.Path]
		totalSize += int64(len(content))
		if part.ContentID == "" || len(content) == 0 {
			continue
		}
		ext := filepath.Ext(part.FileName)
		if ext == "" {
			ext = extFromContentType(part.Type)
		}
		saveInline(dir, part.Path, ext, content, func(localPath, fileName string) {
			cidMap[part.ContentID] = s.staticURL(host, localPath, fileName)
		})
	}

	// 给 cid 图片追加 data-href，保留原始 src（与整封解析路径一致）。
	content = cidImagePattern.ReplaceAllStringFunc(content, func(match string) string {
		sub := cidImagePattern.FindStringSubmatch(match)
		if len(sub) != 4 {
			return match
		}
		target, ok := cidMap[sub[2]]
		if !ok {
			return match
		}
		return fmt.Sprintf(`<img%s src="cid:%s" data-href="%s"%s>`, sub[1], sub[2], target, sub[3])
	})

	return &model.MailDetail{
		Content:     content,
		Attachments: attachments,
		AttachSize:  utils.FormatUnitSize(totalSize),
	}, true
}

// detailPlan 描述分段抓取需要请求哪些 section、哪些部件参与正文/附件/内联图。
type detailPlan struct {
	sections     []mailbox.Section
	textBodies   []*mailbox.Part
	contentParts []*mailbox.Part
	attachments  []*mailbox.Part
	inlines      []*mailbox.Part
	// 单段邮件的根部件：既不是文本也不是 multipart 时，enmime 把它当附件或内联内容。
	rootAttachment *mailbox.Part
	rootInline     *mailbox.Part
}

// buildDetailPlan 校验结构是否适合分段路径，并列出抓取计划。
func buildDetailPlan(root *mailbox.Part) (*detailPlan, bool) {
	if !detailStructureSupported(root) {
		return nil, false
	}

	plan := &detailPlan{}

	// 单段邮件：根既可能是正文，也可能是附件/内联内容。
	if !root.IsMultipart() {
		switch {
		case isDetailTextPart(root):
			plan.textBodies = append(plan.textBodies, root)
			plan.sections = append(plan.sections, mailbox.Section{Path: root.Path})
		case root.Disposition == "inline":
			plan.rootInline = root
			plan.contentParts = append(plan.contentParts, root)
			plan.sections = append(plan.sections, mailbox.Section{Path: root.Path})
		default:
			// 根非文本且非 inline：enmime 一律计入 Attachments（即使带 Content-ID）。
			plan.rootAttachment = root
			plan.sections = append(plan.sections, detailAttachmentSections(root, false)...)
		}
		return plan, true
	}

	// 正文：text/plain 与 text/html，且不是附件。
	var walk func(*mailbox.Part) bool
	walk = func(p *mailbox.Part) bool {
		if p.IsMultipart() {
			for _, child := range p.Children {
				if !walk(child) {
					return false
				}
			}
			return true
		}
		if isDetailTextPart(p) {
			plan.textBodies = append(plan.textBodies, p)
			plan.sections = append(plan.sections, mailbox.Section{Path: p.Path})
		}
		return true
	}
	if !walk(root) {
		return nil, false
	}

	// 附件与内联图按 BFS 收集，保持与整封解析一致的顺序。
	walkBFS(root, func(p *mailbox.Part) {
		if p.IsMultipart() {
			return
		}
		if isDetailAttachment(p) {
			plan.attachments = append(plan.attachments, p)
			needContent := p.ContentID != "" && p.Disposition != "attachment"
			plan.sections = append(plan.sections, detailAttachmentSections(p, needContent)...)
			if needContent {
				plan.contentParts = append(plan.contentParts, p)
			}
			return
		}
		if isDetailInline(p) {
			plan.inlines = append(plan.inlines, p)
			plan.contentParts = append(plan.contentParts, p)
			plan.sections = append(plan.sections, mailbox.Section{Path: p.Path})
		}
	})

	// 内联图尺寸直接从内容长度取，附件才需要推断。
	return plan, true
}

// detailAttachmentSections 返回计算附件尺寸所需的抓取分段。
func detailAttachmentSections(p *mailbox.Part, needContent bool) []mailbox.Section {
	if needContent || strings.EqualFold(p.Encoding, "quoted-printable") {
		return []mailbox.Section{{Path: p.Path}}
	}
	if strings.EqualFold(p.Encoding, "base64") {
		if p.Size <= detailSmallSectionBytes {
			return []mailbox.Section{{Path: p.Path}}
		}
		return []mailbox.Section{
			{Path: p.Path, Offset: 0, Length: detailHeadSampleBytes},
			{Path: p.Path, Offset: p.Size - detailTailSampleBytes, Length: detailTailSampleBytes},
		}
	}
	// 7bit/8bit/binary/未知编码：服务端报告的字节数就是解码后字节数。
	return nil
}

// detailStructureSupported 只拒绝分段路径无法安全重建的结构，其余交给 buildDetailPlan。
func detailStructureSupported(root *mailbox.Part) bool {
	if root == nil {
		return false
	}
	supported := true
	var walk func(*mailbox.Part)
	walk = func(p *mailbox.Part) {
		if !supported || p == nil {
			return
		}
		if p.IsMultipart() {
			// 空 multipart 无法从 BODYSTRUCTURE 得到任何子部件，保持整封兜底。
			if len(p.Children) == 0 {
				supported = false
				return
			}
			for _, child := range p.Children {
				walk(child)
			}
		}
	}
	walk(root)
	return supported
}

func detailCharsetSupported(p *mailbox.Part) bool {
	charset := ""
	for key, value := range p.Params {
		if strings.EqualFold(key, "charset") {
			charset = strings.ToLower(strings.TrimSpace(value))
		}
	}
	switch charset {
	case "", "utf-8", "utf8", "us-ascii", "ascii":
		return true
	default:
		return false
	}
}

func isDetailTextPart(p *mailbox.Part) bool {
	if p == nil || p.IsMultipart() {
		return false
	}
	if p.Disposition == "attachment" {
		return false
	}
	return p.Type == "text/plain" || p.Type == "text/html"
}

func isDetailAttachment(p *mailbox.Part) bool {
	if p == nil || p.IsMultipart() {
		return false
	}
	return p.Disposition == "attachment" || p.Type == "application/octet-stream"
}

func isDetailInline(p *mailbox.Part) bool {
	if p == nil || p.IsMultipart() {
		return false
	}
	return p.Disposition == "inline"
}

// walkBFS 按层遍历，与 enmime BreadthMatchAll 的部件顺序一致。
func walkBFS(root *mailbox.Part, fn func(*mailbox.Part)) {
	if root == nil {
		return
	}
	queue := []*mailbox.Part{root}
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		fn(p)
		queue = append(queue, p.Children...)
	}
}

// walkDFS 按深度优先先序遍历，与 enmime DepthMatchAll 的部件顺序一致。
func walkDFS(root *mailbox.Part, fn func(*mailbox.Part)) {
	if root == nil {
		return
	}
	fn(root)
	for _, child := range root.Children {
		walkDFS(child, fn)
	}
}

// selectDetailContent 复刻 enmime 的正文选择规则：HTML 优先，其次纯文本。
func selectDetailContent(root *mailbox.Part, decoded map[string]string) string {
	if root == nil {
		return ""
	}
	if !root.IsMultipart() {
		if root.Type == "text/html" {
			if content := decoded[root.Path]; content != "" {
				return content
			}
			return ""
		}
		return decoded[root.Path]
	}

	html := ""
	walkDFS(root, func(p *mailbox.Part) {
		if html != "" {
			return
		}
		if p.Type == "text/html" && p.Disposition != "attachment" {
			html = decoded[p.Path]
		}
	})

	text := ""
	if root.Type == "multipart/alternative" {
		walkBFS(root, func(p *mailbox.Part) {
			if text != "" {
				return
			}
			if p.Type == "text/plain" && p.Disposition != "attachment" {
				text = decoded[p.Path]
			}
		})
	} else {
		var parts []string
		walkDFS(root, func(p *mailbox.Part) {
			if p.Type == "text/plain" && p.Disposition != "attachment" {
				parts = append(parts, decoded[p.Path])
			}
		})
		text = strings.Join(parts, "\n--\n")
	}

	if html != "" {
		return html
	}
	return text
}

// decodeDetailBody 按 Content-Transfer-Encoding 解码分段内容。
func decodeDetailBody(p *mailbox.Part, raw []byte) ([]byte, error) {
	var reader io.Reader
	switch strings.ToLower(p.Encoding) {
	case "base64":
		reader = base64.NewDecoder(base64.StdEncoding, bytes.NewReader(raw))
	case "quoted-printable":
		reader = quotedprintable.NewReader(bytes.NewReader(raw))
	default:
		reader = bytes.NewReader(raw)
	}
	return io.ReadAll(reader)
}

// detailPartMessage 用 BODYSTRUCTURE 信息重建一个单部件报文，交给 enmime 解码。
// 正文的字符集探测/转换和 quoted-printable 的软换行处理都由 enmime 原样复刻。
func detailPartMessage(p *mailbox.Part, raw []byte) []byte {
	var buf bytes.Buffer
	buf.WriteString("MIME-Version: 1.0\r\n")
	contentType := mime.FormatMediaType(p.Type, p.Params)
	if contentType == "" {
		contentType = p.Type
	}
	fmt.Fprintf(&buf, "Content-Type: %s\r\n", contentType)
	if p.Encoding != "" {
		fmt.Fprintf(&buf, "Content-Transfer-Encoding: %s\r\n", p.Encoding)
	}
	buf.WriteString("\r\n")
	buf.Write(raw)
	return buf.Bytes()
}

// decodeDetailPartExact 用 enmime 解码单个部件，返回解码后的字节。
func decodeDetailPartExact(p *mailbox.Part, raw []byte) ([]byte, error) {
	part, err := enmime.NewParser().ReadParts(bytes.NewReader(detailPartMessage(p, raw)))
	if err != nil {
		return nil, err
	}
	return part.Content, nil
}

// decodeDetailTextExact 用 enmime 解码单个文本部件，复刻其字符集探测与转换结果。
func decodeDetailTextExact(p *mailbox.Part, raw []byte) ([]byte, error) {
	env, err := enmime.ReadEnvelope(bytes.NewReader(detailPartMessage(p, raw)))
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(p.Type, "text/html") {
		return []byte(env.HTML), nil
	}
	return []byte(env.Text), nil
}

// detailAttachmentSize 计算附件解码后的字节数。
func detailAttachmentSize(p *mailbox.Part, bodies map[string][]byte, decoded map[string][]byte) (int64, bool) {
	if content, ok := decoded[p.Path]; ok {
		return int64(len(content)), true
	}
	if strings.EqualFold(p.Encoding, "quoted-printable") {
		raw, ok := bodies[p.Path]
		if !ok {
			return 0, false
		}
		content, err := decodeDetailPartExact(p, raw)
		if err != nil {
			return 0, false
		}
		return int64(len(content)), true
	}
	if !strings.EqualFold(p.Encoding, "base64") {
		// 7bit/8bit/binary/未知传输编码的字节数与服务端报告一致。
		return p.Size, true
	}
	if content, ok := bodies[p.Path]; ok {
		n, err := decodedLength(content, p.Encoding)
		if err != nil {
			return 0, false
		}
		return n, true
	}
	head := bodies[mailbox.Section{Path: p.Path, Offset: 0, Length: detailHeadSampleBytes}.Key()]
	tail := bodies[mailbox.Section{Path: p.Path, Offset: p.Size - detailTailSampleBytes, Length: detailTailSampleBytes}.Key()]
	size, ok := base64DecodedSize(p.Size, head, tail)
	if !ok {
		return 0, false
	}
	return size, true
}

// decodedLength 只统计解码字节数，不保留解码内容，避免大附件占用内存。
func decodedLength(raw []byte, encoding string) (int64, error) {
	var reader io.Reader
	switch strings.ToLower(encoding) {
	case "base64":
		reader = base64.NewDecoder(base64.StdEncoding, bytes.NewReader(raw))
	case "quoted-printable":
		reader = quotedprintable.NewReader(bytes.NewReader(raw))
	default:
		reader = bytes.NewReader(raw)
	}
	return io.Copy(io.Discard, reader)
}

// base64DecodedSize 根据编码后长度与首尾样本推断解码后字节数。
// 返回 false 表示样本不足以确定，调用方应补抓整段。
func base64DecodedSize(encodedSize int64, head, tail []byte) (int64, bool) {
	if encodedSize <= 0 || len(head) == 0 || len(tail) == 0 {
		return 0, false
	}

	trimmed := len(tail)
	for trimmed > 0 && (tail[trimmed-1] == '\n' || tail[trimmed-1] == '\r') {
		trimmed--
	}
	if trimmed == 0 {
		return 0, false
	}
	tailHasNewline := trimmed < len(tail)

	// 最后一个换行之后的片段就是编码内容的最后一行。
	lastChunk := tail[:trimmed]
	if idx := bytes.LastIndexByte(tail[:trimmed], '\n'); idx >= 0 {
		lastChunk = tail[idx+1 : trimmed]
	}
	pad := int64(0)
	for i := len(lastChunk) - 1; i >= 0 && lastChunk[i] == '='; i-- {
		pad++
	}
	if pad > 2 {
		return 0, false
	}

	lineLen := int64(0)
	for i := 0; i < len(head); i++ {
		if head[i] == '\n' {
			lineLen = int64(i)
			if i > 0 && head[i-1] == '\r' {
				lineLen--
			}
			break
		}
	}
	if lineLen <= 0 || lineLen > 512 {
		lineLen = 0
	}

	var whitespace int64
	if lineLen == 0 {
		// 不折行：整个编码内容是单行，最多末尾带一个 CRLF。
		if tailHasNewline {
			whitespace = 2
		}
	} else {
		lineBytes := lineLen + 2
		full := encodedSize / lineBytes
		rem := encodedSize - full*lineBytes
		lastLen := int64(len(lastChunk))
		switch {
		case rem == lastLen:
			whitespace = 2 * full
		case tailHasNewline && rem == lastLen+2:
			// 末行之后还有一个 CRLF。
			whitespace = 2 * (full + 1)
		default:
			return 0, false
		}
	}

	nonWhitespace := encodedSize - whitespace
	if nonWhitespace <= 0 || nonWhitespace%4 != 0 {
		return 0, false
	}
	decoded := nonWhitespace/4*3 - pad
	if decoded < 0 {
		return 0, false
	}
	return decoded, true
}
