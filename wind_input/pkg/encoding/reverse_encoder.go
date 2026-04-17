package encoding

import "fmt"

// EncodeStatus 编码结果状态。
type EncodeStatus string

const (
	EncodeOK     EncodeStatus = "ok"
	EncodeNoCode EncodeStatus = "no_code" // 码表中无此字
	EncodeNoRule EncodeStatus = "no_rule" // 无匹配的编码规则
)

// EncodeResult 单个词语的编码结果。
type EncodeResult struct {
	Word   string       `json:"word"`
	Code   string       `json:"code"`
	Status EncodeStatus `json:"status"`
	Error  string       `json:"error,omitempty"`
}

// ReverseEncoder 根据反向索引和编码规则，从词语生成编码。
type ReverseEncoder struct {
	reverseIndex map[string][]string
	rules        []Rule
}

// NewReverseEncoder 创建反向编码器。
// reverseIndex: 单字 → 编码列表（来自 CodeTable.BuildReverseIndex）
// rules: 多字词编码规则（来自 Schema.EncoderRules）
func NewReverseEncoder(reverseIndex map[string][]string, rules []Rule) *ReverseEncoder {
	return &ReverseEncoder{
		reverseIndex: reverseIndex,
		rules:        rules,
	}
}

// Encode 为给定词语生成编码。
func (e *ReverseEncoder) Encode(word string) (string, error) {
	runes := []rune(word)
	if len(runes) == 0 {
		return "", fmt.Errorf("空词语")
	}

	if len(runes) == 1 {
		return e.encodeSingleChar(runes[0])
	}

	return e.encodeMultiChar(word, runes)
}

// EncodeBatch 批量编码词语。
func (e *ReverseEncoder) EncodeBatch(words []string) []EncodeResult {
	results := make([]EncodeResult, len(words))
	for i, word := range words {
		results[i].Word = word
		code, err := e.Encode(word)
		if err != nil {
			results[i].Status = classifyEncodeError(word, err)
			results[i].Error = err.Error()
		} else {
			results[i].Code = code
			results[i].Status = EncodeOK
		}
	}
	return results
}

// encodeSingleChar 单字编码：直接从反向索引取最短编码。
func (e *ReverseEncoder) encodeSingleChar(ch rune) (string, error) {
	codes := e.reverseIndex[string(ch)]
	if len(codes) == 0 {
		return "", fmt.Errorf("码表中无此字: %c", ch)
	}
	// 单字取最短编码（简码优先）
	best := codes[0]
	for _, code := range codes[1:] {
		if len(code) < len(best) {
			best = code
		}
	}
	return best, nil
}

// encodeMultiChar 多字词编码：需要编码规则。
func (e *ReverseEncoder) encodeMultiChar(word string, runes []rune) (string, error) {
	if len(e.rules) == 0 {
		return "", fmt.Errorf("当前方案无编码规则")
	}

	// 为每个字查找全码（最长编码）
	charCodes := make(map[string]string, len(runes))
	for _, ch := range runes {
		charStr := string(ch)
		if _, ok := charCodes[charStr]; ok {
			continue
		}
		codes := e.reverseIndex[charStr]
		if len(codes) == 0 {
			return "", fmt.Errorf("码表中无此字: %s", charStr)
		}
		// 取最长编码（全码）
		best := codes[0]
		for _, code := range codes[1:] {
			if len(code) > len(best) {
				best = code
			}
		}
		charCodes[charStr] = best
	}

	return CalcWordCode(word, charCodes, e.rules)
}

// SchemaEncoderRule 是 schema.EncoderRule 的镜像类型，
// 避免 pkg/encoding 引用 internal/schema 产生循环依赖。
type SchemaEncoderRule struct {
	LengthEqual   int
	LengthInRange []int
	Formula       string
}

// ConvertSchemaRules 将 schema 编码规则转换为 encoding.Rule 列表。
func ConvertSchemaRules(schemaRules []SchemaEncoderRule) []Rule {
	rules := make([]Rule, len(schemaRules))
	for i, sr := range schemaRules {
		rules[i] = Rule{
			LengthEqual: sr.LengthEqual,
			Formula:     sr.Formula,
		}
		if len(sr.LengthInRange) == 2 {
			rules[i].LengthRange = [2]int{sr.LengthInRange[0], sr.LengthInRange[1]}
		}
	}
	return rules
}

func classifyEncodeError(word string, err error) EncodeStatus {
	runes := []rune(word)
	if len(runes) > 1 {
		errMsg := err.Error()
		if errMsg == "当前方案无编码规则" || contains(errMsg, "no matching rule") {
			return EncodeNoRule
		}
	}
	return EncodeNoCode
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
