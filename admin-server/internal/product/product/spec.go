package product

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

var (
	ErrTooManySpecs       = errors.New("商品最多支持3个规格维度")
	ErrTooManySpecValues  = errors.New("每个规格维度最多支持20个选项")
	ErrTooManySKUs        = errors.New("单个商品最多支持200条SKU")
	ErrEmptySKUs          = errors.New("商品必须包含至少一个SKU")
	ErrInvalidSpecID      = errors.New("规格ID必须为正整数")
	ErrInvalidValueID     = errors.New("规格值ID必须为正整数")
	ErrEmptySpecName      = errors.New("规格名称不能为空且最多64个字符")
	ErrEmptyValueName     = errors.New("规格值名称不能为空且最多64个字符")
	ErrDuplicateSpecID    = errors.New("商品内规格ID重复")
	ErrDuplicateSpecName  = errors.New("商品内规格名称重复")
	ErrDuplicateValueID   = errors.New("规格维度内选项ID重复")
	ErrDuplicateValueName = errors.New("规格维度内选项名称重复")
	ErrEmptySpecValues    = errors.New("规格维度必须至少包含一个选项")
	ErrDefaultSKURequired = errors.New("无规格商品必须且只能有一个默认SKU")
	ErrDefaultSKUNoValues = errors.New("无规格商品的默认SKU不能包含规格值")
	ErrSKUMismatchSpecs   = errors.New("SKU规格选择数量与商品规格维度不一致")
	ErrDuplicateSKUCombo  = errors.New("商品存在重复的规格组合")
)

// ValidateSpecs verifies specs structure, ensuring <= 3 specs, <= 20 values per spec,
// unique positive IDs and non-empty unique names within each level.
func ValidateSpecs(specs []ProductSpec) error {
	if len(specs) > 3 {
		return ErrTooManySpecs
	}
	seenSpecID := make(map[int64]bool, len(specs))
	seenSpecName := make(map[string]bool, len(specs))

	for _, s := range specs {
		if s.ID <= 0 {
			return ErrInvalidSpecID
		}
		name := strings.TrimSpace(s.Name)
		if name == "" || utf8.RuneCountInString(name) > 64 {
			return ErrEmptySpecName
		}
		if seenSpecID[s.ID] {
			return fmt.Errorf("%w: %d", ErrDuplicateSpecID, s.ID)
		}
		seenSpecID[s.ID] = true

		if seenSpecName[name] {
			return fmt.Errorf("%w: %s", ErrDuplicateSpecName, name)
		}
		seenSpecName[name] = true

		if len(s.Values) == 0 {
			return fmt.Errorf("%w: %s", ErrEmptySpecValues, name)
		}
		if len(s.Values) > 20 {
			return fmt.Errorf("%w: %s", ErrTooManySpecValues, name)
		}

		seenValID := make(map[int64]bool, len(s.Values))
		seenValName := make(map[string]bool, len(s.Values))
		for _, v := range s.Values {
			if v.ID <= 0 {
				return ErrInvalidValueID
			}
			valName := strings.TrimSpace(v.Name)
			if valName == "" || utf8.RuneCountInString(valName) > 64 {
				return ErrEmptyValueName
			}
			if seenValID[v.ID] {
				return fmt.Errorf("%w: %d", ErrDuplicateValueID, v.ID)
			}
			seenValID[v.ID] = true

			if seenValName[valName] {
				return fmt.Errorf("%w: %s", ErrDuplicateValueName, valName)
			}
			seenValName[valName] = true
		}
	}
	return nil
}

// CalculateSKUSpecHash computes the canonical SHA-256 spec_hash.
// For multi-spec: sorts spec_values by spec_id ascending, formats as "1:1|2:4", and hashes to 64 lowercase hex chars.
// For no-spec (specs is empty): hashes the string "default" to 64 lowercase hex chars.
func CalculateSKUSpecHash(specs []ProductSpec, specValues []ProductSKUValue) (string, error) {
	if len(specs) == 0 {
		if len(specValues) != 0 {
			return "", ErrDefaultSKUNoValues
		}
		h := sha256.Sum256([]byte("default"))
		return hex.EncodeToString(h[:]), nil
	}

	if len(specValues) != len(specs) {
		return "", ErrSKUMismatchSpecs
	}

	specMap := make(map[int64]map[int64]bool, len(specs))
	for _, s := range specs {
		valMap := make(map[int64]bool, len(s.Values))
		for _, v := range s.Values {
			valMap[v.ID] = true
		}
		specMap[s.ID] = valMap
	}

	seenSpecs := make(map[int64]bool, len(specValues))
	sortedValues := make([]ProductSKUValue, len(specValues))
	copy(sortedValues, specValues)

	for _, sv := range sortedValues {
		if sv.SpecID <= 0 || sv.ValueID <= 0 {
			return "", errors.New("规格ID和规格值ID必须为正整数")
		}
		valMap, ok := specMap[sv.SpecID]
		if !ok {
			return "", fmt.Errorf("未知的规格ID: %d", sv.SpecID)
		}
		if seenSpecs[sv.SpecID] {
			return "", fmt.Errorf("同一规格不能多选: %d", sv.SpecID)
		}
		seenSpecs[sv.SpecID] = true
		if !valMap[sv.ValueID] {
			return "", fmt.Errorf("未知的规格值ID: %d", sv.ValueID)
		}
	}

	sort.Slice(sortedValues, func(i, j int) bool {
		return sortedValues[i].SpecID < sortedValues[j].SpecID
	})

	var sb strings.Builder
	for i, sv := range sortedValues {
		if i > 0 {
			sb.WriteString("|")
		}
		sb.WriteString(fmt.Sprintf("%d:%d", sv.SpecID, sv.ValueID))
	}

	h := sha256.Sum256([]byte(sb.String()))
	return hex.EncodeToString(h[:]), nil
}
