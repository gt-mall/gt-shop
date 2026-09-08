package util

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var (
	ErrInvalidAmount  = errors.New("金额格式无效，最多两位小数且非负")
	ErrNegativeAmount = errors.New("金额不能为负数")
	ErrAmountTooLarge  = errors.New("金额超出有效范围(最大9999999999.99)")
)

// MaxYuanCents corresponds to DECIMAL(12,2): 9,999,999,999.99 yuan = 999,999,999,999 cents.
const MaxYuanCents int64 = 999999999999

// ParseDecimalYuanToCent converts a decimal yuan string (e.g. "199.90", "0", "12.5")
// into integer cents without any float32/float64 conversions.
func ParseDecimalYuanToCent(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, ErrInvalidAmount
	}
	if strings.HasPrefix(s, "-") {
		return 0, ErrNegativeAmount
	}
	parts := strings.Split(s, ".")
	if len(parts) > 2 {
		return 0, ErrInvalidAmount
	}

	intStr := parts[0]
	if intStr == "" {
		return 0, ErrInvalidAmount
	}
	for i := 0; i < len(intStr); i++ {
		if intStr[i] < '0' || intStr[i] > '9' {
			return 0, ErrInvalidAmount
		}
	}
	intPart, err := strconv.ParseInt(intStr, 10, 64)
	if err != nil {
		return 0, ErrAmountTooLarge
	}
	if intPart > MaxYuanCents/100 {
		return 0, ErrAmountTooLarge
	}

	var centPart int64
	if len(parts) == 2 {
		decStr := parts[1]
		if len(decStr) == 0 || len(decStr) > 2 {
			return 0, ErrInvalidAmount
		}
		for i := 0; i < len(decStr); i++ {
			if decStr[i] < '0' || decStr[i] > '9' {
				return 0, ErrInvalidAmount
			}
		}
		if len(decStr) == 1 {
			centPart = int64(decStr[0]-'0') * 10
		} else {
			centPart = int64(decStr[0]-'0')*10 + int64(decStr[1]-'0')
		}
	}

	totalCents := intPart*100 + centPart
	if totalCents > MaxYuanCents || totalCents < 0 {
		return 0, ErrAmountTooLarge
	}
	return totalCents, nil
}

// FormatCentToYuan converts integer cents into standard 2-decimal yuan string (e.g. 19990 -> "199.90").
func FormatCentToYuan(cents int64) string {
	if cents < 0 {
		return "0.00"
	}
	intPart := cents / 100
	decPart := cents % 100
	return fmt.Sprintf("%d.%02d", intPart, decPart)
}

// NormalizeDecimalYuan parses and normalizes a decimal yuan string into standard 2-decimal format.
func NormalizeDecimalYuan(s string) (string, error) {
	cents, err := ParseDecimalYuanToCent(s)
	if err != nil {
		return "", err
	}
	return FormatCentToYuan(cents), nil
}
