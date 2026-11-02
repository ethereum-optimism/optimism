package sources

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/sources/caching"
)

type L2ClientConfig struct {
	EthClientConfig

	L2BlockRefsCacheSize         int
	L1ConfigsCacheSize           int
	FetchWithdrawalRootFromState bool

	RollupCfg *rollup.Config
}

func L2ClientDefaultConfig(config *rollup.Config, trustRPC bool) *L2ClientConfig {
	// Cache 3/2 worth of sequencing window of payloads, block references, receipts and txs
	span := int(config.SeqWindowSize) * 3 / 2
	// Estimate number of L2 blocks in this span of L1 blocks
	// (there's always one L2 block per L1 block, L1 is thus the minimum, even if block time is very high)
	if config.BlockTime < 12 && config.BlockTime > 0 {
		span *= 12
		span /= int(config.BlockTime)
	}
	fullSpan := span
	if span > 1000 { // sanity cap. If a large sequencing window is configured, do not make the cache too large
		span = 1000
	}
	return L2ClientSimpleConfig(config, trustRPC, span, fullSpan)
}

func L2ClientSimpleConfig(config *rollup.Config, trustRPC bool, span, fullSpan int) *L2ClientConfig {
	return &L2ClientConfig{
		EthClientConfig: EthClientConfig{
			// receipts and transactions are cached per block
			ReceiptsCacheSize:     span,
			TransactionsCacheSize: span,
			HeadersCacheSize:      span,
			PayloadsCacheSize:     span,
			MaxRequestsPerBatch:   20,
			MaxConcurrentRequests: 10,
			BlockRefsCacheSize:    span,
			TrustRPC:              trustRPC,
			MustBePostMerge:       true,
			RPCProviderKind:       RPCKindStandard,
			MethodResetDuration:   time.Minute,
		},
		// Not bounded by span, to cover find-sync-start range fully for speedy recovery after errors.
		L2BlockRefsCacheSize: fullSpan,
		L1ConfigsCacheSize:   span,
		RollupCfg:            config,
	}
}

// L2Client extends EthClient with functions to fetch and cache eth.L2BlockRef values.
type L2Client struct {
	*EthClient
	rollupCfg *rollup.Config

	// cache L2BlockRef by hash
	// common.Hash -> eth.L2BlockRef
	l2BlockRefsCache *caching.LRUCache[common.Hash, eth.L2BlockRef]

	// cache SystemConfig by L2 hash
	// common.Hash -> eth.SystemConfig
	systemConfigsCache *caching.LRUCache[common.Hash, eth.SystemConfig]

	fetchWithdrawalRootFromState bool
}

var _ apis.L2EthClient = (*L2Client)(nil)

// NewL2Client constructs a new L2Client instance. The L2Client is a thin wrapper around the EthClient with added functions
// for fetching and caching eth.L2BlockRef values. This includes fetching an L2BlockRef by block number, label, or hash.
// See: [L2BlockRefByLabel], [L2BlockRefByNumber], [L2BlockRefByHash]
func NewL2Client(client client.RPC, log log.Logger, metrics caching.Metrics, config *L2ClientConfig) (*L2Client, error) {
	ethClient, err := NewEthClient(client, log, metrics, &config.EthClientConfig)
	if err != nil {
		return nil, err
	}

	return &L2Client{
		EthClient:                    ethClient,
		rollupCfg:                    config.RollupCfg,
		l2BlockRefsCache:             caching.NewLRUCache[common.Hash, eth.L2BlockRef](metrics, "blockrefs", config.L2BlockRefsCacheSize),
		systemConfigsCache:           caching.NewLRUCache[common.Hash, eth.SystemConfig](metrics, "systemconfigs", config.L1ConfigsCacheSize),
		fetchWithdrawalRootFromState: config.FetchWithdrawalRootFromState,
	}, nil
}

func (s *L2Client) RollupConfig() *rollup.Config {
	return s.rollupCfg
}

// ErrSyntheticTxInBody is returned by the L2Client accessors that decode a whole block body
// (InfoAndTxsBy*, HeaderAndTxsByHash) when the block holds an OP Stack synthetic transaction
// (0x7E deposit or 0x7D post-exec), which upstream go-ethereum cannot decode. Read L2 block
// bodies through the class-partitioned accessors (InfoAndUserTxsBy*, InfoAndDepositsBy*,
// InfoAndFirstDepositBy*) or PayloadBy* instead.
var ErrSyntheticTxInBody = errors.New("block holds an OP Stack synthetic transaction")

