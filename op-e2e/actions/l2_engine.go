package actions

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/beacon"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	geth "github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

// L2Engine is an in-memory implementation of the Engine API,
// without support for snap-sync, and no concurrency or background processes.
type L2Engine struct {
	log log.Logger

	node *node.Node
	eth  *geth.Ethereum

	rollupGenesis *rollup.Genesis

	// L2 evm / chain
	l2Chain    *core.BlockChain
	l2Database ethdb.Database
	l2Cfg      *core.Genesis
	l2Signer   types.Signer

	// L2 block building data
	l2BuildingHeader *types.Header             // block header that we add txs to for block building
	l2BuildingState  *state.StateDB            // state used for block building
	l2GasPool        *core.GasPool             // track gas used of ongoing building
	pendingIndices   map[common.Address]uint64 // per account, how many txs from the pool were already included in the block, since the pool is lagging behind block mining.
	l2Transactions   []*types.Transaction      // collects txs that were successfully included into current block build
	l2Receipts       []*types.Receipt          // collect receipts of ongoing building
	l2ForceEmpty     bool                      // when no additional txs may be processed (i.e. when sequencer drift runs out)
	l2TxFailed       []*types.Transaction      // log of failed transactions which could not be included

	payloadID beacon.PayloadID // ID of payload that is currently being built

	failL2RPC error // mock error

}

func NewL2Engine(log log.Logger, genesis *core.Genesis, rollupGenesisL1 eth.BlockID, jwtPath string) *L2Engine {
	ethCfg := &ethconfig.Config{
		NetworkId: genesis.Config.ChainID.Uint64(),
		Genesis:   genesis,
	}
	nodeCfg := &node.Config{
		Name:        "l2-geth",
		WSHost:      "127.0.0.1",
		WSPort:      0,
		AuthAddr:    "127.0.0.1",
		AuthPort:    0,
		WSModules:   []string{"debug", "admin", "eth", "txpool", "net", "rpc", "web3", "personal"},
		HTTPModules: []string{"debug", "admin", "eth", "txpool", "net", "rpc", "web3", "personal"},
		JWTSecret:   jwtPath,
	}
	n, err := node.New(nodeCfg)
	if err != nil {
		panic(err)
	}
	backend, err := geth.New(n, ethCfg)
	if err != nil {
		panic(err)
	}
	n.RegisterAPIs(tracers.APIs(backend.APIBackend))

	chain := backend.BlockChain()
	db := backend.ChainDb()
	genesisBlock := chain.Genesis()
	eng := &L2Engine{
		log:  log,
		node: n,
		eth:  backend,

		rollupGenesis: &rollup.Genesis{
			L1:     rollupGenesisL1,
			L2:     eth.BlockID{Hash: genesisBlock.Hash(), Number: genesisBlock.NumberU64()},
			L2Time: genesis.Timestamp,
		},

		l2Chain:    chain,
		l2Database: db,
		l2Cfg:      genesis,
		l2Signer:   types.LatestSigner(genesis.Config),
	}
	// register the custom engine API, so we can serve engine requests while having more control
	// over sequencing of individual txs.
	n.RegisterAPIs([]rpc.API{
		{
			Namespace:     "engine",
			Service:       (*L2EngineAPI)(eng),
			Authenticated: true,
		},
	})
	if err := n.Start(); err != nil {
		panic(fmt.Errorf("failed to start L2 op-geth node: %w", err))
	}

	return eng
}

func (s *L2Engine) RPCClient() *rpc.Client {
	cl, _ := s.node.Attach() // never errors
	return cl
}

var (
	STATUS_INVALID         = &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, PayloadID: nil}
	STATUS_SYNCING         = &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionSyncing}, PayloadID: nil}
	INVALID_TERMINAL_BLOCK = eth.PayloadStatusV1{Status: eth.ExecutionInvalid, LatestValidHash: &common.Hash{}}
)

