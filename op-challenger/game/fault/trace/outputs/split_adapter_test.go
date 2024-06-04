package outputs

import (
	"context"
	"errors"
	"math"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/split"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
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
			topDepth := types.Depth(10)
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

			expectedAgreed := contracts.Proposal{
				L2BlockNumber: big.NewInt(test.expectedAgreedBlockNum),
				OutputRoot:    preClaim.Value,
			}
			expectedClaimed := contracts.Proposal{
				L2BlockNumber: big.NewInt(test.expectedClaimedBlockNum),
				OutputRoot:    postClaim.Value,
			}

			_, err := adapter(context.Background(), 5, preClaim, postClaim)
			require.ErrorIs(t, err, creatorError)
			require.Equal(t, CreateLocalContext(preClaim, postClaim), creator.localContext)
			require.Equal(t, expectedAgreed, creator.agreed)
			require.Equal(t, expectedClaimed, creator.claimed)
		})
	}
}

func TestOutputRootSplitAdapter_FromAbsolutePrestate(t *testing.T) {
	topDepth := types.Depth(10)
	adapter, creator := setupAdapterTest(t, topDepth)

	postClaim := types.Claim{
		ClaimData: types.ClaimData{
			Value:    common.Hash{0xbb},
			Position: types.NewPosition(topDepth, big.NewInt(0)),
		},
		ContractIndex:       7,
		ParentContractIndex: 1,
	}

	expectedAgreed := contracts.Proposal{
		L2BlockNumber: big.NewInt(20),
		OutputRoot:    prestateOutputRoot, // Absolute prestate output root
	}
	expectedClaimed := contracts.Proposal{
		L2BlockNumber: big.NewInt(21),
		OutputRoot:    postClaim.Value,
	}

	_, err := adapter(context.Background(), 5, types.Claim{}, postClaim)
	require.ErrorIs(t, err, creatorError)
	require.Equal(t, CreateLocalContext(types.Claim{}, postClaim), creator.localContext)
	require.Equal(t, expectedAgreed, creator.agreed)
	require.Equal(t, expectedClaimed, creator.claimed)
}

func setupAdapterTest(t *testing.T, topDepth types.Depth) (split.ProviderCreator, *capturingCreator) {
	prestateBlock := uint64(20)
	poststateBlock := uint64(40)
	creator := &capturingCreator{}
	l1Head := eth.BlockID{
		Hash:   common.Hash{0x11, 0x11},
		Number: 11,
	}
	rollupClient := &stubRollupClient{
		outputs: map[uint64]*eth.OutputResponse{
			prestateBlock: {
				OutputRoot: eth.Bytes32(prestateOutputRoot),
			},
		},
		maxSafeHead: math.MaxUint64,
	}
	prestateProvider := &stubPrestateProvider{
		absolutePrestate: prestateOutputRoot,
	}
	topProvider := NewTraceProvider(testlog.Logger(t, log.LevelInfo), prestateProvider, rollupClient, nil, l1Head, topDepth, prestateBlock, poststateBlock)
	adapter := OutputRootSplitAdapter(topProvider, creator.Create)
	return adapter, creator
}

type capturingCreator struct {
	localContext common.Hash
	agreed       contracts.Proposal
	claimed      contracts.Proposal
}

func (c *capturingCreator) Create(_ context.Context, localContext common.Hash, _ types.Depth, agreed contracts.Proposal, claimed contracts.Proposal) (types.TraceProvider, error) {
	c.localContext = localContext
	c.agreed = agreed
	c.claimed = claimed
	return nil, creatorError
}

func TestCreateLocalContext(t *testing.T) {
	tests := []struct {
		name         string
		preValue     common.Hash
		prePosition  types.Position
		postValue    common.Hash
		postPosition types.Position
		expected     []byte
	}{
		{
			name:         "PreAndPost",
			preValue:     common.HexToHash("abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"),
			prePosition:  types.NewPositionFromGIndex(big.NewInt(2)),
			postValue:    common.HexToHash("cc00000000000000000000000000000000000000000000000000000000000000"),
			postPosition: types.NewPositionFromGIndex(big.NewInt(3)),
			expected:     common.FromHex("abcdef0123456789abcdef0123456789abcdef0123456789abcdef01234567890000000000000000000000000000000000000000000000000000000000000002cc000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000003"),
		},
		{
			name:         "LargePositions",
			preValue:     common.HexToHash("abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"),
			prePosition:  types.NewPositionFromGIndex(new(big.Int).SetBytes(common.FromHex("cbcdef0123456789abcdef0123456789abcdef0123456789abcdef012345678c"))),
			postValue:    common.HexToHash("dd00000000000000000000000000000000000000000000000000000000000000"),
			postPosition: types.NewPositionFromGIndex(new(big.Int).SetUint64(math.MaxUint64)),
			expected:     common.FromHex("abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789cbcdef0123456789abcdef0123456789abcdef0123456789abcdef012345678cdd00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000ffffffffffffffff"),
		},
		{
			name:         "AbsolutePreState",
			preValue:     common.Hash{},
			prePosition:  types.Position{},
			postValue:    common.HexToHash("cc00000000000000000000000000000000000000000000000000000000000000"),
			postPosition: types.NewPositionFromGIndex(big.NewInt(3)),
			expected:     common.FromHex("cc000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000003"),
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			pre := types.Claim{
				ClaimData: types.ClaimData{
					Value:    test.preValue,
					Position: test.prePosition,
				},
			}
			post := types.Claim{
				ClaimData: types.ClaimData{
					Value:    test.postValue,
					Position: test.postPosition,
				},
			}
			actualPreimage := localContextPreimage(pre, post)
			require.Equal(t, test.expected, actualPreimage)
			localContext := CreateLocalContext(pre, post)
			require.Equal(t, crypto.Keccak256Hash(test.expected), localContext)
		})
	}
}

type stubPrestateProvider struct {
	errorsOnAbsolutePrestateFetch bool
	absolutePrestate              common.Hash
}

func (s *stubPrestateProvider) AbsolutePreStateCommitment(_ context.Context) (common.Hash, error) {
	if s.errorsOnAbsolutePrestateFetch {
		return common.Hash{}, errNoOutputAtBlock
	}
	return s.absolutePrestate, nil
}
