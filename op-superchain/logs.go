package superchain

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

type LogsProvider interface {
	// FetchLog returns a block's info and logs
	FetchLogs(context.Context, rpc.BlockNumberOrHash) (eth.BlockInfo, []types.Log, error)
}

type logsProvider struct {
	clnt client.RPC
}

func NewLogProvider(clnt client.RPC) LogsProvider {
	return &logsProvider{clnt}
}

func (p *logsProvider) FetchLogs(ctx context.Context, id rpc.BlockNumberOrHash) (eth.BlockInfo, []types.Log, error) {
	var logs []types.Log
	var header *types.Header

	blockId := id.String()
	filterArgs := map[string]interface{}{"fromBlock": blockId, "toBlock": blockId}
	batchElems := make([]rpc.BatchElem, 2)
	batchElems[0] = rpc.BatchElem{Method: "eth_getBlockByNumber", Args: []interface{}{blockId, false}, Result: &header}
	batchElems[1] = rpc.BatchElem{Method: "eth_getLogs", Args: []interface{}{filterArgs}, Result: &logs}
	if err := p.clnt.BatchCallContext(ctx, batchElems); err != nil {
		return nil, nil, fmt.Errorf("unable to request logs: %w", err)
	}
	if batchElems[0].Error != nil || batchElems[1].Error != nil {
		return nil, nil, fmt.Errorf("batch rpc failure: getBlockByNumber: %w, getLogs: %w", batchElems[0].Error, batchElems[1].Error)
	}

	if header == nil {
		return nil, nil, ethereum.NotFound
	}

	return eth.HeaderBlockInfo(header), logs, nil
}
