package engine_controller

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// TestEngineController_RewindToTimestamp tests the rewind functionality of the engine controller
// under various error conditions simulated by embedding a mock L2 which can misbehave in multiple ways.
// The test ensures the method translates thos errors,
// sometimes originating on the other side of an RPC connection
// into the correct sentinel errors to enable handling by the caller of the method.
func TestEngineController_RewindToTimestamp(t *testing.T) {

	type testCase struct {
		name          string
		expectedError error

		// Below are various ways an error condition can be declared.
		// The test strategy is to first construct a mock would
		// pass the test (no error), and then install up to one error
		// condition in the mock L2, When an error condition is installed
		// the test case should declare the appropriate expected sentinel error.

		missingEngineClient, missingRollupConfig bool

		refErr, newPayloadErr, fcuErr error
		newPayloadStatus              *eth.PayloadStatusV1
		fcuResult                     *eth.ForkchoiceUpdatedResult

		incorrectUnsafe, incorrectSafe, incorrectFinalized bool

		targetBeforeGenesis   bool
		targetBeforeFinalized bool
	}

	testCases := []testCase{
		{
			name:          "successful rewind",
			expectedError: nil,
		},
		{
			name:                "nil engine client",
			missingEngineClient: true,
			expectedError:       ErrNoEngineClient,
		},
		{
			name:                "nil rollup config",
			missingRollupConfig: true,
			expectedError:       ErrNoRollupConfig,
		},
		{
			name:          "reference error",
			refErr:        ethereum.NotFound,
			expectedError: ErrRewindTargetBlockNotFound,
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
			name:                "target before genesis",
			targetBeforeGenesis: true,
			expectedError:       ErrRewindTimestampToBlockConversion,
		},
		{
			name:                  "target before finalized",
			targetBeforeFinalized: true,
			expectedError:         ErrRewindOverFinalizedHead,
		},
	}

	// Setup: chain is at block 10, we want to rewind to block 5
	// Block 5 is at timestamp 1000 + 5*2 = 1010 (2s block time)
	genesisTime := uint64(1000)
	targetBlockNum, targetTimestamp, parentHash := uint64(5), genesisTime+(5*2), common.Hash{0x04}
	targetRef := eth.L2BlockRef{
		Number:     targetBlockNum,
		Hash:       common.Hash{byte(targetBlockNum)},
		ParentHash: parentHash,
		Time:       targetTimestamp,
	}
	payloadEnvelope := eth.ExecutionPayloadEnvelope{
		ExecutionPayload: &eth.ExecutionPayload{
			ParentHash:   parentHash,
			BlockNumber:  eth.Uint64Quantity(targetRef.BlockRef().Number),
			Timestamp:    eth.Uint64Quantity(targetRef.BlockRef().Time),
			BlockHash:    targetRef.Hash,
			FeeRecipient: common.Address{0x01},
		},
	}

	createMockL2 := func() mockL2 {
		return mockL2{
			refsByNumber: map[uint64]eth.L2BlockRef{
				targetBlockNum: targetRef,
			},
			// Initial state before rewind — used by computeRewindTargets
			refsByLabel: map[eth.BlockLabel]eth.L2BlockRef{
				eth.Safe:      {Number: 10, Hash: common.Hash{0x0a}},
				eth.Finalized: {Number: 2, Hash: common.Hash{0x08}},
			},
			payloadsByNumber: map[uint64]*eth.ExecutionPayloadEnvelope{
				targetBlockNum: &payloadEnvelope,
			},
		}
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// First, get a "good mock" which would pass the test with no error:
			rollupConfig := rollup.Config{
				Genesis:   rollup.Genesis{L2: eth.BlockID{Number: 0}, L2Time: genesisTime},
				BlockTime: 2,
				L2ChainID: big.NewInt(420),
			}
			l2 := createMockL2()

			// Next, apply the sabotage(s):
			if tc.refErr != nil {
				l2.refErr = tc.refErr
			}
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
			if tc.incorrectUnsafe {
				l2.labelOverrides[eth.Unsafe] = eth.L2BlockRef{Number: targetBlockNum, Hash: common.Hash{0xff}}
			}
			if tc.incorrectSafe {
				l2.labelOverrides[eth.Safe] = eth.L2BlockRef{Number: targetBlockNum, Hash: common.Hash{0xff}}
			}
			if tc.incorrectFinalized {
				l2.labelOverrides[eth.Finalized] = eth.L2BlockRef{Number: targetBlockNum, Hash: common.Hash{0xff}}
			}
			if tc.targetBeforeGenesis {
				rollupConfig.Genesis = rollup.Genesis{L2Time: 2000}
			}
			if tc.targetBeforeFinalized {
				l2.refsByLabel[eth.Finalized] = eth.L2BlockRef{Number: targetBlockNum + 1, Hash: common.Hash{0xff}}
			}

			// Make a "good" engine controller, using a potentially sabotaged mock L2
			ec := &simpleEngineController{l2: &l2, rollup: &rollupConfig, log: testlog.Logger(t, log.LvlDebug)}

			// Apply a couple of potential sabotages directly to the engine controller:
			if tc.missingEngineClient {
				ec.l2 = nil
			}
			if tc.missingRollupConfig {
				ec.rollup = nil
			}

			// Execute the method call under test:
			err := ec.RewindToTimestamp(context.Background(), targetTimestamp)

			// Make the assertions declared in the test case:
			if tc.expectedError != nil {
				require.ErrorIs(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
				// Verify NewPayload was called (for synthetic block)
				require.Equal(t, 1, l2.newPayloadCalls, "NewPayload should be called once for synthetic block")
				require.NotNil(t, l2.lastNewPayload)
				// The synthetic payload should have modified ExtraData to change the block hash
				require.NotEqual(t, payloadEnvelope.ExecutionPayload.ExtraData, l2.lastNewPayload.ExtraData, "Synthetic payload should have modified ExtraData")

				// Verify ForkchoiceUpdate was called twice (once for synthetic, once for target)
				require.Equal(t, 2, l2.fcuCalls, "ForkchoiceUpdate should be called twice")
			}
		})
	}
}
