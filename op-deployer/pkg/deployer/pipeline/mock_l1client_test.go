package pipeline

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
)

// mockL1Client is a test-only L1Client that returns hardcoded values,
// removing the need for a real Sepolia RPC connection.
type mockL1Client struct {
	chainID      *big.Int
	codeByAddr   map[common.Address][]byte
	chainIDErr   error
	codeAtErr    error
	rpcClient    *rpc.Client
}

func newSepoliaMockL1Client() *mockL1Client {
	return &mockL1Client{
		chainID: big.NewInt(11155111),
		codeByAddr: map[common.Address][]byte{
			// DeterministicDeployerAddress has code on Sepolia
			common.HexToAddress("0x4e59b44847b379578588920cA78FbF26c0B4956C"): {0x01},
		},
	}
}

func (m *mockL1Client) ChainID(_ context.Context) (*big.Int, error) {
	if m.chainIDErr != nil {
		return nil, m.chainIDErr
	}
	return m.chainID, nil
}

func (m *mockL1Client) CodeAt(_ context.Context, account common.Address, _ *big.Int) ([]byte, error) {
	if m.codeAtErr != nil {
		return nil, m.codeAtErr
	}
	code, ok := m.codeByAddr[account]
	if !ok {
		return nil, nil
	}
	return code, nil
}

func (m *mockL1Client) Client() *rpc.Client {
	return m.rpcClient
}
