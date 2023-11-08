package cannon

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestFetchLocalInputs(t *testing.T) {
	ctx := context.Background()
	contract := &mockGameInputsSource{
		l1Head: common.Hash{0xcc},
		starting: contracts.Proposal{
			Index:         big.NewInt(6),
			L2BlockNumber: big.NewInt(2222),
			OutputRoot:    common.Hash{0xdd},
		},
		disputed: contracts.Proposal{
			Index:         big.NewInt(7),
			L2BlockNumber: big.NewInt(3333),
			OutputRoot:    common.Hash{0xee},
		},
	}
	l2Client := &mockL2DataSource{
		chainID: big.NewInt(88422),
		header: ethtypes.Header{
			Number: contract.starting.L2BlockNumber,
		},
	}

	inputs, err := fetchLocalInputs(ctx, contract, l2Client)
	require.NoError(t, err)

	require.Equal(t, contract.l1Head, inputs.L1Head)
	require.Equal(t, l2Client.header.Hash(), inputs.L2Head)
	require.EqualValues(t, contract.starting.OutputRoot, inputs.L2OutputRoot)
	require.EqualValues(t, contract.disputed.OutputRoot, inputs.L2Claim)
	require.Equal(t, contract.disputed.L2BlockNumber, inputs.L2BlockNumber)
}

type mockGameInputsSource struct {
	l1Head   common.Hash
	starting contracts.Proposal
	disputed contracts.Proposal
}

func (s *mockGameInputsSource) GetL1Head(_ context.Context) (common.Hash, error) {
	return s.l1Head, nil
}

func (s *mockGameInputsSource) GetProposals(_ context.Context) (contracts.Proposal, contracts.Proposal, error) {
	return s.starting, s.disputed, nil
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
