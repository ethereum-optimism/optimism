package erigon_build_hack_test

import (
	"testing"

	"github.com/onsi/gomega/gexec"
)

func TestBuild(t *testing.T) {
	_, err := gexec.Build("github.com/ledgerwatch/erigon/cmd/erigon")
	if err != nil {
		t.Fatalf("Could not build Erigon: %s", err)
	}
	gexec.CleanupBuildArtifacts()
}
