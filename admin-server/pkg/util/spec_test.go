package util

import (
	"encoding/json"
	"testing"
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
