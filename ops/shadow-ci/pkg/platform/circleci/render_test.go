package circleci

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestYamlSafe_Plain(t *testing.T) {
	assert.Equal(t, "hello", yamlSafe("hello"))
	assert.Equal(t, "op-node", yamlSafe("op-node"))
	assert.Equal(t, "pkg/foo", yamlSafe("pkg/foo"))
}

func TestYamlSafe_Empty(t *testing.T) {
	assert.Equal(t, `""`, yamlSafe(""))
}

func TestYamlSafe_Dangerous(t *testing.T) {
	// Colons, newlines, and other YAML-special chars get quoted.
	assert.Equal(t, `"key: value"`, yamlSafe("key: value"))
	assert.Equal(t, `"line1\nline2"`, yamlSafe("line1\nline2"))
	assert.Equal(t, `"has\"quote"`, yamlSafe(`has"quote`))
	assert.Equal(t, `"{braces}"`, yamlSafe("{braces}"))
}

func TestYamlSafe_BoolLike(t *testing.T) {
	// YAML boolean-like strings must be quoted.
	assert.Equal(t, `"true"`, yamlSafe("true"))
	assert.Equal(t, `"false"`, yamlSafe("false"))
	assert.Equal(t, `"yes"`, yamlSafe("yes"))
	assert.Equal(t, `"no"`, yamlSafe("no"))
	assert.Equal(t, `"null"`, yamlSafe("null"))
	assert.Equal(t, `"TRUE"`, yamlSafe("TRUE"))
}

func TestYamlSafe_LeadingDash(t *testing.T) {
	assert.Equal(t, `"--flag"`, yamlSafe("--flag"))
	assert.Equal(t, `"-v"`, yamlSafe("-v"))
}

func TestYamlSafe_Backslash(t *testing.T) {
	assert.Equal(t, `"back\\slash"`, yamlSafe(`back\slash`))
}
