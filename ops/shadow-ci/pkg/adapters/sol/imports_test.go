package sol

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseImports(t *testing.T) {
	dir := t.TempDir()
	content := `// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import "src/L1/SystemConfig.sol";
import {OptimismPortal} from "src/L1/OptimismPortal2.sol";
import "../libraries/GasPayingToken.sol";
`
	path := filepath.Join(dir, "test.sol")
	os.WriteFile(path, []byte(content), 0o644)

	imports, err := ParseImports(path)
	require.NoError(t, err)
	assert.Len(t, imports, 3)
	assert.Contains(t, imports, "src/L1/SystemConfig.sol")
	assert.Contains(t, imports, "src/L1/OptimismPortal2.sol")
	assert.Contains(t, imports, "../libraries/GasPayingToken.sol")
}

func TestParseRemappings(t *testing.T) {
	dir := t.TempDir()
	content := `@openzeppelin/=lib/openzeppelin-contracts/
forge-std/=lib/forge-std/src/
ds-test/:forge-std/=lib/forge-std/src/
`
	path := filepath.Join(dir, "remappings.txt")
	os.WriteFile(path, []byte(content), 0o644)

	remappings, err := ParseRemappings(path)
	require.NoError(t, err)
	assert.Equal(t, "lib/openzeppelin-contracts/", remappings["@openzeppelin/"])
	assert.Equal(t, "lib/forge-std/src/", remappings["forge-std/"])
}

func TestRemappings_ResolveImport(t *testing.T) {
	r := Remappings{
		"@openzeppelin/": "lib/openzeppelin-contracts/",
		"forge-std/":     "lib/forge-std/src/",
	}

	assert.Equal(t, "lib/openzeppelin-contracts/token/ERC20.sol", r.ResolveImport("@openzeppelin/token/ERC20.sol"))
	assert.Equal(t, "lib/forge-std/src/Test.sol", r.ResolveImport("forge-std/Test.sol"))
	assert.Equal(t, "src/L1/Portal.sol", r.ResolveImport("src/L1/Portal.sol")) // no remapping
}

func TestParseRemappings_NoFile(t *testing.T) {
	r, err := ParseRemappings("/nonexistent/remappings.txt")
	require.NoError(t, err)
	assert.Empty(t, r)
}

func TestCollectSolFiles(t *testing.T) {
	dir := t.TempDir()

	// Create test structure.
	os.MkdirAll(filepath.Join(dir, "src", "L1"), 0o755)
	os.MkdirAll(filepath.Join(dir, "test"), 0o755)
	os.WriteFile(filepath.Join(dir, "src", "L1", "Portal.sol"), []byte("// sol"), 0o644)
	os.WriteFile(filepath.Join(dir, "test", "Portal.t.sol"), []byte("// test"), 0o644)
	os.WriteFile(filepath.Join(dir, "src", "L1", "README.md"), []byte("# readme"), 0o644) // not .sol

	files, err := CollectSolFiles(dir, []string{"src/", "test/"})
	require.NoError(t, err)
	assert.Len(t, files, 2)

	hasSrc := false
	hasTest := false
	for _, f := range files {
		if f == "src/L1/Portal.sol" {
			hasSrc = true
		}
		if f == "test/Portal.t.sol" {
			hasTest = true
		}
	}
	assert.True(t, hasSrc)
	assert.True(t, hasTest)
}