// computePayloadId computes a pseudo-random payloadid, based on the parameters.
func computePayloadId(headBlockHash common.Hash, params *eth.PayloadAttributes) beacon.PayloadID {
	// Hash
	hasher := sha256.New()
	hasher.Write(headBlockHash[:])
	_ = binary.Write(hasher, binary.BigEndian, params.Timestamp)
	hasher.Write(params.PrevRandao[:])
	hasher.Write(params.SuggestedFeeRecipient[:])
	for _, tx := range params.Transactions {
		_ = binary.Write(hasher, binary.BigEndian, uint64(len(tx))) // length-prefix to avoid collisions
		hasher.Write(tx)
	}
	if params.NoTxPool {
		hasher.Write([]byte{1})
	}
	var out beacon.PayloadID
	copy(out[:], hasher.Sum(nil)[:8])
	return out
}

type L2EngineAPI L2Engine

func (s *L2EngineAPI) startBlock(parent common.Hash, params *eth.PayloadAttributes) error {
	if s.l2BuildingHeader != nil {
		s.log.Warn("started building new block without ending previous block", "previous", s.l2BuildingHeader, "prev_payload_id", s.payloadID)
	}

	parentHeader := s.l2Chain.GetHeaderByHash(parent)
	if parentHeader == nil {
		return fmt.Errorf("uknown parent block: %s", parent)
	}
	statedb, err := state.New(parentHeader.Root, state.NewDatabase(s.l2Database), nil)
	if err != nil {
		return fmt.Errorf("failed to init state db around block %s (state %s): %w", parent, parentHeader.Root, err)
	}

	header := &types.Header{
		ParentHash: parent,
		Coinbase:   params.SuggestedFeeRecipient,
		Difficulty: common.Big0,
		Number:     new(big.Int).Add(parentHeader.Number, common.Big1),
		GasLimit:   parentHeader.GasLimit,
		Time:       uint64(params.Timestamp),
		Extra:      nil,
		MixDigest:  common.Hash(params.PrevRandao),
	}

	header.BaseFee = misc.CalcBaseFee(s.l2Cfg.Config, parentHeader)

	s.l2BuildingHeader = header
	s.l2BuildingState = statedb
	s.l2Receipts = make([]*types.Receipt, 0)
	s.l2Transactions = make([]*types.Transaction, 0)
	s.pendingIndices = make(map[common.Address]uint64)
	s.l2ForceEmpty = params.NoTxPool
	s.l2GasPool = new(core.GasPool).AddGas(header.GasLimit)
	s.payloadID = computePayloadId(parent, params)

	// pre-process the deposits
	for i, otx := range params.Transactions {
		var tx types.Transaction
		if err := tx.UnmarshalBinary(otx); err != nil {
			return fmt.Errorf("transaction %d is not valid: %v", i, err)
		}

		receipt, err := core.ApplyTransaction(s.l2Cfg.Config, s.l2Chain, &s.l2BuildingHeader.Coinbase,
			s.l2GasPool, s.l2BuildingState, s.l2BuildingHeader, &tx, &s.l2BuildingHeader.GasUsed, *s.l2Chain.GetVMConfig())
		if err != nil {
			s.l2TxFailed = append(s.l2TxFailed, &tx)
			return fmt.Errorf("failed to apply deposit transaction to L2 block (tx %d): %w", i, err)
		}
		s.l2Receipts = append(s.l2Receipts, receipt)
		s.l2Transactions = append(s.l2Transactions, &tx)
	}
	return nil
}

func (s *L2EngineAPI) endBlock() (*types.Block, error) {
	if s.l2BuildingHeader == nil {
		return nil, fmt.Errorf("no block is being built currently (id %s)", s.payloadID)
	}
	header := s.l2BuildingHeader
	s.l2BuildingHeader = nil

	header.GasUsed = header.GasLimit - uint64(*s.l2GasPool)
	header.Root = s.l2BuildingState.IntermediateRoot(s.l2Cfg.Config.IsEIP158(header.Number))
	block := types.NewBlock(header, s.l2Transactions, nil, s.l2Receipts, trie.NewStackTrie(nil))

	// Write state changes to db
	root, err := s.l2BuildingState.Commit(s.l2Cfg.Config.IsEIP158(header.Number))
	if err != nil {
		return nil, fmt.Errorf("l2 state write error: %v", err)
	}
	if err := s.l2BuildingState.Database().TrieDB().Commit(root, false, nil); err != nil {
		return nil, fmt.Errorf("l2 trie write error: %v", err)
	}
	return block, nil
}

