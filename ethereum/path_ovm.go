// Copyright (C) Immutability, LLC - All Rights Reserved
// Unauthorized copying of this file, via any medium is strictly prohibited
// Proprietary and confidential
// Written by Ino Murko <ino@omg.network>, July 2021

package ethereum

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/omgnetwork/immutability-eth-plugin/contracts/ovm_ctc"
	"github.com/omgnetwork/immutability-eth-plugin/contracts/ovm_scc"
	"github.com/omgnetwork/immutability-eth-plugin/util"
	"golang.org/x/crypto/sha3"
)

const ovm string = "ovm"

type Context struct {
	NumSequencedTransactions       int64 `json:"num_sequenced_transactions"`
	NumSubsequentQueueTransactions int64 `json:"num_subsequent_queue_transactions"`
	Timestamp                      int64 `json:"timestamp"`
	BlockNumber                    int64 `json:"block_number"`
}

func OvmPaths(b *PluginBackend) []*framework.Path {
	return []*framework.Path{
		{
			Pattern: QualifiedPath("encodeAppendSequencerBatch/?"),
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.CreateOperation: b.pathEncodeAppendSequencerBatch,
			},
			HelpSynopsis:    "Encoding for AppendSequencerBatch",
			HelpDescription: `Use this to Encode data for AppendSequencerBatch`,
			Fields: map[string]*framework.FieldSchema{
				"contexts": {
					Type:        framework.TypeStringSlice,
					Description: "An array of objects of num_sequenced_transactions,num_subsequent_queue_transactions,timestamp,block_number.",
				},
				"should_start_at_element": {
					Type:        framework.TypeString,
					Description: "AppendSequencerBatchParams shouldStartAtElement.",
				},
				"total_elements_to_append": {
					Type:        framework.TypeString,
					Description: "AppendSequencerBatchParams totalElementsToAppend.",
				},
				"transactions": {
					Type:        framework.TypeStringSlice,
					Description: "Transaction data.",
				},
			},
			ExistenceCheck: pathExistenceCheck,
		},
		{
			Pattern:         ContractPath(ovm, "appendStateBatch"),
			HelpSynopsis:    "Submits the state batch",
			HelpDescription: "Allows the sequencer to submit the state root batch.",
			Fields: map[string]*framework.FieldSchema{
				"name":    {Type: framework.TypeString, Description: "Name of the wallet."},
				"address": {Type: framework.TypeString, Description: "The address in the wallet."},
				"contract": {
					Type:        framework.TypeString,
					Description: "The address of the Block Controller.",
				},
				"gas_price": {
					Type:        framework.TypeString,
					Description: "The gas price for the transaction in wei.",
				},
				"nonce": {
					Type:        framework.TypeString,
					Description: "The nonce for the transaction.",
				},
				"should_start_at_element": {
					Type:        framework.TypeString,
					Description: "Index of the element at which this batch should start.",
				},
				"batch": {
					Type:        framework.TypeStringSlice,
					Description: "Batch of state roots.",
				},
			},
			ExistenceCheck: pathExistenceCheck,
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.CreateOperation: b.pathOvmAppendStateBatch,
			},
		},
		{
			Pattern:         ContractPath(ovm, "appendSequencerBatch"),
			HelpSynopsis:    "Submits the state batch",
			HelpDescription: "Allows the sequencer to submit the state root batch.",
			Fields: map[string]*framework.FieldSchema{
				"name":    {Type: framework.TypeString, Description: "Name of the wallet."},
				"address": {Type: framework.TypeString, Description: "The address in the wallet."},
				"contract": {
					Type:        framework.TypeString,
					Description: "The address of the Block Controller.",
				},
				"gas_price": {
					Type:        framework.TypeString,
					Description: "The gas price for the transaction in wei.",
				},
				"nonce": {
					Type:        framework.TypeString,
					Description: "The nonce for the transaction.",
				},

				"contexts": {
					Type:        framework.TypeStringSlice,
					Description: "An array of objects of num_sequenced_transactions,num_subsequent_queue_transactions,timestamp,block_number.",
				},
				"should_start_at_element": {
					Type:        framework.TypeString,
					Description: "AppendSequencerBatchParams shouldStartAtElement.",
				},
				"total_elements_to_append": {
					Type:        framework.TypeString,
					Description: "AppendSequencerBatchParams totalElementsToAppend.",
				},
				"transactions": {
					Type:        framework.TypeStringSlice,
					Description: "Transaction data.",
				},
			},
			ExistenceCheck: pathExistenceCheck,
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.CreateOperation: b.pathOvmAppendSequencerBatch,
			},
		},
	}
}

