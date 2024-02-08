package genesis

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/bobanetwork/v3-anchorage/boba-chain-ops/node"
	libcommon "github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon-lib/common/hexutil"
	"github.com/ledgerwatch/erigon-lib/common/hexutility"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/ledgerwatch/erigon/turbo/engineapi/engine_types"
	"github.com/ledgerwatch/log/v3"
)

type BuilderEngine struct {
	ctx                 context.Context
	stop                chan struct{}
	l2PrivateClient     node.RPC
	l2PublicClient      node.RPC
	l2LegacyClient      node.RPC
	rpcTimeout          time.Duration
	pollingInterval     time.Duration
	hardforkBlockNumber int64
}

func NewEngineConfig(l2PrivateEndpoint, l2PublicEndpoint, l2LegacyEndpoint, jwtSecretPath string, rpcTimeout, pollingInterval time.Duration, hardforkBlockNumber int64, logger log.Logger) (*BuilderEngine, error) {
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

	l2LegacyClient, err := node.NewRPC(l2LegacyEndpoint, rpcTimeout, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to dial l2LegacyEndpoint: %w", err)
	}
	log.Info("l2LegacyClient connected", "endpoint", l2LegacyEndpoint)

	return &BuilderEngine{
		ctx:                 context.Background(),
		stop:                make(chan struct{}),
		l2PrivateClient:     l2PrivateClient,
		l2PublicClient:      l2PublicClient,
		l2LegacyClient:      l2LegacyClient,
		rpcTimeout:          rpcTimeout,
		pollingInterval:     pollingInterval,
		hardforkBlockNumber: hardforkBlockNumber,
	}, nil
}

func (b *BuilderEngine) Start() {
	go b.BlockGenerationLoop()
	// b.BlockGenerationLoop()
}

func (b *BuilderEngine) Stop() {
	close(b.stop)
}

func (b *BuilderEngine) Wait() {
	<-b.stop
}

func (b *BuilderEngine) BlockGenerationLoop() {
	timer := time.NewTicker(b.pollingInterval)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			log.Trace("polling", "time", time.Now())
			if err := b.RegenerateBlock(); err != nil {
				log.Error("cannot generate new block", "message", err)
			}
		case <-b.ctx.Done():
			b.Stop()
		}
	}
}

