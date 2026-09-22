// Package projection validates complete public projection ranges before execution.
// Admission is pure; the separate ContextCollector resolves its explicit canonical inputs.
package projection

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"slices"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-private-interop/codec"
	"github.com/ethereum-optimism/optimism/op-private-interop/wire"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	InsecureStub   = "insecure-stub-v1"
	ExecutionMock  = "execution-mock-v1"
	MaxRangeBlocks = 65536
	MaxTxGas       = wire.MaxTxGas
)

// Config is consensus configuration, not an operator-selected verifier override.
type Config struct {
	Verifier          string      `json:"verifier"`
	AllowEvents       bool        `json:"allow_events,omitempty"`
	GenesisOutputRoot common.Hash `json:"genesis_output_root"`
}

func (c *Config) Check() error {
	if c == nil || (c.Verifier != InsecureStub && c.Verifier != ExecutionMock) || c.GenesisOutputRoot == (common.Hash{}) {
		return fmt.Errorf("unsupported projection verifier")
	}
	return nil
}

// Range is the decoded full span, including any already-canonical overlap.
type Range interface {
	GetBlockCount() int
	GetBlockTimestamp(int) uint64
	GetBlockEpochNum(int) uint64
	GetBlockTransactions(int) []hexutil.Bytes
}

type Context struct {
	ChainID                               *big.Int
	GenesisNumber, GenesisTime, BlockTime uint64
	ParentHash                            common.Hash
	Continuation                          Continuation
}

// Statement commits to the public records, not to proof-dependent signatures.
// Claim.Proof is always nil. Continuation binds the surviving checkpoint and
// canonical recovery inputs; a real proof must verify their private execution.
type Statement struct {
	ChainID        common.Hash
	ParentHash     common.Hash
	ProjectionHash common.Hash
	Continuation   Continuation
	Claim          codec.RangeClaim
}

// ProofVerifier implementations must be pure: no I/O, mutation of inputs, clock,
// randomness, or mutable state. Production selection is fixed by consensus config.
type ProofVerifier interface{ Verify(Statement, []byte) error }

// StubVerifier deliberately provides no cryptographic execution guarantee.
type StubVerifier struct{}

func (StubVerifier) Verify(Statement, []byte) error { return nil }

func VerifierFor(c *Config) (ProofVerifier, error) {
	if err := c.Check(); err != nil {
		return nil, err
	}
	if c.Verifier == ExecutionMock {
		return ExecutionMockVerifier{}, nil
	}
	return StubVerifier{}, nil
}

