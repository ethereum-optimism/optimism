package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type PayloadAttributesV2 struct {
	Timestamp             uint64              `json:"timestamp"`
	Random                common.Hash         `json:"prevRandao"`
	SuggestedFeeRecipient common.Address      `json:"suggestedFeeRecipient"`
	Withdrawals           []*types.Withdrawal `json:"withdrawals"`
}

func (p PayloadAttributesV2) MarshalJSON() ([]byte, error) {
	type PayloadAttributes struct {
		Timestamp             hexutil.Uint64      `json:"timestamp"             gencodec:"required"`
		Random                common.Hash         `json:"prevRandao"            gencodec:"required"`
		SuggestedFeeRecipient common.Address      `json:"suggestedFeeRecipient" gencodec:"required"`
		Withdrawals           []*types.Withdrawal `json:"withdrawals"`
	}
	var enc PayloadAttributes
	enc.Timestamp = hexutil.Uint64(p.Timestamp)
	enc.Random = p.Random
	enc.SuggestedFeeRecipient = p.SuggestedFeeRecipient
	enc.Withdrawals = make([]*types.Withdrawal, 0)
	return json.Marshal(&enc)
}

func DialClient(ctx context.Context, endpoint string, jwtSecret [32]byte) (client.RPC, error) {
	auth := node.NewJWTAuth(jwtSecret)

	rpcClient, err := rpc.DialOptions(ctx, endpoint, rpc.WithHTTPAuth(auth))
	if err != nil {
		return nil, fmt.Errorf("failed to dial engine endpoint: %w", err)
	}
	return client.NewBaseRPCClient(rpcClient), nil
}

type RPCBlock struct {
	types.Header
	Transactions []*types.Transaction `json:"transactions"`
}

func getBlock(ctx context.Context, client client.RPC, method string, tag string) (*types.Block, error) {
	var bl *RPCBlock
	err := client.CallContext(ctx, &bl, method, tag, true)
	if err != nil {
		return nil, err
	}
	return types.NewBlockWithHeader(&bl.Header).WithBody(bl.Transactions, nil), nil
}

func getHeader(ctx context.Context, client client.RPC, method string, tag string) (*types.Header, error) {
	var header *types.Header
	err := client.CallContext(ctx, &header, method, tag, false)
	if err != nil {
		return nil, err
	}
	return header, nil
}

func headSafeFinalized(ctx context.Context, client client.RPC) (head *types.Block, safe, finalized *types.Header, err error) {
	head, err = getBlock(ctx, client, "eth_getBlockByNumber", "latest")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get block: %w", err)
	}
	safe, err = getHeader(ctx, client, "eth_getBlockByNumber", "safe")
	if err != nil {
		return head, nil, nil, fmt.Errorf("failed to get safe block: %w", err)
	}
	finalized, err = getHeader(ctx, client, "eth_getBlockByNumber", "finalized")
	if err != nil {
		return head, safe, nil, fmt.Errorf("failed to get finalized block: %w", err)
	}
	return head, safe, finalized, nil
}

func insertBlock(ctx context.Context, client client.RPC, payload *engine.ExecutableData) error {
	var payloadResult *engine.PayloadStatusV1
	if err := client.CallContext(ctx, &payloadResult, "engine_newPayloadV2", payload); err != nil {
		return fmt.Errorf("failed to insert block %d: %w", payload.Number, err)
	}
	if payloadResult.Status != string(eth.ExecutionValid) {
		return fmt.Errorf("block insertion was not valid: %v", payloadResult.ValidationError)
	}
	return nil
}

func updateForkchoice(ctx context.Context, client client.RPC, head, safe, finalized common.Hash) error {
	var post engine.ForkChoiceResponse
	if err := client.CallContext(ctx, &post, "engine_forkchoiceUpdatedV2",
		engine.ForkchoiceStateV1{
			HeadBlockHash:      head,
			SafeBlockHash:      safe,
			FinalizedBlockHash: finalized,
		}, nil); err != nil {
		return fmt.Errorf("failed to set forkchoice with new block %s: %w", head, err)
	}
	if post.PayloadStatus.Status != string(eth.ExecutionValid) {
		return fmt.Errorf("post-block forkchoice update was not valid: %v", post.PayloadStatus.ValidationError)
	}
	return nil
}

