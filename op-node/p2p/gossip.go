package p2p

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"github.com/golang/snappy"
	lru "github.com/hashicorp/golang-lru"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
)

const (
	// maxGossipSize limits the total size of gossip RPC containers as well as decompressed individual messages.
	maxGossipSize = 10 * (1 << 20)
	// minGossipSize is used to make sure that there is at least some data to validate the signature against.
	minGossipSize          = 66
	maxOutboundQueue       = 256
	maxValidateQueue       = 256
	globalValidateThrottle = 512
	gossipHeartbeat        = 500 * time.Millisecond
	// seenMessagesTTL limits the duration that message IDs are remembered for gossip deduplication purposes
	seenMessagesTTL = 80 * gossipHeartbeat
)

// Message domains, the msg id function uncompresses to keep data monomorphic,
// but invalid compressed data will need a unique different id.

var MessageDomainInvalidSnappy = [4]byte{0, 0, 0, 0}
var MessageDomainValidSnappy = [4]byte{1, 0, 0, 0}

type GossipMetricer interface {
	RecordGossipEvent(evType int32)
}

func blocksTopicV1(cfg *rollup.Config) string {
	return fmt.Sprintf("/optimism/%s/0/blocks", cfg.L2ChainID.String())
}

// BuildSubscriptionFilter builds a simple subscription filter,
// to help protect against peers spamming useless subscriptions.
func BuildSubscriptionFilter(cfg *rollup.Config) pubsub.SubscriptionFilter {
	return pubsub.NewAllowlistSubscriptionFilter(blocksTopicV1(cfg)) // add more topics here in the future, if any.
}

var msgBufPool = sync.Pool{New: func() any {
	// note: the topic validator concurrency is limited, so pool won't blow up, even with large pre-allocation.
	x := make([]byte, 0, maxGossipSize)
	return &x
}}

// BuildMsgIdFn builds a generic message ID function for gossipsub that can handle compressed payloads,
// mirroring the eth2 p2p gossip spec.
func BuildMsgIdFn(cfg *rollup.Config) pubsub.MsgIdFunction {
	return func(pmsg *pb.Message) string {
		valid := false
		var data []byte
		// If it's a valid compressed snappy data, then hash the uncompressed contents.
		// The validator can throw away the message later when recognized as invalid,
		// and the unique hash helps detect duplicates.
		dLen, err := snappy.DecodedLen(pmsg.Data)
		if err == nil && dLen <= maxGossipSize {
			res := msgBufPool.Get().(*[]byte)
			defer msgBufPool.Put(res)
			if data, err = snappy.Decode((*res)[:0], pmsg.Data); err == nil {
				*res = data // if we ended up growing the slice capacity, fine, keep the larger one.
				valid = true
			}
		}
		if data == nil {
			data = pmsg.Data
		}
		h := sha256.New()
		if valid {
			h.Write(MessageDomainValidSnappy[:])
		} else {
			h.Write(MessageDomainInvalidSnappy[:])
		}
		// The chain ID is part of the gossip topic, making the msg id unique
		topic := pmsg.GetTopic()
		var topicLen [8]byte
		binary.LittleEndian.PutUint64(topicLen[:], uint64(len(topic)))
		h.Write(topicLen[:])
		h.Write([]byte(topic))
		h.Write(data)
		// the message ID is shortened to save space, a lot of these may be gossiped.
		return string(h.Sum(nil)[:20])
	}
}

func BuildGlobalGossipParams(cfg *rollup.Config) pubsub.GossipSubParams {
	params := pubsub.DefaultGossipSubParams()
	params.D = 8                               // topic stable mesh target count
	params.Dlo = 6                             // topic stable mesh low watermark
	params.Dhi = 12                            // topic stable mesh high watermark
	params.Dlazy = 6                           // gossip target
	params.HeartbeatInterval = gossipHeartbeat // interval of heartbeat
	params.FanoutTTL = 24 * time.Second        // ttl for fanout maps for topics we are not subscribed to but have published to
	params.HistoryLength = 12                  // number of windows to retain full messages in cache for IWANT responses
	params.HistoryGossip = 3                   // number of windows to gossip about

	return params
}

