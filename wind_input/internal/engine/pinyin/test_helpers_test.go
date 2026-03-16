package pinyin

import (
	"github.com/huanfeng/wind_input/internal/dict"
)

// wrapInCompositeDict 将 PinyinDict 包装为 CompositeDict（测试辅助）
func wrapInCompositeDict(pinyinDict *dict.PinyinDict) *dict.CompositeDict {
	cd := dict.NewCompositeDict()
	layer := dict.NewPinyinDictLayer("test-system", dict.LayerTypeSystem, pinyinDict)
	cd.AddLayer(layer)
	return cd
}