//this goes into L1
func (b *PluginBackend) pathOvmAppendStateBatch(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	config, err := b.configured(ctx, req)
	if err != nil {
		return nil, err
	}
	address := data.Get("address").(string)
	name := data.Get("name").(string)
	contractAddress := common.HexToAddress(data.Get("contract").(string))
	accountJSON, err := readAccount(ctx, req, name, address)
	if err != nil || accountJSON == nil {
		return nil, fmt.Errorf("error reading address")
	}

	chainID := util.ValidNumber(config.ChainID)
	if chainID == nil {
		return nil, fmt.Errorf("invalid chain ID")
	}

	client, err := ethclient.Dial(config.getRPCURL())
	if err != nil {
		return nil, err
	}

	walletJSON, err := readWallet(ctx, req, name)
	if err != nil {
		return nil, err
	}

	wallet, account, err := getWalletAndAccount(*walletJSON, accountJSON.Index)
	if err != nil {
		return nil, err
	}

	// get the AppendStateBatch function arguments from JSON
	inputShouldStartAtElement, ok := data.GetOk("should_start_at_element")
	if !ok {
		return nil, fmt.Errorf("invalid should_start_at_element")
	}
	shouldStartAtElement := util.ValidNumber(inputShouldStartAtElement.(string))
	if shouldStartAtElement == nil {
		return nil, fmt.Errorf("invalid should_start_at_element")
	}

	inputBatch, ok := data.GetOk("batch")
	if !ok {
		return nil, fmt.Errorf("invalid batch")
	}
	var inputBatchArr []string = inputBatch.([]string)
	var batch = make([][32]byte, len(inputBatchArr))

	for i, s := range inputBatchArr {
		var buf []byte
		hash := sha3.NewLegacyKeccak256()
		hash.Write([]byte(s))
		buf = hash.Sum(buf)
		if len(buf) != 32 {
			return nil, fmt.Errorf("invalid batch element - not the right size")
		}
		batchByteElement := [32]byte{}
		copy(batchByteElement[:], buf[0:32])
		batch[i] = batchByteElement
	}
	// log.Print(batch)
	// get the AppendStateBatch function arguments from JSON DONE

	instance, err := ovm_scc.NewOvmScc(contractAddress, client)
	if err != nil {
		return nil, err
	}
	callOpts := &bind.CallOpts{}

	transactOpts, err := b.NewWalletTransactor(chainID, wallet, account)
	if err != nil {
		return nil, err
	}
	// transactOpts needs gas etc. Use supplied gas_price
	gasPriceRaw := data.Get("gas_price").(string)
	if gasPriceRaw == "" {
		return nil, fmt.Errorf("invalid gas_price")
	}
	transactOpts.GasPrice = util.ValidNumber(gasPriceRaw)

	// //transactOpts needs nonce. Use supplied nonce
	nonceRaw := data.Get("nonce").(string)
	if nonceRaw == "" {
		return nil, fmt.Errorf("invalid nonce")
	}
	transactOpts.Nonce = util.ValidNumber(nonceRaw)

	sccSession := &ovm_scc.OvmSccSession{
		Contract:     instance,  // Generic contract caller binding to set the session for
		CallOpts:     *callOpts, // Call options to use throughout this session
		TransactOpts: *transactOpts,
	}

	tx, err := sccSession.AppendStateBatch(batch, shouldStartAtElement)
	if err != nil {
		return nil, err
	}

	var signedTxBuff bytes.Buffer
	tx.EncodeRLP(&signedTxBuff)
	return &logical.Response{
		Data: map[string]interface{}{
			"contract":           contractAddress.Hex(),
			"transaction_hash":   tx.Hash().Hex(),
			"signed_transaction": hexutil.Encode(signedTxBuff.Bytes()),
			"from":               account.Address.Hex(),
			"nonce":              tx.Nonce(),
			"gas_price":          tx.GasPrice(),
			"gas_limit":          tx.Gas(),
		},
	}, nil
}