// HeaderAndTxsByHash shadows EthClient.HeaderAndTxsByHash. It returns ErrSyntheticTxInBody for
// a block that holds an OP Stack synthetic transaction, so an L2 caller fails now rather than
// only after the cutover to upstream go-ethereum.
func (s *L2Client) HeaderAndTxsByHash(ctx context.Context, hash common.Hash) (*types.Header, types.Transactions, error) {
	header, rawTxs, err := s.headerAndRawTxsByHash(ctx, hash)
	if err != nil {
		return nil, nil, err
	}
	txs, err := decodeWithoutSynthetic(rawTxs)
	if err != nil {
		return nil, nil, err
	}
	return header, txs, nil
}

// InfoAndTxsByHash shadows EthClient.InfoAndTxsByHash; see HeaderAndTxsByHash.
func (s *L2Client) InfoAndTxsByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, types.Transactions, error) {
	header, txs, err := s.HeaderAndTxsByHash(ctx, hash)
	if err != nil {
		return nil, nil, err
	}
	return eth.HeaderBlockInfoTrusted(hash, header), txs, nil
}

// InfoAndTxsByNumber shadows EthClient.InfoAndTxsByNumber; see HeaderAndTxsByHash.
func (s *L2Client) InfoAndTxsByNumber(ctx context.Context, number uint64) (eth.BlockInfo, types.Transactions, error) {
	// can't hit the cache when querying by number due to reorgs.
	header, rawTxs, err := s.blockCall(ctx, "eth_getBlockByNumber", numberID(number))
	if err != nil {
		return nil, nil, err
	}
	txs, err := decodeWithoutSynthetic(rawTxs)
	if err != nil {
		return nil, nil, err
	}
	return eth.HeaderBlockInfo(header), txs, nil
}

// InfoAndTxsByLabel shadows EthClient.InfoAndTxsByLabel; see HeaderAndTxsByHash.
func (s *L2Client) InfoAndTxsByLabel(ctx context.Context, label eth.BlockLabel) (eth.BlockInfo, types.Transactions, error) {
	// can't hit the cache when querying the head due to reorgs / changes.
	header, rawTxs, err := s.blockCall(ctx, "eth_getBlockByNumber", label)
	if err != nil {
		return nil, nil, err
	}
	txs, err := decodeWithoutSynthetic(rawTxs)
	if err != nil {
		return nil, nil, err
	}
	return eth.HeaderBlockInfo(header), txs, nil
}

// decodeWithoutSynthetic decodes txs into go-ethereum transactions, or returns
// ErrSyntheticTxInBody if any of them is an OP Stack synthetic transaction. It checks the type
// bytes before decoding, so the rejection does not depend on which go-ethereum the build resolves.
func decodeWithoutSynthetic(txs RawTransactions) (types.Transactions, error) {
	for i, tx := range txs {
		switch typ := tx.Type(); typ {
		case optypes.DepositTxType, optypes.PostExecTxType:
			return nil, fmt.Errorf("tx %d has type %#x: %w", i, typ, ErrSyntheticTxInBody)
		}
	}
	return txs.Geth()
}

// L2BlockRefByLabel returns the [eth.L2BlockRef] for the given block label.
func (s *L2Client) L2BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L2BlockRef, error) {
	envelope, err := s.PayloadByLabel(ctx, label)
	if err != nil {
		return eth.L2BlockRef{}, fmt.Errorf("failed to determine L2BlockRef of %s, could not get payload: %w", label, err)
	}
	ref, err := derive.PayloadToBlockRef(s.rollupCfg, envelope.ExecutionPayload)
	if err != nil {
		return eth.L2BlockRef{}, err
	}
	s.l2BlockRefsCache.Add(ref.Hash, ref)
	return ref, nil
}

// L2BlockRefByNumber returns the [eth.L2BlockRef] for the given block number.
func (s *L2Client) L2BlockRefByNumber(ctx context.Context, num uint64) (eth.L2BlockRef, error) {
	envelope, err := s.PayloadByNumber(ctx, num)
	if err != nil {
		return eth.L2BlockRef{}, fmt.Errorf("failed to determine L2BlockRef of height %v, could not get payload: %w", num, err)
	}
	ref, err := derive.PayloadToBlockRef(s.rollupCfg, envelope.ExecutionPayload)
	if err != nil {
		return eth.L2BlockRef{}, err
	}
	s.l2BlockRefsCache.Add(ref.Hash, ref)
	return ref, nil
}

