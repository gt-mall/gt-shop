package util

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestEnrichSKUSpecValues(t *testing.T) {
	prodSpecs := `[
		{
			"id": 1,
			"name": "机身配色",
			"values": [
				{"id": 201, "name": "幻影黑"},
				{"id": 202, "name": "云母白"}
			]
		},
		{
			"id": 2,
			"name": "配置版本",
			"values": [
				{"id": 211, "name": "标准有线充电盒版"},
				{"id": 212, "name": "Qi无线充智能充电盒版"}
			]
		}
	]`

	t.Run("Enriches raw SKU spec values with spec_name and value_name", func(t *testing.T) {
		raw := `[{"spec_id":1,"value_id":201},{"spec_id":2,"value_id":212}]`
		enriched := EnrichSKUSpecValues(raw, &prodSpecs)

		var res []EnrichedSKUSpecValue
		if err := json.Unmarshal([]byte(enriched), &res); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if len(res) != 2 {
			t.Fatalf("expected 2 items, got %d", len(res))
		}
		if res[0].SpecName != "机身配色" || res[0].ValueName != "幻影黑" {
			t.Errorf("res[0] mismatch: %+v", res[0])
		}
		if res[1].SpecName != "配置版本" || res[1].ValueName != "Qi无线充智能充电盒版" {
			t.Errorf("res[1] mismatch: %+v", res[1])
		}
	})

	t.Run("Already enriched values are untouched", func(t *testing.T) {
		raw := `[{"spec_id":1,"spec_name":"机身配色","value_id":201,"value_name":"幻影黑"}]`
		enriched := EnrichSKUSpecValues(raw, &prodSpecs)
		if enriched != raw {
			t.Errorf("expected %s, got %s", raw, enriched)
		}
	})

	t.Run("Empty or invalid inputs handled safely", func(t *testing.T) {
		if res := EnrichSKUSpecValues("", &prodSpecs); res != "" {
			t.Errorf("expected empty string, got %s", res)
		}
		if res := EnrichSKUSpecValues("[]", &prodSpecs); res != "[]" {
			t.Errorf("expected [], got %s", res)
		}
		if res := EnrichSKUSpecValues("invalid", &prodSpecs); res != "invalid" {
			t.Errorf("expected invalid, got %s", res)
		}
		raw := `[{"spec_id":1,"value_id":201}]`
		if res := EnrichSKUSpecValues(raw, nil); res != raw {
			t.Errorf("expected %s, got %s", raw, res)
		}
	})
}

func TestFormatSKUSpecText(t *testing.T) {
	t.Run("Single spec", func(t *testing.T) {
		raw := `[{"spec_id":1,"spec_name":"颜色","value_id":101,"value_name":"曜石黑"}]`
		got := FormatSKUSpecText(raw)
		want := "颜色：曜石黑"
		if got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	})

	t.Run("Three specs preserve order and separator", func(t *testing.T) {
		raw := `[
			{"spec_id":1,"spec_name":"颜色","value_id":101,"value_name":"曜石黑"},
			{"spec_id":2,"spec_name":"内存","value_id":201,"value_name":"12GB+256GB"},
			{"spec_id":3,"spec_name":"套餐","value_id":301,"value_name":"官方标配"}
		]`
		got := FormatSKUSpecText(raw)
		want := "颜色：曜石黑 / 内存：12GB+256GB / 套餐：官方标配"
		if got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	})

	t.Run("Long spec names and values exceeding 255 chars", func(t *testing.T) {
		// 60 chars spec_name, 60 chars value_name for 3 specs
		s1Name := "超长第一规格名称示例用于测试数据库长度边界条件测试测试测试测试" // ~90 bytes
		s1Val := "超长第一规格取值示例用于测试数据库长度边界条件测试测试测试测试"  // ~90 bytes
		s2Name := "超长第二规格名称示例用于测试数据库长度边界条件测试测试测试测试" // ~90 bytes
		s2Val := "超长第二规格取值示例用于测试数据库长度边界条件测试测试测试测试"  // ~90 bytes
		s3Name := "超长第三规格名称示例用于测试数据库长度边界条件测试测试测试测试" // ~90 bytes
		s3Val := "超长第三规格取值示例用于测试数据库长度边界条件测试测试测试测试"  // ~90 bytes

		raw := fmt.Sprintf(`[
			{"spec_name":%q,"value_name":%q},
			{"spec_name":%q,"value_name":%q},
			{"spec_name":%q,"value_name":%q}
		]`, s1Name, s1Val, s2Name, s2Val, s3Name, s3Val)

		got := FormatSKUSpecText(raw)
		if len(got) <= 255 {
			t.Fatalf("expected formatted spec text byte length > 255, got length %d", len(got))
		}
		if utf8.RuneCountInString(got) > 512 {
			t.Fatalf("expected formatted spec text rune count <= 512 (MySQL VARCHAR(512)), got %d", utf8.RuneCountInString(got))
		}
		expectedPrefix := s1Name + "：" + s1Val
		if !strings.HasPrefix(got, expectedPrefix) {
			t.Fatalf("expected prefix %q, got %q", expectedPrefix, got)
		}
	})

	t.Run("Output does not contain JSON keywords or spec_id", func(t *testing.T) {
		raw := `[{"spec_id":1,"spec_name":"颜色","value_id":101,"value_name":"曜石黑"}]`
		got := FormatSKUSpecText(raw)
		if strings.Contains(got, "spec_id") || strings.Contains(got, "value_id") || strings.Contains(got, "{") || strings.Contains(got, "}") {
			t.Fatalf("formatted text should not contain json tokens: %s", got)
		}
	})

	t.Run("Empty or no specs", func(t *testing.T) {
		if res := FormatSKUSpecText(""); res != "" {
			t.Fatalf("expected empty, got %q", res)
		}
		if res := FormatSKUSpecText("[]"); res != "" {
			t.Fatalf("expected empty, got %q", res)
		}
		if res := FormatSKUSpecText("   "); res != "" {
			t.Fatalf("expected empty, got %q", res)
		}
	})

	t.Run("Malformed JSON does not panic", func(t *testing.T) {
		if res := FormatSKUSpecText("[invalid json"); res != "" {
			t.Fatalf("expected empty for invalid json array, got %q", res)
		}
		if res := FormatSKUSpecText("{not array}"); res != "" {
			t.Fatalf("expected empty for invalid json object, got %q", res)
		}
	})

	t.Run("Plain text fallback", func(t *testing.T) {
		plain := "颜色:红色;尺寸:XL"
		if res := FormatSKUSpecText(plain); res != plain {
			t.Fatalf("expected %q, got %q", plain, res)
		}
	})

	t.Run("Value only without spec name", func(t *testing.T) {
		raw := `[{"value_name":"128GB"}]`
		if res := FormatSKUSpecText(raw); res != "128GB" {
			t.Fatalf("expected %q, got %q", "128GB", res)
		}
	})
}
