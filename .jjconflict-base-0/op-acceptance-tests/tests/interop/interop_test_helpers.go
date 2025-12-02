package interop

import (
	"fmt"
	"math/rand"

	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum/go-ethereum/common"
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
