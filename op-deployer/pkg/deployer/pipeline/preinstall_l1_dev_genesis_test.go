package pipeline

import (
	"crypto/sha256"
	"log/slog"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestPreinstallL1DevGenesis_RequestQueues(t *testing.T) {
	t.Parallel()
	_, artifactsFS := testutil.LocalArtifacts(t)
	logger := testlog.Logger(t, slog.LevelDebug)
	host := script.NewHost(logger, &foundry.ArtifactsFS{FS: artifactsFS}, nil, script.DefaultContext)
	require.NoError(t, host.EnableCheats())
	host.SetBalance(params.WithdrawalQueueAddress, uint256.NewInt(42))
	host.SetNonce(params.WithdrawalQueueAddress, 7)
	storageKey, storageValue := common.HexToHash("0x42"), common.HexToHash("0x1234")
	host.SetStorage(params.WithdrawalQueueAddress, storageKey, storageValue)
	env := &Env{Logger: logger, L1ScriptHost: host, Deployer: common.Address{0xdd}}
	intent := &state.Intent{
		L1ChainID: 900,
		L1DevGenesisParams: &state.L1DevGenesisParams{
			BlockParams:      state.L1DevGenesisBlockParams{Timestamp: 1_700_000_000},
			PragueTimeOffset: ptr.New(uint64(0)),
		},
	}
	st := &state.State{}
	require.NoError(t, PreinstallL1DevGenesis(env, intent, st))
	require.NoError(t, SealL1DevGenesis(env, intent, st))

	for _, queue := range []struct {
		name     string
		addr     common.Address
		codeHash common.Hash
		balance  uint64
		nonce    uint64
	}{
		{"withdrawal", params.WithdrawalQueueAddress, common.Hash(sha256.Sum256(params.WithdrawalQueueCode)), 42, 7},
		{"consolidation", params.ConsolidationQueueAddress, common.Hash(sha256.Sum256(params.ConsolidationQueueCode)), 0, 1},
		// SHA-256 digests of the canonical EIP-8282 runtime code in go-ethereum v1.17.7.
		{"builder deposit", common.HexToAddress("0x0000bff46984e3725691fa540a8c7589300d8282"), common.HexToHash("0x2c49dcf745b1304f3dac0ea7487eae6d8fd07812ada980d542f79e8e5e53eb8d"), 0, 1},
		{"builder exit", common.HexToAddress("0x000064d678505ad48f8ccb093bc65613800e8282"), common.HexToHash("0xc889ed88730d157d192aae28c2dee61324d0df3bd01ff0078386808b4adb27aa"), 0, 1},
	} {
		t.Run(queue.name, func(t *testing.T) {
			account, ok := st.L1StateDump.Data.Accounts[queue.addr]
			require.True(t, ok, "request queue must be present in the L1 genesis alloc")
			require.Equal(t, queue.codeHash, common.Hash(sha256.Sum256(account.Code)))
			require.Equal(t, queue.nonce, account.Nonce)
			require.Zero(t, account.Balance.Cmp(uint256.NewInt(queue.balance).ToBig()))
		})
	}

	require.Equal(t, storageValue, st.L1StateDump.Data.Accounts[params.WithdrawalQueueAddress].Storage[storageKey])

	// The exported alloc must include the queues before the genesis state root is sealed.
	genesis := *st.L1DevGenesis
	genesis.StateHash = nil
	genesis.Alloc = st.L1StateDump.Data.Accounts
	require.Equal(t, *st.L1DevGenesis.StateHash, genesis.ToBlock().Root())
}
