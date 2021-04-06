package rollup

import (
	"errors"
	"fmt"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/go-resty/resty/v2"
)

/**
 * GET /enqueue/index/{index}
 * GET /transaction/index/{index}
 * GET /eth/context/latest
 */

type Batch struct {
	Index             uint64         `json:"index"`
	Root              common.Hash    `json:"root,omitempty"`
	Size              uint32         `json:"size,omitempty"`
	PrevTotalElements uint32         `json:"prevTotalElements,omitempty"`
	ExtraData         hexutil.Bytes  `json:"extraData,omitempty"`
	BlockNumber       uint64         `json:"blockNumber"`
	Timestamp         uint64         `json:"timestamp"`
	Submitter         common.Address `json:"submitter"`
}

type EthContext struct {
	BlockNumber uint64      `json:"blockNumber"`
	BlockHash   common.Hash `json:"blockHash"`
	Timestamp   uint64      `json:"timestamp"`
}

type SyncStatus struct {
	Syncing                      bool   `json:"syncing"`
	HighestKnownTransactionIndex uint64 `json:"highestKnownTransactionIndex"`
	CurrentTransactionIndex      uint64 `json:"currentTransactionIndex"`
}

type transaction struct {
	Index       uint64          `json:"index"`
	BatchIndex  uint64          `json:"batchIndex"`
	BlockNumber uint64          `json:"blockNumber"`
	Timestamp   uint64          `json:"timestamp"`
	GasLimit    uint64          `json:"gasLimit"`
	Target      common.Address  `json:"target"`
	Origin      *common.Address `json:"origin"`
	Data        hexutil.Bytes   `json:"data"`
	QueueOrigin string          `json:"queueOrigin"`
	Type        string          `json:"type"`
	QueueIndex  *uint64         `json:"queueIndex"`
	Decoded     *decoded        `json:"decoded"`
}

type Enqueue struct {
	Index       *uint64         `json:"ctcIndex"`
	Target      *common.Address `json:"target"`
	Data        *hexutil.Bytes  `json:"data"`
	GasLimit    *uint64         `json:"gasLimit"`
	Origin      *common.Address `json:"origin"`
	BlockNumber *uint64         `json:"blockNumber"`
	Timestamp   *uint64         `json:"timestamp"`
	QueueIndex  *uint64         `json:"index"`
}

type signature struct {
	R hexutil.Bytes `json:"r"`
	S hexutil.Bytes `json:"s"`
	V uint          `json:"v"`
}

type decoded struct {
	Signature signature      `json:"sig"`
	GasLimit  uint64         `json:"gasLimit"`
	GasPrice  uint64         `json:"gasPrice"`
	Nonce     uint64         `json:"nonce"`
	Target    common.Address `json:"target"`
	Data      hexutil.Bytes  `json:"data"`
}

type RollupClient interface {
	GetEnqueue(index uint64) (*types.Transaction, error)
	GetLatestEnqueue() (*types.Transaction, error)
	GetTransaction(index uint64) (*types.Transaction, error)
	GetLatestTransaction() (*types.Transaction, error)
	GetEthContext(index uint64) (*EthContext, error)
	GetLatestEthContext() (*EthContext, error)
	GetLastConfirmedEnqueue() (*types.Transaction, error)
	SyncStatus() (*SyncStatus, error)
}

type Client struct {
	client *resty.Client
	signer *types.OVMSigner
}

type TransactionResponse struct {
	Transaction *transaction `json:"transaction"`
	Batch       *Batch       `json:"batch"`
}

func NewClient(url string, chainID *big.Int) *Client {
	client := resty.New()
	client.SetHostURL(url)
	signer := types.NewOVMSigner(chainID)

	return &Client{
		client: client,
		signer: &signer,
	}
}

// This needs to return a transaction instead
func (c *Client) GetEnqueue(index uint64) (*types.Transaction, error) {
	str := strconv.FormatUint(index, 10)
	response, err := c.client.R().
		SetPathParams(map[string]string{
			"index": str,
		}).
		SetResult(&Enqueue{}).
		Get("/enqueue/index/{index}")

	if err != nil {
		return nil, err
	}
	enqueue, ok := response.Result().(*Enqueue)
	if !ok {
		return nil, fmt.Errorf("Cannot fetch enqueue %d", index)
	}
	if enqueue == nil {
		return nil, fmt.Errorf("Cannot deserialize enqueue %d", index)
	}
	tx, err := enqueueToTransaction(enqueue)
	if err != nil {
		return nil, fmt.Errorf("Cannot parse enqueue tx :%w", err)
	}
	return tx, nil
}