type BlockBuildingSettings struct {
	BlockTime uint64
	// skip a block; timestamps will still increase in multiples of BlockTime like L1, but there may be gaps.
	AllowGaps    bool
	Random       common.Hash
	FeeRecipient common.Address
	BuildTime    time.Duration
}

func BuildBlock(ctx context.Context, client client.RPC, status *StatusData, settings *BlockBuildingSettings) (*engine.ExecutableData, error) {
	timestamp := status.Head.Time + settings.BlockTime
	if settings.AllowGaps {
		now := uint64(time.Now().Unix())
		if now > timestamp {
			timestamp = now - ((now - timestamp) % settings.BlockTime)
		}
	}
	var pre engine.ForkChoiceResponse
	if err := client.CallContext(ctx, &pre, "engine_forkchoiceUpdatedV2",
		engine.ForkchoiceStateV1{
			HeadBlockHash:      status.Head.Hash,
			SafeBlockHash:      status.Safe.Hash,
			FinalizedBlockHash: status.Finalized.Hash,
		}, PayloadAttributesV2{
			Timestamp:             timestamp,
			Random:                settings.Random,
			SuggestedFeeRecipient: settings.FeeRecipient,
		}); err != nil {
		return nil, fmt.Errorf("failed to set forkchoice when building new block: %w", err)
	}
	if pre.PayloadStatus.Status != string(eth.ExecutionValid) {
		return nil, fmt.Errorf("pre-block forkchoice update was not valid: %v", pre.PayloadStatus.ValidationError)
	}

	// wait some time for the block to get built
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(settings.BuildTime):
	}

	var payload *engine.ExecutionPayloadEnvelope
	if err := client.CallContext(ctx, &payload, "engine_getPayloadV2", pre.PayloadID); err != nil {
		return nil, fmt.Errorf("failed to get payload %v, %d time after instructing engine to build it: %w", pre.PayloadID, settings.BuildTime, err)
	}

	if err := insertBlock(ctx, client, payload.ExecutionPayload); err != nil {
		return nil, err
	}
	if err := updateForkchoice(ctx, client, payload.ExecutionPayload.BlockHash, status.Safe.Hash, status.Finalized.Hash); err != nil {
		return nil, err
	}

	return payload.ExecutionPayload, nil
}

