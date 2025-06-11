package validators

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/bindings"
	"github.com/ethereum-optimism/optimism/devnet-sdk/interfaces"
	"github.com/ethereum-optimism/optimism/devnet-sdk/system"
	"github.com/ethereum-optimism/optimism/devnet-sdk/testing/systest"
	"github.com/ethereum-optimism/optimism/devnet-sdk/types"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

func Uint64Ptr(x uint64) *uint64 {
	return &x
}

// TestSystemTestHelper tests the basic implementation of systemTestHelper
func TestValidators(t *testing.T) {
	t.Run("multiple validators", func(t *testing.T) {
		walletGetter1, validator1 := AcquireL2WalletWithFunds(0, types.NewBalance(big.NewInt(1)))
		walletGetter2, validator2 := AcquireL2WalletWithFunds(0, types.NewBalance(big.NewInt(10)))
		lowLevelSystemGetter, validator3 := AcquireLowLevelSystem()
		chainConfigGetter, l2ForkValidator := AcquireL2WithFork(0, rollup.Isthmus)

		// We create a system that has a low-level L1 chain and at least one wallet
		systestSystem := &mockSystem{
			l1: &mockChain{},
			l2s: []system.L2Chain{
				&mockL2Chain{
					mockChain: mockChain{
						wallets: system.WalletMap{
							"user1": &mockWallet{
								address: types.Address(common.HexToAddress("0x1")),
								balance: types.NewBalance(big.NewInt(2)),
							},
							"user2": &mockWallet{
								address: types.Address(common.HexToAddress("0x2")),
								balance: types.NewBalance(big.NewInt(11)),
							},
						},
						config: &params.ChainConfig{
							Optimism:    &params.OptimismConfig{},
							IsthmusTime: Uint64Ptr(0),
						},
					},
				},
			},
		}

		// Now we apply all validators, accumulating contexts

		systestT := systest.NewT(t)

		ctx1, err := validator1(systestT, systestSystem)
		systestT = systestT.WithContext(ctx1)
		require.NoError(t, err)

		ctx2, err := validator2(systestT, systestSystem)
		systestT = systestT.WithContext(ctx2)
		require.NoError(t, err)

		ctx3, err := validator3(systestT, systestSystem)
		systestT = systestT.WithContext(ctx3)
		require.NoError(t, err)

		ctx4, err := l2ForkValidator(systestT, systestSystem)
		systestT = systestT.WithContext(ctx4)
		require.NoError(t, err)

		ctx := systestT.Context()

		// Now we call all the getters to make sure they work
		wallet1 := walletGetter1(ctx)
		wallet2 := walletGetter2(ctx)
		lowLevelSystem := lowLevelSystemGetter(ctx)
		chainConfig := chainConfigGetter(ctx)

		// And we ensure that the values are not mismatched
		require.NotEqual(t, wallet1, wallet2)
		require.NotEqual(t, wallet1, lowLevelSystem)
		require.NotEqual(t, wallet2, lowLevelSystem)

		// And that we got a lowlevelSystem
		require.NotNil(t, lowLevelSystem)

		// And that we got a chain config
		require.NotNil(t, chainConfig)
	})

	t.Run("test AcquireL2WithFork - fork active", func(t *testing.T) {
		// Create a system with the Isthmus fork active
		systestSystem := &mockSystem{
			l2s: []system.L2Chain{
				&mockL2Chain{
					mockChain: mockChain{
						config: &params.ChainConfig{
							Optimism:    &params.OptimismConfig{},
							IsthmusTime: Uint64Ptr(50),
						},
					},
				},
			},
		}

		// Get the validator for requiring Isthmus fork to be active
		chainConfigGetter, validator := AcquireL2WithFork(0, rollup.Isthmus)
		systestT := systest.NewT(t)

		// Apply the validator
		ctx, err := validator(systestT, systestSystem)
		require.NoError(t, err, "Validator should pass when fork is active")

		// Verify the chain config getter works
		chainConfig := chainConfigGetter(ctx)
		require.NotNil(t, chainConfig)
		isActive, err := IsForkActivated(chainConfig, rollup.Isthmus, 100)
		require.NoError(t, err)
		require.True(t, isActive)
	})

	t.Run("test AcquireL2WithFork - fork not active", func(t *testing.T) {
		// Create a system where the Isthmus fork is not yet active
		systestSystem := &mockSystem{
			l2s: []system.L2Chain{
				&mockL2Chain{
					mockChain: mockChain{
						config: &params.ChainConfig{
							Optimism:    &params.OptimismConfig{},
							IsthmusTime: Uint64Ptr(150),
						},
					},
				},
			},
		}

		// Get the validator for requiring Isthmus fork to be active
		_, validator := AcquireL2WithFork(0, rollup.Isthmus)
		systestT := systest.NewT(t)

		// Apply the validator - should fail since fork is not active
		_, err := validator(systestT, systestSystem)
		require.Error(t, err, "Validator should fail when fork is not active")
		require.Contains(t, err.Error(), "does not have fork", "Error message should indicate fork is not active")
	})

	t.Run("test AcquireRequiresNotL2Fork - fork not active", func(t *testing.T) {
		// Create a system where the Isthmus fork is not yet active
		systestSystem := &mockSystem{
			l2s: []system.L2Chain{
				&mockL2Chain{
					mockChain: mockChain{
						config: &params.ChainConfig{
							Optimism:    &params.OptimismConfig{},
							IsthmusTime: Uint64Ptr(150), // Activates after current timestamp
						},
					},
				},
			},
		}

		// Get the validator for requiring Isthmus fork to not be active
		chainConfigGetter, validator := AcquireL2WithoutFork(0, rollup.Isthmus)
		systestT := systest.NewT(t)

		// Apply the validator
		ctx, err := validator(systestT, systestSystem)
		require.NoError(t, err, "Validator should pass when fork is not active")

		// Verify the chain config getter works
		chainConfig := chainConfigGetter(ctx)
		require.NotNil(t, chainConfig)
		isActive, err := IsForkActivated(chainConfig, rollup.Isthmus, 100)
		require.NoError(t, err)
		require.False(t, isActive)
	})

	t.Run("test AcquireRequiresNotL2Fork - fork active", func(t *testing.T) {
		// Create a system with the Isthmus fork active
		systestSystem := &mockSystem{
			l2s: []system.L2Chain{
				&mockL2Chain{
					mockChain: mockChain{
						config: &params.ChainConfig{
							Optimism:    &params.OptimismConfig{},
							IsthmusTime: Uint64Ptr(50),
						},
					},
				},
			},
		}

		// Get the validator for requiring Isthmus fork to not be active
		_, validator := AcquireL2WithoutFork(0, rollup.Isthmus)
		systestT := systest.NewT(t)

		// Apply the validator - should fail since fork is active
		_, err := validator(systestT, systestSystem)
		require.Error(t, err, "Validator should fail when fork is active")
		require.Contains(t, err.Error(), "has fork", "Error message should indicate fork is active")
	})

	t.Run("chain index out of range", func(t *testing.T) {
		// Create a system with no L2 chains
		systestSystem := &mockSystem{
			l2s: []system.L2Chain{},
		}

		// Try to get chain config for an invalid chain index
		_, validator := AcquireL2WithFork(0, rollup.Isthmus)
		systestT := systest.NewT(t)

		// Apply the validator - should fail since chain index is out of range
		_, err := validator(systestT, systestSystem)
		require.Error(t, err, "Validator should fail when chain index is out of range")
		require.Contains(t, err.Error(), "chain index 0 out of range", "Error message should indicate chain index out of range")
	})
}