func enqueueToTransaction(enqueue *Enqueue) (*types.Transaction, error) {
	// When the queue index is nil, is means that the enqueue'd transaction
	// does not exist.
	if enqueue.QueueIndex == nil {
		return nil, nil
	}
	// The queue index is the nonce
	nonce := *enqueue.QueueIndex

	if enqueue.Target == nil {
		return nil, errors.New("Target not found for enqueue tx")
	}
	target := *enqueue.Target

	if enqueue.GasLimit == nil {
		return nil, errors.New("Gas limit not found for enqueue tx")
	}
	gasLimit := *enqueue.GasLimit
	if enqueue.Origin == nil {
		return nil, errors.New("Origin not found for enqueue tx")
	}
	origin := *enqueue.Origin
	if enqueue.BlockNumber == nil {
		return nil, errors.New("Blocknumber not found for enqueue tx")
	}
	blockNumber := new(big.Int).SetUint64(*enqueue.BlockNumber)
	if enqueue.Timestamp == nil {
		return nil, errors.New("Timestamp not found for enqueue tx")
	}
	timestamp := *enqueue.Timestamp

	if enqueue.Data == nil {
		return nil, errors.New("Data not found for enqueue tx")
	}
	data := *enqueue.Data

	value := big.NewInt(0)
	tx := types.NewTransaction(nonce, target, value, gasLimit, big.NewInt(0), data)

	// The index does not get a check as it is allowed to be nil in the context
	// of an enqueue transaction that has yet to be included into the CTC
	txMeta := types.NewTransactionMeta(
		blockNumber,
		timestamp,
		&origin,
		types.SighashEIP155,
		types.QueueOriginL1ToL2,
		enqueue.Index,
		enqueue.QueueIndex,
		data,
	)
	tx.SetTransactionMeta(txMeta)

	return tx, nil
}

func (c *Client) GetLatestEnqueue() (*types.Transaction, error) {
	response, err := c.client.R().
		SetResult(&Enqueue{}).
		Get("/enqueue/latest")

	if err != nil {
		return nil, err
	}
	enqueue, ok := response.Result().(*Enqueue)
	if !ok {
		return nil, errors.New("Cannot fetch latest enqueue")
	}
	tx, err := enqueueToTransaction(enqueue)
	if err != nil {
		return nil, fmt.Errorf("Cannot parse enqueue tx :%w", err)
	}
	return tx, nil
}

func transactionResponseToTransaction(res *TransactionResponse, signer *types.OVMSigner) (*types.Transaction, error) {
	// `nil` transactions are not found
	if res.Transaction == nil {
		return nil, nil
	}
	// The queue origin must be either sequencer of l1, otherwise
	// it is considered an unknown queue origin and will not be processed
	var queueOrigin types.QueueOrigin
	if res.Transaction.QueueOrigin == "sequencer" {
		queueOrigin = types.QueueOriginSequencer
	} else if res.Transaction.QueueOrigin == "l1" {
		queueOrigin = types.QueueOriginL1ToL2
	} else {
		return nil, fmt.Errorf("Unknown queue origin: %s", res.Transaction.QueueOrigin)
	}
	// The transaction type must be EIP155 or EthSign. Throughout this
	// codebase, it is referred to as "sighash type" but it could actually
	// be generalized to transaction type. Right now the only different
	// types use a different signature hashing scheme.
	var sighashType types.SignatureHashType
	if res.Transaction.Type == "EIP155" {
		sighashType = types.SighashEIP155
	} else if res.Transaction.Type == "ETH_SIGN" {
		sighashType = types.SighashEthSign
	} else {
		return nil, fmt.Errorf("Unknown transaction type: %s", res.Transaction.Type)
	}
	// Transactions that have been decoded are
	// Queue Origin Sequencer transactions
	if res.Transaction.Decoded != nil {
		nonce := res.Transaction.Decoded.Nonce
		to := res.Transaction.Decoded.Target
		value := new(big.Int)
		// Note: there are two gas limits, one top level and
		// another on the raw transaction itself. Maybe maxGasLimit
		// for the top level?
		gasLimit := res.Transaction.Decoded.GasLimit
		gasPrice := new(big.Int).SetUint64(res.Transaction.Decoded.GasPrice)
		data := res.Transaction.Decoded.Data

		var tx *types.Transaction
		if to == (common.Address{}) {
			tx = types.NewContractCreation(nonce, value, gasLimit, gasPrice, data)
		} else {
			tx = types.NewTransaction(nonce, to, value, gasLimit, gasPrice, data)
		}

		txMeta := types.NewTransactionMeta(
			new(big.Int).SetUint64(res.Transaction.BlockNumber),
			res.Transaction.Timestamp,
			res.Transaction.Origin,
			sighashType,
			queueOrigin,
			&res.Transaction.Index,
			res.Transaction.QueueIndex,
			res.Transaction.Data,
		)
		tx.SetTransactionMeta(txMeta)

		r, s := res.Transaction.Decoded.Signature.R, res.Transaction.Decoded.Signature.S
		sig := make([]byte, crypto.SignatureLength)
		copy(sig[32-len(r):32], r)
		copy(sig[64-len(s):64], s)
		sig[64] = byte(res.Transaction.Decoded.Signature.V)

		tx, err := tx.WithSignature(signer, sig[:])
		if err != nil {
			return nil, fmt.Errorf("Cannot add signature to transaction: %w", err)
		}

		return tx, nil
	}

	// The transaction is  either an L1 to L2 transaction or it does not have a
	// known deserialization
	nonce := uint64(0)
	if res.Transaction.QueueOrigin == "l1" {
		if res.Transaction.QueueIndex == nil {
			return nil, errors.New("Queue origin L1 to L2 without a queue index")
		}
		nonce = *res.Transaction.QueueIndex
	}
	target := res.Transaction.Target
	gasLimit := res.Transaction.GasLimit
	data := res.Transaction.Data
	origin := res.Transaction.Origin
	tx := types.NewTransaction(nonce, target, big.NewInt(0), gasLimit, big.NewInt(0), data)
	txMeta := types.NewTransactionMeta(
		new(big.Int).SetUint64(res.Transaction.BlockNumber),
		res.Transaction.Timestamp,
		origin,
		sighashType,
		queueOrigin,
		&res.Transaction.Index,
		res.Transaction.QueueIndex,
		res.Transaction.Data,
	)
	tx.SetTransactionMeta(txMeta)
	return tx, nil
}