func NewGossipSub(p2pCtx context.Context, h host.Host, cfg *rollup.Config, m GossipMetricer) (*pubsub.PubSub, error) {
	denyList, err := pubsub.NewTimeCachedBlacklist(30 * time.Second)
	if err != nil {
		return nil, err
	}
	return pubsub.NewGossipSub(p2pCtx, h,
		pubsub.WithMaxMessageSize(maxGossipSize),
		pubsub.WithMessageIdFn(BuildMsgIdFn(cfg)),
		pubsub.WithNoAuthor(),
		pubsub.WithMessageSignaturePolicy(pubsub.StrictNoSign),
		pubsub.WithSubscriptionFilter(BuildSubscriptionFilter(cfg)),
		pubsub.WithValidateQueueSize(maxValidateQueue),
		pubsub.WithPeerOutboundQueueSize(maxOutboundQueue),
		pubsub.WithValidateThrottle(globalValidateThrottle),
		pubsub.WithSeenMessagesTTL(seenMessagesTTL),
		pubsub.WithPeerExchange(false),
		pubsub.WithBlacklist(denyList),
		pubsub.WithGossipSubParams(BuildGlobalGossipParams(cfg)),
		pubsub.WithEventTracer(&gossipTracer{m: m}),
	)
	// TODO: pubsub.WithPeerScoreInspect(inspect, InspectInterval) to update peerstore scores with gossip scores
}

func validationResultString(v pubsub.ValidationResult) string {
	switch v {
	case pubsub.ValidationAccept:
		return "ACCEPT"
	case pubsub.ValidationIgnore:
		return "IGNORE"
	case pubsub.ValidationReject:
		return "REJECT"
	default:
		return fmt.Sprintf("UNKNOWN_%d", v)
	}
}

func logValidationResult(self peer.ID, msg string, log log.Logger, fn pubsub.ValidatorEx) pubsub.ValidatorEx {
	return func(ctx context.Context, id peer.ID, message *pubsub.Message) pubsub.ValidationResult {
		res := fn(ctx, id, message)
		var src interface{}
		src = id
		if id == self {
			src = "self"
		}
		log.Debug(msg, "result", validationResultString(res), "from", src)
		return res
	}
}

type seenBlocks struct {
	sync.Mutex
	blockHashes []common.Hash
}

// hasSeen checks if the hash has been marked as seen, and how many have been seen.
func (sb *seenBlocks) hasSeen(h common.Hash) (count int, hasSeen bool) {
	sb.Lock()
	defer sb.Unlock()
	for _, prev := range sb.blockHashes {
		if prev == h {
			return len(sb.blockHashes), true
		}
	}
	return len(sb.blockHashes), false
}

// markSeen marks the block hash as seen
func (sb *seenBlocks) markSeen(h common.Hash) {
	sb.Lock()
	defer sb.Unlock()
	sb.blockHashes = append(sb.blockHashes, h)
}

