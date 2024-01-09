package driver

import (
	"context"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

const DefaultMessagesBuffer = 10

// InteropMessageFetcher watches for remote changes in finalized
// state and buffers interop messages intended for this chain.
// Whenever possible, messages are flushed into the provided sink.
//
// TODOs:
//
//  1. Accept or compute starting heights for the remote.
//     - Starting height when interop relationship has already been
//     established as well as already bridged messages
//     - This needs to timed with the InteropHardFork marker as well
//     as the L1 heights of when the interop relationship was established
//
//  2. Establish what happens in errors state w.r.t to derivation (liveness)
//
// Questions:
//
//  1. Should finalized L1Origin be kept consistent between local/remote for consistency
//
//  2. Should the remote be processed on a per-block basis or block-range
type InteropMessageFetcher struct {
	log log.Logger

	localChain  common.Hash
	remoteChain common.Hash

	// TODO: we should probably utilize `source.EthClient` however, that fetches
	// all receipts per block and we're only interested in the Outbox and dont want
	// to unneccesarily cause cashing of uninteresting receipts.
	clnt client.RPC

	// New messages are buffered in the background and
	// flushed into the provided sink when possible
	buffer chan derive.InteropMessages
	sink   chan<- derive.InteropMessages

	finalized chan eth.L1BlockRef
}

func NewInteropMessageFetcher(log log.Logger, localChain, remoteChain common.Hash, clnt client.RPC, sink chan<- derive.InteropMessages) *InteropMessageFetcher {
	log = log.New("module", "interop_fetcher", "remote_chain_id", remoteChain)
	f := &InteropMessageFetcher{
		log:         log,
		localChain:  localChain,
		remoteChain: remoteChain,
		clnt:        clnt,
		finalized:   make(chan eth.L1BlockRef, 5),

		sink:   sink,
		buffer: make(chan derive.InteropMessages, DefaultMessagesBuffer),
	}

	go f.pollMessages()
	go f.flushMessages()
	return f
}

func (f *InteropMessageFetcher) HandleNewFinalizedBlock(finalized eth.L1BlockRef) error {
	select {
	case f.finalized <- finalized:
	default:
		// we dont want to block if the fetcher is already busy processing prior updates.
		// Since this channel is buffered, the fetcher should already be aware of additional
		// work that needs to be processed prior so we'll just skip over this update
		f.log.Warn("unable to handle finalized update", "hash", finalized.Hash, "number", finalized.Number)

		// TODO: maybe at some point we need to signal upwards a problem with an interop peer
	}

	return nil
}

func (f *InteropMessageFetcher) flushMessages() {
	for newMsgs := range f.buffer {
		f.log.Info("flushing messages", "size", len(newMsgs.Messages))
		f.sink <- newMsgs
	}
}

func (f *InteropMessageFetcher) pollMessages() {
	lastFinalized := eth.L1BlockRef{}

	// After each finalization event, there's a range of
	// blocks for which the outbox needs to be checked.
	for finalized := range f.finalized {
		var fromBlockNumber, toBlockNumber uint64 = 0, finalized.Number
		if (lastFinalized != eth.L1BlockRef{}) {
			fromBlockNumber = lastFinalized.Number + 1
		}

		// Query for new messages. (to be replaced with the specific interop rpc)

		var logs []types.Log
		filterArgs := map[string]interface{}{}
		filterArgs["addresses"] = []common.Address{predeploys.CrossL2OutboxAddr}
		filterArgs["topics"] = []common.Hash{derive.OutboxMessagePassedABIHash}
		filterArgs["fromBlock"] = hexutil.EncodeUint64(fromBlockNumber)
		filterArgs["toBlock"] = hexutil.EncodeUint64(toBlockNumber)

		err := f.clnt.CallContext(context.Background(), &logs, "eth_getLogs", filterArgs)
		if err != nil {
			f.log.Error("unable to fetch for logs", "err", err)
			continue
		}

		newMessages := make([]derive.InteropMessage, 0, len(logs))
		for _, log := range logs {
			msg, err := derive.UnmarshalInteropMessageLog(f.remoteChain, &log)
			if err != nil {
				f.log.Error("unable to process outbox messages", "err", err)
				panic(err) // fix
			}
			if msg.TargetChain != f.localChain {
				f.log.Debug("skipping message for a different chain", "nonce", msg.Nonce)
				continue
			}

			newMessages = append(newMessages, *msg)
		}

		if len(newMessages) == 0 {
			f.log.Debug("no messages for this chain in the outbox")
			continue
		}

		f.buffer <- derive.InteropMessages{
			Messages: newMessages,
			SourceInfo: derive.InteropMessageSourceInfo{
				RemoteChain:     f.remoteChain,
				FromBlockNumber: big.NewInt(int64(fromBlockNumber)),
				ToBlockNumber:   big.NewInt(int64(toBlockNumber)),
			},
		}

		lastFinalized = finalized
	}
}