func (e *L2EngineAPI) GetPayloadV1(ctx context.Context, payloadId eth.PayloadID) (*eth.ExecutionPayload, error) {
	e.log.Trace("L2Engine API request received", "method", "GetPayload", "id", payloadId)
	if e.payloadID != payloadId {
		e.log.Warn("unexpected payload ID requested for block building", "expected", e.payloadID, "got", payloadId)
		return nil, beacon.UnknownPayload
	}
	bl, err := e.endBlock()
	if err != nil {
		e.log.Error("failed to finish block building", "err", err)
		return nil, beacon.UnknownPayload
	}
	return eth.BlockAsPayload(bl)
}

func (e *L2EngineAPI) ForkchoiceUpdatedV1(ctx context.Context, state *eth.ForkchoiceState, attr *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error) {
	e.log.Trace("L2Engine API request received", "method", "ForkchoiceUpdated", "head", state.HeadBlockHash, "finalized", state.FinalizedBlockHash, "safe", state.SafeBlockHash)
	if state.HeadBlockHash == (common.Hash{}) {
		e.log.Warn("Forkchoice requested update to zero hash")
		return STATUS_INVALID, nil
	}
	// Check whether we have the block yet in our database or not. If not, we'll
	// need to either trigger a sync, or to reject this forkchoice update for a
	// reason.
	block := e.l2Chain.GetBlockByHash(state.HeadBlockHash)
	if block == nil {
		// TODO: syncing not supported yet
		return STATUS_SYNCING, nil
	}
	// Block is known locally, just sanity check that the beacon client does not
	// attempt to push us back to before the merge.
	if block.Difficulty().BitLen() > 0 || block.NumberU64() == 0 {
		var (
			td  = e.l2Chain.GetTd(state.HeadBlockHash, block.NumberU64())
			ptd = e.l2Chain.GetTd(block.ParentHash(), block.NumberU64()-1)
			ttd = e.l2Chain.Config().TerminalTotalDifficulty
		)
		if td == nil || (block.NumberU64() > 0 && ptd == nil) {
			e.log.Error("TDs unavailable for TTD check", "number", block.NumberU64(), "hash", state.HeadBlockHash, "td", td, "parent", block.ParentHash(), "ptd", ptd)
			return STATUS_INVALID, errors.New("TDs unavailable for TDD check")
		}
		if td.Cmp(ttd) < 0 {
			e.log.Error("Refusing beacon update to pre-merge", "number", block.NumberU64(), "hash", state.HeadBlockHash, "diff", block.Difficulty(), "age", common.PrettyAge(time.Unix(int64(block.Time()), 0)))
			return &eth.ForkchoiceUpdatedResult{PayloadStatus: INVALID_TERMINAL_BLOCK, PayloadID: nil}, nil
		}
		if block.NumberU64() > 0 && ptd.Cmp(ttd) >= 0 {
			e.log.Error("Parent block is already post-ttd", "number", block.NumberU64(), "hash", state.HeadBlockHash, "diff", block.Difficulty(), "age", common.PrettyAge(time.Unix(int64(block.Time()), 0)))
			return &eth.ForkchoiceUpdatedResult{PayloadStatus: INVALID_TERMINAL_BLOCK, PayloadID: nil}, nil
		}
	}
	valid := func(id *beacon.PayloadID) *eth.ForkchoiceUpdatedResult {
		return &eth.ForkchoiceUpdatedResult{
			PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionValid, LatestValidHash: &state.HeadBlockHash},
			PayloadID:     id,
		}
	}
	if rawdb.ReadCanonicalHash(e.l2Database, block.NumberU64()) != state.HeadBlockHash {
		// Block is not canonical, set head.
		if latestValid, err := e.l2Chain.SetCanonical(block); err != nil {
			return &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionInvalid, LatestValidHash: &latestValid}}, err
		}
	} else if e.l2Chain.CurrentBlock().Hash() == state.HeadBlockHash {
		// If the specified head matches with our local head, do nothing and keep
		// generating the payload. It's a special corner case that a few slots are
		// missing and we are requested to generate the payload in slot.
	} else if e.l2Chain.Config().Optimism == nil { // minor L2Engine API divergence: allow proposers to reorg their own chain
		panic("engine not configured as optimism engine")
	}

	// If the beacon client also advertised a finalized block, mark the local
	// chain final and completely in PoS mode.
	if state.FinalizedBlockHash != (common.Hash{}) {
		// If the finalized block is not in our canonical tree, somethings wrong
		finalBlock := e.l2Chain.GetBlockByHash(state.FinalizedBlockHash)
		if finalBlock == nil {
			e.log.Warn("Final block not available in database", "hash", state.FinalizedBlockHash)
			return STATUS_INVALID, beacon.InvalidForkChoiceState.With(errors.New("final block not available in database"))
		} else if rawdb.ReadCanonicalHash(e.l2Database, finalBlock.NumberU64()) != state.FinalizedBlockHash {
			e.log.Warn("Final block not in canonical chain", "number", block.NumberU64(), "hash", state.HeadBlockHash)
			return STATUS_INVALID, beacon.InvalidForkChoiceState.With(errors.New("final block not in canonical chain"))
		}
		// Set the finalized block
		e.l2Chain.SetFinalized(finalBlock)
	}
	// Check if the safe block hash is in our canonical tree, if not somethings wrong
	if state.SafeBlockHash != (common.Hash{}) {
		safeBlock := e.l2Chain.GetBlockByHash(state.SafeBlockHash)
		if safeBlock == nil {
			e.log.Warn("Safe block not available in database")
			return STATUS_INVALID, beacon.InvalidForkChoiceState.With(errors.New("safe block not available in database"))
		}
		if rawdb.ReadCanonicalHash(e.l2Database, safeBlock.NumberU64()) != state.SafeBlockHash {
			e.log.Warn("Safe block not in canonical chain")
			return STATUS_INVALID, beacon.InvalidForkChoiceState.With(errors.New("safe block not in canonical chain"))
		}
		// Set the safe block
		e.l2Chain.SetSafe(safeBlock)
	}
	// If payload generation was requested, create a new block to be potentially
	// sealed by the beacon client. The payload will be requested later, and we
	// might replace it arbitrarily many times in between.
	if attr != nil {
		err := e.startBlock(state.HeadBlockHash, attr)
		if err != nil {
			e.log.Error("Failed to start block building", "err", err, "noTxPool", attr.NoTxPool, "txs", len(attr.Transactions), "timestamp", attr.Timestamp)
			return STATUS_INVALID, beacon.InvalidPayloadAttributes.With(err)
		}

		return valid(&e.payloadID), nil
	}
	return valid(nil), nil
}

