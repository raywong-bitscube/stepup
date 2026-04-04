package prompttemplate

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

// KeyPaperAnalyzeChatUser is the prompt_template.key for the OpenAI-compatible chat
// "user" message when analyzing a student exam paper (text + optional vision image).
const KeyPaperAnalyzeChatUser = "paper_analyze_chat_user"

// DefaultPaperAnalyzeChatUser is used when the DB row is missing, inactive, or unreadable.
func DefaultPaperAnalyzeChatUser() string {
	return `试卷上传元信息：科目=%subject，阶段=%stage，原始文件名=%file_name。
请只输出一段合法 JSON（不要用 markdown 代码围栏），严格符合下列键：summary (string)、weak_points (string 数组)、improvement_plan (string 数组)、raw_content (string，可为试卷要点摘录或空字符串)。
内容针对中国学生试卷分析场景，用语简洁专业。`
}

// Expand replaces named placeholders in template text. Supported placeholders:
// %subject, %stage, %file_name; optional legacy / advanced %file_content (replaced by empty string unless you supply it in vars).
func Expand(tpl string, vars map[string]string) string {
	sub := func(k string) string {
		if vars == nil {
			return ""
		}
		return vars[k]
	}
	r := strings.NewReplacer(
		"%file_content", sub("file_content"),
		"%file_name", sub("file_name"),
		"%stage", sub("stage"),
		"%subject", sub("subject"),
	)
	return r.Replace(tpl)
}

// GetActiveContent returns prompt_template.content for key when status=1 and not deleted.
func GetActiveContent(ctx context.Context, db *sql.DB, key string) (string, error) {
	if db == nil {
		return "", sql.ErrNoRows
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return "", sql.ErrNoRows
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	var content string
	err := db.QueryRowContext(ctx, `
SELECT content
FROM prompt_template
WHERE `+"`key`"+` = ? AND status = 1 AND is_deleted = 0
LIMIT 1
`, key).Scan(&content)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(content), nil
}