func Auto(ctx context.Context, metrics Metricer, client client.RPC, log log.Logger, shutdown <-chan struct{}, settings *BlockBuildingSettings) error {
	ticker := time.NewTicker(time.Millisecond * 100)
	defer ticker.Stop()

	var lastPayload *engine.ExecutableData
	var buildErr error
	for {
		select {
		case <-shutdown:
			log.Info("shutting down")
			return nil
		case <-ctx.Done():
			log.Info("context closed", "err", ctx.Err())
			return ctx.Err()
		case now := <-ticker.C:
			blockTime := time.Duration(settings.BlockTime) * time.Second
			lastTime := uint64(0)
			if lastPayload != nil {
				lastTime = lastPayload.Timestamp
			}
			buildTriggerTime := time.Unix(int64(lastTime), 0).Add(blockTime - settings.BuildTime)

			if lastPayload == nil || now.After(buildTriggerTime) {
				buildTime := settings.BuildTime
				// don't waste time on trying to include txs if we are lagging behind at least a block,
				// but don't go ham if we are failing to build blocks already.
				if buildErr == nil && now.After(buildTriggerTime.Add(blockTime)) {
					buildTime = 10 * time.Millisecond
				}
				buildErr = nil
				status, err := Status(ctx, client)
				if err != nil {
					log.Error("failed to get pre-block engine status", "err", err)
					metrics.RecordBlockFail()
					buildErr = err
					continue
				}
				log.Info("status", "head", status.Head, "safe", status.Safe, "finalized", status.Finalized,
					"head_time", status.Head.Time, "txs", status.Txs, "gas", status.Gas, "basefee", status.Gas)

				// On a mocked "beacon epoch transition", update finalization and justification checkpoints.
				// There are no gap slots, so we just go back 32 blocks.
				if status.Head.Number%32 == 0 {
					if status.Safe.Number+32 <= status.Head.Number {
						safe, err := getHeader(ctx, client, "eth_getBlockByNumber", hexutil.Uint64(status.Head.Number-32).String())
						if err != nil {
							buildErr = err
							log.Error("failed to find block for new safe block progress", "err", err)
							continue
						}
						status.Safe = eth.L1BlockRef{Hash: safe.Hash(), Number: safe.Number.Uint64(), Time: safe.Time, ParentHash: safe.ParentHash}
					}
					if status.Finalized.Number+32 <= status.Safe.Number {
						finalized, err := getHeader(ctx, client, "eth_getBlockByNumber", hexutil.Uint64(status.Safe.Number-32).String())
						if err != nil {
							buildErr = err
							log.Error("failed to find block for new finalized block progress", "err", err)
							continue
						}
						status.Finalized = eth.L1BlockRef{Hash: finalized.Hash(), Number: finalized.Number.Uint64(), Time: finalized.Time, ParentHash: finalized.ParentHash}
					}
				}

				payload, err := BuildBlock(ctx, client, status, &BlockBuildingSettings{
					BlockTime:    settings.BlockTime,
					AllowGaps:    settings.AllowGaps,
					Random:       settings.Random,
					FeeRecipient: settings.FeeRecipient,
					BuildTime:    buildTime,
				})
				if err != nil {
					buildErr = err
					log.Error("failed to produce block", "err", err)
					metrics.RecordBlockFail()
				} else {
					lastPayload = payload
					log.Info("created block", "hash", payload.BlockHash, "number", payload.Number,
						"timestamp", payload.Timestamp, "txs", len(payload.Transactions),
						"gas", payload.GasUsed, "basefee", payload.BaseFeePerGas)
					basefee, _ := new(big.Float).SetInt(payload.BaseFeePerGas).Float64()
					metrics.RecordBlockStats(payload.BlockHash, payload.Number, payload.Timestamp, uint64(len(payload.Transactions)), payload.GasUsed, basefee)
				}
			}
		}
	}
}

type StatusData struct {
	Head      eth.L1BlockRef `json:"head"`
	Safe      eth.L1BlockRef `json:"safe"`
	Finalized eth.L1BlockRef `json:"finalized"`
	Txs       uint64         `json:"txs"`
	Gas       uint64         `json:"gas"`
	StateRoot common.Hash    `json:"stateRoot"`
	BaseFee   *big.Int       `json:"baseFee"`
}

func Status(ctx context.Context, client client.RPC) (*StatusData, error) {
	head, safe, finalized, err := headSafeFinalized(ctx, client)
	if err != nil {
		return nil, err
	}
	return &StatusData{
		Head:      eth.L1BlockRef{Hash: head.Hash(), Number: head.NumberU64(), Time: head.Time(), ParentHash: head.ParentHash()},
		Safe:      eth.L1BlockRef{Hash: safe.Hash(), Number: safe.Number.Uint64(), Time: safe.Time, ParentHash: safe.ParentHash},
		Finalized: eth.L1BlockRef{Hash: finalized.Hash(), Number: finalized.Number.Uint64(), Time: finalized.Time, ParentHash: finalized.ParentHash},
		Txs:       uint64(len(head.Transactions())),
		Gas:       head.GasUsed(),
		StateRoot: head.Root(),
		BaseFee:   head.BaseFee(),
	}, nil
}