func (e *L2EngineAPI) NewPayloadV1(ctx context.Context, payload *eth.ExecutionPayload) (*eth.PayloadStatusV1, error) {
	e.log.Trace("L2Engine API request received", "method", "ExecutePayload", "number", payload.BlockNumber, "hash", payload.BlockHash)
	txs := make([][]byte, len(payload.Transactions))
	for i, tx := range payload.Transactions {
		txs[i] = tx
	}
	block, err := beacon.ExecutableDataToBlock(beacon.ExecutableDataV1{
		ParentHash:    payload.ParentHash,
		FeeRecipient:  payload.FeeRecipient,
		StateRoot:     common.Hash(payload.StateRoot),
		ReceiptsRoot:  common.Hash(payload.ReceiptsRoot),
		LogsBloom:     payload.LogsBloom[:],
		Random:        common.Hash(payload.PrevRandao),
		Number:        uint64(payload.BlockNumber),
		GasLimit:      uint64(payload.GasLimit),
		GasUsed:       uint64(payload.GasUsed),
		Timestamp:     uint64(payload.Timestamp),
		ExtraData:     payload.ExtraData,
		BaseFeePerGas: payload.BaseFeePerGas.ToBig(),
		BlockHash:     payload.BlockHash,
		Transactions:  txs,
	})
	if err != nil {
		log.Debug("Invalid NewPayload params", "params", payload, "error", err)
		return &eth.PayloadStatusV1{Status: eth.ExecutionInvalidBlockHash}, nil
	}
	// If we already have the block locally, ignore the entire execution and just
	// return a fake success.
	if block := e.l2Chain.GetBlockByHash(payload.BlockHash); block != nil {
		e.log.Warn("Ignoring already known beacon payload", "number", payload.BlockNumber, "hash", payload.BlockHash, "age", common.PrettyAge(time.Unix(int64(block.Time()), 0)))
		hash := block.Hash()
		return &eth.PayloadStatusV1{Status: eth.ExecutionValid, LatestValidHash: &hash}, nil
	}

	// TODO: skipping invalid ancestor check (i.e. not remembering previously failed blocks)

	parent := e.l2Chain.GetBlock(block.ParentHash(), block.NumberU64()-1)
	if parent == nil {
		// TODO: hack, saying we accepted if we don't know the parent block. Might want to return critical error if we can't actually sync.
		return &eth.PayloadStatusV1{Status: eth.ExecutionAccepted, LatestValidHash: nil}, nil
	}
	if !e.l2Chain.HasBlockAndState(block.ParentHash(), block.NumberU64()-1) {
		e.log.Warn("State not available, ignoring new payload")
		return &eth.PayloadStatusV1{Status: eth.ExecutionAccepted}, nil
	}
	if err := e.l2Chain.InsertBlockWithoutSetHead(block); err != nil {
		e.log.Warn("NewPayloadV1: inserting block failed", "error", err)
		// TODO not remembering the payload as invalid
		return e.invalid(err, parent.Header()), nil
	}
	hash := block.Hash()
	return &eth.PayloadStatusV1{Status: eth.ExecutionValid, LatestValidHash: &hash}, nil
}