// ValidateProjectionRange checks every block before invoking the proof verifier.
// It never releases a valid prefix of a structurally invalid range.
func ValidateProjectionRange(c *Config, ctx Context, span Range, verifier ProofVerifier) (*Statement, error) {
	if err := c.Check(); err != nil {
		return nil, err
	}
	if verifier == nil || ctx.ChainID == nil || ctx.ChainID.Sign() <= 0 || ctx.ChainID.BitLen() > 256 || ctx.BlockTime == 0 {
		return nil, fmt.Errorf("invalid projection context or verifier")
	}
	first, _, err := RangeBounds(ctx.GenesisNumber, ctx.GenesisTime, ctx.BlockTime, span)
	if err != nil {
		return nil, err
	}
	count, start := span.GetBlockCount(), span.GetBlockTimestamp(0)
	if ctx.Continuation.Anchor.Number >= first || ctx.Continuation.Anchor.Hash == (common.Hash{}) || ctx.Continuation.OutputRoot == (common.Hash{}) {
		return nil, fmt.Errorf("invalid authenticated private checkpoint")
	}
	var leaves []common.Hash
	var claim *codec.RangeClaim
	var transcript bytes.Buffer
	put := func(n uint64) { var b [8]byte; binary.BigEndian.PutUint64(b[:], n); transcript.Write(b[:]) }
	signer := types.LatestSignerForChainID(ctx.ChainID)
	for i := 0; i < count; i++ {
		if span.GetBlockTimestamp(i) != start+uint64(i)*ctx.BlockTime {
			return nil, fmt.Errorf("noncontiguous projection timestamps")
		}
		blockStart := transcript.Len()
		put(first + uint64(i))
		put(span.GetBlockTimestamp(i))
		put(span.GetBlockEpochNum(i))
		txs := span.GetBlockTransactions(i)
		if i == 0 && len(txs) == 0 {
			return nil, fmt.Errorf("missing opening claim")
		}
		put(uint64(len(txs)))
		outputSeen := false
		outputPosition := 0
		if i == 0 {
			outputPosition = 1
		}
		for j, raw := range txs {
			var tx types.Transaction
			if len(raw) > wire.MaxMessageBytes+4096 {
				return nil, fmt.Errorf("projection transaction too large")
			}
			if err := tx.UnmarshalBinary(raw); err != nil {
				return nil, fmt.Errorf("decode projection transaction: %w", err)
			}
			canonical, err := tx.MarshalBinary()
			if err != nil || !bytes.Equal(canonical, raw) {
				return nil, fmt.Errorf("noncanonical transaction")
			}
			if tx.Type() != types.DynamicFeeTxType || tx.To() == nil || tx.ChainId().Cmp(ctx.ChainID) != 0 || tx.Value().Sign() != 0 || tx.GasFeeCap().Sign() != 0 || tx.GasTipCap().Sign() != 0 || tx.Gas() == 0 || tx.Gas() > MaxTxGas {
				return nil, fmt.Errorf("invalid projection transaction envelope")
			}
			sender, err := types.Sender(signer, &tx)
			if err != nil {
				return nil, fmt.Errorf("invalid projection signature: %w", err)
			}
			data := tx.Data()
			var expected types.AccessList
			switch *tx.To() {
			case predeploys.ClaimRegistryAddr:
				if len(data) >= 4 && bytes.Equal(data[:4], wire.OutputSelector) {
					if outputSeen || j != outputPosition {
						return nil, fmt.Errorf("duplicate or misplaced private output")
					}
					if _, err := wire.DecodeOutput(data); err != nil {
						return nil, err
					}
					outputSeen = true
					break
				}
				if i != 0 || j != 0 {
					return nil, fmt.Errorf("duplicate or misplaced claim")
				}
				claim, err = wire.DecodeClaim(data)
				if err != nil {
					return nil, err
				}
				expected := ctx.Continuation
				if claim.AnchorBlock != expected.Anchor.Number || claim.AnchorOutputRoot != expected.OutputRoot || claim.RecoveryHash != expected.RecoveryHash || claim.ParentOutputRoot == (common.Hash{}) {
					return nil, fmt.Errorf("claim does not match canonical continuation")
				}
				if expected.Anchor.Number == first-1 {
					if expected.Anchor.Hash != ctx.ParentHash || expected.RecoveryHash != (common.Hash{}) || claim.ParentOutputRoot != expected.OutputRoot {
						return nil, fmt.Errorf("private parent differs from canonical checkpoint")
					}
				} else if expected.RecoveryHash == (common.Hash{}) {
					return nil, fmt.Errorf("missing canonical recovery inputs")
				}
				if claim.FirstBlock != first || claim.LastBlock != first+uint64(count-1) {
					return nil, fmt.Errorf("claim does not cover exactly the span")
				}
				normalized := *claim
				normalized.Proof = nil
				data, err = wire.EncodePostClaim(&normalized)
				if err != nil {
					return nil, err
				}
			case predeploys.L2toL2CrossDomainMessengerAddr:
				m, err := wire.DecodeReplaySentMessage(data)
				if err != nil {
					return nil, err
				}
				if m.Sender == predeploys.SuperchainETHBridgeAddr || m.Target == predeploys.SuperchainETHBridgeAddr {
					return nil, fmt.Errorf("private native bridge replay is forbidden")
				}
			case predeploys.CrossL2InboxAddr:
				m, err := wire.DecodeValidateMessage(data)
				if err != nil {
					return nil, err
				}
				expected = types.AccessList{{Address: predeploys.CrossL2InboxAddr, StorageKeys: messages.EncodeAccessList([]messages.Access{m.Access()})}}
			case predeploys.EventReplayerAddr:
				if !c.AllowEvents {
					return nil, fmt.Errorf("extra event replay is disabled")
				}
				if err := wire.CheckReplayEvent(data); err != nil {
					return nil, err
				}
			default:
				return nil, fmt.Errorf("unexpected projection destination")
			}
			if i == 0 && j == 0 && claim == nil {
				return nil, fmt.Errorf("opening transaction is not a claim")
			}
			al := tx.AccessList()
			if len(al) != len(expected) {
				return nil, fmt.Errorf("unexpected projection access list")
			}
			for k := range al {
				if al[k].Address != expected[k].Address || !slices.Equal(al[k].StorageKeys, expected[k].StorageKeys) {
					return nil, fmt.Errorf("incorrect import checksum access list")
				}
			}
			transcript.Write(sender.Bytes())
			put(tx.Nonce())
			put(tx.Gas())
			transcript.Write(tx.To().Bytes())
			put(uint64(len(data)))
			transcript.Write(data)
		}
		if !outputSeen {
			return nil, fmt.Errorf("missing private output record")
		}
		leaves = append(leaves, crypto.Keccak256Hash([]byte{0}, transcript.Bytes()[blockStart:]))
	}
	if claim == nil {
		return nil, fmt.Errorf("missing claim")
	}
	statement := &Statement{ChainID: common.BigToHash(ctx.ChainID), ParentHash: ctx.ParentHash, ProjectionHash: RecordsRoot(leaves), Continuation: ctx.Continuation, Claim: *claim}
	statement.Claim.Proof = nil
	if err := verifier.Verify(*statement, claim.Proof); err != nil {
		return nil, fmt.Errorf("projection proof: %w", err)
	}
	return statement, nil
}

