package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_hasConcreteContract(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name: "concrete contract",
			content: `
				pragma solidity 0.8.15;
				contract MyContract {
				}
			`,
			expected: true,
		},
		{
			name: "abstract contract only",
			content: `
				pragma solidity 0.8.15;
				abstract contract MyContract {
				}
			`,
			expected: false,
		},
		{
			name: "library only",
			content: `
				pragma solidity 0.8.15;
				library MyLibrary {
				}
			`,
			expected: false,
		},
		{
			name: "interface only",
			content: `
				pragma solidity 0.8.15;
				interface IMyInterface {
				}
			`,
			expected: false,
		},
		{
			name: "abstract and concrete contract",
			content: `
				pragma solidity 0.8.15;
				abstract contract Base {
				}
				contract MyContract is Base {
				}
			`,
			expected: true,
		},
		{
			name: "library and concrete contract",
			content: `
				pragma solidity 0.8.15;
				library MyLibrary {
				}
				contract MyContract {
				}
			`,
			expected: true,
		},
		{
			name: "contract in comment",
			content: `
				pragma solidity 0.8.15;
				// contract NotReal {
				// }
				library MyLibrary {
				}
			`,
			expected: false,
		},
		{
			name: "contract in multiline comment",
			content: `
				pragma solidity 0.8.15;
				/*
				contract NotReal {
				}
				*/
				library MyLibrary {
				}
			`,
			expected: false,
		},
		{
			name:     "empty content",
			content:  "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasConcreteContract(tt.content)
			require.Equal(t, tt.expected, result)
		})
	}
}

func Test_extractPragma(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name: "strict pragma",
			content: `
				pragma solidity 0.8.15;
				contract MyContract {}
			`,
			expected: "0.8.15",
		},
		{
			name: "caret pragma",
			content: `
				pragma solidity ^0.8.0;
				contract MyContract {}
			`,
			expected: "^0.8.0",
		},
		{
			name: "range pragma",
			content: `
				pragma solidity >=0.8.0 <0.9.0;
				contract MyContract {}
			`,
			expected: ">=0.8.0 <0.9.0",
		},
		{
			name: "greater than pragma",
			content: `
				pragma solidity >=0.8.0;
				contract MyContract {}
			`,
			expected: ">=0.8.0",
		},
		{
			name:     "no pragma",
			content:  "contract MyContract {}",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPragma(tt.content)
			require.Equal(t, tt.expected, result)
		})
	}
}

func Test_isStrictPragma(t *testing.T) {
	tests := []struct {
		name     string
		pragma   string
		expected bool
	}{
		{
			name:     "strict version",
			pragma:   "0.8.15",
			expected: true,
		},
		{
			name:     "strict version with different numbers",
			pragma:   "0.8.28",
			expected: true,
		},
		{
			name:     "caret version",
			pragma:   "^0.8.0",
			expected: false,
		},
		{
			name:     "greater than or equal",
			pragma:   ">=0.8.0",
			expected: false,
		},
		{
			name:     "less than or equal",
			pragma:   "<=0.9.0",
			expected: false,
		},
		{
			name:     "range",
			pragma:   ">=0.8.0 <0.9.0",
			expected: false,
		},
		{
			name:     "tilde version",
			pragma:   "~0.8.0",
			expected: false,
		},
		{
			name:     "wildcard x",
			pragma:   "0.8.x",
			expected: false,
		},
		{
			name:     "wildcard X",
			pragma:   "0.8.X",
			expected: false,
		},
		{
			name:     "wildcard star",
			pragma:   "0.8.*",
			expected: false,
		},
		{
			name:     "empty pragma",
			pragma:   "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isStrictPragma(tt.pragma)
			require.Equal(t, tt.expected, result)
		})
	}
}

func Test_removeComments(t *testing.T) {
	tests := []struct {
		name             string
		content          string
		shouldNotContain string
		shouldContain    string
	}{
		{
			name: "single line comment",
			content: `contract MyContract {
				// this is a comment
				uint256 value;
			}`,
			shouldNotContain: "this is a comment",
			shouldContain:    "uint256 value",
		},
		{
			name: "multi line comment",
			content: `contract MyContract {
				/* this is
				   a multi-line
				   comment */
				uint256 value;
			}`,
			shouldNotContain: "multi-line",
			shouldContain:    "uint256 value",
		},
		{
			name: "inline comment",
			content: `contract MyContract {
				uint256 value; // inline comment
			}`,
			shouldNotContain: "inline comment",
			shouldContain:    "uint256 value",
		},
		{
			name:             "no comments",
			content:          "contract MyContract {}",
			shouldNotContain: "",
			shouldContain:    "contract MyContract",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeComments(tt.content)
			if tt.shouldNotContain != "" {
				require.NotContains(t, result, tt.shouldNotContain)
			}
			require.Contains(t, result, tt.shouldContain)
		})
	}
}
