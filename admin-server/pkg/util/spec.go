package util

import (
	"encoding/json"
	"fmt"
	"strings"
)

type rawSpecValueItem struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type rawSpecItem struct {
	ID     int64              `json:"id"`
	Name   string             `json:"name"`
	Values []rawSpecValueItem `json:"values"`
}

// EnrichedSKUSpecValue represents a SKU spec value item enriched with readable names.
type EnrichedSKUSpecValue struct {
	SpecID    int64  `json:"spec_id"`
	SpecName  string `json:"spec_name,omitempty"`
	ValueID   int64  `json:"value_id"`
	ValueName string `json:"value_name,omitempty"`
}

// EnrichSKUSpecValues parses the raw SKU spec_values JSON (e.g. [{"spec_id":1,"value_id":2}])
// and matches it against the product's specs JSON definition to add spec_name and value_name.
// If the input already contains value_name or if product specs are not provided, it handles gracefully.
func EnrichSKUSpecValues(rawSpecValues string, prodSpecsJSON *string) string {
	rawSpecValues = strings.TrimSpace(rawSpecValues)
	if rawSpecValues == "" || rawSpecValues == "[]" {
		return rawSpecValues
	}

	var skuValues []EnrichedSKUSpecValue
	if err := json.Unmarshal([]byte(rawSpecValues), &skuValues); err != nil {
		return rawSpecValues
	}
	if len(skuValues) == 0 {
		return rawSpecValues
	}

	allEnriched := true
	for _, sv := range skuValues {
		if sv.ValueName == "" || sv.SpecName == "" {
			allEnriched = false
			break
		}
	}
	if allEnriched {
		return rawSpecValues
	}

	if prodSpecsJSON == nil || strings.TrimSpace(*prodSpecsJSON) == "" {
		return rawSpecValues
	}

	var specsDef []rawSpecItem
	if err := json.Unmarshal([]byte(*prodSpecsJSON), &specsDef); err != nil {
		return rawSpecValues
	}

	specNameMap := make(map[int64]string, len(specsDef))
	valNameMap := make(map[int64]map[int64]string, len(specsDef))
	for _, s := range specsDef {
		specNameMap[s.ID] = s.Name
		valMap := make(map[int64]string, len(s.Values))
		for _, v := range s.Values {
			valMap[v.ID] = v.Name
		}
		valNameMap[s.ID] = valMap
	}

	for i := range skuValues {
		if skuValues[i].SpecName == "" {
			skuValues[i].SpecName = specNameMap[skuValues[i].SpecID]
		}
		if skuValues[i].ValueName == "" {
			if vm, ok := valNameMap[skuValues[i].SpecID]; ok {
				skuValues[i].ValueName = vm[skuValues[i].ValueID]
			}
		}
	}

	enrichedBytes, err := json.Marshal(skuValues)
	if err != nil {
		return rawSpecValues
	}
	return string(enrichedBytes)
}

// FormatSKUSpecText converts raw SKU spec JSON into a human-readable string.
// E.g. [{"spec_name":"颜色","value_name":"曜石黑"},{"spec_name":"内存","value_name":"12GB+256GB"}]
// -> "颜色：曜石黑 / 内存：12GB+256GB"
//
// Rules:
// 1. Prefers spec_name + value_name ("颜色：曜石黑").
// 2. Preserves order from original JSON array.
// 3. Trims whitespace and skips empty items.
// 4. If only value_name exists, displays value_name.
// 5. If only spec_name exists, displays spec_name.
// 6. If not valid JSON, returns trimmed raw string if plain text, or empty if corrupted JSON.
// 7. Does not query database or mutate external state.
func FormatSKUSpecText(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return ""
	}

	type specItem struct {
		SpecID    int64  `json:"spec_id"`
		SpecName  string `json:"spec_name"`
		Name      string `json:"name"`
		ValueID   int64  `json:"value_id"`
		ValueName string `json:"value_name"`
		Value     string `json:"value"`
	}

	var items []specItem
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		// If input starts with JSON delimiters, it's corrupted JSON
		if strings.HasPrefix(raw, "[") || strings.HasPrefix(raw, "{") {
			return ""
		}
		// Plain text fallback
		return raw
	}

	if len(items) == 0 {
		return ""
	}

	var parts []string
	for _, it := range items {
		specName := strings.TrimSpace(it.SpecName)
		if specName == "" {
			specName = strings.TrimSpace(it.Name)
		}
		valName := strings.TrimSpace(it.ValueName)
		if valName == "" {
			valName = strings.TrimSpace(it.Value)
		}

		if specName != "" && valName != "" {
			parts = append(parts, fmt.Sprintf("%s：%s", specName, valName))
		} else if valName != "" {
			parts = append(parts, valName)
		} else if specName != "" {
			parts = append(parts, specName)
		}
	}

	return strings.Join(parts, " / ")
}