// RecordsRoot authenticates every ordered block record, including intermediate
// private outputs. Leaves use H(0x00 || canonicalBlock); nodes H(0x01 || left ||
// right), duplicating the last node at odd levels. The leaf count is committed to
// separately so duplication cannot alias spans of different lengths.
func RecordsRoot(leaves []common.Hash) common.Hash {
	var count [8]byte
	binary.BigEndian.PutUint64(count[:], uint64(len(leaves)))
	level := slices.Clone(leaves)
	if len(level) == 0 {
		return common.Hash{}
	}
	for len(level) > 1 {
		next := make([]common.Hash, 0, (len(level)+1)/2)
		for i := 0; i < len(level); i += 2 {
			right := min(i+1, len(level)-1)
			next = append(next, crypto.Keccak256Hash([]byte{1}, level[i][:], level[right][:]))
		}
		level = next
	}
	return crypto.Keccak256Hash([]byte("optimism.private-projection.v2\x00"), count[:], level[0][:])
}

// RangeBounds rejects malformed geometry before derivation performs any parent
// lookup. In particular, genesis cannot underflow to an unresolvable parent.
func RangeBounds(genesisNumber, genesisTime, blockTime uint64, span Range) (uint64, uint64, error) {
	count := span.GetBlockCount()
	if count < 1 || count > MaxRangeBlocks || blockTime == 0 {
		return 0, 0, fmt.Errorf("invalid projection range length or block time")
	}
	start := span.GetBlockTimestamp(0)
	if start < genesisTime || (start-genesisTime)%blockTime != 0 {
		return 0, 0, fmt.Errorf("unaligned projection range")
	}
	offset := (start - genesisTime) / blockTime
	if offset == 0 || offset > math.MaxUint64-genesisNumber {
		return 0, 0, fmt.Errorf("invalid first projection height")
	}
	first := genesisNumber + offset
	if uint64(count-1) > math.MaxUint64-first || uint64(count-1) > (math.MaxUint64-start)/blockTime {
		return 0, 0, fmt.Errorf("projection range overflow")
	}
	for i := 0; i < count; i++ {
		if span.GetBlockTimestamp(i) != start+uint64(i)*blockTime {
			return 0, 0, fmt.Errorf("noncontiguous projection timestamps")
		}
	}
	return first, first + uint64(count-1), nil
}
