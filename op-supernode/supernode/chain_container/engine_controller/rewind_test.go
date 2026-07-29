package engine_controller

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// TestEngineController_Rewind tests the rewind functionality of the engine controller
// under various error conditions simulated by embedding a mock L2 which can misbehave in
// multiple ways. The test ensures the method translates those errors — sometimes originating
// on the other side of an RPC connection — into the correct sentinel errors to enable handling
// by the caller of the method.
func TestEngineController_Rewind(t *testing.T) {
	type testCase struct {
		name          string
		expectedError error

		// Below are various ways an error condition can be declared.
		// The test strategy is to first construct a mock that passes the test (no error),
		// and then install up to one error condition. When an error condition is installed
		// the test case should declare the appropriate expected sentinel error.

		missingEngineClient bool
		nilTarget           bool

		newPayloadErr, fcuErr error
		newPayloadStatus      *eth.PayloadStatusV1
		fcuResult             *eth.ForkchoiceUpdatedResult
		labelErr              error

		incorrectUnsafe, incorrectSafe, incorrectFinalized bool

		// finalizedAheadOfTarget puts the finalized head ahead of the target — rewind must refuse.
		finalizedAheadOfTarget bool

		// payloadTimestampMismatch sabotages the target envelope so its timestamp does not
		// correspond to its block number per the rollup config.
		payloadTimestampMismatch bool

		// unsafeMatchesTarget causes the mock to report the unsafe head at the target
		// block (same hash), exercising the no-op path.
		unsafeMatchesTarget bool
		// canonicalReinsertBadStatus simulates the EL rejecting the re-inserted canonical
		// payload after the synthetic FCU.
		canonicalReinsertBadStatus bool

		// expectedNewPayloadCalls overrides the default count of NewPayload calls
		// expected on a successful run.
		expectedNewPayloadCalls *int
		expectedFCUCalls        *int
	}

	testCases := []testCase{
		{
			name: "successful rewind",
		},
		{
			name:                "nil engine client",
			missingEngineClient: true,
			expectedError:       ErrNoEngineClient,
		},
		{
			name:          "nil target",
			nilTarget:     true,
			expectedError: ErrRewindNilTarget,
		},
		{
			name:                     "target payload inconsistent with rollup config",
			payloadTimestampMismatch: true,
			expectedError:            ErrRewindTargetMismatch,
		},
		{
			name:          "new payload error",
			newPayloadErr: errors.New("engine unavailable"),
			expectedError: ErrRewindInsertSyntheticFailed,
		},
		{
			name: "new payload bad status",
			newPayloadStatus: &eth.PayloadStatusV1{
				Status: eth.ExecutionInvalid,
			},
			expectedError: ErrRewindSyntheticPayloadRejected,
		},
		{
			name:          "FCU error",
			fcuErr:        errors.New("FCU failed"),
			expectedError: ErrRewindFCUSyntheticFailed,
		},
		{
			name: "FCU invalid",
			fcuResult: &eth.ForkchoiceUpdatedResult{
				PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionInvalid},
			},
			expectedError: ErrRewindFCURejected,
		},
		{
			name:            "incorrect post-state (unsafe)",
			incorrectUnsafe: true,
			expectedError:   ErrRewindFCUHeadMismatch,
		},
		{
			name:          "incorrect post-state (safe)",
			incorrectSafe: true,
			expectedError: ErrRewindFCUHeadMismatch,
		},
		{
			name:               "incorrect post-state (finalized)",
			incorrectFinalized: true,
			expectedError:      ErrRewindFCUHeadMismatch,
		},
		{
			name:                   "target before finalized",
			finalizedAheadOfTarget: true,
			expectedError:          ErrRewindOverFinalizedHead,
		},
		{
			name:                    "no-op when unsafe head matches target",
			unsafeMatchesTarget:     true,
			expectedNewPayloadCalls: intPtr(0),
			expectedFCUCalls:        intPtr(0),
		},
		{
			name:                       "canonical re-insert bad status",
			canonicalReinsertBadStatus: true,
			expectedError:              ErrRewindCanonicalPayloadRejected,
		},
		{
			name:          "current unsafe label fetch fails",
			labelErr:      errors.New("transient RPC error"),
			expectedError: ErrRewindCurrentUnsafeFailed,
		},
	}

	// Setup: chain is at block 10, we want to rewind to block 5.
	// Block 5 is at timestamp 1000 + 5*2 = 1010 (2s block time).
	genesisTime := uint64(1000)
	targetBlockNum, targetTimestamp, parentHash := uint64(5), genesisTime+(5*2), common.Hash{0x04}
	targetHash := common.Hash{byte(targetBlockNum)}
	canonicalPayload := func() *eth.ExecutionPayloadEnvelope {
		return &eth.ExecutionPayloadEnvelope{
			ExecutionPayload: &eth.ExecutionPayload{
				ParentHash:   parentHash,
				BlockNumber:  eth.Uint64Quantity(targetBlockNum),
				Timestamp:    eth.Uint64Quantity(targetTimestamp),
				BlockHash:    targetHash,
				FeeRecipient: common.Address{0x01},
			},
		}
	}

	createMockL2 := func() mockL2 {
		return mockL2{
			refsByLabel: map[eth.BlockLabel]eth.L2BlockRef{
				eth.Unsafe:    {Number: 10, Hash: common.Hash{0xee}},
				eth.Safe:      {Number: 10, Hash: common.Hash{0x0a}},
				eth.Finalized: {Number: 2, Hash: common.Hash{0x08}},
			},
		}
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rollupConfig := rollup.Config{
				Genesis:   rollup.Genesis{L2: eth.BlockID{Number: 0}, L2Time: genesisTime},
				BlockTime: 2,
				L2ChainID: big.NewInt(420),
			}
			l2 := createMockL2()

			if tc.newPayloadErr != nil {
				l2.newPayloadErr = tc.newPayloadErr
			}
			if tc.newPayloadStatus != nil {
				l2.newPayloadStatus = tc.newPayloadStatus
			}
			if tc.fcuErr != nil {
				l2.fcuErr = tc.fcuErr
			}
			if tc.fcuResult != nil {
				l2.fcuResult = tc.fcuResult
			}
			l2.labelOverrides = make(map[eth.BlockLabel]eth.L2BlockRef)
			if tc.labelErr != nil {
				l2.refByLabelErr = tc.labelErr
			}
			if tc.incorrectUnsafe {
				l2.labelOverrides[eth.Unsafe] = eth.L2BlockRef{Number: targetBlockNum, Hash: common.Hash{0xff}}
			}
			if tc.incorrectSafe {
				l2.labelOverrides[eth.Safe] = eth.L2BlockRef{Number: targetBlockNum, Hash: common.Hash{0xff}}
			}
			if tc.incorrectFinalized {
				l2.labelOverrides[eth.Finalized] = eth.L2BlockRef{Number: targetBlockNum, Hash: common.Hash{0xff}}
			}
			if tc.finalizedAheadOfTarget {
				l2.refsByLabel[eth.Finalized] = eth.L2BlockRef{Number: targetBlockNum + 1, Hash: common.Hash{0xff}}
			}
			if tc.unsafeMatchesTarget {
				l2.refsByLabel[eth.Unsafe] = eth.L2BlockRef{Number: targetBlockNum, Hash: targetHash}
			}
			if tc.canonicalReinsertBadStatus {
				l2.newPayloadStatuses = []*eth.PayloadStatusV1{nil, {Status: eth.ExecutionInvalid}}
			}

			ec := &simpleEngineController{l2: &l2, rollup: &rollupConfig, log: testlog.Logger(t, log.LvlDebug)}
			if tc.missingEngineClient {
				ec.l2 = nil
			}

			var target *eth.ExecutionPayloadEnvelope
			if !tc.nilTarget {
				target = canonicalPayload()
			}
			if tc.payloadTimestampMismatch {
				// Shift block number while keeping timestamp; rollup-derived expected number won't match.
				target.ExecutionPayload.BlockNumber = eth.Uint64Quantity(targetBlockNum + 1)
			}

			err := ec.Rewind(context.Background(), target)

			if tc.expectedError != nil {
				require.ErrorIs(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
				wantNewPayloadCalls := 2
				if tc.expectedNewPayloadCalls != nil {
					wantNewPayloadCalls = *tc.expectedNewPayloadCalls
				}
				wantFCUCalls := 2
				if tc.expectedFCUCalls != nil {
					wantFCUCalls = *tc.expectedFCUCalls
				}
				require.Equal(t, wantNewPayloadCalls, l2.newPayloadCalls,
					"NewPayload call count mismatch (synthetic + canonical re-insert)")
				require.Equal(t, wantFCUCalls, l2.fcuCalls,
					"ForkchoiceUpdate call count mismatch (synthetic + target)")
				if wantNewPayloadCalls == 2 {
					require.NotNil(t, l2.lastNewPayload)
					require.Equal(t, canonicalPayload().ExecutionPayload.ExtraData, l2.lastNewPayload.ExtraData,
						"final NewPayload should be the canonical re-insert with original ExtraData")
				}
			}
		})
	}
}

func intPtr(v int) *int { return &v }