type mockSystem struct {
	l1  system.Chain
	l2s []system.L2Chain
}

func (sys *mockSystem) Identifier() string {
	return "mock"
}

func (sys *mockSystem) L1() system.Chain {
	return sys.l1
}

func (sys *mockSystem) L2s() []system.L2Chain {
	return sys.l2s
}

type mockChain struct {
	wallets system.WalletMap
	config  *params.ChainConfig
}

func (m *mockChain) RPCURL() string                                  { return "http://localhost:8545" }
func (m *mockChain) Client() (*sources.EthClient, error)             { return nil, nil }
func (m *mockChain) GethClient() (*ethclient.Client, error)          { return nil, nil }
func (m *mockChain) ID() types.ChainID                               { return types.ChainID(big.NewInt(1)) }
func (m *mockChain) ContractsRegistry() interfaces.ContractsRegistry { return nil }
func (m *mockChain) Wallets() system.WalletMap {
	return m.wallets
}
func (m *mockChain) GasPrice(ctx context.Context) (*big.Int, error) {
	return big.NewInt(1), nil
}
func (m *mockChain) GasLimit(ctx context.Context, tx system.TransactionData) (uint64, error) {
	return 1000000, nil
}
func (m *mockChain) PendingNonceAt(ctx context.Context, address common.Address) (uint64, error) {
	return 0, nil
}
func (m *mockChain) SupportsEIP(ctx context.Context, eip uint64) bool {
	return true
}
func (m *mockChain) Config() (*params.ChainConfig, error) {
	if m.config == nil {
		return nil, fmt.Errorf("chain config not implemented")
	}
	return m.config, nil
}
func (m *mockChain) Node() system.Node {
	return newMockNode(m.config)
}

