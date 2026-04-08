package solidity

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/ops/checks/graph"
)

func createTestSolProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// src/L1/Foo.sol imports src/libraries/Bar.sol
	os.MkdirAll(filepath.Join(dir, "src", "L1"), 0755)
	os.MkdirAll(filepath.Join(dir, "src", "libraries"), 0755)
	os.MkdirAll(filepath.Join(dir, "test", "L1"), 0755)

	os.WriteFile(filepath.Join(dir, "src", "libraries", "Bar.sol"), []byte(`
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

library Bar {
    function hello() internal pure returns (string memory) {
        return "hello";
    }
}
`), 0644)

	os.WriteFile(filepath.Join(dir, "src", "L1", "Foo.sol"), []byte(`
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import "../libraries/Bar.sol";

contract Foo {
    function greet() public pure returns (string memory) {
        return Bar.hello();
    }
}
`), 0644)

	os.WriteFile(filepath.Join(dir, "test", "L1", "Foo.t.sol"), []byte(`
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import "../../src/L1/Foo.sol";

contract FooTest {
    function test_greet() public {
        Foo foo = new Foo();
        foo.greet();
    }
}
`), 0644)

	return dir
}

func TestAnalyze_SolidityProject(t *testing.T) {
	dir := createTestSolProject(t)
	g := graph.NewGraph()
	a := New()

	if err := a.Analyze(g, dir); err != nil {
		t.Fatal(err)
	}

	nodes := g.NodesOfKind(graph.KindSource)
	if len(nodes) != 3 {
		t.Errorf("expected 3 source nodes, got %d", len(nodes))
		for _, n := range nodes {
			t.Logf("  %s", n.ID)
		}
	}

	// Foo.sol should import Bar.sol
	fooEdges := g.EdgesFrom("sol:src/L1/Foo.sol")
	foundBar := false
	for _, e := range fooEdges {
		if e.To == "sol:src/libraries/Bar.sol" {
			foundBar = true
		}
	}
	if !foundBar {
		t.Error("expected import edge from Foo.sol to Bar.sol")
	}

	// Foo.t.sol should import Foo.sol
	testEdges := g.EdgesFrom("sol:test/L1/Foo.t.sol")
	foundFoo := false
	for _, e := range testEdges {
		if e.To == "sol:src/L1/Foo.sol" {
			foundFoo = true
		}
	}
	if !foundFoo {
		t.Error("expected import edge from Foo.t.sol to Foo.sol")
	}
}

func TestAnalyze_WithRemappings(t *testing.T) {
	dir := t.TempDir()

	// Create foundry.toml with remappings
	os.WriteFile(filepath.Join(dir, "foundry.toml"), []byte(`
[profile.default]
remappings = [
    'mylib/=src/libraries/',
]
`), 0644)

	os.MkdirAll(filepath.Join(dir, "src", "libraries"), 0755)
	os.MkdirAll(filepath.Join(dir, "src", "L1"), 0755)

	os.WriteFile(filepath.Join(dir, "src", "libraries", "Util.sol"), []byte(`
pragma solidity ^0.8.0;
library Util {}
`), 0644)

	os.WriteFile(filepath.Join(dir, "src", "L1", "Main.sol"), []byte(`
pragma solidity ^0.8.0;
import "mylib/Util.sol";
contract Main {}
`), 0644)

	g := graph.NewGraph()
	a := New()
	if err := a.Analyze(g, dir); err != nil {
		t.Fatal(err)
	}

	edges := g.EdgesFrom("sol:src/L1/Main.sol")
	found := false
	for _, e := range edges {
		if e.To == "sol:src/libraries/Util.sol" {
			found = true
		}
	}
	if !found {
		t.Error("expected remapped import edge from Main.sol to Util.sol")
		for _, e := range edges {
			t.Logf("  edge: %s -> %s", e.From, e.To)
		}
	}
}

func TestParseImports(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.sol")
	os.WriteFile(path, []byte(`
import "./Foo.sol";
import {Bar} from "../Bar.sol";
import {A, B} from "lib/C.sol";
`), 0644)

	imports := parseImports(path)
	if len(imports) != 3 {
		t.Fatalf("expected 3 imports, got %d: %v", len(imports), imports)
	}
}

func TestName(t *testing.T) {
	a := New()
	if a.Name() != "solidity" {
		t.Errorf("expected 'solidity', got %q", a.Name())
	}
}
