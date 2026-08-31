package batcher

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/dial"
)

// The production RangeSource.
//
// It answers the one question the seam cannot answer for itself, and that question is a WAIT on
// something outside this process: what does this range continue from? The rendering node
// (component 3 of the ratified operator topology) has to have DERIVED AND EXECUTED the previous
// range first. Nothing here guesses: a wrong parent check is a BatchDrop at every verifier.
//
// Every failure is a plain error. The batcher's stock retry path re-runs the range; a range built
// on a guess is a range every verifier drops.

// RenderingBlock is the part of a rendering block the range source reads: its identity, and
// nothing else.
//
// The block's transactions are deliberately NOT here. They used to be, for a decoder that read the
// L1 origin and sequence number out of the block's attributes deposit — machinery origin-copy
// deleted. Carrying bodies nothing decodes costs bandwidth and buys a decode hazard for free: the
// rendering's first transaction is an L1-attributes DEPOSIT, which a general-purpose eth client
// cannot decode at all.
type RenderingBlock struct {
	Hash   common.Hash
	Number uint64
}

// RenderingFollower is the node following the rendering.
type RenderingFollower interface {
	// BlockByNumber returns the rendering's block at number. It fails while the rendering has not
	// derived that far, which is the wait the range source exists to perform.
	BlockByNumber(ctx context.Context, number uint64) (*RenderingBlock, error)
	// NonceAt returns account's transaction count as of that block.
	NonceAt(ctx context.Context, account common.Address, number uint64) (uint64, error)
}

// rpcRenderingFollower is a RenderingFollower over a plain JSON-RPC connection to the rendering's
// execution client: eth_getBlockByNumber and eth_getTransactionCount, nothing more.
type rpcRenderingFollower struct {
	rpc     *rpc.Client
	timeout time.Duration
}

var _ RenderingFollower = (*rpcRenderingFollower)(nil)

// NewRPCRenderingFollower dials the rendering's execution client.
func NewRPCRenderingFollower(ctx context.Context, lgr log.Logger, url string, timeout time.Duration) (RenderingFollower, error) {
	cl, err := dial.DialRPCClientWithTimeout(ctx, lgr, url)
	if err != nil {
		return nil, fmt.Errorf("dialling the rendering node at %s: %w", url, err)
	}
	return &rpcRenderingFollower{rpc: cl, timeout: timeout}, nil
}

// rpcBlock is the subset of eth_getBlockByNumber's result the follower needs.
//
// The call asks for fullTx=FALSE, so no transaction is ever decoded on this path. That is not just
// thrift: with fullTx=true the transactions arrive as objects, and the T1 devstack caught the
// original hexutil.Bytes decode failing on every non-empty block. Nothing reads them any more, so
// nothing has to survive decoding them.
type rpcBlock struct {
	Hash   common.Hash    `json:"hash"`
	Number hexutil.Uint64 `json:"number"`
}

func (f *rpcRenderingFollower) BlockByNumber(ctx context.Context, number uint64) (*RenderingBlock, error) {
	ctx, cancel := context.WithTimeout(ctx, f.timeout)
	defer cancel()
	var out *rpcBlock
	if err := f.rpc.CallContext(ctx, &out, "eth_getBlockByNumber", hexutil.Uint64(number), false); err != nil {
		return nil, err
	}
	if out == nil {
		return nil, fmt.Errorf("the rendering has no block %d yet", number)
	}
	return &RenderingBlock{Hash: out.Hash, Number: uint64(out.Number)}, nil
}

func (f *rpcRenderingFollower) NonceAt(ctx context.Context, account common.Address, number uint64) (uint64, error) {
	ctx, cancel := context.WithTimeout(ctx, f.timeout)
	defer cancel()
	var out hexutil.Uint64
	if err := f.rpc.CallContext(ctx, &out, "eth_getTransactionCount", account, hexutil.Uint64(number)); err != nil {
		return 0, err
	}
	return uint64(out), nil
}

// PrivateInteropRangeSourceConfig configures the production RangeSource.
type PrivateInteropRangeSourceConfig struct {
	Log log.Logger
	// RenderingRollup is the RENDERING's rollup config: it names the rendering's genesis, which is
	// the one block whose origin and sequence number are not read from an attributes deposit.
	RenderingRollup *rollup.Config
	// Rendering is the rendering follower.
	Rendering RenderingFollower
	// Operator is the EOA that signs the rendering's transactions; its nonce continues across
	// ranges and is read from the rendering, never remembered.
	Operator common.Address
	// NetworkTimeout bounds each L1 call.
	NetworkTimeout time.Duration
}

// privateInteropRangeSource is the production RangeSource. It is stateless: since origin-copy there
// is no L1 view to bound, no confirmation depth to hold back from, and no origin floor to remember
// between calls — every answer is read from the rendering when asked.
type privateInteropRangeSource struct {
	cfg PrivateInteropRangeSourceConfig
}

var _ RangeSource = (*privateInteropRangeSource)(nil)

// NewPrivateInteropRangeSource builds the production range source.
func NewPrivateInteropRangeSource(cfg PrivateInteropRangeSourceConfig) (RangeSource, error) {
	if cfg.RenderingRollup == nil {
		return nil, errors.New("private interop range source: no rendering rollup config")
	}
	if cfg.Rendering == nil {
		return nil, errors.New("private interop range source: no rendering follower")
	}
	if cfg.Operator == (common.Address{}) {
		return nil, errors.New("private interop range source: no operator address")
	}
	if cfg.NetworkTimeout == 0 {
		cfg.NetworkTimeout = 10 * time.Second
	}
	if cfg.Log == nil {
		cfg.Log = log.New()
	}
	return &privateInteropRangeSource{cfg: cfg}, nil
}

// RangeStart reads everything the range beginning at firstBlock continues from off the rendering.
//
// The rendering's block at firstBlock-1 carries both answers: its hash is the span's parent check,
// and the operator's nonce as of it is the range's starting nonce. Reading the nonce from the chain
// rather than remembering it is what makes a restarted batcher rebuild the same range.
//
// It used to read the origin and sequence number to continue from out of that block's attributes
// deposit as well. Since origin-copy there is nothing to continue: a rendering block's origin and
// sequence number are its private block's own, so the bookkeeping has no state to carry across a
// range boundary.
func (s *privateInteropRangeSource) RangeStart(ctx context.Context, firstBlock uint64) (RangeStart, error) {
	genesis := s.cfg.RenderingRollup.Genesis.L2.Number
	if firstBlock <= genesis {
		return RangeStart{}, fmt.Errorf("range cannot start at %d: the rendering's genesis is block %d", firstBlock, genesis)
	}
	prev := firstBlock - 1

	blk, err := s.cfg.Rendering.BlockByNumber(ctx, prev)
	if err != nil {
		return RangeStart{}, fmt.Errorf("reading the rendering's block %d: %w", prev, err)
	}
	if blk.Number != prev {
		return RangeStart{}, fmt.Errorf("asked the rendering for block %d and got %d", prev, blk.Number)
	}

	nonce, err := s.cfg.Rendering.NonceAt(ctx, s.cfg.Operator, prev)
	if err != nil {
		return RangeStart{}, fmt.Errorf("reading the operator's nonce at rendering block %d: %w", prev, err)
	}

	s.cfg.Log.Info("Private interop range start resolved",
		"first_block", firstBlock, "parent_check", blk.Hash, "start_nonce", nonce)
	return RangeStart{
		PrevTerminalRenderingHash: blk.Hash,
		StartNonce:                nonce,
	}, nil
}
