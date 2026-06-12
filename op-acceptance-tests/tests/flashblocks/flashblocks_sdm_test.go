package flashblocks

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/pkg/sdm"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type flashblocksIncludedTx struct {
	receipt  *types.Receipt
	blockNum uint64
}

type observedSdmFlashblock struct {
	index      uint64
	postExecTx []byte
	payload    *sdm.PostExecPayload
	// blockHash is diff.block_hash: the hash op-rbuilder computed over the materialized view
	// (base + all diff.transactions + this post_exec_tx). Matches the canonical block hash only
	// when rollup-boost serves the builder payload, not an EL fallback block.
	blockHash common.Hash
}

// TestFlashblocksSDMMaterializesPostExecBlock proves that op-rbuilder publishes the current SDM
// PostExec transaction in flashblocks and rollup-boost materializes exactly that trailing tx in
// the final execution payload.
func TestFlashblocksSDMMaterializesPostExecBlock(gt *testing.T) {
	t := devtest.SerialT(gt)
	sysgo.SkipOnKonaNode(t, "flashblocks acceptance preset requires user RPC")
	sysgo.SkipOnOpGeth(t, "SDM flashblocks require op-reth post-exec support")

	// SDM rides Interop: activating Interop at genesis turns SDM on for op-reth execution
	// and the op-rbuilder payload builder consistently with op-node derivation. The preset
	// also provisions the DependencySet required by op-node whenever Interop is scheduled.
	sys := presets.NewSingleChainWithFlashblocks(t, presets.WithInteropAtGenesis())

	driveViaTestSequencer(t, sys, 2)

	// SDM PostExec production is gated by an in-memory operator flag that starts
	// disabled on every process boot — protocol activation alone is not enough.
	// Opt in on both the sequencer EL (op-reth fallback) and op-rbuilder (the
	// flashblocks producer), since rollup-boost may route to either.
	//
	// Unlike op-node (which persists the opt-in in its config file — see
	// op-node/config/config_persistence.go), op-reth and op-rbuilder hold the
	// flag in a process-local Arc<AtomicBool> with no disk backing (see the
	// "persistence is deliberately out of scope" notes in
	// rust/op-reth/crates/rpc/src/sdm_admin.rs and
	// rust/op-rbuilder/crates/op-rbuilder/src/sdm_admin.rs). Any restart of
	// either Rust process drops the opt-in, so callers — tests here, operators
	// in prod — must re-issue admin_setSdmPostExecOptIn after every boot.
	setFlashblocksSDMEnabled(t, sys.L2EL.Escape().L2EthClient().RPC(), true)
	setFlashblocksSDMEnabled(t, sys.L2OPRBuilder.Escape().L2EthClient().RPC(), true)

	fbClient := sources.NewFlashblockClient(
		sys.L2OPRBuilder.FlashblocksClient(),
		t.Logger().With("stream_source", "op-rbuilder"),
		200,
	)
	startClient(t, fbClient)

	alice := sys.FunderL2.NewFundedEOA(eth.OneEther)
	stateBloatAddr := flashblocksDeployContract(t, alice, sdm.StateBloatBin)

	const batchSize = 50
	const slotCount = 20
	startNonce := alice.PendingNonce()
	plannedTxs := make([]*txplan.PlannedTx, 0, batchSize)
	for i := 0; i < batchSize; i++ {
		plannedTxs = append(plannedTxs, flashblocksSubmitTxWithoutWait(
			t,
			alice,
			startNonce+uint64(i),
			txplan.WithTo(&stateBloatAddr),
			txplan.WithData(sdm.EncodeRun(slotCount)),
			txplan.WithGasLimit(1_000_000),
		))
	}

	type receiptResult struct {
		idx     int
		receipt *types.Receipt
		err     error
	}
	receiptCh := make(chan receiptResult, batchSize)
	var wg sync.WaitGroup
	defer wg.Wait()
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(receiptCh)
		for i, ptx := range plannedTxs {
			receipt, err := ptx.Included.Eval(t.Ctx())
			receiptCh <- receiptResult{idx: i, receipt: receipt, err: err}
			if err != nil {
				return
			}
		}
	}()

	observedByBlock := make(map[uint64]observedSdmFlashblock)
	includedByBlock := make(map[uint64][]flashblocksIncludedTx)
	var targetBlockNum uint64
	// candidate is a block that satisfies the "2+ workload receipts AND an observed
	// post_exec_tx flashblock" criteria. We don't commit it as targetBlockNum until we
	// see a flashblock for a later block, which is op-rbuilder's de-facto seal signal —
	// flashblock payloads carry no explicit "is_final" flag, and post_exec_tx accumulates
	// entries with every new flashblock for the same block. Committing the candidate
	// earlier risks comparing the final materialized block against an in-progress
	// flashblock snapshot.
	var candidate uint64
	var postReceiptTimer *time.Timer
	var postReceiptDeadline <-chan time.Time
	defer func() {
		if postReceiptTimer != nil {
			postReceiptTimer.Stop()
		}
	}()

	tryMarkCandidate := func(blockNum uint64) {
		if candidate != 0 {
			return
		}
		if len(includedByBlock[blockNum]) < 2 {
			return
		}
		if _, ok := observedByBlock[blockNum]; !ok {
			return
		}
		candidate = blockNum
	}

	for targetBlockNum == 0 {
		select {
		case fb, ok := <-fbClient.Next():
			t.Require().True(ok, "flashblock client closed before observing SDM post_exec_tx")
			t.Require().NotNil(fb)
			blockNum := uint64(fb.Metadata.BlockNumber)
			if candidate != 0 && blockNum > candidate {
				targetBlockNum = candidate
				continue
			}
			if fb.Diff.PostExecTx == nil {
				continue
			}
			t.Require().False(flashblockTransactionsContainPostExec(fb),
				"SDM post-exec tx must be carried in diff.post_exec_tx, not diff.transactions")
			payload, raw := decodeFlashblockPostExecTx(t, fb)
			observedByBlock[blockNum] = observedSdmFlashblock{
				index:      uint64(fb.Index),
				postExecTx: raw,
				payload:    payload,
				blockHash:  common.HexToHash(fb.Diff.BlockHash),
			}
			tryMarkCandidate(blockNum)
		case result, ok := <-receiptCh:
			if !ok {
				receiptCh = nil
				postReceiptTimer = time.NewTimer(15 * time.Second)
				postReceiptDeadline = postReceiptTimer.C
				continue
			}
			t.Require().NoError(result.err, "workload tx %d failed before inclusion", result.idx)
			t.Require().Equal(types.ReceiptStatusSuccessful, result.receipt.Status,
				"workload tx %d must succeed", result.idx)
			blockNum := bigs.Uint64Strict(result.receipt.BlockNumber)
			includedByBlock[blockNum] = append(includedByBlock[blockNum], flashblocksIncludedTx{
				receipt:  result.receipt,
				blockNum: blockNum,
			})
			tryMarkCandidate(blockNum)
		case <-postReceiptDeadline:
			if candidate != 0 {
				targetBlockNum = candidate
				continue
			}
			t.Require().Fail("post-receipt wait expired", "all workload receipts arrived but no matching SDM flashblock post_exec_tx was observed")
		case <-t.Ctx().Done():
			t.Require().NoError(t.Ctx().Err(), "never found a multi-tx workload block with SDM flashblock post_exec_tx")
		}
	}

	observed := observedByBlock[targetBlockNum]
	t.Require().NotEmpty(includedByBlock[targetBlockNum], "target block must include workload txs")

	block := getFlashblocksBlockWithTxs(t, sys.L2EL, targetBlockNum)
	t.Require().NotEmpty(block.Transactions, "final materialized block must have transactions")

	// Pin the canonical block to the rollup-boost materialization path: its hash must equal the
	// builder's diff.block_hash. A plain "block has a trailing PostExec tx" check would also pass
	// under the EL fallback (the sequencer EL runs SDM too), so it cannot distinguish a dropped
	// post_exec_tx during materialization from the real builder payload.
	t.Require().Equal(observed.blockHash, block.Hash,
		"canonical block must be the rollup-boost-materialized flashblock view (diff.block_hash), not an EL fallback block")

	postExecTx, postExecPos := sdm.FindPostExecTransaction(block)
	t.Require().NotNil(postExecTx, "final materialized block must contain one PostExec tx")
	t.Require().Equal(len(block.Transactions)-1, postExecPos, "PostExec tx must be last in the final block")
	secondPostExecTx, _ := sdm.FindPostExecTransaction(&sdm.RPCBlock{Transactions: block.Transactions[:postExecPos]})
	t.Require().Nil(secondPostExecTx, "final materialized block must contain exactly one PostExec tx")

	payload, err := sdm.DecodePayload(postExecTx.Input)
	t.Require().NoError(err, "final PostExec tx input must decode")
	t.Require().Equal(targetBlockNum, payload.BlockNumber, "final payload must target selected block")
	t.Require().NotEmpty(payload.GasRefundEntries, "final payload must contain SDM refund entries")
	t.Require().Equal(observed.payload, payload, "observed flashblock payload must match final payload")
	t.Require().Equal(observed.postExecTx[1:], []byte(postExecTx.Input),
		"final PostExec tx input must match latest observed flashblock post_exec_tx")

	validation, err := sdm.ValidatePostExecBlock(
		t.Ctx(),
		sys.L2EL.Escape().L2EthClient().RPC(),
		targetBlockNum,
		sdm.DefaultValidationOptions(),
	)
	t.Require().NoError(err, "final post-exec block must validate")
	replay := validation.Replay
	t.Require().NotNil(replay, "SDM validation must include replay result")

	t.Logger().Info("TestFlashblocksSDMMaterializesPostExecBlock passed",
		"block_num", targetBlockNum,
		"block_hash", block.Hash,
		"flashblock_index", observed.index,
		"workload_txs", len(includedByBlock[targetBlockNum]),
		"payload_entries", len(payload.GasRefundEntries),
		"post_exec_tx_index", postExecPos)
}