func (e *L2EngineAPI) invalid(err error, latestValid *types.Header) *eth.PayloadStatusV1 {
	currentHash := e.l2Chain.CurrentBlock().Hash()
	if latestValid != nil {
		// Set latest valid hash to 0x0 if parent is PoW block
		currentHash = common.Hash{}
		if latestValid.Difficulty.BitLen() == 0 {
			// Otherwise set latest valid hash to parent hash
			currentHash = latestValid.Hash()
		}
	}
	errorMsg := err.Error()
	return &eth.PayloadStatusV1{Status: eth.ExecutionInvalid, LatestValidHash: &currentHash, ValidationError: &errorMsg}
}

// make next L2 request fail
func (s *L2Engine) actL2RPCFail(t Testing) {
	if s.failL2RPC != nil { // already set to fail?
		t.InvalidAction("already set a mock L2 rpc error")
		return
	}
	s.failL2RPC = errors.New("mock L2 RPC error")
}

// add next tx from L2 tx queue
func (s *L2Engine) actL2IncludeTx(from common.Address) Action {
	return func(t Testing) {
		if s.l2BuildingHeader == nil {
			t.InvalidAction("not currently building a block, cannot include tx from queue")
			return
		}
		if s.l2ForceEmpty {
			t.InvalidAction("cannot include any sequencer txs")
			return
		}

		i := s.pendingIndices[from]
		txs, q := s.eth.TxPool().ContentFrom(from)
		if uint64(len(txs)) <= i {
			t.Fatalf("no pending txs from %s, and have %d unprocessable queued txs from this account", from, len(q))
		}
		tx := txs[i]
		if tx.Gas() > s.l2BuildingHeader.GasLimit {
			t.Fatalf("tx consumes %d gas, more than available in L2 block %d", tx.Gas(), s.l2BuildingHeader.GasLimit)
		}
		if tx.Gas() > uint64(*s.l2GasPool) {
			t.InvalidAction("action takes too much gas: %d, only have %d", tx.Gas(), uint64(*s.l2GasPool))
			return
		}
		s.pendingIndices[from] = i + 1 // won't retry the tx
		receipt, err := core.ApplyTransaction(s.l2Cfg.Config, s.l2Chain, &s.l2BuildingHeader.Coinbase,
			s.l2GasPool, s.l2BuildingState, s.l2BuildingHeader, tx, &s.l2BuildingHeader.GasUsed, *s.l2Chain.GetVMConfig())
		if err != nil {
			s.l2TxFailed = append(s.l2TxFailed, tx)
			t.Fatalf("failed to apply transaction to L1 block (tx %d): %v", len(s.l2Transactions), err)
		}
		s.l2Receipts = append(s.l2Receipts, receipt)
		s.l2Transactions = append(s.l2Transactions, tx)
	}
}
