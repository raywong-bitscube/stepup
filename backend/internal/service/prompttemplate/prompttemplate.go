package prompttemplate

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/raywong-bitscube/stepup/backend/internal/dbutil"
)

// KeyPaperAnalyzeChatUser is the prompt_template.key for the OpenAI-compatible chat
// "user" message when analyzing a student exam paper (text + optional vision image).
const KeyPaperAnalyzeChatUser = "paper_analyze_chat_user"

const KeyEssayOutlineGenerateTopic = "essay_outline_generate_topic"
const KeyEssayOutlineReview = "essay_outline_review"
const KeyEssayOutlineOCRTopic = "essay_outline_ocr_topic"

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
		"%genre", sub("genre"),
		"%task_type", sub("task_type"),
		"%topic_text", sub("topic_text"),
		"%outline_text", sub("outline_text"),
	)
	return r.Replace(tpl)
}

// DefaultEssayOutlineGenerateTopic fallback when DB prompt missing.
func DefaultEssayOutlineGenerateTopic() string {
	return `你是一名有10年高中语文教学经验的资深教师，熟悉高考作文命题趋势。
用户选择的文体形式为：%genre；命题方式为：%task_type。
请生成1道符合近年高考趋势的作文题目。要求：题目需明确文体/命题类型，内容贴合高中生认知，具有思辨性或情感表达空间，避免偏题怪题。
请严格用一行输出，格式为：{题目全文} | {文体/命题类型标签}。不要其它说明或换行。`
}

// DefaultEssayOutlineReview fallback when DB prompt missing.
func DefaultEssayOutlineReview() string {
	return `你是一名高考作文阅卷专家，请对用户的作文提纲进行专业点评。
题目为：%topic_text
用户提纲为：%outline_text
请从以下维度分析：1.题目匹配度（是否紧扣文体/命题要求）；2.结构合理性（层次是否清晰，逻辑是否连贯）；3.素材适配性（素材是否典型、支撑中心）。
请严格用一段连续文本输出三段，段与段之间用英文竖线 | 分隔，格式如下：
{总体评价}|{维度评分：匹配度X星/结构X星/素材X星}|{详细建议：1.xxx；2.xxx}
其中 X 为 1-5 的整数。不要 markdown 代码围栏。
仅输出上述三段中文正文；不要输出思考过程、英文推演或「Thinking」类内容。`
}

// DefaultEssayOutlineOCRTopic fallback when DB prompt missing.
func DefaultEssayOutlineOCRTopic() string {
	return `请识别图片中的作文题目或材料内容，只输出应作为「题目文本」交给学生看的正文本身；不要加「题目：」等前缀，不要解释。若材料为多段，保留合理换行。`
}

// GetActiveContent returns prompt_template.content for key when status=1 and not deleted.
func GetActiveContent(ctx context.Context, db *sqlx.DB, key string) (string, error) {
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
	err := db.QueryRowContext(ctx, dbutil.Rebind(`
SELECT content
FROM prompt_template
WHERE "key" = ? AND status = 1 AND is_deleted = 0
LIMIT 1
`), key).Scan(&content)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(content), nil
}
