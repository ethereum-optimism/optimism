package p2p

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/golang/snappy"
	lru "github.com/hashicorp/golang-lru/v2"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	opsigner "github.com/ethereum-optimism/optimism/op-service/signer"
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
	// 130 * gossipHeartbeat
	seenMessagesTTL  = 130 * gossipHeartbeat
	DefaultMeshD     = 8  // topic stable mesh target count
	DefaultMeshDlo   = 6  // topic stable mesh low watermark
	DefaultMeshDhi   = 12 // topic stable mesh high watermark
	DefaultMeshDlazy = 6  // gossip target
	// peerScoreInspectFrequency is the frequency at which peer scores are inspected
	peerScoreInspectFrequency = 15 * time.Second
)

// Message domains, the msg id function uncompresses to keep data monomorphic,
// but invalid compressed data will need a unique different id.

var MessageDomainInvalidSnappy = [4]byte{0, 0, 0, 0}
var MessageDomainValidSnappy = [4]byte{1, 0, 0, 0}

type GossipSetupConfigurables interface {
	PeerScoringParams() *ScoringParams
	// ConfigureGossip creates configuration options to apply to the GossipSub setup
	ConfigureGossip() []pubsub.Option
}

type GossipRuntimeConfig interface {
	P2PSequencerAddress() common.Address
}

//go:generate mockery --name GossipMetricer
type GossipMetricer interface {
	RecordGossipEvent(evType int32)
}

type BlockVersioning interface {
	BlockVersion(chainID eth.ChainID, timestamp uint64) (eth.BlockVersion, error)
}

type BlockRuntimeAuth interface {
	// The result may change over time.
	P2PSequencerAddress(chainID eth.ChainID) (common.Address, error)
}

type GossipChainConfig interface {
	// ChainIDs returns the lists of chains to subscribe to
	ChainIDs() []eth.ChainID
	BlockRuntimeAuth
	BlockVersioning
}

type SingleChainGossip struct {
	RollupCfg        *rollup.Config
	P2PSequencerAuth GossipRuntimeConfig
}

func (s *SingleChainGossip) P2PSequencerAddress(chainID eth.ChainID) (common.Address, error) {
	if eth.ChainIDFromBig(s.RollupCfg.L2ChainID) != chainID {
		return common.Address{}, fmt.Errorf("unknown chain: %s", chainID)
	}
	return s.P2PSequencerAuth.P2PSequencerAddress(), nil
}

func (s *SingleChainGossip) ChainIDs() []eth.ChainID {
	return []eth.ChainID{eth.ChainIDFromBig(s.RollupCfg.L2ChainID)}
}

func (s *SingleChainGossip) BlockTime(chainID eth.ChainID) time.Duration {
	return time.Duration(s.RollupCfg.BlockTime) * time.Second
}

func (s *SingleChainGossip) BlockVersion(chainID eth.ChainID, timestamp uint64) (eth.BlockVersion, error) {
	if eth.ChainIDFromBig(s.RollupCfg.L2ChainID) != chainID {
		return 0, fmt.Errorf("unknown chain: %s", chainID)
	}
	if s.RollupCfg.IsIsthmus(timestamp) {
		return eth.BlockV4, nil
	} else if s.RollupCfg.IsEcotone(timestamp) {
		return eth.BlockV3, nil
	} else if s.RollupCfg.IsCanyon(timestamp) {
		return eth.BlockV2, nil
	} else {
		return eth.BlockV1, nil
	}
}

var _ GossipChainConfig = (*SingleChainGossip)(nil)

// BuildSubscriptionFilter builds a simple subscription filter,
// to help protect against peers spamming useless subscriptions.
func BuildSubscriptionFilter(chains []eth.ChainID) pubsub.SubscriptionFilter {
	var topics []string
	for _, chainID := range chains {
		topics = append(topics,
			eth.BlockV1.BlocksTopic(chainID),
			eth.BlockV2.BlocksTopic(chainID),
			eth.BlockV3.BlocksTopic(chainID),
			eth.BlockV4.BlocksTopic(chainID))
	}
	// add more topics here in the future, if any.
	return pubsub.NewAllowlistSubscriptionFilter(topics...)
}

