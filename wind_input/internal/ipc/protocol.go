package ipc

// 请求类型
type RequestType string

const (
	RequestTypeConvert RequestType = "convert"
)

// 请求结构
type Request struct {
	Type RequestType `json:"type"`
	Data ConvertData `json:"data"`
}

// 转换请求数据
type ConvertData struct {
	Input         string `json:"input"`
	Context       string `json:"context"`
	MaxCandidates int    `json:"max_candidates"`
}

// 响应结构
type Response struct {
	Status     string      `json:"status"`
	Candidates []Candidate `json:"candidates,omitempty"`
	Preedit    string      `json:"preedit,omitempty"`     // 预编辑文本 (例如: "wkdi")
	ShowInline bool        `json:"show_inline,omitempty"` // 是否显示在嵌入式编码行
	Error      string      `json:"error,omitempty"`
}

// 候选词
type Candidate struct {
	Text   string `json:"text"`
	Weight int    `json:"weight"`
}
