package driver

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimistic-specs/opnode/eth"
	"github.com/ethereum-optimism/optimistic-specs/opnode/rollup/sync"
)

func fakeID(id rune, num uint64) eth.BlockID {
	var h common.Hash
	copy(h[:], string(id))
	return eth.BlockID{Hash: h, Number: uint64(num)}
}

func fakeL1Block(self rune, parent rune, num uint64) eth.L1Node {
	var parentID eth.BlockID
	if num != 0 {
		parentID = fakeID(parent, num-1)
	}
	return eth.L1Node{Self: fakeID(self, num), Parent: parentID}
}

func fakeL2Block(self rune, parent rune, l1parent eth.BlockID, num uint64) eth.L2Node {
	var parentID eth.BlockID
	if num != 0 {
		parentID = fakeID(parent, num-1)
	}
	return eth.L2Node{Self: fakeID(self, num), L2Parent: parentID, L1Parent: l1parent}
}

func chainL1(offset uint64, ids string) (out []eth.L1Node) {
	var prevID rune
	for i, id := range ids {
		out = append(out, fakeL1Block(id, prevID, offset+uint64(i)))
		prevID = id
	}
	return
}

func chainL2(l1 []eth.L1Node, ids string) (out []eth.L2Node) {
	var prevID rune
	for i, id := range ids {
		out = append(out, fakeL2Block(id, prevID, l1[i].Self, uint64(i)))
		prevID = id
	}
	return
}

func NewFakeChainSource(l1 []string, l2 []string, log log.Logger) *fakeChainSource {
	var l1s [][]eth.L1Node
	for _, l1string := range l1 {
		l1s = append(l1s, chainL1(0, l1string))
	}
	var l2s [][]eth.L2Node
	for i, l2string := range l2 {
		l2s = append(l2s, chainL2(l1s[i], l2string))
	}
	return &fakeChainSource{
		l1s: l1s,
		l2s: l2s,
		log: log,
	}
}

// fakeChainSource implements the ChainSource interface with the ability to control
// what the head block is of the L1 and L2 chains. In addition, it enables re-orgs
// to easily be implemented
type fakeChainSource struct {
	reorg  int            // Index of which chain to be operating on
	l1head int            // Head block of the L1 chain
	l2head int            // Head block of the L2 chain
	l1s    [][]eth.L1Node // l1s[reorg] is the L1 chain in that specific re-org configuration
	l2s    [][]eth.L2Node // l2s[reorg] is the L2 chain in that specific re-org configuration
	log    log.Logger
}

func (m *fakeChainSource) L1NodeByNumber(ctx context.Context, l1Num uint64) (eth.L1Node, error) {
	m.log.Trace("L1NodeByNumber", "l1Num", l1Num, "l1Head", m.l1head, "reorg", m.reorg)
	if l1Num > uint64(m.l1head) {
		return eth.L1Node{}, ethereum.NotFound
	}
	return m.l1s[m.reorg][l1Num], nil
}

func (m *fakeChainSource) L1HeadNode(ctx context.Context) (eth.L1Node, error) {
	m.log.Trace("L1HeadNode", "l1Head", m.l1head, "reorg", m.reorg)
	l := len(m.l1s[m.reorg])
	if l == 0 {
		return eth.L1Node{}, ethereum.NotFound
	}
	return m.l1s[m.reorg][m.l1head], nil
}

func (m *fakeChainSource) L2NodeByNumber(ctx context.Context, l2Num *big.Int) (eth.L2Node, error) {
	m.log.Trace("L2NodeByNumber", "l2Num", l2Num, "l2Head", m.l2head, "reorg", m.reorg)
	if len(m.l2s[m.reorg]) == 0 {
		panic("bad test, no l2 chain")
	}
	if l2Num == nil {
		return m.l2s[m.reorg][m.l2head], nil
	}
	i := int(l2Num.Int64())
	if i > m.l2head {
		return eth.L2Node{}, ethereum.NotFound
	}
	return m.l2s[m.reorg][i], nil
}

func (m *fakeChainSource) L2NodeByHash(ctx context.Context, l2Hash common.Hash) (eth.L2Node, error) {
	m.log.Trace("L2NodeByHash", "l2Hash", l2Hash, "l2Head", m.l2head, "reorg", m.reorg)
	for i, bl := range m.l2s[m.reorg] {
		if bl.Self.Hash == l2Hash {
			return m.L2NodeByNumber(ctx, big.NewInt(int64(i)))
		}
	}
	return eth.L2Node{}, ethereum.NotFound
}

var _ sync.ChainSource = (*fakeChainSource)(nil)

func (m *fakeChainSource) reorgChains(reorgBase int) {
	m.log.Trace("Reorg", "new_reorg", m.reorg+1, "old_reorg", m.reorg)
	m.reorg++
	if m.reorg >= len(m.l1s) {
		panic("No more re-org chains available")
	}
	m.l2head = reorgBase
}

func (m *fakeChainSource) advanceL1() eth.L1Node {
	m.log.Trace("Advance L1", "new_head", m.l1head+1, "old_head", m.l1head)
	m.l1head++
	if m.l1head >= len(m.l1s[m.reorg]) {
		panic("Cannot advance L1 past end of chain")
	}
	return m.l1s[m.reorg][m.l1head]
}

func (m *fakeChainSource) l1Head() eth.L1Node {
	m.log.Trace("L1 Head", "head", m.l1head)
	return m.l1s[m.reorg][m.l1head]
}

func (m *fakeChainSource) advanceL2() eth.L2Node {
	m.log.Trace("Advance L2", "new_head", m.l2head+1, "old_head", m.l2head)
	m.l2head++
	if m.l2head >= len(m.l2s[m.reorg]) {
		panic("Cannot advance L2 past end of chain")
	}
	return m.l2s[m.reorg][m.l2head]
}

// unused
// func (m *fakeChainSource) l2Head() eth.L2Node {
// 	m.log.Trace("L2 Head", "head", m.l2head)
// 	return m.l2s[m.reorg][m.l2head]
// }