var msgBufPool = sync.Pool{New: func() any {
	// note: the topic validator concurrency is limited, so pool won't blow up, even with large pre-allocation.
	x := make([]byte, 0, maxGossipSize)
	return &x
}}

// BuildMsgIdFn builds a generic message ID function for gossipsub that can handle compressed payloads,
// mirroring the eth2 p2p gossip spec.
func BuildMsgIdFn() pubsub.MsgIdFunction {
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
			if data, err = snappy.Decode((*res)[:cap(*res)], pmsg.Data); err == nil {
				if cap(data) > cap(*res) {
					// if we ended up growing the slice capacity, fine, keep the larger one.
					*res = data[:cap(data)]
				}
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

func (p *Config) ConfigureGossip() []pubsub.Option {
	params := BuildGlobalGossipParams()

	// override with CLI changes
	params.D = p.MeshD
	params.Dlo = p.MeshDLo
	params.Dhi = p.MeshDHi
	params.Dlazy = p.MeshDLazy

	// in the future we may add more advanced options like scoring and PX / direct-mesh / episub
	return []pubsub.Option{
		pubsub.WithGossipSubParams(params),
		pubsub.WithFloodPublish(p.FloodPublish),
	}
}

func BuildGlobalGossipParams() pubsub.GossipSubParams {
	params := pubsub.DefaultGossipSubParams()
	params.D = DefaultMeshD                    // topic stable mesh target count
	params.Dlo = DefaultMeshDlo                // topic stable mesh low watermark
	params.Dhi = DefaultMeshDhi                // topic stable mesh high watermark
	params.Dlazy = DefaultMeshDlazy            // gossip target
	params.HeartbeatInterval = gossipHeartbeat // interval of heartbeat
	params.FanoutTTL = 24 * time.Second        // ttl for fanout maps for topics we are not subscribed to but have published to
	params.HistoryLength = 12                  // number of windows to retain full messages in cache for IWANT responses
	params.HistoryGossip = 3                   // number of windows to gossip about

	return params
}

// NewGossipSub configures a new pubsub instance with the specified parameters.
// PubSub uses a GossipSubRouter as it's router under the hood.
func NewGossipSub(p2pCtx context.Context, h host.Host, cfg GossipChainConfig,
	gossipConf GossipSetupConfigurables, scorer Scorer, m GossipMetricer, log log.Logger) (*pubsub.PubSub, error) {
	denyList, err := pubsub.NewTimeCachedBlacklist(30 * time.Second)
	if err != nil {
		return nil, err
	}
	gossipOpts := []pubsub.Option{
		pubsub.WithMaxMessageSize(maxGossipSize),
		pubsub.WithMessageIdFn(BuildMsgIdFn()),
		pubsub.WithNoAuthor(),
		pubsub.WithMessageSignaturePolicy(pubsub.StrictNoSign),
		pubsub.WithSubscriptionFilter(BuildSubscriptionFilter(cfg.ChainIDs())),
		pubsub.WithValidateQueueSize(maxValidateQueue),
		pubsub.WithPeerOutboundQueueSize(maxOutboundQueue),
		pubsub.WithValidateThrottle(globalValidateThrottle),
		pubsub.WithSeenMessagesTTL(seenMessagesTTL),
		pubsub.WithPeerExchange(false),
		pubsub.WithBlacklist(denyList),
		pubsub.WithEventTracer(&gossipTracer{m: m}),
	}
	gossipOpts = append(gossipOpts, ConfigurePeerScoring(gossipConf, scorer, log)...)
	gossipOpts = append(gossipOpts, gossipConf.ConfigureGossip()...)
	return pubsub.NewGossipSub(p2pCtx, h, gossipOpts...)
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
		var src any
		src = id
		if id == self {
			src = "self"
		}
		log.Debug(msg, "result", validationResultString(res), "from", src)
		return res
	}
}

func guardGossipValidator(log log.Logger, fn pubsub.ValidatorEx) pubsub.ValidatorEx {
	return func(ctx context.Context, id peer.ID, message *pubsub.Message) (result pubsub.ValidationResult) {
		defer func() {
			if err := recover(); err != nil {
				log.Error("gossip validation panic", "err", err, "peer", id)
				result = pubsub.ValidationReject
			}
		}()
		return fn(ctx, id, message)
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

func BuildBlocksValidator(log log.Logger, chainID eth.ChainID, runCfg BlockRuntimeAuth, blockVersion eth.BlockVersion) pubsub.ValidatorEx {
	// Seen block hashes per block height
	// uint64 -> *seenBlocks
	blockHeightLRU, err := lru.New[uint64, *seenBlocks](1000)
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
		data, err := snappy.Decode((*res)[:cap(*res)], message.Data)
		if err != nil {
			log.Warn("invalid snappy compression", "err", err, "peer", id)
			return pubsub.ValidationReject
		}
		// if we ended up growing the slice capacity, fine, keep the larger one.
		if cap(data) > cap(*res) {
			*res = data[:cap(data)]
		}

		// message starts with compact-encoding secp256k1 encoded signature
		signature := eth.Bytes65(data[:65])
		payloadBytes := data[65:]

		// [REJECT] if the signature by the sequencer is not valid
		result := verifyBlockSignature(log, chainID, runCfg, id, signature, payloadBytes)
		if result != pubsub.ValidationAccept {
			return result
		}

		var envelope eth.ExecutionPayloadEnvelope

		// [REJECT] if the block encoding is not valid
		if blockVersion.HasParentBeaconBlockRoot() {
			if err := envelope.UnmarshalSSZ(blockVersion, uint32(len(payloadBytes)), bytes.NewReader(payloadBytes)); err != nil {
				log.Warn("invalid envelope payload", "err", err, "peer", id)
				return pubsub.ValidationReject
			}
		} else {
			var payload eth.ExecutionPayload
			if err := payload.UnmarshalSSZ(blockVersion, uint32(len(payloadBytes)), bytes.NewReader(payloadBytes)); err != nil {
				log.Warn("invalid execution payload", "err", err, "peer", id)
				return pubsub.ValidationReject
			}
			envelope = eth.ExecutionPayloadEnvelope{ExecutionPayload: &payload}
		}

		payload := envelope.ExecutionPayload

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
		if actual, ok := envelope.CheckBlockHash(); !ok {
			log.Warn("payload has bad block hash", "bad_hash", payload.BlockHash.String(), "actual", actual.String())
			return pubsub.ValidationReject
		}

		// [REJECT] if a V1 Block has withdrawals
		if !blockVersion.HasWithdrawals() && payload.Withdrawals != nil {
			log.Warn("payload is on v1 topic, but has withdrawals", "bad_hash", payload.BlockHash.String())
			return pubsub.ValidationReject
		}

		// [REJECT] if a >= V2 Block does not have withdrawals
		if blockVersion.HasWithdrawals() && payload.Withdrawals == nil {
			log.Warn("payload is on v2/v3 topic, but does not have withdrawals", "bad_hash", payload.BlockHash.String())
			return pubsub.ValidationReject
		}

		// [REJECT] if a >= V2 Block has non-empty withdrawals
		if blockVersion.HasWithdrawals() && len(*payload.Withdrawals) != 0 {
			log.Warn("payload is on v2/v3 topic, but has non-empty withdrawals", "bad_hash", payload.BlockHash.String(), "withdrawal_count", len(*payload.Withdrawals))
			return pubsub.ValidationReject
		}

		// [REJECT] if the block is on a topic <= V2 and has a blob gas value set
		if !blockVersion.HasBlobProperties() && payload.BlobGasUsed != nil {
			log.Warn("payload is on v1/v2 topic, but has blob gas used", "bad_hash", payload.BlockHash.String())
			return pubsub.ValidationReject
		}

		// [REJECT] if the block is on a topic <= V2 and has an excess blob gas value set
		if !blockVersion.HasBlobProperties() && payload.ExcessBlobGas != nil {
			log.Warn("payload is on v1/v2 topic, but has excess blob gas", "bad_hash", payload.BlockHash.String())
			return pubsub.ValidationReject
		}

		if blockVersion.HasBlobProperties() {
			// [REJECT] if the block is on a topic >= V3 and has a blob gas used value that is not zero
			if payload.BlobGasUsed == nil || *payload.BlobGasUsed != 0 {
				log.Warn("payload is on v3 topic, but has non-zero blob gas used", "bad_hash", payload.BlockHash.String(), "blob_gas_used", payload.BlobGasUsed)
				return pubsub.ValidationReject
			}

			// [REJECT] if the block is on a topic >= V3 and has an excess blob gas value that is not zero
			if payload.ExcessBlobGas == nil || *payload.ExcessBlobGas != 0 {
				log.Warn("payload is on v3 topic, but has non-zero excess blob gas", "bad_hash", payload.BlockHash.String(), "excess_blob_gas", payload.ExcessBlobGas)
				return pubsub.ValidationReject
			}
		}

		// [REJECT] if the block is on a topic >= V3 and the parent beacon block root is nil
		if blockVersion.HasParentBeaconBlockRoot() && envelope.ParentBeaconBlockRoot == nil {
			log.Warn("payload is on v3 topic, but has nil parent beacon block root", "bad_hash", payload.BlockHash.String())
			return pubsub.ValidationReject
		}

		if blockVersion.HasWithdrawalsRoot() && payload.WithdrawalsRoot == nil {
			log.Warn("payload is on v4 topic, but has nil withdrawals root", "bad_hash", payload.BlockHash.String())
			return pubsub.ValidationReject
		}

		seen, ok := blockHeightLRU.Get(uint64(payload.BlockNumber))
		if !ok {
			seen = new(seenBlocks)
			blockHeightLRU.Add(uint64(payload.BlockNumber), seen)
		}

		if count, hasSeen := seen.hasSeen(payload.BlockHash); count > 5 {
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
		seen.markSeen(payload.BlockHash)

		// remember the decoded payload for later usage in topic subscriber.
		message.ValidatorData = &envelope
		return pubsub.ValidationAccept
	}
}

func verifyBlockSignature(log log.Logger, chainID eth.ChainID, runCfg BlockRuntimeAuth, id peer.ID, signature eth.Bytes65, payloadBytes []byte) pubsub.ValidationResult {
	seqAddress, err := runCfg.P2PSequencerAddress(chainID)
	if err != nil {
		log.Warn("Cannot determine P2P Sequencer address to auth against", "chainID", chainID, "err", err)
		return pubsub.ValidationReject
	}
	authCtx := &opsigner.OPStackP2PBlockAuthV1{
		Allowed: seqAddress,
		Chain:   chainID,
	}
	if authCtx.Allowed == (common.Address{}) {
		log.Warn("no configured p2p sequencer address, ignoring gossiped block", "peer", id, "addr", authCtx.Allowed)
		return pubsub.ValidationIgnore
	}
	block := opsigner.SignedP2PBlock{
		Raw:       payloadBytes,
		Signature: signature,
	}
	if err := block.VerifySignature(authCtx); err != nil {
		log.Warn("invalid block signature", "err", err, "peer", id)
		return pubsub.ValidationReject
	}
	return pubsub.ValidationAccept
}

type GossipIn interface {
	OnUnsafeL2Payload(ctx context.Context, chainID eth.ChainID, from peer.ID, msg *eth.ExecutionPayloadEnvelope) error
}

type GossipOut interface {
	SignAndPublishL2Payload(ctx context.Context, chainID eth.ChainID, msg *eth.ExecutionPayloadEnvelope, signer Signer) error
	PublishSignedL2Payload(ctx context.Context, chainID eth.ChainID, signedEnvelope *opsigner.SignedExecutionPayloadEnvelope) error
	Close() error
}

type blockTopic struct {
	// blocks topic, main handle on block gossip
	topic *pubsub.Topic
	// block events handler, to be cancelled before closing the blocks topic.
	events *pubsub.TopicEventHandler
	// block subscriptions, to be cancelled before closing blocks topic.
	sub *pubsub.Subscription
}

func (bt *blockTopic) Close() error {
	bt.events.Cancel()
	bt.sub.Cancel()
	return bt.topic.Close()
}

type blockTopicKey struct {
	eth.BlockVersion
	eth.ChainID
}

type publisher struct {
	log log.Logger

	blockVersioning BlockVersioning

	// p2pCancel cancels the downstream gossip event-handling functions, independent of the sources.
	// A closed gossip event source (event handler or subscription) does not stop any open event iteration,
	// thus we have to stop it ourselves this way.
	p2pCancel context.CancelFunc

	blockTopics map[blockTopicKey]*blockTopic
}

var _ GossipOut = (*publisher)(nil)

func (p *publisher) PublishSignedL2Payload(ctx context.Context, chainID eth.ChainID, signedEnvelope *opsigner.SignedExecutionPayloadEnvelope) error {
	res := msgBufPool.Get().(*[]byte)
	buf := bytes.NewBuffer((*res)[:0])
	defer func() {
		*res = buf.Bytes()
		defer msgBufPool.Put(res)
	}()

	buf.Write(signedEnvelope.Signature[:])

	if signedEnvelope.Envelope.ParentBeaconBlockRoot != nil {
		if _, err := signedEnvelope.Envelope.MarshalSSZ(buf); err != nil {
			return fmt.Errorf("failed to encoded execution payload envelope to publish: %w", err)
		}
	} else {
		if _, err := signedEnvelope.Envelope.ExecutionPayload.MarshalSSZ(buf); err != nil {
			return fmt.Errorf("failed to encoded execution payload to publish: %w", err)
		}
	}

	data := buf.Bytes()
	timestamp := uint64(signedEnvelope.Envelope.ExecutionPayload.Timestamp)
	return p.publishRawSignedPayload(ctx, chainID, timestamp, data)
}

func (p *publisher) SignAndPublishL2Payload(ctx context.Context, chainID eth.ChainID, envelope *eth.ExecutionPayloadEnvelope, signer Signer) error {
	res := msgBufPool.Get().(*[]byte)
	buf := bytes.NewBuffer((*res)[:0])
	defer func() {
		*res = buf.Bytes()
		defer msgBufPool.Put(res)
	}()

	buf.Write(make([]byte, 65))

	if envelope.ParentBeaconBlockRoot != nil {
		if _, err := envelope.MarshalSSZ(buf); err != nil {
			return fmt.Errorf("failed to encoded execution payload envelope to publish: %w", err)
		}
	} else {
		if _, err := envelope.ExecutionPayload.MarshalSSZ(buf); err != nil {
			return fmt.Errorf("failed to encoded execution payload to publish: %w", err)
		}
	}
	data := buf.Bytes()
	payloadData := data[65:]
	payloadHash := opsigner.PayloadHash(payloadData)
	sig, err := signer.SignBlockV1(ctx, chainID, payloadHash)
	if err != nil {
		return fmt.Errorf("failed to sign execution payload with signer: %w", err)
	}
	copy(data[:65], sig[:])
	return p.publishRawSignedPayload(ctx, chainID, uint64(envelope.ExecutionPayload.Timestamp), data)
}

func (p *publisher) publishRawSignedPayload(ctx context.Context, chainID eth.ChainID, timestamp uint64, data []byte) error {
	// compress the full message
	// This also copies the data, freeing up the original buffer to go back into the pool
	out := snappy.Encode(nil, data)

	blockVersion, err := p.blockVersioning.BlockVersion(chainID, timestamp)
	if err != nil {
		return fmt.Errorf("unable to determine block version to publish on (chain %s, timestamp %d): %w", chainID, timestamp, err)
	}
	k := blockTopicKey{
		BlockVersion: blockVersion,
		ChainID:      chainID,
	}
	top, ok := p.blockTopics[k]
	if !ok {
		return fmt.Errorf("cannot publish on unknown block version %s / chain ID %s", blockVersion, chainID)
	}
	return top.topic.Publish(ctx, out)
}

func (p *publisher) Close() error {
	p.p2pCancel()
	var out error
	for key, v := range p.blockTopics {
		if err := v.Close(); err != nil {
			out = errors.Join(out, fmt.Errorf("failed to close topic blocks %s %s: %w", key.ChainID, key.BlockVersion, err))
		}
	}
	return out
}

func setupBlockTopic(p2pCtx context.Context, ps *pubsub.PubSub, self peer.ID, chainID eth.ChainID, runCfg BlockRuntimeAuth, blockVersion eth.BlockVersion, logger log.Logger, gossipIn GossipIn) (*blockTopic, error) {
	logger = logger.New("topic", "blocks"+strings.ToUpper(blockVersion.String()))
	validator := BuildBlocksValidator(logger, chainID, runCfg, blockVersion)
	topicName := blockVersion.BlocksTopic(chainID)
	guardedValidator := guardGossipValidator(logger, logValidationResult(self, "validated "+blockVersion.String(), logger, validator))
	blTopic, err := newBlockTopic(p2pCtx, chainID, topicName, ps, logger, gossipIn, guardedValidator)
	if err != nil {
		return nil, fmt.Errorf("failed to setup blocks %s topic p2p: %w", blockVersion, err)
	}
	return blTopic, nil
}

func JoinGossip(self peer.ID, ps *pubsub.PubSub, logger log.Logger, cfg GossipChainConfig, gossipIn GossipIn) (GossipOut, error) {
	p2pCtx, p2pCancel := context.WithCancel(context.Background())

	versions := []eth.BlockVersion{eth.BlockV1, eth.BlockV2, eth.BlockV3, eth.BlockV4}
	chainIDs := cfg.ChainIDs()
	blockTopics := make(map[blockTopicKey]*blockTopic)
	for _, blockVersion := range versions {
		for _, chainID := range chainIDs {
			// TODO: we can stop subscribing to old inactive topics

			blTopic, err := setupBlockTopic(p2pCtx, ps, self, chainID, cfg, blockVersion, logger, gossipIn)
			if err != nil {
				p2pCancel()
				return nil, err
			}
			key := blockTopicKey{
				BlockVersion: blockVersion,
				ChainID:      chainID,
			}
			blockTopics[key] = blTopic
		}
	}

	return &publisher{
		log:             logger,
		blockVersioning: cfg,
		p2pCancel:       p2pCancel,
		blockTopics:     blockTopics,
	}, nil
}

func newBlockTopic(ctx context.Context, chainID eth.ChainID, topicId string, ps *pubsub.PubSub, log log.Logger, gossipIn GossipIn, validator pubsub.ValidatorEx) (*blockTopic, error) {
	err := ps.RegisterTopicValidator(topicId,
		validator,
		pubsub.WithValidatorTimeout(3*time.Second),
		pubsub.WithValidatorConcurrency(4))

	if err != nil {
		return nil, fmt.Errorf("failed to register gossip topic: %w", err)
	}

	blocksTopic, err := ps.Join(topicId)
	if err != nil {
		return nil, fmt.Errorf("failed to join gossip topic: %w", err)
	}

	blocksTopicEvents, err := blocksTopic.EventHandler()
	if err != nil {
		return nil, fmt.Errorf("failed to create blocks gossip topic handler: %w", err)
	}

	go LogTopicEvents(ctx, log, blocksTopicEvents)

	subscription, err := blocksTopic.Subscribe()
	if err != nil {
		err = errors.Join(err, blocksTopic.Close())
		return nil, fmt.Errorf("failed to subscribe to blocks gossip topic: %w", err)
	}

	subscriber := MakeSubscriber(log, BlocksHandler(chainID, gossipIn.OnUnsafeL2Payload))
	go subscriber(ctx, subscription)

	return &blockTopic{
		topic:  blocksTopic,
		events: blocksTopicEvents,
		sub:    subscription,
	}, nil
}

type TopicSubscriber func(ctx context.Context, sub *pubsub.Subscription)
type MessageHandler func(ctx context.Context, from peer.ID, msg any) error

func BlocksHandler(chainID eth.ChainID, onBlock func(ctx context.Context, chainID eth.ChainID, from peer.ID, msg *eth.ExecutionPayloadEnvelope) error) MessageHandler {
	return func(ctx context.Context, from peer.ID, msg any) error {
		payload, ok := msg.(*eth.ExecutionPayloadEnvelope)
		if !ok {
			return fmt.Errorf("expected topic validator to parse and validate data into execution payload, but got %T", msg)
		}
		return onBlock(ctx, chainID, from, payload)
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
