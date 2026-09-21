// Package projection validates complete public projection ranges before execution.
// It has no dependency on derivation, private execution, RPC, or mutable node state.
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
	MaxRangeBlocks = 65536
	MaxTxGas       = wire.MaxTxGas
)

// Config is consensus configuration, not an operator-selected verifier override.
type Config struct {
	Verifier    string `json:"verifier"`
	AllowEvents bool   `json:"allow_events,omitempty"`
}

func (c *Config) Check() error {
	if c == nil || c.Verifier != InsecureStub {
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
}

// Statement commits to the public records, not to proof-dependent signatures.
// Claim.Proof is always nil. A real private execution proof needs a separately
// versioned statement specifying private prestate and transition continuity.
type Statement struct {
	ChainID        common.Hash
	ParentHash     common.Hash
	ProjectionHash common.Hash
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
	count := span.GetBlockCount()
	if count < 1 || count > MaxRangeBlocks {
		return nil, fmt.Errorf("invalid projection range length")
	}
	start := span.GetBlockTimestamp(0)
	if start < ctx.GenesisTime || (start-ctx.GenesisTime)%ctx.BlockTime != 0 {
		return nil, fmt.Errorf("unaligned projection range")
	}
	offset := (start - ctx.GenesisTime) / ctx.BlockTime
	if offset == 0 || offset > math.MaxUint64-ctx.GenesisNumber {
		return nil, fmt.Errorf("invalid first projection height")
	}
	first := ctx.GenesisNumber + offset
	if uint64(count-1) > math.MaxUint64-first || uint64(count-1) > (math.MaxUint64-start)/ctx.BlockTime {
		return nil, fmt.Errorf("projection range overflow")
	}
	var claim *codec.RangeClaim
	var transcript bytes.Buffer
	transcript.WriteString("optimism.private-projection.v1\x00")
	put := func(n uint64) { var b [8]byte; binary.BigEndian.PutUint64(b[:], n); transcript.Write(b[:]) }
	put(uint64(count))
	signer := types.LatestSignerForChainID(ctx.ChainID)
	for i := 0; i < count; i++ {
		if span.GetBlockTimestamp(i) != start+uint64(i)*ctx.BlockTime {
			return nil, fmt.Errorf("noncontiguous projection timestamps")
		}
		put(first + uint64(i))
		put(span.GetBlockTimestamp(i))
		put(span.GetBlockEpochNum(i))
		txs := span.GetBlockTransactions(i)
		if i == 0 && len(txs) == 0 {
			return nil, fmt.Errorf("missing opening claim")
		}
		put(uint64(len(txs)))
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
				if i != 0 || j != 0 {
					return nil, fmt.Errorf("duplicate or misplaced claim")
				}
				claim, err = wire.DecodeClaim(data)
				if err != nil {
					return nil, err
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
	}
	if claim == nil {
		return nil, fmt.Errorf("missing claim")
	}
	statement := &Statement{ChainID: common.BigToHash(ctx.ChainID), ParentHash: ctx.ParentHash, ProjectionHash: crypto.Keccak256Hash(transcript.Bytes()), Claim: *claim}
	statement.Claim.Proof = nil
	if err := verifier.Verify(*statement, claim.Proof); err != nil {
		return nil, fmt.Errorf("projection proof: %w", err)
	}
	return statement, nil
}
