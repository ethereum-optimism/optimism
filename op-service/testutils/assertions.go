package testutils

import (
	"math/big"
	"testing"
)

func BigEqual(a, b *big.Int) bool {
	if a == nil || b == nil {
		return a == b
	} else {
		return a.Cmp(b) == 0
	}
}

func RequireBigEqual(t *testing.T, exp, actual *big.Int) {
	t.Helper()
	if !BigEqual(exp, actual) {
		t.Fatalf("expected %s to be equal to %s", exp.String(), actual.String())
	}
}
