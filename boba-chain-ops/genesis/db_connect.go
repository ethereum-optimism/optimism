package genesis

import (
	"fmt"
	"time"

	"github.com/bobanetwork/v3-anchorage/boba-chain-ops/node"
	libcommon "github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon-lib/common/hexutility"
	"github.com/ledgerwatch/erigon/cmd/rpcdaemon/commands"
	"github.com/ledgerwatch/erigon/common/hexutil"
	"github.com/ledgerwatch/log/v3"
)

type ConnectEngine struct {
	l2PrivateClient   node.RPC
	l2PublicClient    node.RPC
	startingTimestamp int
	rpcTimeout        time.Duration
}

func NewConnectConfig(l2PrivateEndpoint, l2PublicEndpoint string, startingTimestamp int, jwtSecretPath string, rpcTimeout time.Duration, logger log.Logger) (*ConnectEngine, error) {
	jwtSecret, err := node.ReadJWTAuthSecret(jwtSecretPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read jwt secret: %w", err)
	}

	l2PrivateClient, err := node.NewRPC(l2PrivateEndpoint, rpcTimeout, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to dial l2PrivateEndpoint: %w", err)
	}
	err = l2PrivateClient.SetJWTAuth(jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to set jwt auth for l2PrivateEndpoint: %w", err)
	}
	log.Info("l2PrivateClient connected", "endpoint", l2PrivateEndpoint)

	l2PublicClient, err := node.NewRPC(l2PublicEndpoint, rpcTimeout, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to dial l2PublicEndpoint: %w", err)
	}
	log.Info("l2PublicClient connected", "endpoint", l2PublicEndpoint)

	return &ConnectEngine{
		l2PrivateClient:   l2PrivateClient,
		l2PublicClient:    l2PublicClient,
		startingTimestamp: startingTimestamp,
		rpcTimeout:        rpcTimeout,
	}, nil
}

func (c *ConnectEngine) Start() error {
	latestBlock, err := c.l2PublicClient.GetLatestBlock()
	if err != nil {
		return err
	}

	difficulty := latestBlock.Difficulty.ToInt()
	if difficulty.Cmp(libcommon.Big2) != 0 {
		return fmt.Errorf("difficulty is not 2, got %s", difficulty.String())
	}

	transactions := make([]hexutility.Bytes, 0)

	// Step 1: Get new payloadID
	// engine_forkchoiceUpdatedV1 -> Get payloadID
	fc := &commands.ForkChoiceState{
		HeadHash:           latestBlock.Hash,
		SafeBlockHash:      latestBlock.Hash,
		FinalizedBlockHash: latestBlock.Hash,
	}

	attributes := &commands.PayloadAttributes{
		Timestamp:             hexutil.Uint64(c.startingTimestamp),
		PrevRandao:            [32]byte{},
		SuggestedFeeRecipient: libcommon.HexToAddress("0x4200000000000000000000000000000000000011"),
		Transactions:          transactions,
		NoTxPool:              true,
	}

	fcUpdateRes, err := c.l2PrivateClient.ForkchoiceUpdateV1(fc, attributes)
	if err != nil {
		return err
	}
	log.Info("Got forkchoice update", "status", fcUpdateRes.PayloadStatus.Status, "payloadID", fcUpdateRes.PayloadID)

	// Step 2: Get next block information
	// engine_getPayloadV1 -> Get executionPayload
	executionPayload, err := c.l2PrivateClient.GetPayloadV1(fcUpdateRes.PayloadID)
	if err != nil {
		return err
	}
	if len(executionPayload.Transactions) != 0 {
		log.Warn("Pending transaction length is not 0", "length", len(executionPayload.Transactions))
		return fmt.Errorf("pending transaction length is not 0")
	}
	if uint64(executionPayload.Timestamp) != uint64(c.startingTimestamp) {
		log.Warn("Timestamp is not expected", "timestamp", executionPayload.Timestamp)
		return fmt.Errorf("timestamp is not expected")
	}

	// Step 3: Process new block
	// engine_newPayloadV1 -> Execute payload
	payloadStatus, err := c.l2PrivateClient.NewPayloadV1(executionPayload)
	if err != nil {
		return err
	}
	if payloadStatus.Status != "VALID" {
		log.Warn("Payload is invalid", "status", payloadStatus.Status)
		return fmt.Errorf("payload is invalid")
	}
	if *payloadStatus.LatestValidHash != executionPayload.BlockHash {
		log.Warn("Latest valid hash is not correct", "pending", executionPayload.BlockHash, "latest", payloadStatus.LatestValidHash)
		return fmt.Errorf("latest valid hash is not correct")
	}

	// Step 4: Submit block
	// engine_executePayloadV1 -> Submit block
	updatedFc := &commands.ForkChoiceState{
		HeadHash:           executionPayload.BlockHash,
		SafeBlockHash:      executionPayload.BlockHash,
		FinalizedBlockHash: executionPayload.BlockHash,
	}
	fcFinalRes, err := c.l2PrivateClient.ForkchoiceUpdateV1(updatedFc, nil)
	if err != nil {
		return err
	}
	if fcFinalRes.PayloadStatus.Status != "VALID" {
		log.Warn("Payload is invalid", "status", fcFinalRes.PayloadStatus.Status)
		return fmt.Errorf("payload is invalid")
	}

	log.Info("Connection block mined", "blockNumber", uint64(executionPayload.BlockNumber))

	return nil
}
