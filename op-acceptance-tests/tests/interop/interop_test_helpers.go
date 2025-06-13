package interop

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/constants"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func RandomTopicAndData(rng *rand.Rand, cnt, len int) ([][32]byte, []byte) {
	topics := [][32]byte{}
	for range cnt {
		var topic [32]byte
		copy(topic[:], testutils.RandomData(rng, 32))
		topics = append(topics, topic)
	}
	data := testutils.RandomData(rng, len)
	return topics, data
}

func RandomInitTrigger(rng *rand.Rand, eventLoggerAddress common.Address, cnt, len int) *txintent.InitTrigger {
	if cnt >= 5 {
		panic(fmt.Sprintf("log holds at most 4 topics, got %d", cnt))
	}
	topics, data := RandomTopicAndData(rng, cnt, len)
	return &txintent.InitTrigger{
		Emitter:    eventLoggerAddress,
		Topics:     topics,
		OpaqueData: data,
	}
}

// ExecTriggerFromInitTrigger returns corresponding execTrigger with necessary information
func ExecTriggerFromInitTrigger(init *txintent.InitTrigger, logIndex uint, targetNum, targetTime uint64, chainID eth.ChainID) (*txintent.ExecTrigger, error) {
	topics := []common.Hash{}
	for _, topic := range init.Topics {
		topics = append(topics, topic)
	}
	log := &types.Log{Address: init.Emitter, Topics: topics,
		Data: init.OpaqueData, BlockNumber: targetNum, Index: logIndex}
	logs := make([]*types.Log, logIndex+1)
	for i := range logs {
		// dummy logs to fit in log index
		logs[i] = &types.Log{}
	}
	logs[logIndex] = log
	rec := &types.Receipt{Logs: logs}
	includedIn := eth.BlockRef{Time: targetTime}
	output := &txintent.InteropOutput{}
	err := output.FromReceipt(context.TODO(), rec, includedIn, chainID)
	if err != nil {
		return nil, err
	}
	if x := len(output.Entries); x <= int(logIndex) {
		return nil, fmt.Errorf("invalid index: %d, only have %d events", logIndex, x)
	}
	return &txintent.ExecTrigger{Executor: constants.CrossL2Inbox, Msg: output.Entries[logIndex]}, nil
}
