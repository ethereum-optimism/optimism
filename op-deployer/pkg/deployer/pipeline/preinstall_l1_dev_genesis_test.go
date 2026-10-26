package pipeline

import (
	"crypto/sha256"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestPreinstallL1DevGenesis_RequestQueues(t *testing.T) {
	t.Parallel()
	env := newTestGenesisEnv(t, common.Address{0xdd})
	storageKey, storageValue := common.HexToHash("0x42"), common.HexToHash("0x1234")
	env.Forge.Genesis.Allocs.Accounts[params.WithdrawalQueueAddress] = types.Account{
		Balance: big.NewInt(42),
		Nonce:   7,
		Storage: map[common.Hash]common.Hash{storageKey: storageValue},
	}
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
			account, ok := st.L1DevGenesis.Alloc[queue.addr]
			require.True(t, ok, "request queue must be present in the L1 genesis alloc")
			require.Equal(t, queue.codeHash, common.Hash(sha256.Sum256(account.Code)))
			require.Equal(t, queue.nonce, account.Nonce)
			require.Zero(t, account.Balance.Cmp(uint256.NewInt(queue.balance).ToBig()))
		})
	}

	require.Equal(t, storageValue, st.L1DevGenesis.Alloc[params.WithdrawalQueueAddress].Storage[storageKey])
}
