package bigint

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestBigIntUint64(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analyzers, err := (&BigIntPlugin{}).BuildAnalyzers()
	if err != nil {
		t.Fatalf("build analyzers: %v", err)
	}

	analysistest.Run(t, testdata, analyzers[0], "bigint")
}
