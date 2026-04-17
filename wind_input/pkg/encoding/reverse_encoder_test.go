package encoding

import "testing"

func testReverseIndex() map[string][]string {
	return map[string][]string{
		"工": {"aaaa", "a"},
		"式": {"aad", "aa"},
		"中": {"khkh", "kh"},
		"国": {"lgyi", "lg"},
		"输": {"lwgj"},
		"入": {"tyyy", "ty"},
		"法": {"ifcl", "if"},
	}
}

func testRules() []Rule {
	return []Rule{
		{LengthEqual: 2, Formula: "AaAbBaBb"},
		{LengthEqual: 3, Formula: "AaAbBaCa"},
		{LengthRange: [2]int{4, 99}, Formula: "AaBaZaZb"}, // 4字及以上
	}
}

func TestReverseEncoderSingleChar(t *testing.T) {
	enc := NewReverseEncoder(testReverseIndex(), testRules())

	// 单字取最短编码
	code, err := enc.Encode("工")
	if err != nil {
		t.Fatalf("Encode 工: %v", err)
	}
	if code != "a" {
		t.Errorf("Encode 工 = %q, want 'a'", code)
	}
}

func TestReverseEncoderTwoChar(t *testing.T) {
	enc := NewReverseEncoder(testReverseIndex(), testRules())

	// 二字词: AaAbBaBb → 工(aaaa) 式(aad) → a,a,a,a = "aaaa"
	code, err := enc.Encode("工式")
	if err != nil {
		t.Fatalf("Encode 工式: %v", err)
	}
	// A=工(aaaa), B=式(aad): Aa=a, Ab=a, Ba=a, Bb=a → "aaaa"
	if code != "aaaa" {
		t.Errorf("Encode 工式 = %q, want 'aaaa'", code)
	}
}

func TestReverseEncoderThreeChar(t *testing.T) {
	enc := NewReverseEncoder(testReverseIndex(), testRules())

	// 三字词: AaAbBaCa → 输(lwgj) 入(tyyy) 法(ifcl)
	// Aa=l, Ab=w, Ba=t, Ca=i → "lwti"
	code, err := enc.Encode("输入法")
	if err != nil {
		t.Fatalf("Encode 输入法: %v", err)
	}
	if code != "lwti" {
		t.Errorf("Encode 输入法 = %q, want 'lwti'", code)
	}
}

func TestReverseEncoderNoCode(t *testing.T) {
	enc := NewReverseEncoder(testReverseIndex(), testRules())

	_, err := enc.Encode("鑫")
	if err == nil {
		t.Error("expected error for unknown char")
	}
}

func TestReverseEncoderNoRule(t *testing.T) {
	// 无规则
	enc := NewReverseEncoder(testReverseIndex(), nil)

	_, err := enc.Encode("工式")
	if err == nil {
		t.Error("expected error for no rules")
	}
}

func TestEncodeBatch(t *testing.T) {
	enc := NewReverseEncoder(testReverseIndex(), testRules())

	results := enc.EncodeBatch([]string{"工", "中国", "鑫", "输入法"})

	if len(results) != 4 {
		t.Fatalf("results count = %d, want 4", len(results))
	}

	// 工: ok
	if results[0].Status != EncodeOK || results[0].Code != "a" {
		t.Errorf("工: %+v", results[0])
	}

	// 中国: ok
	if results[1].Status != EncodeOK {
		t.Errorf("中国: %+v", results[1])
	}

	// 鑫: no_code
	if results[2].Status != EncodeNoCode {
		t.Errorf("鑫: status=%v, want no_code", results[2].Status)
	}

	// 输入法: ok
	if results[3].Status != EncodeOK || results[3].Code != "lwti" {
		t.Errorf("输入法: %+v", results[3])
	}
}

func TestConvertSchemaRules(t *testing.T) {
	schemaRules := []SchemaEncoderRule{
		{LengthEqual: 2, Formula: "AaAbBaBb"},
		{LengthInRange: []int{3, 99}, Formula: "AaBaZaZb"},
	}

	rules := ConvertSchemaRules(schemaRules)
	if len(rules) != 2 {
		t.Fatalf("rules count = %d, want 2", len(rules))
	}
	if rules[0].LengthEqual != 2 || rules[0].Formula != "AaAbBaBb" {
		t.Errorf("rule[0] = %+v", rules[0])
	}
	if rules[1].LengthRange != [2]int{3, 99} {
		t.Errorf("rule[1].LengthRange = %v, want [3, 99]", rules[1].LengthRange)
	}
}
