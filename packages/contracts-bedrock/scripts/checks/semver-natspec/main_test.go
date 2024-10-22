package main

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegexes(t *testing.T) {
	t.Run("ConstantVersionPattern", func(t *testing.T) {
		testRegex(t, ConstantVersionPattern, []regexTest{
			{
				name:    "constant version",
				input:   `string constant version = "1.2.3";`,
				capture: "1.2.3",
			},
			{
				name:    "constant version with weird spaces",
				input:   ` string   constant   version =  "1.2.3";`,
				capture: "1.2.3",
			},
			{
				name:    "constant version with visibility",
				input:   `string public constant version = "1.2.3";`,
				capture: "1.2.3",
			},
			{
				name:    "different variable name",
				input:   `string constant VERSION = "1.2.3";`,
				capture: "",
			},
			{
				name:    "different type",
				input:   `uint constant version = 1;`,
				capture: "",
			},
			{
				name:    "not constant",
				input:   `string version = "1.2.3";`,
				capture: "",
			},
			{
				name:    "unterminated",
				input:   `string constant version = "1.2.3"`,
				capture: "",
			},
		})
	})

	t.Run("FunctionVersionPattern", func(t *testing.T) {
		testRegex(t, FunctionVersionPattern, []regexTest{
			{
				name:    "function version",
				input:   `    return "1.2.3";`,
				capture: "1.2.3",
			},
			{
				name:    "function version with weird spaces",
				input:   `    return   "1.2.3";`,
				capture: "1.2.3",
			},
			{
				name:    "function version with prerelease",
				input:   `    return "1.2.3-alpha.1";`,
				capture: "1.2.3-alpha.1",
			},
			{
				name:    "invalid semver",
				input:   `    return "1.2.cabdab";`,
				capture: "",
			},
			{
				name:    "not a return statement",
				input:   `function foo()`,
				capture: "",
			},
		})
	})

	t.Run("InteropVersionPattern", func(t *testing.T) {
		testRegex(t, InteropVersionPattern, []regexTest{
			{
				name:    "interop version",
				input:   `    return string.concat(super.version(), "+interop");`,
				capture: "+interop",
			},
			{
				name:    "interop version but as a valid semver",
				input:   `    return string.concat(super.version(), "0.0.0+interop");`,
				capture: "0.0.0+interop",
			},
			{
				name:    "not an interop version",
				input:   `	return string.concat(super.version(), "hello!");`,
				capture: "",
			},
			{
				name:    "invalid syntax",
				input:   `	return string.concat(super.version(), "0.0.0+interop`,
				capture: "",
			},
			{
				name:    "something else is concatted",
				input:   `	return string.concat("superduper", "mart");`,
				capture: "",
			},
		})
	})
}

type regexTest struct {
	name    string
	input   string
	capture string
}

func testRegex(t *testing.T, re *regexp.Regexp, tests []regexTest) {
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.capture, findLine([]byte(test.input), re))
		})
	}
}
