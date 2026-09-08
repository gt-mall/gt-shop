package util

import (
	"errors"
	"testing"
)

func TestDecimalHandling(t *testing.T) {
	tests := []struct {
		input       string
		expectCents int64
		expectYuan  string
		expectErr   error
	}{
		{"0", 0, "0.00", nil},
		{"0.0", 0, "0.00", nil},
		{"0.00", 0, "0.00", nil},
		{"199.9", 19990, "199.90", nil},
		{"199.90", 19990, "199.90", nil},
		{"0.05", 5, "0.05", nil},
		{"9999999999.99", 999999999999, "9999999999.99", nil},
		{"-1", 0, "", ErrNegativeAmount},
		{"-0.01", 0, "", ErrNegativeAmount},
		{"199.999", 0, "", ErrInvalidAmount},
		{"abc", 0, "", ErrInvalidAmount},
		{"12.3.4", 0, "", ErrInvalidAmount},
		{"10000000000.00", 0, "", ErrAmountTooLarge},
		{"", 0, "", ErrInvalidAmount},
		{"   88.8   ", 8880, "88.80", nil},
	}

	for _, tc := range tests {
		cents, err := ParseDecimalYuanToCent(tc.input)
		if tc.expectErr != nil {
			if !errors.Is(err, tc.expectErr) {
				t.Errorf("input %q expected err %v, got %v", tc.input, tc.expectErr, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("input %q unexpected err: %v", tc.input, err)
			continue
		}
		if cents != tc.expectCents {
			t.Errorf("input %q expected cents %d, got %d", tc.input, tc.expectCents, cents)
		}
		yuan := FormatCentToYuan(cents)
		if yuan != tc.expectYuan {
			t.Errorf("input %q expected yuan %q, got %q", tc.input, tc.expectYuan, yuan)
		}
		norm, err := NormalizeDecimalYuan(tc.input)
		if err != nil || norm != tc.expectYuan {
			t.Errorf("input %q normalize expected %q, got %q (err %v)", tc.input, tc.expectYuan, norm, err)
		}
	}
}
