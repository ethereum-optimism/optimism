package genesis

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/triedb"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
)

// ToBlock returns the genesis block. It is go-ethereum's genesis block for GethGenesis,
// except that from Isthmus the header's withdrawals root is the storage root of the
// L2ToL1MessagePasser predeploy instead of the empty withdrawals root.
func (g *Genesis) ToBlock() *types.Block {
	block := g.GethGenesis().ToBlock()
	if !g.Config.IsIsthmus(g.Timestamp) {
		return block
	}
	header := block.Header()
	root := storageRoot(g.Alloc, predeploys.L2ToL1MessagePasserAddr)
	header.WithdrawalsHash = &root
	return block.WithSeal(header)
}

// storageRoot returns the storage root of the account at addr in alloc, or the zero hash
// if alloc has no such account.
func storageRoot(alloc types.GenesisAlloc, addr common.Address) common.Hash {
	account, ok := alloc[addr]
	if !ok {
		return common.Hash{}
	}
	db := state.NewDatabase(triedb.NewDatabase(rawdb.NewMemoryDatabase(), nil), nil)
	statedb, err := state.New(types.EmptyRootHash, db)
	if err != nil {
		panic(fmt.Errorf("creating in-memory state: %w", err))
	}
	statedb.CreateAccount(addr)
	for key, value := range account.Storage {
		statedb.SetState(addr, key, value)
	}
	if _, err := statedb.Commit(0, false, false); err != nil {
		panic(fmt.Errorf("committing genesis storage of %s: %w", addr, err))
	}
	return statedb.GetStorageRoot(addr)
}
