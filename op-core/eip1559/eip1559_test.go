package eip1559

import (
	"testing"
)

func TestValidateHolocene1559Params(t *testing.T) {
	tests := []struct {
		name     string
		params   []byte
		expected string
	}{
		{
			name:     "Wrong Length",
			params:   []byte{0x00, 0x01},
			expected: "holocene eip-1559 params should be 8 bytes, got 2",
		},
		{
			name:     "Zero denominator, non-zero elasticity",
			params:   []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
			expected: "holocene params cannot have a 0 denominator unless elasticity is also 0",
		},
		{
			name:     "Zero elasticity, non-zero denominator",
			params:   []byte{0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00},
			expected: "holocene params cannot have a 0 elasticity unless denominator is also 0",
		},
		{
			name:   "Both zero (valid)",
			params: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		},
		{
			name:   "Both non-zero (valid)",
			params: []byte{0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateHolocene1559Params(tc.params)
			if tc.expected == "" && err != nil {
				t.Errorf("Expected no error, but got: %v", err)
			}
			if tc.expected != "" && (err == nil || err.Error() != tc.expected) {
				t.Errorf("Expected error: %s, but got: %v", tc.expected, err)
			}
		})
	}
}

func TestValidateHoloceneExtraData(t *testing.T) {
	tests := []struct {
		name     string
		extra    []byte
		expected string
	}{
		{
			name:     "Wrong Length",
			extra:    []byte{0x00, 0x01},
			expected: "holocene extraData should be 9 bytes, got 2",
		},
		{
			name:     "Wrong Version",
			extra:    []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			expected: "holocene extraData version byte should be 0, got 1",
		},
		{
			name:     "Zero denominator, non-zero elasticity",
			extra:    []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
			expected: "holocene extraData must encode a non-zero denominator",
		},
		{
			name:     "Zero elasticity, non-zero denominator",
			extra:    []byte{0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00},
			expected: "holocene extraData must encode a non-zero elasticity",
		},
		{
			name:     "Both zero (invalid)",
			extra:    []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			expected: "holocene extraData must encode a non-zero denominator",
		},
		{
			name:  "Both non-zero (valid)",
			extra: []byte{0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateHoloceneExtraData(tc.extra)
			if tc.expected == "" && err != nil {
				t.Errorf("Expected no error, but got: %v", err)
			}
			if tc.expected != "" && (err == nil || err.Error() != tc.expected) {
				t.Errorf("Expected error: %s, but got: %v", tc.expected, err)
			}
		})
	}
}