func BuildBlocksValidator(log log.Logger, cfg *rollup.Config) pubsub.ValidatorEx {

	// Seen block hashes per block height
	// uint64 -> *seenBlocks
	blockHeightLRU, err := lru.New(100)
	if err != nil {
		panic(fmt.Errorf("failed to set up block height LRU cache: %w", err))
	}

	return func(ctx context.Context, id peer.ID, message *pubsub.Message) pubsub.ValidationResult {
		// [REJECT] if the compression is not valid
		outLen, err := snappy.DecodedLen(message.Data)
		if err != nil {
			log.Warn("invalid snappy compression length data", "err", err, "peer", id)
			return pubsub.ValidationReject
		}
		if outLen > maxGossipSize {
			log.Warn("possible snappy zip bomb, decoded length is too large", "decoded_length", outLen, "peer", id)
			return pubsub.ValidationReject
		}
		if outLen < minGossipSize {
			log.Warn("rejecting undersized gossip payload")
			return pubsub.ValidationReject
		}

		res := msgBufPool.Get().(*[]byte)
		defer msgBufPool.Put(res)
		data, err := snappy.Decode((*res)[:0], message.Data)
		if err != nil {
			log.Warn("invalid snappy compression", "err", err, "peer", id)
			return pubsub.ValidationReject
		}
		*res = data // if we ended up growing the slice capacity, fine, keep the larger one.

		// message starts with compact-encoding secp256k1 encoded signature
		signatureBytes, payloadBytes := data[:65], data[65:]

		// [REJECT] if the signature by the sequencer is not valid
		signingHash, err := BlockSigningHash(cfg, payloadBytes)
		if err != nil {
			log.Warn("failed to compute block signing hash", "err", err, "peer", id)
			return pubsub.ValidationReject
		}

		pub, err := crypto.SigToPub(signingHash[:], signatureBytes)
		if err != nil {
			log.Warn("invalid block signature", "err", err, "peer", id)
			return pubsub.ValidationReject
		}
		addr := crypto.PubkeyToAddress(*pub)

		// TODO: in the future we can support multiple valid p2p addresses.
		if addr != cfg.P2PSequencerAddress {
			log.Warn("unexpected block author", "err", err, "peer", id)
			return pubsub.ValidationReject
		}

		// [REJECT] if the block encoding is not valid
		var payload eth.ExecutionPayload
		if err := payload.UnmarshalSSZ(uint32(len(payloadBytes)), bytes.NewReader(payloadBytes)); err != nil {
			log.Warn("invalid payload", "err", err, "peer", id)
			return pubsub.ValidationReject
		}

		// rounding down to seconds is fine here.
		now := uint64(time.Now().Unix())

		// [REJECT] if the `payload.timestamp` is older than 60 seconds in the past
		if uint64(payload.Timestamp) < now-60 {
			log.Warn("payload is too old", "timestamp", uint64(payload.Timestamp))
			return pubsub.ValidationReject
		}

		// [REJECT] if the `payload.timestamp` is more than 5 seconds into the future
		if uint64(payload.Timestamp) > now+5 {
			log.Warn("payload is too new", "timestamp", uint64(payload.Timestamp))
			return pubsub.ValidationReject
		}

		// [REJECT] if the `block_hash` in the `payload` is not valid
		if actual, ok := payload.CheckBlockHash(); !ok {
			log.Warn("payload has bad block hash", "bad_hash", payload.BlockHash.String(), "actual", actual.String())
			return pubsub.ValidationReject
		}

		seen, ok := blockHeightLRU.Get(uint64(payload.BlockNumber))
		if !ok {
			seen = new(seenBlocks)
			blockHeightLRU.Add(uint64(payload.BlockNumber), seen)
		}

		if count, hasSeen := seen.(*seenBlocks).hasSeen(payload.BlockHash); count > 5 {
			// [REJECT] if more than 5 blocks have been seen with the same block height
			log.Warn("seen too many different blocks at same height", "height", payload.BlockNumber)
			return pubsub.ValidationReject
		} else if hasSeen {
			// [IGNORE] if the block has already been seen
			log.Warn("validated already seen message again")
			return pubsub.ValidationIgnore
		}

		// mark it as seen. (note: with concurrent validation more than 5 blocks may be marked as seen still,
		// but validator concurrency is limited anyway)
		seen.(*seenBlocks).markSeen(payload.BlockHash)

		// remember the decoded payload for later usage in topic subscriber.
		message.ValidatorData = &payload
		return pubsub.ValidationAccept
	}
}

type GossipIn interface {
	OnUnsafeL2Payload(ctx context.Context, from peer.ID, msg *eth.ExecutionPayload) error
}

type GossipTopicInfo interface {
	BlocksTopicPeers() []peer.ID
}

type GossipOut interface {
	GossipTopicInfo
	PublishL2Payload(ctx context.Context, msg *eth.ExecutionPayload, signer Signer) error
	Close() error
}

type publisher struct {
	log         log.Logger
	cfg         *rollup.Config
	blocksTopic *pubsub.Topic
}

var _ GossipOut = (*publisher)(nil)

func (p *publisher) BlocksTopicPeers() []peer.ID {
	return p.blocksTopic.ListPeers()
}

func (p *publisher) PublishL2Payload(ctx context.Context, payload *eth.ExecutionPayload, signer Signer) error {
	res := msgBufPool.Get().(*[]byte)
	buf := bytes.NewBuffer((*res)[:0])
	defer func() {
		*res = buf.Bytes()
		defer msgBufPool.Put(res)
	}()

	buf.Write(make([]byte, 65))
	if _, err := payload.MarshalSSZ(buf); err != nil {
		return fmt.Errorf("failed to encoded execution payload to publish: %w", err)
	}
	data := buf.Bytes()
	payloadData := data[65:]
	sig, err := signer.Sign(ctx, SigningDomainBlocksV1, p.cfg.L2ChainID, payloadData)
	if err != nil {
		return fmt.Errorf("failed to sign execution payload with signer: %w", err)
	}
	copy(data[:65], sig[:])

	// compress the full message
	// This also copies the data, freeing up the original buffer to go back into the pool
	out := snappy.Encode(nil, data)

	return p.blocksTopic.Publish(ctx, out)
}