func decodeFlashblockPostExecTx(t devtest.T, fb *sources.Flashblock) (*sdm.PostExecPayload, []byte) {
	t.Helper()
	t.Require().NotNil(fb.Diff.PostExecTx, "flashblock must include post_exec_tx")
	raw := append([]byte(nil), (*fb.Diff.PostExecTx)...)
	t.Require().NotEmpty(raw, "post_exec_tx must not be empty")
	t.Require().Equal(byte(sdm.SDMTxType), raw[0], "post_exec_tx must use type 0x7d")
	payload, err := sdm.DecodePayload(raw[1:])
	t.Require().NoError(err, "post_exec_tx payload must decode")
	t.Require().Equal(uint64(fb.Metadata.BlockNumber), payload.BlockNumber,
		"post_exec_tx payload block number must match flashblock metadata")
	t.Require().NotEmpty(payload.GasRefundEntries, "SDM flashblock payload must contain refund entries")
	return payload, raw
}

func flashblockTransactionsContainPostExec(fb *sources.Flashblock) bool {
	for _, tx := range fb.Diff.Transactions {
		switch value := tx.(type) {
		case string:
			if strings.HasPrefix(strings.ToLower(value), "0x7d") {
				return true
			}
		case map[string]any:
			if ty, ok := value["type"].(string); ok && strings.EqualFold(ty, "0x7d") {
				return true
			}
		default:
			encoded, err := json.Marshal(value)
			if err == nil && strings.Contains(strings.ToLower(string(encoded)), "\"type\":\"0x7d\"") {
				return true
			}
		}
	}
	return false
}