func (b *PluginBackend) pathEncodeAppendSequencerBatch(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	//log.Print(util.PrettyPrint(data))

	encodedData, err := encode(data)
	if err != nil {
		return nil, err
	}
	return &logical.Response{
		Data: map[string]interface{}{
			"data": encodedData,
		},
	}, nil
}

func (b *PluginBackend) pathOvmAppendSequencerBatch(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	//log.Print(util.PrettyPrint(data))

	config, err := b.configured(ctx, req)
	if err != nil {
		return nil, err
	}
	address := data.Get("address").(string)
	name := data.Get("name").(string)
	contractAddress := common.HexToAddress(data.Get("contract").(string))
	accountJSON, err := readAccount(ctx, req, name, address)
	if err != nil || accountJSON == nil {
		return nil, fmt.Errorf("error reading address")
	}

	chainID := util.ValidNumber(config.ChainID)
	if chainID == nil {
		return nil, fmt.Errorf("invalid chain ID")
	}

	client, err := ethclient.Dial(config.getRPCURL())
	if err != nil {
		return nil, err
	}

	walletJSON, err := readWallet(ctx, req, name)
	if err != nil {
		return nil, err
	}

	wallet, account, err := getWalletAndAccount(*walletJSON, accountJSON.Index)
	if err != nil {
		return nil, err
	}

	instance, err := ovm_ctc.NewOvmCtc(contractAddress, client)
	//instance.OvmCtcTransactor.
	if err != nil {
		return nil, err
	}
	callOpts := &bind.CallOpts{}

	transactOpts, err := b.NewWalletTransactor(chainID, wallet, account)
	if err != nil {
		return nil, err
	}
	// transactOpts needs gas etc. Use supplied gas_price
	gasPriceRaw := data.Get("gas_price").(string)
	if gasPriceRaw == "" {
		return nil, fmt.Errorf("invalid gas_price")
	}
	transactOpts.GasPrice = util.ValidNumber(gasPriceRaw)

	// //transactOpts needs nonce. Use supplied nonce
	nonceRaw := data.Get("nonce").(string)
	if nonceRaw == "" {
		return nil, fmt.Errorf("invalid nonce")
	}
	transactOpts.Nonce = util.ValidNumber(nonceRaw)

	ctcSession := &ovm_ctc.OvmCtcSession{
		Contract:     instance,  // Generic contract caller binding to set the session for
		CallOpts:     *callOpts, // Call options to use throughout this session
		TransactOpts: *transactOpts,
	}

	encodedData, err := encode(data)
	if err != nil {
		return nil, err
	}

	tx, err := ctcSession.RawAppendSequencerBatch([]byte(encodedData))

	if err != nil {
		return nil, err
	}

	var signedTxBuff bytes.Buffer
	tx.EncodeRLP(&signedTxBuff)
	return &logical.Response{
		Data: map[string]interface{}{
			"contract":           contractAddress.Hex(),
			"transaction_hash":   tx.Hash().Hex(),
			"signed_transaction": hexutil.Encode(signedTxBuff.Bytes()),
			"from":               account.Address.Hex(),
			"nonce":              tx.Nonce(),
			"gas_price":          tx.GasPrice(),
			"gas_limit":          tx.Gas(),
		},
	}, nil
}

