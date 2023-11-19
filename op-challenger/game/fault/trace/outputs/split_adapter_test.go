package outputs

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/split"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var creatorError = errors.New("captured args")

func TestOutputRootSplitAdapter(t *testing.T) {
	tests := []struct {
		name                    string
		preTraceIndex           int64
		postTraceIndex          int64
		expectedAgreedBlockNum  int64
		expectedClaimedBlockNum int64
	}{
		{
			name:                    "middleOfBlockRange",
			preTraceIndex:           5,
			postTraceIndex:          9,
			expectedAgreedBlockNum:  26,
			expectedClaimedBlockNum: 30,
		},
		{
			name:                    "beyondPostBlock",
			preTraceIndex:           5,
			postTraceIndex:          50,
			expectedAgreedBlockNum:  26,
			expectedClaimedBlockNum: 40,
		},
		{
			name:                    "firstBlock",
			preTraceIndex:           0,
			postTraceIndex:          1,
			expectedAgreedBlockNum:  21,
			expectedClaimedBlockNum: 22,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			topDepth := 10
			adapter, creator := setupAdapterTest(t, topDepth)
			preClaim := types.Claim{
				ClaimData: types.ClaimData{
					Value:    common.Hash{0xaa},
					Position: types.NewPosition(topDepth, big.NewInt(test.preTraceIndex)),
				},
				ContractIndex:       3,
				ParentContractIndex: 2,
			}
			postClaim := types.Claim{
				ClaimData: types.ClaimData{
					Value:    common.Hash{0xbb},
					Position: types.NewPosition(topDepth, big.NewInt(test.postTraceIndex)),
				},
				ContractIndex:       7,
				ParentContractIndex: 1,
			}

			expectedLocalContext := uint64(0x300000007)
			expectedAgreed := contracts.Proposal{
				L2BlockNumber: big.NewInt(test.expectedAgreedBlockNum),
				OutputRoot:    preClaim.Value,
			}
			expectedClaimed := contracts.Proposal{
				L2BlockNumber: big.NewInt(test.expectedClaimedBlockNum),
				OutputRoot:    postClaim.Value,
			}

			_, err := adapter(context.Background(), preClaim, postClaim)
			require.ErrorIs(t, err, creatorError)
			require.Equal(t, expectedLocalContext, creator.localContext)
			require.Equal(t, expectedAgreed, creator.agreed)
			require.Equal(t, expectedClaimed, creator.claimed)
		})
	}
}

func TestOutputRootSplitAdapter_FromAbsolutePrestate(t *testing.T) {
	topDepth := 10
	adapter, creator := setupAdapterTest(t, topDepth)

	postClaim := types.Claim{
		ClaimData: types.ClaimData{
			Value:    common.Hash{0xbb},
			Position: types.NewPosition(topDepth, big.NewInt(0)),
		},
		ContractIndex:       7,
		ParentContractIndex: 1,
	}

	// Use MaxUint32 for the pre-state block contract index.
	expectedLocalContext := uint64(0xffffffff00000007)
	expectedAgreed := contracts.Proposal{
		L2BlockNumber: big.NewInt(20),
		OutputRoot:    prestateOutputRoot, // Absolute prestate output root
	}
	expectedClaimed := contracts.Proposal{
		L2BlockNumber: big.NewInt(21),
		OutputRoot:    postClaim.Value,
	}

	_, err := adapter(context.Background(), types.Claim{}, postClaim)
	require.ErrorIs(t, err, creatorError)
	require.Equal(t, expectedLocalContext, creator.localContext)
	require.Equal(t, expectedAgreed, creator.agreed)
	require.Equal(t, expectedClaimed, creator.claimed)
}

func setupAdapterTest(t *testing.T, topDepth int) (split.ProviderCreator, *capturingCreator) {
	prestateBlock := uint64(20)
	poststateBlock := uint64(40)
	creator := &capturingCreator{}
	rollupClient := &stubRollupClient{
		outputs: map[uint64]*eth.OutputResponse{
			prestateBlock: {
				OutputRoot: eth.Bytes32(prestateOutputRoot),
			},
		},
	}
	topProvider := NewTraceProviderFromInputs(testlog.Logger(t, log.LvlInfo), rollupClient, uint64(topDepth), prestateBlock, poststateBlock)
	adapter := OutputRootSplitAdapter(topProvider, creator.Create)
	return adapter, creator
}

type capturingCreator struct {
	localContext uint64
	agreed       contracts.Proposal
	claimed      contracts.Proposal
}

func (c *capturingCreator) Create(_ context.Context, localContext uint64, agreed contracts.Proposal, claimed contracts.Proposal) (types.TraceProvider, error) {
	c.localContext = localContext
	c.agreed = agreed
	c.claimed = claimed
	return nil, creatorError
}