func (b *BuilderEngine) RegenerateBlock() error {
	latestBlock, err := b.l2PublicClient.GetLatestBlock()
	if err != nil {
		return err
	}
	nextBlockNumber := uint64(latestBlock.Number) + 1

	if b.hardforkBlockNumber != 0 && nextBlockNumber > uint64(b.hardforkBlockNumber) {
		log.Info("hardfork block number reached, exiting")
		b.Stop()
		return nil
	}

	legacyBlock, err := b.l2LegacyClient.GetBlockByNumber(big.NewInt(int64(nextBlockNumber)))
	if err != nil {
		return err
	}

	if legacyBlock == nil {
		log.Info("No new block to be mined", "blockNumber", nextBlockNumber)
		return nil
	}

	gasLimit := legacyBlock.GasLimit
	txHash := legacyBlock.Transactions[0]

	legacyTransaction, err := b.l2LegacyClient.GetTransactionByHash(txHash)
	if err != nil {
		return err
	}

	// Verify that legacy transaction has the same txHash
	if legacyTransaction.Hash() != *txHash {
		return fmt.Errorf("transaction hashs from legacy endpoint do not match")
	}

	// Build binary transaction input
	var buf bytes.Buffer
	err = legacyTransaction.MarshalBinary(&buf)
	if err != nil {
		return err
	}
	transactions := make([]hexutility.Bytes, 1)
	transactions[0] = hexutility.Bytes(buf.Bytes())

	// Step 1: Get new payloadID
	// engine_forkchoiceUpdatedV1 -> Get payloadID
	fc := &engine_types.ForkChoiceState{
		HeadHash:           latestBlock.Hash,
		SafeBlockHash:      latestBlock.Hash,
		FinalizedBlockHash: latestBlock.Hash,
	}
	attributes := &engine_types.PayloadAttributes{
		Timestamp:             hexutil.Uint64(legacyBlock.Time),
		PrevRandao:            [32]byte{},
		SuggestedFeeRecipient: libcommon.HexToAddress("0x4200000000000000000000000000000000000011"),
		Transactions:          transactions,
		NoTxPool:              true,
		GasLimit:              &gasLimit,
	}

	fcUpdateRes, err := b.l2PrivateClient.ForkchoiceUpdateV1(fc, attributes)
	if err != nil {
		return err
	}
	log.Info("Got forkchoice update", "status", fcUpdateRes.PayloadStatus.Status, "payloadID", fcUpdateRes.PayloadID)

	// Step 2: Get next block information
	// engine_getPayloadV1 -> Get executionPayload
	executionPayload, err := b.l2PrivateClient.GetPayloadV1(fcUpdateRes.PayloadID)
	if err != nil {
		return err
	}
	if len(executionPayload.Transactions) != 1 {
		log.Warn("Pending transaction length is not 1", "length", len(executionPayload.Transactions))
		return fmt.Errorf("pending transaction length is not 1")
	}
	tx, err := types.UnmarshalTransactionFromBinary(executionPayload.Transactions[0])
	if err != nil {
		return fmt.Errorf("failed to unmarshal transaction: %w", err)
	}
	if tx.Hash() != *txHash {
		log.Warn("Pending transaction hash is not correct", "pending", tx.Hash(), "latest", txHash)
		return fmt.Errorf("pending transaction hash is not correct")
	}
	if executionPayload.BlockHash != legacyBlock.Hash {
		log.Warn("Pending block hash is not correct", "pending", executionPayload.BlockHash, "latest", legacyBlock.Hash)
		return fmt.Errorf("pending block hash is not correct")
	}
	log.Info("Got execution payload", "blockNumber", uint64(executionPayload.BlockNumber))

	// Step 3: Process new block
	// engine_newPayloadV1 -> Execute payload
	payloadStatus, err := b.l2PrivateClient.NewPayloadV1(executionPayload)
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
	updatedFc := &engine_types.ForkChoiceState{
		HeadHash:           executionPayload.BlockHash,
		SafeBlockHash:      executionPayload.BlockHash,
		FinalizedBlockHash: executionPayload.BlockHash,
	}
	fcFinalRes, err := b.l2PrivateClient.ForkchoiceUpdateV1(updatedFc, nil)
	if err != nil {
		return err
	}
	if fcFinalRes.PayloadStatus.Status != "VALID" {
		log.Warn("Payload is invalid", "status", fcFinalRes.PayloadStatus.Status)
		return fmt.Errorf("payload is invalid")
	}

	// Verify that the block is submitted
	latestBlock, err = b.l2PublicClient.GetLatestBlock()
	if err != nil {
		log.Warn("Failed to get latest block", "error", err)

	}
	// database is corrupted if the following checks fail
	if latestBlock.Root != legacyBlock.Root {
		log.Warn("Block root is not correct", "pending", legacyBlock.Root, "latest", latestBlock.Root)
		b.Stop()
		return fmt.Errorf("Block root is not correct")
	}
	if latestBlock.ReceiptHash != legacyBlock.ReceiptHash {
		log.Warn("Receipt hash is not correct", "pending", legacyBlock.ReceiptHash, "latest", latestBlock.ReceiptHash)
		b.Stop()
		return fmt.Errorf("Receipt hash is not correct")
	}

	log.Info("Block mined", "blockNumber", uint64(executionPayload.BlockNumber))
	log.Info("Waiting for next block to be mined", "blockNumber", uint64(executionPayload.BlockNumber+1))

	return nil
}
