package l1

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

const (
	DefaultConnectionTimeout        = 20 * time.Second
	DefaultConfDepth         uint64 = 20
	DefaultMaxBatchSize      uint64 = 100
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

func HeaderByNumber(ctx context.Context, client *rpc.Client, height *big.Int) (*NewHeader, error) {
	var head *NewHeader
	err := client.CallContext(ctx, &head, "eth_getBlockByNumber", toBlockNumArg(height), false)
	if err == nil && head == nil {
		err = ethereum.NotFound
	}
	return head, err
}

func (f *ConfirmedHeaderSelector) NewHead(
	ctx context.Context,
	lowest uint64,
	header *types.Header,
	client *rpc.Client,
) []*NewHeader {

	number := header.Number.Uint64()
	blockHash := header.Hash

	logger.Info("New block", "block", number, "hash", blockHash)

	if number < f.cfg.ConfDepth {
		return nil
	}
	endHeight := number - f.cfg.ConfDepth + 1

	minNextHeight := lowest + f.cfg.ConfDepth
	if minNextHeight > number {
		log.Info("Fork block ", "block", number, "hash", blockHash)
		return nil
	}
	startHeight := lowest + 1

	// Clamp to max batch size
	if startHeight+f.cfg.MaxBatchSize < endHeight+1 {
		endHeight = startHeight + f.cfg.MaxBatchSize - 1
	}

	nHeaders := endHeight - startHeight + 1
	if nHeaders > 1 {
		logger.Info("Loading block batch ",
			"startHeight", startHeight, "endHeight", endHeight)
	}

	headers := make([]*NewHeader, nHeaders)
	var wg sync.WaitGroup
	for i := uint64(0); i < nHeaders; i++ {
		wg.Add(1)
		go func(ii uint64) {
			defer wg.Done()

			ctxt, cancel := context.WithTimeout(ctx, DefaultConnectionTimeout)
			defer cancel()

			height := startHeight + ii
			bigHeight := new(big.Int).SetUint64(height)
			header, err := HeaderByNumber(ctxt, client, bigHeight)
			if err != nil {
				log.Error("Unable to load block ", "block", height, "err", err)
				return
			}

			headers[ii] = header
		}(i)
	}
	wg.Wait()

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

	return headers
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
