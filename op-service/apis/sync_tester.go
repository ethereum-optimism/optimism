package apis

import (
	"context"
	"encoding/json"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

type SyncTester interface {
	// Only expose sync namespace for encapsulation
	SyncAPI
	// ChainID for minimal sanity check
	ChainID(ctx context.Context) (eth.ChainID, error)
}

type SyncAPI interface {
	GetSession(ctx context.Context) (*eth.SyncTesterSession, error)
	DeleteSession(ctx context.Context) error
	ResetSession(ctx context.Context) error
	ListSessions(ctx context.Context) ([]string, error)
}

type EthAPI interface {
	GetBlockByNumber(ctx context.Context, number rpc.BlockNumber, fullTx bool) (json.RawMessage, error)
	GetBlockByHash(ctx context.Context, hash common.Hash, fullTx bool) (json.RawMessage, error)
	// GetBlockReceipts returns the raw eth_getBlockReceipts payload, relayed exactly as the
	// backing execution layer served it. Raw rather than geth's []*types.Receipt: the consensus
	// receipt type does not carry the `from`/`to` fields the execution-apis spec requires, so a
	// typed round-trip strips them and breaks clients that require them.
	GetBlockReceipts(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (json.RawMessage, error)
	ChainId(ctx context.Context) (hexutil.Big, error)
}

type EngineAPI interface {
	GetPayloadV1(ctx context.Context, payloadID eth.PayloadID) (*eth.ExecutionPayloadEnvelope, error)
	GetPayloadV2(ctx context.Context, payloadID eth.PayloadID) (*eth.ExecutionPayloadEnvelope, error)
	GetPayloadV3(ctx context.Context, payloadID eth.PayloadID) (*eth.ExecutionPayloadEnvelope, error)
	GetPayloadV4(ctx context.Context, payloadID eth.PayloadID) (*eth.ExecutionPayloadEnvelope, error)
	GetPayloadV5(ctx context.Context, payloadID eth.PayloadID) (*eth.ExecutionPayloadEnvelope, error)

	ForkchoiceUpdatedV1(ctx context.Context, state *eth.ForkchoiceState, attr *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error)
	ForkchoiceUpdatedV2(ctx context.Context, state *eth.ForkchoiceState, attr *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error)
	ForkchoiceUpdatedV3(ctx context.Context, state *eth.ForkchoiceState, attr *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error)

	NewPayloadV1(ctx context.Context, payload *eth.ExecutionPayload) (*eth.PayloadStatusV1, error)
	NewPayloadV2(ctx context.Context, payload *eth.ExecutionPayload) (*eth.PayloadStatusV1, error)
	NewPayloadV3(ctx context.Context, payload *eth.ExecutionPayload, versionedHashes []common.Hash, beaconRoot *common.Hash) (*eth.PayloadStatusV1, error)
	NewPayloadV4(ctx context.Context, payload *eth.ExecutionPayload, versionedHashes []common.Hash, beaconRoot *common.Hash, executionRequests []hexutil.Bytes) (*eth.PayloadStatusV1, error)

	ExchangeCapabilities(ctx context.Context, _ []string) []string
}
