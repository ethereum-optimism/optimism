package genesis

import (
	"bytes"
	"fmt"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"

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
	type leaf struct{ key, value []byte }
	leaves := make([]leaf, 0, len(account.Storage))
	for slot, value := range account.Storage {
		if value == (common.Hash{}) {
			continue // a zero value is the absence of the slot
		}
		enc, err := rlp.EncodeToBytes(common.TrimLeftZeroes(value[:]))
		if err != nil {
			panic(fmt.Errorf("encoding storage slot %s of %s: %w", slot, addr, err))
		}
		leaves = append(leaves, leaf{crypto.Keccak256(slot[:]), enc})
	}
	// A stack trie takes its keys in ascending order.
	slices.SortFunc(leaves, func(a, b leaf) int { return bytes.Compare(a.key, b.key) })
	st := trie.NewStackTrie(nil)
	for _, l := range leaves {
		if err := st.Update(l.key, l.value); err != nil {
			panic(fmt.Errorf("building genesis storage trie of %s: %w", addr, err))
		}
	}
	return st.Hash()
}
