package schema

// PinyinScheme 拼音方案类型（全拼/双拼）。
type PinyinScheme string

const (
	PinyinSchemeFull      PinyinScheme = "full"      // 全拼
	PinyinSchemeShuangpin PinyinScheme = "shuangpin" // 双拼
)

// Valid 校验取值是否在合法集合内（空串/未知值返回 false）
func (s PinyinScheme) Valid() bool {
	switch s {
	case PinyinSchemeFull, PinyinSchemeShuangpin:
		return true
	}
	return false
}

// DictType 词库类型。
type DictType string

const (
	DictTypeCodetable     DictType = "codetable"      // 传统单文件码表
	DictTypeRimeCodetable DictType = "rime_codetable" // RIME 多文件码表
	DictTypeRimePinyin    DictType = "rime_pinyin"    // RIME 拼音词库
)

// Valid 校验取值是否在合法集合内（空串/未知值返回 false）
func (t DictType) Valid() bool {
	switch t {
	case DictTypeCodetable, DictTypeRimeCodetable, DictTypeRimePinyin:
		return true
	}
	return false
}

// DictFormat 词库二进制格式。
type DictFormat string

const (
	DictFormatDAT DictFormat = "dat" // DAT 双数组 Trie（默认）
	DictFormatWDB DictFormat = "wdb" // 旧版 WDB 格式
)

// Valid 校验取值是否在合法集合内（空串/未知值返回 false）
func (f DictFormat) Valid() bool {
	switch f {
	case DictFormatDAT, DictFormatWDB:
		return true
	}
	return false
}
