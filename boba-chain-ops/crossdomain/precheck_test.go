package crossdomain

import (
	"math/big"
	"testing"

	"github.com/bobanetwork/boba/boba-bindings/predeploys"
	"github.com/ledgerwatch/erigon-lib/chain"
	"github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon/core/types"
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
	g := &types.Genesis{
		Config: &chain.Config{
			ChainID: big.NewInt(2888),
		},
		Alloc: types.GenesisAlloc{
			predeploys.LegacyMessagePasserAddr: types.GenesisAccount{
				Storage: map[common.Hash]common.Hash{
					{0: 0xff}: {0: 0xff},
				},
			},
		},
	}

	_, err := PreCheckWithdrawals(g, nil, nil)
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
	g := &types.Genesis{
		Config: &chain.Config{
			ChainID: big.NewInt(2888),
		},
		Alloc: types.GenesisAlloc{
			predeploys.LegacyMessagePasserAddr: types.GenesisAccount{},
		},
	}
	for _, wd := range dbWds {
		slot, err := wd.StorageSlot()
		require.NoError(t, err)
		if g.Alloc[predeploys.LegacyMessagePasserAddr].Storage == nil {
			storage := types.GenesisAccount{
				Storage: map[common.Hash]common.Hash{
					slot: abiTrue,
				},
			}
			g.Alloc[predeploys.LegacyMessagePasserAddr] = storage
		} else {
			g.Alloc[predeploys.LegacyMessagePasserAddr].Storage[slot] = abiTrue
		}
	}
	return PreCheckWithdrawals(g, witnessWds, nil)
}
