package crossdomain

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/stretchr/testify/require"
)

func TestPreCheckWithdrawals_Filtering(t *testing.T) {
	dbWds := []*LegacyWithdrawal{
		// Random legacy WD to something other than the L2XDM.
		{
			MessageSender: common.Address{19: 0xFF},
			XDomainTarget: common.Address{19: 0x01},
			XDomainSender: common.Address{19: 0x02},
			XDomainData:   []byte{0x01, 0x02, 0x03},
			XDomainNonce:  big.NewInt(0),
		},
		// Random legacy WD to the L2XDM. Should be the only thing
		// returned by the prechecker.
		{
			MessageSender: predeploys.L2CrossDomainMessengerAddr,
			XDomainTarget: common.Address{19: 0x01},
			XDomainSender: common.Address{19: 0x02},
			XDomainData:   []byte{0x01, 0x02, 0x03},
			XDomainNonce:  big.NewInt(1),
		},
	}

	// Add an additional witness to the witnesses list to
	// test how the prechecker handles witness data that
	// isn't in state.
	witnessWds := append([]*LegacyWithdrawal{
		{
			MessageSender: common.Address{19: 0xAA},
			XDomainTarget: common.Address{19: 0x03},
			XDomainSender: predeploys.L2CrossDomainMessengerAddr,
			XDomainData:   []byte{0x01, 0x02, 0x03},
			XDomainNonce:  big.NewInt(0),
		},
	}, dbWds...)

	filteredWds, err := runPrecheck(t, dbWds, witnessWds)
	require.NoError(t, err)
	require.EqualValues(t, []*LegacyWithdrawal{dbWds[1]}, filteredWds)
}

func TestPreCheckWithdrawals_InvalidSlotInStorage(t *testing.T) {
	rawDB := rawdb.NewMemoryDatabase()
	rawStateDB := state.NewDatabaseWithConfig(rawDB, &trie.Config{
		Preimages: true,
		Cache:     1024,
	})
	stateDB, err := state.New(common.Hash{}, rawStateDB, nil)
	require.NoError(t, err)

	// Create account, and set a random storage slot to a value
	// other than abiTrue.
	stateDB.CreateAccount(predeploys.LegacyMessagePasserAddr)
	stateDB.SetState(predeploys.LegacyMessagePasserAddr, common.Hash{0: 0xff}, common.Hash{0: 0xff})

	root, err := stateDB.Commit(false)
	require.NoError(t, err)

	err = stateDB.Database().TrieDB().Commit(root, true)
	require.NoError(t, err)

	_, err = PreCheckWithdrawals(stateDB, nil, nil)
	require.ErrorIs(t, err, ErrUnknownSlotInMessagePasser)
}

func TestPreCheckWithdrawals_MissingStorageSlot(t *testing.T) {
	// Add a legacy WD to state that does not appear in witness data.
	dbWds := []*LegacyWithdrawal{
		{
			XDomainTarget: common.Address{19: 0x01},
			XDomainSender: predeploys.L2CrossDomainMessengerAddr,
			XDomainData:   []byte{0x01, 0x02, 0x03},
			XDomainNonce:  big.NewInt(1),
		},
	}

	// Create some witness data that includes both a valid
	// and an invalid witness, but neither of which correspond
	// to the value above in state.
	witnessWds := []*LegacyWithdrawal{
		{
			XDomainTarget: common.Address{19: 0x01},
			XDomainSender: common.Address{19: 0x02},
			XDomainData:   []byte{0x01, 0x02, 0x03},
			XDomainNonce:  big.NewInt(0),
		},
		{
			XDomainTarget: common.Address{19: 0x03},
			XDomainSender: predeploys.L2CrossDomainMessengerAddr,
			XDomainData:   []byte{0x01, 0x02, 0x03},
			XDomainNonce:  big.NewInt(0),
		},
	}

	_, err := runPrecheck(t, dbWds, witnessWds)
	require.ErrorIs(t, err, ErrMissingSlotInWitness)
}

func runPrecheck(t *testing.T, dbWds []*LegacyWithdrawal, witnessWds []*LegacyWithdrawal) ([]*LegacyWithdrawal, error) {
	rawDB := rawdb.NewMemoryDatabase()
	rawStateDB := state.NewDatabaseWithConfig(rawDB, &trie.Config{
		Preimages: true,
		Cache:     1024,
	})
	stateDB, err := state.New(common.Hash{}, rawStateDB, nil)
	require.NoError(t, err)

	stateDB.CreateAccount(predeploys.LegacyMessagePasserAddr)
	for _, wd := range dbWds {
		slot, err := wd.StorageSlot()
		require.NoError(t, err)
		stateDB.SetState(predeploys.LegacyMessagePasserAddr, slot, abiTrue)
	}

	root, err := stateDB.Commit(false)
	require.NoError(t, err)

	err = stateDB.Database().TrieDB().Commit(root, true)
	require.NoError(t, err)

	return PreCheckWithdrawals(stateDB, witnessWds, nil)
}
