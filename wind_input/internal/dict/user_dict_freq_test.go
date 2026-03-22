package dict

import (
	"path/filepath"
	"testing"
)

func TestIncreaseWeight_MaxDynamicWeight(t *testing.T) {
	ud := NewUserDict("test", filepath.Join(t.TempDir(), "ud.txt"))
	defer ud.Close()

	// 添加一个接近上限的词条
	ud.Add("nihao", "你好", MaxDynamicWeight-5)

	// 增加 10，应被上限截断
	ud.IncreaseWeight("nihao", "你好", 10)

	results := ud.Search("nihao", 10)
	if len(results) == 0 {
		t.Fatal("应有结果")
	}
	if results[0].Weight != MaxDynamicWeight {
		t.Errorf("权重应被截断为 %d, 实际=%d", MaxDynamicWeight, results[0].Weight)
	}
}

func TestIncreaseWeight_CountIncrement(t *testing.T) {
	ud := NewUserDict("test", filepath.Join(t.TempDir(), "ud.txt"))
	defer ud.Close()

	ud.Add("nihao", "你好", 100)

	// 初始 count=0
	ud.mu.RLock()
	w := ud.entries["nihao"][0]
	ud.mu.RUnlock()
	if w.Count != 0 {
		t.Errorf("初始 count 应为 0, 实际=%d", w.Count)
	}

	// IncreaseWeight 应递增 count
	ud.IncreaseWeight("nihao", "你好", 10)

	ud.mu.RLock()
	w = ud.entries["nihao"][0]
	ud.mu.RUnlock()
	if w.Count != 1 {
		t.Errorf("IncreaseWeight 后 count 应为 1, 实际=%d", w.Count)
	}
	if w.Weight != 110 {
		t.Errorf("权重应为 110, 实际=%d", w.Weight)
	}
}

func TestIncreaseWeight_NonExistentWord(t *testing.T) {
	ud := NewUserDict("test", filepath.Join(t.TempDir(), "ud.txt"))
	defer ud.Close()

	// 不存在的编码，不应 panic
	ud.IncreaseWeight("xyz", "不存在", 10)

	// 编码存在但词不存在
	ud.Add("nihao", "你好", 100)
	ud.IncreaseWeight("nihao", "不存在的词", 10)

	results := ud.Search("nihao", 10)
	if len(results) != 1 || results[0].Weight != 100 {
		t.Errorf("不匹配的词不应被修改")
	}
}

func TestOnWordSelected_NewWord(t *testing.T) {
	ud := NewUserDict("test", filepath.Join(t.TempDir(), "ud.txt"))
	defer ud.Close()

	// 新词应被添加，count=1
	ud.OnWordSelected("nihao", "你好", 50, 10, 3)

	results := ud.Search("nihao", 10)
	if len(results) == 0 {
		t.Fatal("新词应被添加")
	}
	if results[0].Weight != 50 {
		t.Errorf("新词初始权重应为 50, 实际=%d", results[0].Weight)
	}

	ud.mu.RLock()
	w := ud.entries["nihao"][0]
	ud.mu.RUnlock()
	if w.Count != 1 {
		t.Errorf("新词 count 应为 1, 实际=%d", w.Count)
	}
}

func TestOnWordSelected_BelowThreshold(t *testing.T) {
	ud := NewUserDict("test", filepath.Join(t.TempDir(), "ud.txt"))
	defer ud.Close()

	ud.Add("nihao", "你好", 50)

	// 选中第1次 (count 从 0→1)：未达阈值3，不提权
	ud.OnWordSelected("nihao", "你好", 50, 10, 3)
	results := ud.Search("nihao", 10)
	if results[0].Weight != 50 {
		t.Errorf("count=1 未达阈值，权重应不变 50, 实际=%d", results[0].Weight)
	}

	// 选中第2次 (count 1→2)：仍未达阈值
	ud.OnWordSelected("nihao", "你好", 50, 10, 3)
	results = ud.Search("nihao", 10)
	if results[0].Weight != 50 {
		t.Errorf("count=2 未达阈值，权重应不变 50, 实际=%d", results[0].Weight)
	}

	// 选中第3次 (count 2→3)：达到阈值，开始提权
	ud.OnWordSelected("nihao", "你好", 50, 10, 3)
	results = ud.Search("nihao", 10)
	if results[0].Weight != 60 {
		t.Errorf("count=3 达到阈值，权重应为 60, 实际=%d", results[0].Weight)
	}

	// 选中第4次 (count 3→4)：继续提权
	ud.OnWordSelected("nihao", "你好", 50, 10, 3)
	results = ud.Search("nihao", 10)
	if results[0].Weight != 70 {
		t.Errorf("count=4, 权重应为 70, 实际=%d", results[0].Weight)
	}
}

func TestOnWordSelected_MaxWeight(t *testing.T) {
	ud := NewUserDict("test", filepath.Join(t.TempDir(), "ud.txt"))
	defer ud.Close()

	ud.Add("nihao", "你好", MaxDynamicWeight-5)

	// 模拟已达阈值（手动设置 count）
	ud.mu.Lock()
	ud.entries["nihao"][0].Count = 10
	ud.mu.Unlock()

	// 提权应被上限截断
	ud.OnWordSelected("nihao", "你好", 50, 10, 3)
	results := ud.Search("nihao", 10)
	if results[0].Weight != MaxDynamicWeight {
		t.Errorf("权重应被截断为 %d, 实际=%d", MaxDynamicWeight, results[0].Weight)
	}
}

func TestOnWordSelected_CaseInsensitive(t *testing.T) {
	ud := NewUserDict("test", filepath.Join(t.TempDir(), "ud.txt"))
	defer ud.Close()

	ud.Add("nihao", "你好", 100)

	// 大写编码应匹配
	ud.OnWordSelected("NIHAO", "你好", 50, 10, 3)

	ud.mu.RLock()
	w := ud.entries["nihao"][0]
	ud.mu.RUnlock()
	if w.Count != 1 {
		t.Errorf("大写编码应匹配已有词条, count 应为 1, 实际=%d", w.Count)
	}
}

func TestUserDict_SaveLoadWithCount(t *testing.T) {
	tmpPath := filepath.Join(t.TempDir(), "ud.txt")

	// 创建并保存带 count 的词条
	ud1 := NewUserDict("test", tmpPath)
	ud1.Add("nihao", "你好", 100)
	ud1.mu.Lock()
	ud1.entries["nihao"][0].Count = 5
	ud1.mu.Unlock()
	ud1.Save()
	ud1.Close()

	// 重新加载，验证 count 被持久化
	ud2 := NewUserDict("test", tmpPath)
	ud2.Load()
	defer ud2.Close()

	ud2.mu.RLock()
	words := ud2.entries["nihao"]
	ud2.mu.RUnlock()

	if len(words) == 0 {
		t.Fatal("应有词条")
	}
	if words[0].Count != 5 {
		t.Errorf("重新加载后 count 应为 5, 实际=%d", words[0].Count)
	}
	if words[0].Weight != 100 {
		t.Errorf("重新加载后 weight 应为 100, 实际=%d", words[0].Weight)
	}
}
