package test

import (
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	_ "unsafe"
)

//go:linkname importPath testing/internal/testdeps.ImportPath
var importPath string

var withMain atomic.Bool

func checkMain() {
	if !withMain.Load() {
		panic("TestMain missing, cannot plan test")
	}
}

func Main(m *testing.M) {
	withMain.Store(true)

	rootName := strings.ReplaceAll(importPath, "/", ".")

	// The test-binary main-function sets the import-path, it's not immediately available.
	plan.ImportPath = importPath

	// TODO validate flags

	// run the actual test-cases
	code := m.Run()
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	fmt.Println("working dir: " + wd)
	err = postProcess(rootName)
	if err != nil {
		fmt.Printf("\n\ntest post-processing fail: %v\n\n\n", err)
		code = 1
	}
	os.Exit(code)
}

func postProcess(name string) error {
	// TODO configurable output dir
	err := WritePlans("out/" + name + ".json")
	if err != nil {
		return fmt.Errorf("failed to write test plan: %w", err)
	}
	return nil
}
