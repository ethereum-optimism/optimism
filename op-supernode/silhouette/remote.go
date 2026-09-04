package silhouette

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-service/client"
)

// InteropSource is the narrow, receipt-free interface the supernode needs from a proof-rendering
// execution client. The production implementation is RPCInteropSource; FactStore implements it too
// so the wire and container rules can be unit-tested without a socket.
type InteropSource interface {
	InteropBlock(ctx context.Context, number uint64) (*InteropBlock, error)
	MarkDenied(ctx context.Context, number uint64, hash common.Hash) error
	PruneDenied(ctx context.Context, removed map[uint64][]common.Hash) error
}

// RPCInteropSource consumes the public silhouette_ namespace of a standalone op-silhouette-el.
// It deliberately knows nothing about L1, proof verification, or private execution.
type RPCInteropSource struct{ rpc client.RPC }

func NewRPCInteropSource(rpc client.RPC) *RPCInteropSource { return &RPCInteropSource{rpc: rpc} }

func (s *RPCInteropSource) Status(ctx context.Context) (*Status, error) {
	var out *Status
	if err := s.rpc.CallContext(ctx, &out, "silhouette_status"); err != nil {
		return nil, fmt.Errorf("read silhouette status: %w", err)
	}
	if out == nil {
		return nil, fmt.Errorf("silhouette status returned null")
	}
	return out, nil
}

func (s *RPCInteropSource) InteropBlock(ctx context.Context, number uint64) (*InteropBlock, error) {
	var out *InteropBlock
	if err := s.rpc.CallContext(ctx, &out, "silhouette_interopBlock", hexutil.Uint64(number)); err != nil {
		return nil, fmt.Errorf("read silhouette interop block %d: %w", number, err)
	}
	return out, nil
}

func (s *RPCInteropSource) MarkDenied(ctx context.Context, number uint64, hash common.Hash) error {
	var out any
	if err := s.rpc.CallContext(ctx, &out, "silhouette_markDenied", hexutil.Uint64(number), hash); err != nil {
		return fmt.Errorf("mark silhouette block %d (%s) denied: %w", number, hash, err)
	}
	return nil
}

func (s *RPCInteropSource) PruneDenied(ctx context.Context, removed map[uint64][]common.Hash) error {
	blocks := make([]DeniedBlock, 0)
	for number, hashes := range removed {
		for _, hash := range hashes {
			blocks = append(blocks, DeniedBlock{Number: hexutil.Uint64(number), Hash: hash})
		}
	}
	var out any
	if err := s.rpc.CallContext(ctx, &out, "silhouette_pruneDenied", blocks); err != nil {
		return fmt.Errorf("prune silhouette denial cache: %w", err)
	}
	return nil
}

func (f *FactStore) InteropBlock(ctx context.Context, number uint64) (*InteropBlock, error) {
	fact, ok := f.ByNumber(number)
	if !ok {
		return nil, nil
	}
	denied, err := f.Denied(number, fact.Hash)
	if err != nil {
		return nil, err
	}
	return &InteropBlock{
		Number:        hexutil.Uint64(fact.Number),
		Timestamp:     hexutil.Uint64(fact.Timestamp),
		ParentHash:    fact.ParentHash,
		Hash:          fact.Hash,
		Logs:          cloneLogExports(fact.LogExports),
		ExecMsgs:      append([]messages.ExecutingMessage(nil), fact.ExecMsgs...),
		ExecMsgsKnown: fact.ExecMsgsKnown,
		Denied:        denied,
		Replacement:   fact.Replacement,
	}, nil
}

// localInteropSource adapts the legacy in-memory FactStore methods to InteropSource. Production
// supernodes never use it; keeping it private makes that distinction visible.
type localInteropSource struct{ facts *FactStore }

func LocalInteropSource(facts *FactStore) InteropSource { return &localInteropSource{facts: facts} }

func (s *localInteropSource) InteropBlock(ctx context.Context, number uint64) (*InteropBlock, error) {
	return s.facts.InteropBlock(ctx, number)
}

func (s *localInteropSource) MarkDenied(ctx context.Context, number uint64, hash common.Hash) error {
	return s.facts.MarkDenied(number, hash)
}

func (s *localInteropSource) PruneDenied(ctx context.Context, removed map[uint64][]common.Hash) error {
	s.facts.PruneDenied(removed)
	return nil
}
