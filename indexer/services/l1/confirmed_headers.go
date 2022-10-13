package l1

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/indexer/services/util"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

const (
	DefaultConnectionTimeout = 30 * time.Second
	DefaultMaxBatchSize      = 100
)

type NewHeader struct {
	types.Header
	Hash common.Hash
}

func (h *NewHeader) UnmarshalJSON(input []byte) error {
	type NewHeader struct {
		Hash        *common.Hash      `json:"hash"             gencodec:"required"`
		ParentHash  *common.Hash      `json:"parentHash"       gencodec:"required"`
		UncleHash   *common.Hash      `json:"sha3Uncles"       gencodec:"required"`
		Coinbase    *common.Address   `json:"miner"            gencodec:"required"`
		Root        *common.Hash      `json:"stateRoot"        gencodec:"required"`
		TxHash      *common.Hash      `json:"transactionsRoot" gencodec:"required"`
		ReceiptHash *common.Hash      `json:"receiptsRoot"     gencodec:"required"`
		Bloom       *types.Bloom      `json:"logsBloom"        gencodec:"required"`
		Difficulty  *hexutil.Big      `json:"difficulty"       gencodec:"required"`
		Number      *hexutil.Big      `json:"number"           gencodec:"required"`
		GasLimit    *hexutil.Uint64   `json:"gasLimit"         gencodec:"required"`
		GasUsed     *hexutil.Uint64   `json:"gasUsed"          gencodec:"required"`
		Time        *hexutil.Uint64   `json:"timestamp"        gencodec:"required"`
		Extra       *hexutil.Bytes    `json:"extraData"        gencodec:"required"`
		MixDigest   *common.Hash      `json:"mixHash"`
		Nonce       *types.BlockNonce `json:"nonce"`
		BaseFee     *hexutil.Big      `json:"baseFeePerGas" rlp:"optional"`
	}
	var dec NewHeader
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.Hash == nil {
		return errors.New("missing required field 'hash' for Header")
	}
	h.Hash = *dec.Hash
	if dec.ParentHash == nil {
		return errors.New("missing required field 'parentHash' for Header")
	}
	h.ParentHash = *dec.ParentHash
	if dec.UncleHash == nil {
		return errors.New("missing required field 'sha3Uncles' for Header")
	}
	h.UncleHash = *dec.UncleHash
	if dec.Coinbase == nil {
		return errors.New("missing required field 'miner' for Header")
	}
	h.Coinbase = *dec.Coinbase
	if dec.Root == nil {
		return errors.New("missing required field 'stateRoot' for Header")
	}
	h.Root = *dec.Root
	if dec.TxHash == nil {
		return errors.New("missing required field 'transactionsRoot' for Header")
	}
	h.TxHash = *dec.TxHash
	if dec.ReceiptHash == nil {
		return errors.New("missing required field 'receiptsRoot' for Header")
	}
	h.ReceiptHash = *dec.ReceiptHash
	if dec.Bloom == nil {
		return errors.New("missing required field 'logsBloom' for Header")
	}
	h.Bloom = *dec.Bloom
	if dec.Difficulty == nil {
		return errors.New("missing required field 'difficulty' for Header")
	}
	h.Difficulty = (*big.Int)(dec.Difficulty)
	if dec.Number == nil {
		return errors.New("missing required field 'number' for Header")
	}
	h.Number = (*big.Int)(dec.Number)
	if dec.GasLimit == nil {
		return errors.New("missing required field 'gasLimit' for Header")
	}
	h.GasLimit = uint64(*dec.GasLimit)
	if dec.GasUsed == nil {
		return errors.New("missing required field 'gasUsed' for Header")
	}
	h.GasUsed = uint64(*dec.GasUsed)
	if dec.Time == nil {
		return errors.New("missing required field 'timestamp' for Header")
	}
	h.Time = uint64(*dec.Time)
	if dec.Extra == nil {
		return errors.New("missing required field 'extraData' for Header")
	}
	h.Extra = *dec.Extra
	if dec.MixDigest != nil {
		h.MixDigest = *dec.MixDigest
	}
	if dec.Nonce != nil {
		h.Nonce = *dec.Nonce
	}
	if dec.BaseFee != nil {
		h.BaseFee = (*big.Int)(dec.BaseFee)
	}
	return nil
}