func (p *publisher) Close() error {
	return p.blocksTopic.Close()
}

func JoinGossip(p2pCtx context.Context, self peer.ID, ps *pubsub.PubSub, log log.Logger, cfg *rollup.Config, gossipIn GossipIn) (GossipOut, error) {
	val := logValidationResult(self, "validated block", log, BuildBlocksValidator(log, cfg))
	blocksTopicName := blocksTopicV1(cfg)
	err := ps.RegisterTopicValidator(blocksTopicName,
		val,
		pubsub.WithValidatorTimeout(3*time.Second),
		pubsub.WithValidatorConcurrency(4))
	if err != nil {
		return nil, fmt.Errorf("failed to register blocks gossip topic: %w", err)
	}
	blocksTopic, err := ps.Join(blocksTopicName)
	if err != nil {
		return nil, fmt.Errorf("failed to join blocks gossip topic: %w", err)
	}
	blocksTopicEvents, err := blocksTopic.EventHandler()
	if err != nil {
		return nil, fmt.Errorf("failed to create blocks gossip topic handler: %w", err)
	}
	go LogTopicEvents(p2pCtx, log.New("topic", "blocks"), blocksTopicEvents)

	// TODO: block topic scoring parameters
	// See prysm: https://github.com/prysmaticlabs/prysm/blob/develop/beacon-chain/p2p/gossip_scoring_params.go
	// And research from lighthouse: https://gist.github.com/blacktemplar/5c1862cb3f0e32a1a7fb0b25e79e6e2c
	// And docs: https://github.com/libp2p/specs/blob/master/pubsub/gossipsub/gossipsub-v1.1.md#topic-parameter-calculation-and-decay
	//err := blocksTopic.SetScoreParams(&pubsub.TopicScoreParams{......})

	subscription, err := blocksTopic.Subscribe()
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to blocks gossip topic: %w", err)
	}

	subscriber := MakeSubscriber(log, BlocksHandler(gossipIn.OnUnsafeL2Payload))
	go subscriber(p2pCtx, subscription)

	return &publisher{log: log, cfg: cfg, blocksTopic: blocksTopic}, nil
}

type TopicSubscriber func(ctx context.Context, sub *pubsub.Subscription)
type MessageHandler func(ctx context.Context, from peer.ID, msg interface{}) error

func BlocksHandler(onBlock func(ctx context.Context, from peer.ID, msg *eth.ExecutionPayload) error) MessageHandler {
	return func(ctx context.Context, from peer.ID, msg interface{}) error {
		payload, ok := msg.(*eth.ExecutionPayload)
		if !ok {
			return fmt.Errorf("expected topic validator to parse and validate data into execution payload, but got %T", msg)
		}
		return onBlock(ctx, from, payload)
	}
}

func MakeSubscriber(log log.Logger, msgHandler MessageHandler) TopicSubscriber {
	return func(ctx context.Context, sub *pubsub.Subscription) {
		topicLog := log.New("topic", sub.Topic())
		for {
			msg, err := sub.Next(ctx)
			if err != nil { // ctx was closed, or subscription was closed
				topicLog.Debug("stopped subscriber")
				return
			}
			if msg.ValidatorData == nil {
				topicLog.Error("gossip message with no data", "from", msg.ReceivedFrom)
				continue
			}
			if err := msgHandler(ctx, msg.ReceivedFrom, msg.ValidatorData); err != nil {
				topicLog.Error("failed to process gossip message", "err", err)
			}
		}
	}
}

func LogTopicEvents(ctx context.Context, log log.Logger, evHandler *pubsub.TopicEventHandler) {
	defer evHandler.Cancel()
	for {
		ev, err := evHandler.NextPeerEvent(ctx)
		if err != nil {
			return // ctx closed
		}
		switch ev.Type {
		case pubsub.PeerJoin:
			log.Debug("peer joined topic", "peer", ev.Peer)
		case pubsub.PeerLeave:
			log.Debug("peer left topic", "peer", ev.Peer)
		default:
			log.Warn("unrecognized topic event", "ev", ev)
		}
	}
}

type gossipTracer struct {
	m GossipMetricer
}

func (g *gossipTracer) Trace(evt *pb.TraceEvent) {
	if g.m != nil {
		g.m.RecordGossipEvent(int32(*evt.Type))
	}
}