// Copy takes the forkchoice state of copyFrom, and applies it to copyTo, and inserts the head-block.
// The destination engine should then start syncing to this new chain if it has peers to do so.
func Copy(ctx context.Context, copyFrom client.RPC, copyTo client.RPC) error {
	copyHead, copySafe, copyFinalized, err := headSafeFinalized(ctx, copyFrom)
	if err != nil {
		return err
	}
	payloadEnv := engine.BlockToExecutableData(copyHead, nil, nil)
	if err := updateForkchoice(ctx, copyTo, copyHead.ParentHash(), copySafe.Hash(), copyFinalized.Hash()); err != nil {
		return err
	}
	payload := payloadEnv.ExecutionPayload
	if err := insertBlock(ctx, copyTo, payload); err != nil {
		return err
	}
	if err := updateForkchoice(ctx, copyTo, payload.BlockHash, copySafe.Hash(), copyFinalized.Hash()); err != nil {
		return err
	}
	return nil
}

func SetForkchoice(ctx context.Context, client client.RPC, finalizedNum, safeNum, unsafeNum uint64) error {
	if unsafeNum < safeNum {
		return fmt.Errorf("cannot set unsafe (%d) < safe (%d)", unsafeNum, safeNum)
	}
	if safeNum < finalizedNum {
		return fmt.Errorf("cannot set safe (%d) < finalized (%d)", safeNum, finalizedNum)
	}
	head, err := getHeader(ctx, client, "eth_getBlockByNumber", "latest")
	if err != nil {
		return fmt.Errorf("failed to get latest block: %w", err)
	}
	if unsafeNum > head.Number.Uint64() {
		return fmt.Errorf("cannot set unsafe (%d) > latest (%d)", unsafeNum, head.Number.Uint64())
	}
	finalizedHeader, err := getHeader(ctx, client, "eth_getBlockByNumber", hexutil.Uint64(finalizedNum).String())
	if err != nil {
		return fmt.Errorf("failed to get block %d to mark finalized: %w", finalizedNum, err)
	}
	safeHeader, err := getHeader(ctx, client, "eth_getBlockByNumber", hexutil.Uint64(safeNum).String())
	if err != nil {
		return fmt.Errorf("failed to get block %d to mark safe: %w", safeNum, err)
	}
	if err := updateForkchoice(ctx, client, head.Hash(), safeHeader.Hash(), finalizedHeader.Hash()); err != nil {
		return fmt.Errorf("failed to update forkchoice: %w", err)
	}
	return nil
}

func RawJSONInteraction(ctx context.Context, client client.RPC, method string, args []string, input io.Reader, output io.Writer) error {
	var params []any
	if input != nil {
		r := json.NewDecoder(input)
		for {
			var param json.RawMessage
			if err := r.Decode(&param); err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				return fmt.Errorf("unexpected error while reading json params: %w", err)
			}
			params = append(params, param)
		}
	} else {
		for _, arg := range args {
			// add quotes to unquoted strings, but not to other json data
			if isUnquotedJsonString(arg) {
				arg = fmt.Sprintf("%q", arg)
			}
			params = append(params, json.RawMessage(arg))
		}
	}
	var result json.RawMessage
	if err := client.CallContext(ctx, &result, method, params...); err != nil {
		return fmt.Errorf("failed RPC call: %w", err)
	}
	if _, err := output.Write(result); err != nil {
		return fmt.Errorf("failed to write RPC output: %w", err)
	}
	return nil
}

func isUnquotedJsonString(v string) bool {
	v = strings.TrimSpace(v)
	// check if empty string (must get quotes)
	if len(v) == 0 {
		return true
	}
	// check if special value
	switch v {
	case "null", "true", "false":
		return false
	}
	// check if it looks like a json structure
	switch v[0] {
	case '[', '{', '"':
		return false
	}
	// check if a number
	var n json.Number
	if err := json.Unmarshal([]byte(v), &n); err == nil {
		return false
	}
	return true
}