func (m *mockChain) Addresses() system.AddressMap {
	return system.AddressMap{}
}

type mockL2Chain struct {
	mockChain
	l1Wallets system.WalletMap
}

func (m *mockL2Chain) L1Addresses() system.AddressMap {
	return system.AddressMap{}
}
func (m *mockL2Chain) L1Wallets() system.WalletMap {
	return m.l1Wallets
}

type mockNode struct {
	chainConfig *params.ChainConfig
}

func newMockNode(chainConfig *params.ChainConfig) *mockNode {
	return &mockNode{
		chainConfig: chainConfig,
	}
}

func (m *mockNode) GasPrice(ctx context.Context) (*big.Int, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockNode) GasLimit(ctx context.Context, tx system.TransactionData) (uint64, error) {
	return 0, fmt.Errorf("not implemented")
}

func (m *mockNode) PendingNonceAt(ctx context.Context, addr common.Address) (uint64, error) {
	return 0, fmt.Errorf("not implemented")
}

func (m *mockNode) BlockByNumber(ctx context.Context, number *big.Int) (eth.BlockInfo, error) {
	header := ethtypes.Header{Time: 100}
	info := eth.HeaderBlockInfo(&header)
	return info, nil
}

type mockWallet struct {
	balance types.Balance
	address types.Address
}

func (m mockWallet) Balance() types.Balance {
	return m.balance
}

func (m mockWallet) Address() types.Address {
	return m.address
}

func (m mockWallet) PrivateKey() types.Key {
	key, _ := crypto.HexToECDSA("123")
	return types.Key(key)
}

func (m mockWallet) SendETH(to types.Address, amount types.Balance) types.WriteInvocation[any] {
	panic("not implemented")
}

func (m mockWallet) InitiateMessage(chainID types.ChainID, target common.Address, message []byte) types.WriteInvocation[any] {
	panic("not implemented")
}

func (m mockWallet) ExecuteMessage(identifier bindings.Identifier, sentMessage []byte) types.WriteInvocation[any] {
	panic("not implemented")
}

func (m mockWallet) Nonce() uint64 {
	return 0
}

func (m mockWallet) Sign(tx system.Transaction) (system.Transaction, error) {
	return tx, nil
}

func (m mockWallet) Send(ctx context.Context, tx system.Transaction) error {
	return nil
}

func (m mockWallet) Transactor() *bind.TransactOpts {
	return nil
}

var (
	_ system.Chain           = (*mockChain)(nil)
	_ system.LowLevelChain   = (*mockChain)(nil)
	_ system.L2Chain         = (*mockL2Chain)(nil)
	_ system.LowLevelL2Chain = (*mockL2Chain)(nil)
	_ system.System          = (*mockSystem)(nil)
	_ system.Wallet          = (*mockWallet)(nil)
)
