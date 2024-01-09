package driver

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type interopMessages struct {
	derive.InteropMessages
	read chan<- interface{}
}

// InteropMessageQueue returns the next set of interop
// message to include per chain.
//
// TODOs:
//  1. Establish a priority queue for incoming messages.
//     All messages per chains cannot be consumed at once
//     and should instead be ordered & limited by gas.
type InteropMessageQueue struct {
	log log.Logger
	mu  sync.Mutex

	newMsgs     []interopMessages
	msgFetchers map[uint64]*InteropMessageFetcher
}

func NewInteropMessageQueue(log log.Logger, localChainId *big.Int, remoteChains map[uint64]client.RPC) *InteropMessageQueue {
	q := InteropMessageQueue{
		log: log.New("module", "interop_msg_queue"),

		newMsgs:     nil,
		msgFetchers: map[uint64]*InteropMessageFetcher{},
	}

	localChain := common.BigToHash(localChainId)

	for chainId, rpc := range remoteChains {
		q.log.Info("initializing remote", "chain_id", chainId)
		remoteChain := common.BigToHash(big.NewInt(int64(chainId)))

		msgsSink := make(chan derive.InteropMessages)
		go q.onNewMessages(msgsSink)
		q.msgFetchers[chainId] = NewInteropMessageFetcher(log, localChain, remoteChain, rpc, msgsSink)
	}

	return &q
}

func (q *InteropMessageQueue) HandleNewFinalizedBlock(chain uint64, finalized eth.L1BlockRef) error {
	f, ok := q.msgFetchers[chain]
	if !ok {
		return fmt.Errorf("interop rpc not found for peer chain: %d", chain)
	}

	return f.HandleNewFinalizedBlock(finalized)
}

// NewMessages will return the next set of available messages per chain
func (q *InteropMessageQueue) NewMessages() []derive.InteropMessages {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.newMsgs) == 0 {
		return nil
	}

	result := make([]derive.InteropMessages, len(q.newMsgs))
	for i, msgs := range q.newMsgs {
		result[i] = msgs.InteropMessages
		msgs.read <- struct{}{}
	}

	q.newMsgs = nil
	return result
}

func (q *InteropMessageQueue) onNewMessages(msgs <-chan derive.InteropMessages) {
	for newMsgs := range msgs {
		read := make(chan interface{}, 1)
		item := interopMessages{
			InteropMessages: newMsgs,
			read:            read,
		}

		q.mu.Lock()
		q.newMsgs = append(q.newMsgs, item)
		q.mu.Unlock()

		// only move on once the item has been consumed
		<-read
	}
}
