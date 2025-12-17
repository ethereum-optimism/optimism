package binary

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSearch(t *testing.T) {
	tests := []struct {
		name           string
		n              int
		pattern        []bool
		values         []string
		expectedError  string
		expectedIndexL int
		expectedValueL string
		expectedIndexR int
		expectedValueR string
	}{
		{"empty", 0, []bool{}, []string{}, "",
			-1, "",
			0, ""},
		{"all_false", 5, []bool{false, false, false, false, false}, []string{"a", "b", "c", "d", "e"}, "",
			-1, "",
			0, "a"},
		{"all_true", 5, []bool{true, true, true, true, true}, []string{"a", "b", "c", "d", "e"}, "",
			4, "e",
			5, ""},
		{"single_true", 1, []bool{true}, []string{"x"}, "",
			0, "x",
			1, ""},
		{"single_false", 1, []bool{false}, []string{"x"}, "",
			-1, "",
			0, "x"},
		{"classic_pattern", 8, []bool{true, true, true, true, false, false, false, false}, []string{"a", "b", "c", "d", "e", "f", "g", "h"}, "",
			3, "d",
			4, "e"},
		{"first_only", 5, []bool{true, false, false, false, false}, []string{"first", "b", "c", "d", "e"}, "",
			0, "first",
			1, "b"},
		{"last_only", 5, []bool{true, true, true, true, false}, []string{"a", "b", "c", "d", "last"}, "",
			3, "d",
			4, "last"},
		{"error_case", 5, []bool{true, true, false, false, false}, []string{"a", "b", "c", "d", "e"}, "test error",
			-1, "",
			-1, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := func(i int) (bool, string, error) {
				if tt.expectedError != "" {
					return false, "", errors.New(tt.expectedError)
				}
				return tt.pattern[i], tt.values[i], nil
			}

			indexL, valueL, errL := SearchL(tt.n, f)
			indexR, valueR, errR := SearchR(tt.n, f)

			if tt.expectedError != "" {
				require.Error(t, errL)
				require.Contains(t, errL.Error(), tt.expectedError)
				require.Error(t, errR)
				require.Contains(t, errR.Error(), tt.expectedError)
			} else {
				require.NoError(t, errL)
				require.Equal(t, tt.expectedIndexL, indexL)
				require.Equal(t, tt.expectedValueL, valueL)

				require.NoError(t, errR)
				require.Equal(t, tt.expectedIndexR, indexR)
				require.Equal(t, tt.expectedValueR, valueR)
			}
		})
	}
}
