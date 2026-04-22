package datformat

import (
	"testing"
)

func TestDAT_Build_And_ExactLookup(t *testing.T) {
	b := NewDATBuilder()
	keys := []string{"shi", "shui", "si", "sha", "she", "shu"}
	for i, k := range keys {
		b.Add(k, uint32(i))
	}
	dat, err := b.Build()
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}

	// 所有已添加的 key 必须能精确匹配到对应 dataIndex
	for i, k := range keys {
		idx, found := dat.ExactMatch(k)
		if !found {
			t.Errorf("key %q should be found", k)
			continue
		}
		if idx != uint32(i) {
			t.Errorf("key %q: want dataIndex %d, got %d", k, i, idx)
		}
	}

	// 不存在的 key 应返回 false
	notExist := []string{"s", "sh", "shia", "x", "", "shuix"}
	for _, k := range notExist {
		_, found := dat.ExactMatch(k)
		if found {
			t.Errorf("key %q should NOT be found", k)
		}
	}
}

func TestDAT_Build_Empty(t *testing.T) {
	b := NewDATBuilder()
	dat, err := b.Build()
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}
	_, found := dat.ExactMatch("any")
	if found {
		t.Error("empty DAT should not match any key")
	}
	_, found = dat.ExactMatch("")
	if found {
		t.Error("empty DAT should not match empty key")
	}
}

func TestDAT_Build_SingleKey(t *testing.T) {
	b := NewDATBuilder()
	b.Add("hello", 42)
	dat, err := b.Build()
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}

	idx, found := dat.ExactMatch("hello")
	if !found {
		t.Fatal("key 'hello' should be found")
	}
	if idx != 42 {
		t.Errorf("want dataIndex 42, got %d", idx)
	}

	_, found = dat.ExactMatch("hell")
	if found {
		t.Error("prefix 'hell' should NOT be found")
	}
	_, found = dat.ExactMatch("helloo")
	if found {
		t.Error("extended key 'helloo' should NOT be found")
	}
}
