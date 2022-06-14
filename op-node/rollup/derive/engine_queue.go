package derive

import (
	"context"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

type Engine interface {
	GetPayload(ctx context.Context, payloadId eth.PayloadID) (*eth.ExecutionPayload, error)
	ForkchoiceUpdate(ctx context.Context, state *eth.ForkchoiceState, attr *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error)
	NewPayload(ctx context.Context, payload *eth.ExecutionPayload) error
	PayloadByHash(context.Context, common.Hash) (*eth.ExecutionPayload, error)
	PayloadByNumber(context.Context, *big.Int) (*eth.ExecutionPayload, error)
}

// EngineQueue queues up payload attributes to consolidate or process with the provided Engine
type EngineQueue struct {
	attributes []*eth.PayloadAttributes
	engine     Engine
}

func (eq *EngineQueue) AddAttributes(attributes *eth.PayloadAttributes) {
	eq.attributes = append(eq.attributes)
}

func (eq *EngineQueue) Step() error {
	// TODO: implement below spec
	// 1. return io.EOF if there are no payload attributes buffered
	// 2. peek into first payload attributes
	// 3. check if the engine has synced past these attributes
	//     3.1 if yes, compare the engine attributes
	//           -> see derive.VerifySafeBlock
	//        3.1.1 mark the attributes as safe (forkchoice update, without changing unsafe head) (with timeout)
	//             or log error and return nil if this fails
	//        3.1.2 pop the attributes from buffer
	//        3.1.3 log what we just consolidated
	//        3.1.4 pop from the buffer
	//        3.1.5 return nil
	//     3.2 if not, re-apply the engine attributes
	//           -> see derive.InsertHeadBlock
	//        3.2.1 forkchoice update to make the safe block the head (with timeout)
	//             or log error and return nil if this fails
	//        3.2.2 try to apply the payload attributes to the engine (with timeout)
	//             if RPC error: log it, and return nil
	//        3.2.3 if invalid payload: log it (err level), pop it, and then return nil
	//        3.2.4 if valid payload: log it (info level), pop it, and then return nil
	return nil
}
