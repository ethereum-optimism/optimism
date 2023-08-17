package cannon

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestFetchLocalInputs(t *testing.T) {
	ctx := context.Background()
	gameAddr := common.Address{0xab}
	l1Client := &mockGameInputsSource{
		l1Head: common.Hash{0xcc},
		starting: bindings.IFaultDisputeGameOutputProposal{
			Index:         big.NewInt(6),
			L2BlockNumber: big.NewInt(2222),
			OutputRoot:    common.Hash{0xdd},
		},
		disputed: bindings.IFaultDisputeGameOutputProposal{
			Index:         big.NewInt(7),
			L2BlockNumber: big.NewInt(3333),
			OutputRoot:    common.Hash{0xee},
		},
	}
	l2Client := &mockL2DataSource{
		chainID: big.NewInt(88422),
		header: ethtypes.Header{
			Number: l1Client.starting.L2BlockNumber,
		},
	}

	inputs, err := fetchLocalInputs(ctx, gameAddr, l1Client, l2Client)
	require.NoError(t, err)

	require.Equal(t, l1Client.l1Head, inputs.L1Head)
	require.Equal(t, l2Client.header.Hash(), inputs.L2Head)
	require.EqualValues(t, l1Client.starting.OutputRoot, inputs.L2OutputRoot)
	require.EqualValues(t, l1Client.disputed.OutputRoot, inputs.L2Claim)
	require.Equal(t, l1Client.disputed.L2BlockNumber, inputs.L2BlockNumber)
}

type mockGameInputsSource struct {
	l1Head   common.Hash
	starting bindings.IFaultDisputeGameOutputProposal
	disputed bindings.IFaultDisputeGameOutputProposal
}

func (s *mockGameInputsSource) L1Head(opts *bind.CallOpts) ([32]byte, error) {
	return s.l1Head, nil
}

func (s *mockGameInputsSource) Proposals(opts *bind.CallOpts) (struct {
	Starting bindings.IFaultDisputeGameOutputProposal
	Disputed bindings.IFaultDisputeGameOutputProposal
}, error) {
	return struct {
		Starting bindings.IFaultDisputeGameOutputProposal
		Disputed bindings.IFaultDisputeGameOutputProposal
	}{
		Starting: s.starting,
		Disputed: s.disputed,
	}, nil
}

type mockL2DataSource struct {
	chainID *big.Int
	header  ethtypes.Header
}

func (s *mockL2DataSource) ChainID(ctx context.Context) (*big.Int, error) {
	return s.chainID, nil
}

func (s *mockL2DataSource) HeaderByNumber(ctx context.Context, num *big.Int) (*ethtypes.Header, error) {
	if s.header.Number.Cmp(num) == 0 {
		return &s.header, nil
	}
	return nil, ethereum.NotFound
}