// L2BlockRefByHash returns the [eth.L2BlockRef] for the given block hash.
// The returned BlockRef may not be in the canonical chain.
func (s *L2Client) L2BlockRefByHash(ctx context.Context, hash common.Hash) (eth.L2BlockRef, error) {
	if ref, ok := s.l2BlockRefsCache.Get(hash); ok {
		return ref, nil
	}

	envelope, err := s.PayloadByHash(ctx, hash)
	if err != nil {
		return eth.L2BlockRef{}, fmt.Errorf("failed to determine block-hash of hash %v, could not get payload: %w", hash, err)
	}
	ref, err := derive.PayloadToBlockRef(s.rollupCfg, envelope.ExecutionPayload)
	if err != nil {
		return eth.L2BlockRef{}, err
	}
	s.l2BlockRefsCache.Add(ref.Hash, ref)
	return ref, nil
}

// SystemConfigByL2Hash returns the [eth.SystemConfig] (matching the config updates up to and including the L1 origin) for the given L2 block hash.
// The returned [eth.SystemConfig] may not be in the canonical chain when the hash is not canonical.
func (s *L2Client) SystemConfigByL2Hash(ctx context.Context, hash common.Hash) (eth.SystemConfig, error) {
	if ref, ok := s.systemConfigsCache.Get(hash); ok {
		return ref, nil
	}

	envelope, err := s.PayloadByHash(ctx, hash)
	if err != nil {
		return eth.SystemConfig{}, fmt.Errorf("failed to determine block-hash of hash %v, could not get payload: %w", hash, err)
	}
	cfg, err := derive.PayloadToSystemConfig(s.rollupCfg, envelope.ExecutionPayload)
	if err != nil {
		return eth.SystemConfig{}, err
	}
	s.systemConfigsCache.Add(hash, cfg)
	return cfg, nil
}

func (s *L2Client) OutputV0AtBlockNumber(ctx context.Context, blockNum uint64) (*eth.OutputV0, error) {
	head, err := s.InfoByNumber(ctx, blockNum)
	if err != nil {
		return nil, fmt.Errorf("failed to get L2 block by hash: %w", err)
	}
	return s.outputV0(ctx, head)
}

func (s *L2Client) OutputV0AtBlock(ctx context.Context, blockHash common.Hash) (*eth.OutputV0, error) {
	head, err := s.InfoByHash(ctx, blockHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get L2 block by hash: %w", err)
	}
	return s.outputV0(ctx, head)
}

func (s *L2Client) outputV0(ctx context.Context, block eth.BlockInfo) (*eth.OutputV0, error) {
	if block == nil {
		return nil, ethereum.NotFound
	}

	blockHash := block.Hash()
	var messagePasserStorageRoot eth.Bytes32
	if s.rollupCfg.IsIsthmus(block.Time()) && !s.fetchWithdrawalRootFromState {
		s.log.Debug("Retrieving withdrawal root from block header")
		// If Isthmus hard fork has activated, we can get the withdrawal root directly from the header
		// instead of having to compute it from the contract storage trie.
		messagePasserStorageRoot = eth.Bytes32(*block.WithdrawalsRoot())

	} else {
		s.log.Debug("Bypassing block header, retrieving withdrawal root from state")
		proof, err := s.GetProof(ctx, predeploys.L2ToL1MessagePasserAddr, []common.Hash{}, blockHash.String())
		if err != nil {
			return nil, fmt.Errorf("failed to get contract proof at block %s: %w", blockHash, err)
		}
		if proof == nil {
			return nil, fmt.Errorf("proof %w", ethereum.NotFound)
		}
		// make sure that the proof (including storage hash) that we retrieved is correct by verifying it against the state-root
		if err := proof.Verify(block.Root()); err != nil {
			return nil, fmt.Errorf("invalid withdrawal root hash, state root was %s: %w", block.Root(), err)
		}
		messagePasserStorageRoot = eth.Bytes32(proof.StorageHash)
	}

	stateRoot := eth.Bytes32(block.Root())
	return &eth.OutputV0{
		StateRoot:                stateRoot,
		MessagePasserStorageRoot: messagePasserStorageRoot,
		BlockHash:                blockHash,
	}, nil
}