func getFlashblocksBlockWithTxs(t devtest.T, l2EL *dsl.L2ELNode, blockNum uint64) *sdm.RPCBlock {
	t.Helper()
	block, err := sdm.GetBlockWithTxs(t.Ctx(), l2EL.Escape().L2EthClient().RPC(), blockNum)
	t.Require().NoError(err, "eth_getBlockByNumber RPC failed for block %d", blockNum)
	return block
}

func flashblocksSubmitTxWithoutWait(
	t devtest.T,
	alice *dsl.EOA,
	nonce uint64,
	opts ...txplan.Option,
) *txplan.PlannedTx {
	t.Helper()
	combined := append([]txplan.Option{alice.Plan(), txplan.WithNonce(nonce)}, opts...)
	ptx := txplan.NewPlannedTx(combined...)
	_, err := ptx.Submitted.Eval(t.Ctx())
	t.Require().NoError(err, "failed to submit tx with nonce %d", nonce)
	return ptx
}

func flashblocksDeployContract(t devtest.T, eoa *dsl.EOA, hexBytecode string) common.Address {
	t.Helper()
	tx := txplan.NewPlannedTx(eoa.Plan(), txplan.WithData(common.FromHex(hexBytecode)))
	res, err := tx.Included.Eval(t.Ctx())
	t.Require().NoError(err, "failed to deploy contract")
	return res.ContractAddress
}

func setFlashblocksSDMEnabled(t devtest.T, rpcClient client.RPC, enabled bool) {
	t.Helper()
	err := rpcClient.CallContext(t.Ctx(), nil, "admin_setSdmPostExecOptIn", enabled)
	t.Require().NoError(err, "admin_setSdmPostExecOptIn(%v) RPC failed", enabled)
}
