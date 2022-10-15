package metrics

import (
	"math/big"
	"testing"
)

type weiToEthTestCase struct {
	input  *big.Int
	output float64
}

func TestWeiToEther(t *testing.T) {
	tests := []weiToEthTestCase{
		{
			input:  big.NewInt(1_000_000_000_000_000_000),
			output: 1.0,
		},
		{
			input:  big.NewInt(3_000_000_000_000_000_000),
			output: 3.0,
		},
		{
			input:  big.NewInt(3_456_789_000_000_000_000),
			output: 3.456789,
		},
		{
			input:  new(big.Int).Mul(big.NewInt(1_000_000), big.NewInt(1_000_000_000_000_000_000)),
			output: 1_000_000,
		},
	}

	for i, tc := range tests {
		out := weiToEther(tc.input)
		if out != tc.output {
			t.Fatalf("test %v: expected %v but got %v", i, tc.output, out)
		}
	}

}
