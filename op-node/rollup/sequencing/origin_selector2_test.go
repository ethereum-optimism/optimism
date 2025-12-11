package sequencing

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestFindL1OriginOfNextL2Block(t *testing.T) {
	cfg := &rollup.Config{
		MaxSequencerDrift: 500,
		BlockTime:         2,
	}

	require.Panics(t, func() {
		FindL1OriginOfNextL2Block(
			cfg,
			nil,
			nil,
			nil,
			false)
	})

	type testCase struct {
		name                string
		l2Head              *eth.L2BlockRef
		currentL1Origin     *eth.L1BlockRef
		nextL1Origin        *eth.L1BlockRef
		matchAutoderivation bool
		expectedResult      *eth.L1BlockRef
		expectedError       error
	}

	// L1 chain: a100(1200) <-a101(1212)
	//            /\
	// L2 chain    \_ b1000 (1220)
	a100 := &eth.L1BlockRef{
		Number: 100,
		Hash:   common.Hash{'a', '1', '0', '0'},
		Time:   1200,
	}
	a101 := &eth.L1BlockRef{
		Number:     101,
		ParentHash: a100.Hash,
		Hash:       common.Hash{'a', '0', '0'},
		Time:       1212,
	}
	b1000 := &eth.L2BlockRef{
		Number:   1000,
		Hash:     common.Hash{'b', '1', '0', '0', '0'},
		L1Origin: a100.ID(),
		Time:     1220,
	}

	tcs := []testCase{
		{
			name:            "normal operation, progress because we can",
			l2Head:          b1000,
			currentL1Origin: a100,
			nextL1Origin:    a101,
			expectedResult:  a101,
		},
		{
			name:                "recover mode, progress because we can",
			l2Head:              b1000,
			currentL1Origin:     a100,
			nextL1Origin:        a101,
			expectedResult:      a101,
			matchAutoderivation: true,
		},
		{
			name:            "normal operation, don't need to progress",
			l2Head:          b1000,
			currentL1Origin: a100,
			nextL1Origin:    nil,
			expectedResult:  a100,
		},
		{
			name:                "recover mode, need to progress but can't",
			l2Head:              b1000,
			currentL1Origin:     a100,
			nextL1Origin:        nil,
			matchAutoderivation: true,
			expectedError:       ErrNextL1OriginRequired,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			result, err := FindL1OriginOfNextL2Block(
				cfg,
				tc.l2Head,
				tc.currentL1Origin,
				tc.nextL1Origin,
				tc.matchAutoderivation)
			if tc.expectedError != nil {
				require.ErrorIs(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
			}
			if result != tc.expectedResult {
				t.Errorf("expected result %v, got %v", tc.expectedResult, result)
			}
		})
	}
}
