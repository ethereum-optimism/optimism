package derive

import (
	"context"
	"sync"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-node/rollup"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/log"
)

type BlocksSource interface {
	PayloadByHash(context.Context, common.Hash) (*eth.ExecutionPayload, error)
}

type UnsafeBlocksSource interface {
	BlocksSource
	UnsafeBlockIDs(ctx context.Context, safeHead eth.BlockID, max uint64) ([]eth.BlockID, error)
}

type OutputMetaData struct {
	// Current safe-point of the L2 chain, derived from L1 data.
	SafeHead eth.L2BlockRef `json:"safe_head"`

	// Current tip of the L2 chain
	UnsafeHead eth.L2BlockRef `json:"unsafe_head"`

	// Number of blocks that have been included in a channel, but not finished yet.
	// Within the channel timeout.
	// This may include non-canonical L2 blocks, if L1 reorged the L2 chain.
	OpenedBlocks uint64 `json:"opened_blocks"`

	// Number of blocks that have been fully submitted (i.e. closed channel).
	// Within the channel timeout.
	// This may include non-canonical L2 blocks, if L1 reorged the L2 chain.
	ClosedBlocks uint64 `json:"closed_blocks"`
}

type BatcherChannelData struct {
	// Channels identifies all channels that were involved in this output, with their last frame ID.
	// Empty if no new data was produced.
	Channels map[ChannelID]uint64 `json:"channels"`

	// Data to post to L1, encodes channel version byte and one or more frames.
	Data []byte `json:"data"`

	Meta OutputMetaData `json:"meta"`
}

// ChannelEmitter maintains open channels and emits data with channel frames to confirm the L2 unsafe blocks.
type ChannelEmitter struct {
	mu sync.Mutex

	log log.Logger

	cfg *rollup.Config

	source UnsafeBlocksSource

	// pruned when timed out. We keep track of fully read channels to avoid resubmitting data.
	channels map[ChannelID]*ChannelOut

	// context used to fetch data for channels, might outlive a single request
	ctx context.Context
	// cancels above context
	cancel context.CancelFunc

	l1Time       uint64
	l2SafeHead   eth.L2BlockRef
	l2UnsafeHead eth.L2BlockRef
}