func (c *Client) GetTransaction(index uint64) (*types.Transaction, error) {
	str := strconv.FormatUint(index, 10)
	response, err := c.client.R().
		SetPathParams(map[string]string{
			"index": str,
		}).
		SetResult(&TransactionResponse{}).
		Get("/transaction/index/{index}")

	if err != nil {
		return nil, err
	}
	res, ok := response.Result().(*TransactionResponse)
	if !ok {
		return nil, fmt.Errorf("could not get tx with index %d", index)
	}

	return transactionResponseToTransaction(res, c.signer)
}

func (c *Client) GetLatestTransaction() (*types.Transaction, error) {
	response, err := c.client.R().
		SetResult(&TransactionResponse{}).
		Get("/transaction/latest")

	if err != nil {
		return nil, err
	}
	res, ok := response.Result().(*TransactionResponse)
	if !ok {
		return nil, errors.New("")
	}

	return transactionResponseToTransaction(res, c.signer)
}

func (c *Client) GetEthContext(blockNumber uint64) (*EthContext, error) {
	str := strconv.FormatUint(blockNumber, 10)
	response, err := c.client.R().
		SetPathParams(map[string]string{
			"blocknumber": str,
		}).
		SetResult(&EthContext{}).
		Get("/eth/context/blocknumber/{blocknumber}")

	if err != nil {
		return nil, err
	}

	context, ok := response.Result().(*EthContext)
	if !ok {
		return nil, errors.New("Cannot parse EthContext")
	}

	return context, nil
}

func (c *Client) GetLatestEthContext() (*EthContext, error) {
	response, err := c.client.R().
		SetResult(&EthContext{}).
		Get("/eth/context/latest")

	if err != nil {
		return nil, fmt.Errorf("Cannot fetch eth context: %w", err)
	}

	context, ok := response.Result().(*EthContext)
	if !ok {
		return nil, errors.New("Cannot parse EthContext")
	}

	return context, nil
}

func (c *Client) GetLastConfirmedEnqueue() (*types.Transaction, error) {
	enqueue, err := c.GetLatestEnqueue()
	if err != nil {
		return nil, fmt.Errorf("Cannot get latest enqueue: %w", err)
	}
	// This should only happen if the database is empty
	if enqueue == nil {
		return nil, nil
	}
	// Work backwards looking for the first enqueue
	// to have an index, which means it has been included
	// in the canonical transaction chain.
	for {
		meta := enqueue.GetMeta()
		if meta.Index != nil {
			return enqueue, nil
		}
		if meta.QueueIndex == nil {
			return nil, fmt.Errorf("queue index is nil")
		}
		if *meta.QueueIndex == uint64(0) {
			return enqueue, nil
		}
		next, err := c.GetEnqueue(*meta.QueueIndex - 1)
		if err != nil {
			return nil, fmt.Errorf("cannot get enqueue %d: %w", *meta.Index, err)
		}
		enqueue = next
	}
}

func (c *Client) SyncStatus() (*SyncStatus, error) {
	response, err := c.client.R().
		SetResult(&SyncStatus{}).
		Get("/eth/syncing")

	if err != nil {
		return nil, fmt.Errorf("Cannot fetch sync status: %w", err)
	}

	status, ok := response.Result().(*SyncStatus)
	if !ok {
		return nil, fmt.Errorf("Cannot parse sync status")
	}

	return status, nil
}
