package main

import (
	"os"
	"path/filepath"
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

func Test_rangeErrors(t *testing.T) {
	tests := []struct {
		name    string
		pragmas map[string][]string
		errs    []string
	}{
		{
			name: "ranges that allow a pinned version",
			pragmas: map[string][]string{
				"src/Pinned.sol":       {"0.8.15"},
				"src/PinnedNew.sol":    {"0.8.25"},
				"src/Caret.sol":        {"^0.8.0"},
				"src/CaretNewer.sol":   {"^0.8.24"},
				"src/Bounded.sol":      {">=0.6.2 <0.9.0"},
				"src/BoundedComma.sol": {">=0.8.13, <0.9.0"},
			},
		},
		{
			name: "caret above every pinned version",
			pragmas: map[string][]string{
				"src/Pinned.sol":    {"0.8.15"},
				"src/PinnedNew.sol": {"0.8.25"},
				"src/Float.sol":     {"^0.8.26"},
			},
			errs: []string{"src/Float.sol: pragma solidity ^0.8.26 allows none of the pinned compiler versions (0.8.15, 0.8.25)"},
		},
		{
			name: "bound between pinned versions",
			pragmas: map[string][]string{
				"src/Pinned.sol":    {"0.8.15"},
				"src/PinnedNew.sol": {"0.8.25"},
				"src/Between.sol":   {">=0.8.16 <0.8.25"},
			},
			errs: []string{"src/Between.sol: pragma solidity >=0.8.16 <0.8.25 allows none"},
		},
		{
			name: "every pragma of a file",
			pragmas: map[string][]string{
				"src/Pinned.sol": {"0.8.15"},
				"src/Two.sol":    {"^0.8.0", ">=0.8.20"},
			},
			errs: []string{"src/Two.sol: pragma solidity >=0.8.20 allows none"},
		},
		{
			name: "unparseable range",
			pragmas: map[string][]string{
				"src/Pinned.sol": {"0.8.15"},
				"src/Bad.sol":    {"^zero"},
			},
			errs: []string{"src/Bad.sol: cannot parse pragma solidity ^zero"},
		},
		{
			name: "unparseable pin",
			pragmas: map[string][]string{
				"src/Pinned.sol":    {"0.8.15"},
				"src/BadPinned.sol": {"0.08.15"},
			},
			errs: []string{"src/BadPinned.sol: cannot parse pragma solidity 0.08.15"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := rangeErrors(tt.pragmas)
			require.Len(t, errs, len(tt.errs))
			for i, want := range tt.errs {
				require.ErrorContains(t, errs[i], want)
			}
		})
	}
}

func Test_readPragmas(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Lib.sol")
	content := "// pragma solidity ^0.8.30;\npragma solidity ^0.8.0;\n/* pragma solidity 0.7.6; */\npragma  solidity >=0.8.13 <0.9.0 ;\nlibrary Lib {}\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	pragmas, errs := readPragmas(path)
	require.Empty(t, errs)
	require.Equal(t, []string{"^0.8.0", ">=0.8.13 <0.9.0"}, pragmas)
}