func encode(data *framework.FieldData) (string, error) {
	shouldStartAtElement, err := encodeShouldStartAtElement(data)
	if err != nil {
		return "", err
	}
	totalElementsToAppend, err := encodeTotalElementsToAppend(data)
	if err != nil {
		return "", err
	}
	contexts, err := encodeContexts(data)
	if err != nil {
		return "", err
	}
	transaction, err := encodeTransactionData(data)
	if err != nil {
		return "", err
	}

	return shouldStartAtElement +
		totalElementsToAppend +
		contexts +
		transaction, nil
}

func encodeTransactionData(data *framework.FieldData) (string, error) {
	inputTransactions, ok := data.GetOk("transactions")
	if !ok {
		return "", fmt.Errorf("invalid transactions")
	}

	var encodedTransactionData = ""
	for _, s := range inputTransactions.([]string) {
		if len(s)%2 != 0 {
			return "", fmt.Errorf("unexpected uneven hex string value in transactions")
		}
		encodedTransactionData += fmt.Sprintf("%06s", remove0x(fmt.Sprintf("%x", len(remove0x(s))/2))) + remove0x(s)
	}
	return encodedTransactionData, nil
}

func encodeContexts(data *framework.FieldData) (string, error) {
	inputContexts, ok := data.GetOk("contexts")
	if !ok {
		return "", fmt.Errorf("invalid contexts")
	}
	//contexts
	var contexts = make([]Context, len(inputContexts.([]string)))
	for i, s := range inputContexts.([]string) {
		var context Context
		json.Unmarshal([]byte(s), &context)
		contexts[i] = context
	}
	encodedContextsHeader := encodeHex(int64(len(contexts)), 6)
	var encodedContexts = ""
	for _, s := range contexts {
		encodedContexts += encodeBatchContext(s)
	}
	encodedContexts = encodedContextsHeader + encodedContexts
	return encodedContexts, nil
}

func encodeTotalElementsToAppend(data *framework.FieldData) (string, error) {
	dataTotalElementsToAppend := data.Get("total_elements_to_append").(string)
	validNumber := util.ValidNumber(dataTotalElementsToAppend)
	if validNumber == nil {
		return "", fmt.Errorf("invalid total_elements_to_append")
	}
	inputTotalElementsToAppend, err := strconv.ParseInt(dataTotalElementsToAppend, 10, 64)
	if err != nil {
		return "", fmt.Errorf("%d total_elements_to_append of type %T", inputTotalElementsToAppend, inputTotalElementsToAppend)
	}
	encodedTotalElementsToAppend := encodeHex(inputTotalElementsToAppend, 6)
	return encodedTotalElementsToAppend, nil
}

func encodeShouldStartAtElement(data *framework.FieldData) (string, error) {
	dataEncodeShouldStartAtElement := data.Get("should_start_at_element").(string)
	validNumber := util.ValidNumber(dataEncodeShouldStartAtElement)
	if validNumber == nil {
		return "", fmt.Errorf("invalid should_start_at_element")
	}
	inputEncodeShouldStartAtElement, err := strconv.ParseInt(dataEncodeShouldStartAtElement, 10, 64)
	if err != nil {
		return "", fmt.Errorf("%d should_start_at_element of type %T", inputEncodeShouldStartAtElement, inputEncodeShouldStartAtElement)
	}
	encodeShouldStartAtElement := encodeHex(inputEncodeShouldStartAtElement, 10)
	return encodeShouldStartAtElement, nil
}

func remove0x(val string) string {
	return strings.Replace(val, "0x", "", -1)
}

func encodeHex(val int64, len int) string {
	hex := fmt.Sprintf("%x", val)
	return fmt.Sprintf("%0"+strconv.Itoa(len)+"s", hex)
}

func encodeBatchContext(context Context) string {
	return (encodeHex(context.NumSequencedTransactions, 6) + encodeHex(context.NumSubsequentQueueTransactions, 6) + encodeHex(context.Timestamp, 10) + encodeHex(context.BlockNumber, 10))
}