type HeaderSelectorConfig struct {
	ConfDepth    uint64
	MaxBatchSize uint64
}

type ConfirmedHeaderSelector struct {
	cfg HeaderSelectorConfig
}

func HeadersByRange(ctx context.Context, client *rpc.Client, startHeight uint64, count int) ([]*NewHeader, error) {
	height := startHeight
	batchElems := make([]rpc.BatchElem, count)
	for i := 0; i < count; i++ {
		batchElems[i] = rpc.BatchElem{
			Method: "eth_getBlockByNumber",
			Args: []interface{}{
				util.ToBlockNumArg(new(big.Int).SetUint64(height + uint64(i))),
				false,
			},
			Result: new(NewHeader),
			Error:  nil,
		}
	}

	if err := client.BatchCallContext(ctx, batchElems); err != nil {
		return nil, err
	}

	out := make([]*NewHeader, count)
	for i := 0; i < len(batchElems); i++ {
		if batchElems[i].Error != nil {
			return nil, batchElems[i].Error
		}
		out[i] = batchElems[i].Result.(*NewHeader)
	}

	return out, nil
}

func (f *ConfirmedHeaderSelector) NewHead(
	ctx context.Context,
	lowest uint64,
	header *types.Header,
	client *rpc.Client,
) ([]*NewHeader, error) {

	number := header.Number.Uint64()
	blockHash := header.Hash

	logger.Info("New block", "block", number, "hash", blockHash)

	if number < f.cfg.ConfDepth {
		return nil, nil
	}
	endHeight := number - f.cfg.ConfDepth + 1

	minNextHeight := lowest + f.cfg.ConfDepth
	if minNextHeight > number {
		log.Info("Fork block ", "block", number, "hash", blockHash)
		return nil, nil
	}
	startHeight := lowest + 1

	// Clamp to max batch size
	if startHeight+f.cfg.MaxBatchSize < endHeight+1 {
		endHeight = startHeight + f.cfg.MaxBatchSize - 1
	}

	nHeaders := int(endHeight - startHeight + 1)
	if nHeaders > 1 {
		logger.Info("Loading blocks",
			"startHeight", startHeight, "endHeight", endHeight)
	}

	headers := make([]*NewHeader, 0)
	height := startHeight
	left := nHeaders - len(headers)
	for left > 0 {
		count := DefaultMaxBatchSize
		if count > left {
			count = left
		}

		logger.Info("Loading block batch",
			"height", height, "count", count)

		ctxt, cancel := context.WithTimeout(ctx, DefaultConnectionTimeout)
		fetched, err := HeadersByRange(ctxt, client, height, count)
		cancel()
		if err != nil {
			return nil, err
		}

		headers = append(headers, fetched...)
		left = nHeaders - len(headers)
		height += uint64(count)
	}

	logger.Debug("Verifying block range ",
		"startHeight", startHeight, "endHeight", endHeight)

	for i, header := range headers {
		// Trim the returned headers if any of the lookups failed.
		if header == nil {
			headers = headers[:i]
			break
		}

		// Assert that each header builds on the parent before it, trim if there
		// are any discontinuities.
		if i > 0 {
			prevHeader := headers[i-1]
			if prevHeader.Hash != header.ParentHash {
				log.Error("Parent hash does not connect to ",
					"block", header.Number.Uint64(), "hash", header.Hash,
					"prev", prevHeader.Number.Uint64(), "hash", prevHeader.Hash)
				headers = headers[:i]
				break
			}
		}

		log.Debug("Confirmed block ",
			"block", header.Number.Uint64(), "hash", header.Hash)
	}

	return headers, nil
}

func NewConfirmedHeaderSelector(cfg HeaderSelectorConfig) (*ConfirmedHeaderSelector,
	error) {
	if cfg.ConfDepth == 0 {
		return nil, errors.New("ConfDepth must be greater than zero")
	}
	if cfg.MaxBatchSize == 0 {
		return nil, errors.New("MaxBatchSize must be greater than zero")
	}

	return &ConfirmedHeaderSelector{
		cfg: cfg,
	}, nil
}
