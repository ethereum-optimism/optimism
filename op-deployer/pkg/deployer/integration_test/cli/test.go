package cli

import (
	"context"
	"crypto/ecdsa"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/integration_test/shared"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils/devnet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// SetupAnvilTest sets up an Anvil test environment and returns test utilities
func SetupAnvilTest(t *testing.T) (log.Logger, string, *ethclient.Client, string, *ecdsa.PrivateKey, *devkeys.MnemonicDevKeys, *uint256.Int) {
	lgr := testlog.Logger(t, log.LevelInfo)
	l1RPC, l1Client := devnet.DefaultAnvilRPC(t, lgr)
	pkHex, pk, dk := shared.DefaultPrivkey(t)
	l1ChainID := uint256.NewInt(devnet.DefaultChainID)

	return lgr, l1RPC, l1Client, pkHex, pk, dk, l1ChainID
}

// NewIntent creates a new intent for testing
func NewIntent(t *testing.T, l1ChainID *uint256.Int, dk *devkeys.MnemonicDevKeys, l2ChainID *uint256.Int, l1Loc *artifacts.Locator, l2Loc *artifacts.Locator) (*state.Intent, *state.State) {
	return shared.NewIntent(t, (*uint256.Int)(l1ChainID).ToBig(), dk, l2ChainID, l1Loc, l2Loc, 30000000)
}

// CodeGetter provides a way to get code at an address for validation
type CodeGetter func(t *testing.T, addr common.Address) []byte

// EthClientCodeGetter returns a function that gets code from an Ethereum client
func EthClientCodeGetter(ctx context.Context, client *ethclient.Client) CodeGetter {
	return func(t *testing.T, addr common.Address) []byte {
		code, err := client.CodeAt(ctx, addr, nil)
		require.NoError(t, err)
		return code
	}
}

// NewChainIntent creates a new chain intent for testing
func NewChainIntent(t *testing.T, dk *devkeys.MnemonicDevKeys, l1ChainID *uint256.Int, l2ChainID *uint256.Int) *state.ChainIntent {
	return shared.NewChainIntent(t, dk, l1ChainID.ToBig(), l2ChainID, 30000000)
}
