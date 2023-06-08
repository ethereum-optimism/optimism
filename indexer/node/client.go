package node

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

const (
	// defaultDialTimeout is default duration the processor will wait on
	// startup to make a connection to the backend
	defaultDialTimeout = 5 * time.Second

	// defaultRequestTimeout is the default duration the processor will
	// wait for a request to be fulfilled
	defaultRequestTimeout = 10 * time.Second
)

type EthClient interface {
	FinalizedBlockHeight() (*big.Int, error)

	BlockHeadersByRange(*big.Int, *big.Int) ([]*types.Header, error)
	BlockHeaderByHash(common.Hash) (*types.Header, error)

	RawRpcClient() *rpc.Client
}

type client struct {
	rpcClient *rpc.Client
}

func NewEthClient(rpcUrl string) (EthClient, error) {
	ctxwt, cancel := context.WithTimeout(context.Background(), defaultDialTimeout)
	defer cancel()

	rpcClient, err := rpc.DialContext(ctxwt, rpcUrl)
	if err != nil {
		return nil, err
	}

	client := &client{rpcClient: rpcClient}
	return client, nil
}

func (c *client) RawRpcClient() *rpc.Client {
	return c.rpcClient
}

// FinalizedBlockHeight retrieves the latest block height in a finalized state
func (c *client) FinalizedBlockHeight() (*big.Int, error) {
	ctxwt, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	// Local devnet is having issues with the "finalized" block tag. Switch to "latest"
	// to iterate faster locally but this needs to be updated
	header := new(types.Header)
	err := c.rpcClient.CallContext(ctxwt, header, "eth_getBlockByNumber", "latest", false)
	if err != nil {
		return nil, err
	}

	return header.Number, nil
}

// BlockHeaderByHash retrieves the block header attributed to the supplied hash
func (c *client) BlockHeaderByHash(hash common.Hash) (*types.Header, error) {
	ctxwt, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	header, err := ethclient.NewClient(c.rpcClient).HeaderByHash(ctxwt, hash)
	if err != nil {
		return nil, err
	}

	// sanity check on the data returned
	if header.Hash() != hash {
		return nil, errors.New("header mismatch")
	}

	return header, nil
}

// BlockHeadersByRange will retrieve block headers within the specified range -- includsive. No restrictions
// are placed on the range such as blocks in the "latest", "safe" or "finalized" states. If the specified
// range is too large, `endHeight > latest`, the resulting list is truncated to the available headers
func (c *client) BlockHeadersByRange(startHeight, endHeight *big.Int) ([]*types.Header, error) {
	count := new(big.Int).Sub(endHeight, startHeight).Uint64()
	batchElems := make([]rpc.BatchElem, count)
	for i := uint64(0); i < count; i++ {
		height := new(big.Int).Add(startHeight, new(big.Int).SetUint64(i))
		batchElems[i] = rpc.BatchElem{
			Method: "eth_getBlockByNumber",
			Args:   []interface{}{toBlockNumArg(height), false},
			Result: new(types.Header),
			Error:  nil,
		}
	}

	ctxwt, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()
	err := c.rpcClient.BatchCallContext(ctxwt, batchElems)
	if err != nil {
		return nil, err
	}

	// Parse the headers.
	//  - Ensure integrity that they build on top of each other
	//  - Truncate out headers that do not exist (endHeight > "latest")
	size := 0
	headers := make([]*types.Header, count)
	for i, batchElem := range batchElems {
		if batchElem.Error != nil {
			return nil, batchElem.Error
		} else if batchElem.Result == nil {
			break
		}

		header := batchElem.Result.(*types.Header)
		if i > 0 && header.ParentHash != headers[i-1].Hash() {
			// Warn here that we got a bad (malicious?) response
			break
		}

		headers[i] = header
		size = size + 1
	}
	headers = headers[:size]

	return headers, nil
}

func toBlockNumArg(number *big.Int) string {
	if number == nil {
		return "latest"
	}

	pending := big.NewInt(-1)
	if number.Cmp(pending) == 0 {
		return "pending"
	}

	return hexutil.EncodeBig(number)
}
