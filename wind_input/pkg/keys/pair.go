// pair.go — 组合键群（PairGroup）类型与表
//
// PairGroup 把"组合按键群"（如翻页键、候选选择键、以词定字键、移动高亮键等
// 配置项里指定的成对/成组按键）类型化为单一权威表，避免在多处重复 switch case
// 字符串字面量造成 drift。
package keys

// PairGroup 是"组合按键群"的规范名 —— 用于翻页键、选择键等配置项里指定一对/一组按键。
type PairGroup string

const (
	PairSemicolonQuote PairGroup = "semicolon_quote"
	PairCommaPeriod    PairGroup = "comma_period"
	PairLRShift        PairGroup = "lrshift"
	PairLRCtrl         PairGroup = "lrctrl"
	PairPageUpDown     PairGroup = "pageupdown"
	PairMinusEqual     PairGroup = "minus_equal"
	PairBrackets       PairGroup = "brackets"
	PairShiftTab       PairGroup = "shift_tab"
	PairTab            PairGroup = "tab"    // 实际配对 [KeyShiftTab, KeyTab]，与 PairShiftTab 同义但用于 HighlightKeys
	PairArrows         PairGroup = "arrows" // 4 个键（上/下方向键），不在 pairGroupKeys 表内
)

// pairGroupKeys 把 PairGroup 映射到组成它的两个 Key（前一个=上一页/选择上, 后一个=下一页/选择下）。
//
// 注意：
//   - PairTab 实际配对是 [KeyShiftTab, KeyTab]，与 PairShiftTab 等价（只是分别用于
//     翻页键和移动高亮两个不同配置项的语义）。
//   - PairArrows 是 4 方向键，不在此表内（由调用方单独处理 VK_UP/VK_DOWN）。
var pairGroupKeys = map[PairGroup][2]Key{
	PairSemicolonQuote: {KeySemicolon, KeyQuote},
	PairCommaPeriod:    {KeyComma, KeyPeriod},
	PairLRShift:        {KeyLShift, KeyRShift},
	PairLRCtrl:         {KeyLCtrl, KeyRCtrl},
	PairPageUpDown:     {KeyPageUp, KeyPageDown},
	PairMinusEqual:     {KeyMinus, KeyEqual},
	PairBrackets:       {KeyLBracket, KeyRBracket},
	PairShiftTab:       {KeyShiftTab, KeyTab},
	PairTab:            {KeyShiftTab, KeyTab},
}

// Keys 返回组成 PairGroup 的两个 Key（前一个=上一页/选择上, 后一个=下一页/选择下）。
// 第二返回值表示是否为已知 group（PairArrows 会返回 false，因其为四向键）。
func (g PairGroup) Keys() (prev, next Key, ok bool) {
	pair, exists := pairGroupKeys[g]
	if !exists {
		return "", "", false
	}
	return pair[0], pair[1], true
}

// Valid 返回 PairGroup 是否为已知规范名（包括 PairArrows）。
func (g PairGroup) Valid() bool {
	if _, ok := pairGroupKeys[g]; ok {
		return true
	}
	return g == PairArrows
}
