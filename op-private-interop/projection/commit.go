package projection

import (
	"encoding/binary"
	"fmt"
	"slices"

	"github.com/ethereum-optimism/optimism/op-private-interop/wire"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// Domain tags of the sp1-private-projection-v1 statement (§C.3.1). Byte-identical in Kona.
const (
	ConfigDomain        = "optimism.private-projection-config.v1\x00"
	PrivateConfigDomain = "optimism.private-config.v1\x00"
	DepSetDomain        = "optimism.private-dependency-set.v1\x00"
	OutputsDomain       = "optimism.private-outputs.v1\x00"
	MessagesDomain      = "optimism.private-messages.v1\x00"
)

// Message leaf kinds (§C.3.7). Kind 0x03 (generic EventReplayer) is not defined in v1.
const (
	MessageKindInit byte = 0x01 // export: SentMessage
	MessageKindExec byte = 0x02 // import: ExecutingMessage
)

// commitmentTop is the unwrapped tree top: zero for no leaves, else the RecordsRoot-shaped tree
// (nodes H(0x01 ‖ left ‖ right), duplicating the last node at odd levels).
func commitmentTop(leaves []common.Hash) common.Hash {
	if len(leaves) == 0 {
		return common.Hash{}
	}
	level := slices.Clone(leaves)
	for len(level) > 1 {
		next := make([]common.Hash, 0, (len(level)+1)/2)
		for i := 0; i < len(level); i += 2 {
			right := level[min(i+1, len(level)-1)]
			next = append(next, crypto.Keccak256Hash([]byte{1}, level[i][:], right[:]))
		}
		level = next
	}
	return level[0]
}

func wrapCommitment(domain string, n uint64, top common.Hash) common.Hash {
	var count [8]byte
	binary.BigEndian.PutUint64(count[:], n)
	return crypto.Keccak256Hash([]byte(domain), count[:], top[:])
}

// CommitmentRoot is keccak256(domain ‖ u64be(n) ‖ top) over the §C.3.5 tree. The leaf count is
// committed separately so odd-level duplication cannot alias different lengths.
func CommitmentRoot(domain string, leaves []common.Hash) common.Hash {
	return wrapCommitment(domain, uint64(len(leaves)), commitmentTop(leaves))
}

// CommitmentProof returns the bottom-up siblings of leaves[index]. At a level where the node is
// the last element of an odd-length level its sibling is itself, and no element is emitted.
func CommitmentProof(leaves []common.Hash, index int) ([]common.Hash, error) {
	if index < 0 || index >= len(leaves) {
		return nil, fmt.Errorf("commitment proof index %d out of range [0,%d)", index, len(leaves))
	}
	var siblings []common.Hash
	level := slices.Clone(leaves)
	for len(level) > 1 {
		if sib := index ^ 1; sib < len(level) {
			siblings = append(siblings, level[sib])
		}
		next := make([]common.Hash, 0, (len(level)+1)/2)
		for i := 0; i < len(level); i += 2 {
			right := level[min(i+1, len(level)-1)]
			next = append(next, crypto.Keccak256Hash([]byte{1}, level[i][:], right[:]))
		}
		level = next
		index /= 2
	}
	return siblings, nil
}

// VerifyCommitmentProof recomputes the root from one leaf. Level lengths are derived from n, the
// odd-level rule is applied identically, and every sibling must be consumed.
func VerifyCommitmentProof(domain string, n, index uint64, leaf common.Hash, siblings []common.Hash, root common.Hash) bool {
	if n == 0 || index >= n {
		return false
	}
	node, width, used := leaf, n, 0
	for width > 1 {
		if index%2 == 1 {
			if used >= len(siblings) {
				return false
			}
			node = crypto.Keccak256Hash([]byte{1}, siblings[used][:], node[:])
			used++
		} else if index+1 < width {
			if used >= len(siblings) {
				return false
			}
			node = crypto.Keccak256Hash([]byte{1}, node[:], siblings[used][:])
			used++
		} else {
			node = crypto.Keccak256Hash([]byte{1}, node[:], node[:])
		}
		index /= 2
		width = (width + 1) / 2
	}
	return used == len(siblings) && wrapCommitment(domain, n, node) == root
}

// OutputLeaf = keccak256(0x00 ‖ u64be(blockNumber) ‖ outputRoot).
func OutputLeaf(blockNumber uint64, outputRoot common.Hash) common.Hash {
	var n [8]byte
	binary.BigEndian.PutUint64(n[:], blockNumber)
	return crypto.Keccak256Hash([]byte{0}, n[:], outputRoot[:])
}

// MessageLeaf = keccak256(0x00 ‖ u64be(blockNumber) ‖ u32be(renderedIndex) ‖ u8(kind) ‖ messageHash).
func MessageLeaf(blockNumber uint64, renderedIndex uint32, kind byte, messageHash common.Hash) common.Hash {
	var n [8]byte
	var r [4]byte
	binary.BigEndian.PutUint64(n[:], blockNumber)
	binary.BigEndian.PutUint32(r[:], renderedIndex)
	return crypto.Keccak256Hash([]byte{0}, n[:], r[:], []byte{kind}, messageHash[:])
}

// ExportMessageHash is the interop payload hash keccak256(topic0 ‖ … ‖ topic3 ‖ data) of the
// SentMessage log the replay messenger emits for m.
func ExportMessageHash(m *wire.SentMessage) common.Hash {
	topics, data := wire.SentMessageLog(m)
	h := crypto.NewKeccakState()
	for _, t := range topics {
		_, _ = h.Write(t[:])
	}
	_, _ = h.Write(data)
	var out common.Hash
	_, _ = h.Read(out[:])
	return out
}

// ImportMessageHash = keccak256(identifierWords(160) ‖ payloadHash(32)): the 192 argument bytes of
// validateMessage(Identifier,bytes32), equal by ABI to ExecutingMessage's data ‖ topics[1].
func ImportMessageHash(args [192]byte) common.Hash {
	return crypto.Keccak256Hash(args[:])
}
