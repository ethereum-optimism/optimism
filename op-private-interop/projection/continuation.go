package projection

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-private-interop/wire"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// Continuation is independently resolved from canonical projection history.
// Output records authenticate operator assertions in stub mode, not execution.
type Continuation struct {
	Anchor       eth.BlockID
	OutputRoot   common.Hash
	RecoveryHash common.Hash
}

// CanonicalOutput interprets already-admitted canonical block data. Protocol
// deposits and post-execution transactions cannot impersonate checkpoints.
// A block with sequencer transactions but no correctly placed output is invalid
// context, never a fallback block. Admission validates the rest of its grammar.
func CanonicalOutput(txs []hexutil.Bytes) (common.Hash, error) {
	var root common.Hash
	position := 0
	outputPosition := 0
	for _, raw := range txs {
		if len(raw) == 0 {
			return root, fmt.Errorf("empty canonical transaction")
		}
		if raw[0] == optypes.DepositTxType || raw[0] == optypes.PostExecTxType {
			continue
		}
		var tx types.Transaction
		if err := tx.UnmarshalBinary(raw); err != nil {
			return root, err
		}
		if tx.To() != nil && *tx.To() == predeploys.ClaimRegistryAddr {
			if len(tx.Data()) >= 4 && bytes.Equal(tx.Data()[:4], wire.OutputSelector) {
				if position != outputPosition || root != (common.Hash{}) {
					return root, fmt.Errorf("duplicate or misplaced canonical output")
				}
				var err error
				root, err = wire.DecodeOutput(tx.Data())
				if err != nil {
					return root, err
				}
			} else {
				if position != 0 {
					return root, fmt.Errorf("misplaced canonical claim")
				}
				if _, err := wire.DecodeClaim(tx.Data()); err != nil {
					return root, err
				}
				outputPosition = 1
			}
		}
		position++
	}
	if position != 0 && root == (common.Hash{}) {
		return root, fmt.Errorf("canonical sequencer block has no output record")
	}
	return root, nil
}

// RecoveryStep commits the exact public inputs in reverse height order, starting
// at the publication parent. Constant-space folding permits arbitrarily long
// outages without imposing a consensus maximum recovery interval. The future
// proof must verify private deposit execution, including suppressed public logs.
func RecoveryStep(previous common.Hash, payload *eth.ExecutionPayload) common.Hash {
	h := crypto.NewKeccakState()
	_, _ = h.Write([]byte("optimism.private-recovery.v1\x00"))
	_, _ = h.Write(previous[:])
	_, _ = h.Write(payload.BlockHash[:])
	_, _ = h.Write(payload.ParentHash[:])
	put := func(n uint64) { var b [8]byte; binary.BigEndian.PutUint64(b[:], n); _, _ = h.Write(b[:]) }
	put(uint64(payload.BlockNumber))
	put(uint64(payload.Timestamp))
	put(uint64(len(payload.Transactions)))
	for _, tx := range payload.Transactions {
		put(uint64(len(tx)))
		_, _ = h.Write(tx)
	}
	var out common.Hash
	_, _ = h.Read(out[:])
	return out
}

// ErrContextUnavailable retains the candidate and retries. It is not a consensus
// rejection. This includes an incremental scan that has more work remaining.
var ErrContextUnavailable = errors.New("canonical projection context unavailable")

type PayloadSource interface {
	PayloadByNumber(context.Context, uint64) (*eth.ExecutionPayloadEnvelope, error)
}

// ContextCollector owns only RPC collection, outside the pure admission function.
// One constant-sized cursor is retained between calls. Reset discards it. Each
// attempt rechecks the target hash so a reorg cannot mix branches or reuse a stale
// completed context. The caller supplies a canonical parent, never an unsafe tip.
type ContextCollector struct {
	parent            eth.BlockID
	genesis           eth.BlockID
	genesisOutput     common.Hash
	cursor            eth.BlockID
	result            Continuation
	started, complete bool
}

func (c *ContextCollector) Reset() { *c = ContextCollector{} }

func (c *ContextCollector) Resolve(ctx context.Context, src PayloadSource, parent, genesis eth.BlockID, genesisOutput common.Hash) (Continuation, error) {
	if parent.Number < genesis.Number || genesisOutput == (common.Hash{}) {
		return Continuation{}, fmt.Errorf("invalid projection genesis context")
	}
	if !c.started || c.parent != parent || c.genesis != genesis || c.genesisOutput != genesisOutput {
		*c = ContextCollector{parent: parent, genesis: genesis, genesisOutput: genesisOutput, cursor: parent, started: true}
	}
	// Pin even a completed result to the current canonical parent.
	if parent != genesis {
		env, err := src.PayloadByNumber(ctx, parent.Number)
		if err != nil {
			return Continuation{}, fmt.Errorf("%w: parent %d: %w", ErrContextUnavailable, parent.Number, err)
		}
		if env == nil || env.ExecutionPayload == nil {
			return Continuation{}, fmt.Errorf("%w: missing parent payload", ErrContextUnavailable)
		}
		if env.ExecutionPayload.ID() != parent {
			c.Reset()
			return Continuation{}, fmt.Errorf("%w: canonical parent changed", ErrContextUnavailable)
		}
	}
	if c.complete {
		return c.result, nil
	}
	for reads := 0; reads < 128; reads++ {
		if c.cursor.Number == genesis.Number {
			if c.cursor != genesis {
				return Continuation{}, fmt.Errorf("recovery does not descend from projection genesis")
			}
			c.result.Anchor, c.result.OutputRoot = genesis, genesisOutput
			c.complete = true
			return c.result, nil
		}
		env, err := src.PayloadByNumber(ctx, c.cursor.Number)
		if err != nil {
			return Continuation{}, fmt.Errorf("%w: block %d: %w", ErrContextUnavailable, c.cursor.Number, err)
		}
		if env == nil || env.ExecutionPayload == nil {
			return Continuation{}, fmt.Errorf("%w: missing recovery payload", ErrContextUnavailable)
		}
		p := env.ExecutionPayload
		if p.ID() != c.cursor {
			c.Reset()
			return Continuation{}, fmt.Errorf("%w: recovery ancestry changed", ErrContextUnavailable)
		}
		root, err := CanonicalOutput(p.Transactions)
		if err != nil {
			return Continuation{}, err
		}
		if root != (common.Hash{}) {
			c.result.Anchor, c.result.OutputRoot = c.cursor, root
			c.complete = true
			return c.result, nil
		}
		c.result.RecoveryHash = RecoveryStep(c.result.RecoveryHash, p)
		c.cursor = eth.BlockID{Hash: p.ParentHash, Number: c.cursor.Number - 1}
	}
	return Continuation{}, ErrContextUnavailable
}
