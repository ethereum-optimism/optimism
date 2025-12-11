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
		MaxSequencerDrift: 1800, // Use Fjord constant value
		BlockTime:         2,
	}

	require.Panics(t, func() {
		_, _ = FindL1OriginOfNextL2Block(
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

	tcs := []testCase{}

	// Scenarios with valid data, no drift concerns
	// but availability of next l1 origin is modulated.
	//
	// L1 chain: a100(1200) <- [ a101(1212) ]
	//            /\
	// L2 chain    \_ b1000(1220)
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

	tcs = append(tcs,
		testCase{
			name:            "normal operation, progress because we can",
			l2Head:          b1000,
			currentL1Origin: a100,
			nextL1Origin:    a101,
			expectedResult:  a101,
		},
		testCase{
			name:                "recover mode, progress because we can",
			l2Head:              b1000,
			currentL1Origin:     a100,
			nextL1Origin:        a101,
			expectedResult:      a101,
			matchAutoderivation: true,
		},
		testCase{
			name:            "normal operation, don't need to progress",
			l2Head:          b1000,
			currentL1Origin: a100,
			nextL1Origin:    nil,
			expectedResult:  a100,
		},
		testCase{
			name:                "recover mode, need to progress but can't",
			l2Head:              b1000,
			currentL1Origin:     a100,
			nextL1Origin:        nil,
			matchAutoderivation: true,
			expectedError:       ErrNextL1OriginRequired,
		},
	)

	// Bad input data / reorg scenarios
	// L1 chain: c100(1200) <-[x]- c101(1212)
	//            /\
	// L2 chain    \_[x]- d/e1000(1220)
	c100 := &eth.L1BlockRef{
		Number: 100,
		Hash:   common.Hash{'a', '1', '0', '0'},
		Time:   1200,
	}
	c101 := &eth.L1BlockRef{
		Number:     101,
		ParentHash: common.Hash{}, // does not point to c100
		Hash:       common.Hash{'a', '0', '0'},
		Time:       1212,
	}
	d1000 := &eth.L2BlockRef{
		Number:   1000,
		L1Origin: c100.ID(),
		Hash:     common.Hash{'d', '1', '0', '0', '0'},
		Time:     1220,
	}
	e1000 := &eth.L2BlockRef{
		Number:   1000,
		L1Origin: eth.BlockID{}, // does not point to c100
		Hash:     common.Hash{'d', '1', '0', '0', '0'},
		Time:     1220,
	}
	tcs = append(tcs,
		testCase{
			name:            "L1 reorg",
			currentL1Origin: c100,
			nextL1Origin:    c101,
			l2Head:          d1000,
			expectedError:   ErrNextL1OriginOrphaned,
		},
		testCase{
			name:            "Invalid l1 origin",
			currentL1Origin: c100,
			nextL1Origin:    c101,
			l2Head:          e1000,
			expectedError:   ErrInvalidL1Origin,
		})

	// Drift at maximum,
	// L1 chain: a100(1200) <- [ a101(1212) ]
	//            /\
	// L2 chain    \_ f1000(3000)
	f1000 := &eth.L2BlockRef{
		Number:   1000,
		L1Origin: a100.ID(),
		Hash:     common.Hash{'f', '1', '0', '0', '0'},
		Time:     3000,
	}
	tcs = append(tcs,
		testCase{
			name:            "Drift at maximum, nextL1Origin available",
			currentL1Origin: a100,
			nextL1Origin:    a101,
			l2Head:          f1000,
			expectedResult:  a101,
		},
		testCase{
			name:            "Drift at maximum, nextL1Origin unavailable",
			currentL1Origin: a100,
			l2Head:          f1000,
			expectedError:   ErrNextL1OriginRequired,
		})

	// Negative drift,
	// L1 chain: a100(1200) <- a101(1212)
	//            /\
	// L2 chain    \_ g1000(1200)
	// Current drift is 0
	// adopting the nextLOrigin would make it negative (add 2 subtract 12)
	g1000 := &eth.L2BlockRef{
		Number:   1000,
		L1Origin: a100.ID(),
		Hash:     common.Hash{'g', '1', '0', '0', '0'},
		Time:     1200,
	}
	tcs = append(tcs,
		testCase{
			name:            "Negative drift",
			currentL1Origin: a100,
			nextL1Origin:    a101,
			l2Head:          g1000,
			expectedResult:  a100,
		})

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
