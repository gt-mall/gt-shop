package util

import (
	"encoding/json"
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
