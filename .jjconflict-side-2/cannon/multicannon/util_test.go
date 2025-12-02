package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseFlag(t *testing.T) {
	cases := []struct {
		name      string
		args      string
		flag      string
		expect    string
		expectErr string
	}{
		{
			name:   "bar=one",
			args:   "--foo --bar=one --baz",
			flag:   "--bar",
			expect: "one",
		},
		{
			name:   "bar one",
			args:   "--foo --bar one --baz",
			flag:   "--bar",
			expect: "one",
		},
		{
			name:   "bar one first flag",
			args:   "--bar one --foo two --baz three",
			flag:   "--bar",
			expect: "one",
		},
		{
			name:   "bar one last flag",
			args:   "--foo --baz --bar one",
			flag:   "--bar",
			expect: "one",
		},
		{
			name:      "non-existent flag",
			args:      "--foo one",
			flag:      "--bar",
			expectErr: "missing flag",
		},
		{
			name:      "empty args",
			args:      "",
			flag:      "--foo",
			expectErr: "missing flag",
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			args := strings.Split(tt.args, " ")
			result, err := parseFlag(args, tt.flag)
			if tt.expectErr != "" {
				require.ErrorContains(t, err, tt.expectErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expect, result)
			}
		})
	}
}
