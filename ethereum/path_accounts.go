// Copyright (C) OmiseGO - All Rights Reserved
// Unauthorized copying of this file, via any medium is strictly prohibited
// Proprietary and confidential
// Written by Jeff Ploughman <jeff@immutability.io>, October 2019

package ethereum

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/omisego/immutability-eth-plugin/util"
)

// AccountJSON is what we store for an Ethereum address
type AccountJSON struct {
	Index     int      `json:"index"`
	Whitelist []string `json:"whitelist"`
	Blacklist []string `json:"blacklist"`
}

// BlackListed returns an error if the address is blacklisted
func (account *AccountJSON) BlackListed(toAddress *common.Address) error {
	if util.Contains(account.Blacklist, toAddress.Hex()) {
		return fmt.Errorf("%s is blacklisted by this account", toAddress.Hex())
	}
	return nil
}

// AccountPaths are the path handlers for Ethereum wallets
func AccountPaths(b *PluginBackend) []*framework.Path {
	return []*framework.Path{
		&framework.Path{
			Pattern: QualifiedPath("wallets/" + framework.GenericNameRegex("name") + "/accounts/?"),
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.ListOperation:   b.pathAccountsList,
				logical.CreateOperation: b.pathAccountsCreate,
				logical.UpdateOperation: b.pathAccountsCreate,
			},
			Fields: map[string]*framework.FieldSchema{
				"name": &framework.FieldSchema{Type: framework.TypeString},
				"whitelist": &framework.FieldSchema{
					Type:        framework.TypeCommaStringSlice,
					Description: "The list of accounts that any account can send ETH to.",
				},
				"blacklist": &framework.FieldSchema{
					Type:        framework.TypeCommaStringSlice,
					Description: "The list of accounts that any account can't send ETH to.",
				},
			},
			HelpSynopsis: "List all the Ethereum accounts for a wallet",
			HelpDescription: `
			All the accounts for an Ethereum wallet will be listed.
			`,
		},
		&framework.Path{
			Pattern:      QualifiedPath("wallets/" + framework.GenericNameRegex("name") + "/accounts/" + framework.GenericNameRegex("address")),
			HelpSynopsis: "Create an address.",
			HelpDescription: `

Creates (or updates) an Ethereum wallet: an wallet controlled by a private key. Also
The generator produces a high-entropy passphrase with the provided length and requirements.

`,
			Fields: map[string]*framework.FieldSchema{
				"name":    &framework.FieldSchema{Type: framework.TypeString},
				"address": &framework.FieldSchema{Type: framework.TypeString},
				"whitelist": &framework.FieldSchema{
					Type:        framework.TypeCommaStringSlice,
					Description: "The list of accounts that any account can send ETH to.",
				},
				"blacklist": &framework.FieldSchema{
					Type:        framework.TypeCommaStringSlice,
					Description: "The list of accounts that any account can't send ETH to.",
				},
			},
			ExistenceCheck: pathExistenceCheck,
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.ReadOperation:   b.pathAccountsRead,
				logical.UpdateOperation: b.pathAccountsUpdate,
				logical.DeleteOperation: b.pathAccountsDelete,
			},
		},
		&framework.Path{
			Pattern:      QualifiedPath("wallets/" + framework.GenericNameRegex("name") + "/accounts/" + framework.GenericNameRegex("address") + "/sign-tx"),
			HelpSynopsis: "Sign a transaction.",
			HelpDescription: `

Sign a transaction.

`,
			Fields: map[string]*framework.FieldSchema{
				"name":    &framework.FieldSchema{Type: framework.TypeString},
				"address": &framework.FieldSchema{Type: framework.TypeString},
				"to": &framework.FieldSchema{
					Type:        framework.TypeString,
					Description: "The address of the wallet to send ETH to.",
				},
				"data": &framework.FieldSchema{
					Type:        framework.TypeString,
					Description: "The data to sign.",
				},
				"encoding": &framework.FieldSchema{
					Type:        framework.TypeString,
					Default:     "utf8",
					Description: "The encoding of the data to sign.",
				},
				"amount": &framework.FieldSchema{
					Type:        framework.TypeString,
					Description: "Amount of ETH (in wei).",
				},
				"nonce": &framework.FieldSchema{
					Type:        framework.TypeString,
					Description: "The transaction nonce.",
				},
				"gas_limit": &framework.FieldSchema{
					Type:        framework.TypeString,
					Description: "The gas limit for the transaction - defaults to 21000.",
					Default:     "21000",
				},
				"gas_price": &framework.FieldSchema{
					Type:        framework.TypeString,
					Description: "The gas price for the transaction in wei.",
					Default:     "0",
				},
			},
			ExistenceCheck: pathExistenceCheck,
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.CreateOperation: b.pathSignTx,
				logical.UpdateOperation: b.pathSignTx,
			},
		},
		&framework.Path{
			Pattern:      QualifiedPath("wallets/" + framework.GenericNameRegex("name") + "/accounts/" + framework.GenericNameRegex("address") + "/debit"),
			HelpSynopsis: "Send ETH from an account.",
			HelpDescription: `

Send ETH from an account.

`,
			Fields: map[string]*framework.FieldSchema{
				"name":    &framework.FieldSchema{Type: framework.TypeString},
				"address": &framework.FieldSchema{Type: framework.TypeString},
				"to": &framework.FieldSchema{
					Type:        framework.TypeString,
					Description: "The address of the wallet to send ETH to.",
				},
				"amount": &framework.FieldSchema{
					Type:        framework.TypeString,
					Description: "Amount of ETH (in wei).",
				},
				"gas_limit": &framework.FieldSchema{
					Type:        framework.TypeString,
					Description: "The gas limit for the transaction - defaults to 21000.",
					Default:     "21000",
				},
				"gas_price": &framework.FieldSchema{
					Type:        framework.TypeString,
					Description: "The gas price for the transaction in wei.",
					Default:     "0",
				},
			},
			ExistenceCheck: pathExistenceCheck,
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.UpdateOperation: b.pathDebit,
				logical.CreateOperation: b.pathDebit,
			},
		},
		&framework.Path{
			Pattern:      QualifiedPath("wallets/" + framework.GenericNameRegex("name") + "/accounts/" + framework.GenericNameRegex("address") + "/balance"),
			HelpSynopsis: "Return the balance for an account.",
			HelpDescription: `

Return the balance in wei for an address.

`,
			Fields: map[string]*framework.FieldSchema{
				"name":    &framework.FieldSchema{Type: framework.TypeString},
				"address": &framework.FieldSchema{Type: framework.TypeString},
			},
			ExistenceCheck: pathExistenceCheck,
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.ReadOperation: b.pathReadBalance,
			},
		},
		&framework.Path{
			Pattern:      QualifiedPath("wallets/" + framework.GenericNameRegex("name") + "/accounts/" + framework.GenericNameRegex("address") + "/deploy"),
			HelpSynopsis: "Deploy a smart contract from an account.",
			HelpDescription: `

Send ETH from an account.

`,
			Fields: map[string]*framework.FieldSchema{
				"name":    &framework.FieldSchema{Type: framework.TypeString},
				"address": &framework.FieldSchema{Type: framework.TypeString},
				"version": &framework.FieldSchema{
					Type:        framework.TypeString,
					Description: "The smart contract version.",
				},
				"abi": &framework.FieldSchema{
					Type:        framework.TypeString,
					Description: "The contract ABI.",
				},
				"bin": &framework.FieldSchema{
					Type:        framework.TypeString,
					Description: "The compiled smart contract.",
				},
				"gas_limit": &framework.FieldSchema{
					Type:        framework.TypeString,
					Description: "The gas limit for the transaction - defaults to 0 meaning estimate.",
					Default:     "0",
				},
			},
			ExistenceCheck: pathExistenceCheck,
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.UpdateOperation: b.pathDeploy,
				logical.CreateOperation: b.pathDeploy,
			},
		},
	}
}

func readAccount(ctx context.Context, req *logical.Request, name string, address string) (*AccountJSON, error) {
	path := QualifiedPath("wallets/" + name + "/accounts/" + address)
	entry, err := req.Storage.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, nil
	}

	var accountJSON AccountJSON
	err = entry.DecodeJSON(&accountJSON)

	if entry == nil {
		return nil, fmt.Errorf("failed to deserialize address at %s", path)
	}
	return &accountJSON, nil
}

func (b *PluginBackend) pathAccountsList(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	vals, err := req.Storage.List(ctx, req.Path)
	if err != nil {
		return nil, err
	}
	return logical.ListResponse(vals), nil
}

func (b *PluginBackend) pathAccountsRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	_, err := b.configured(ctx, req)
	if err != nil {
		return nil, err
	}
	name := data.Get("name").(string)
	address := data.Get("address").(string)
	accountJSON, err := readAccount(ctx, req, name, address)
	if err != nil || accountJSON == nil {
		return nil, fmt.Errorf("error reading address")
	}

	return &logical.Response{
		Data: map[string]interface{}{
			"address":   address,
			"whitelist": accountJSON.Whitelist,
			"blacklist": accountJSON.Blacklist,
			"index":     accountJSON.Index,
		},
	}, nil
}

func (b *PluginBackend) pathAccountsDelete(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	_, err := b.configured(ctx, req)
	if err != nil {
		return nil, err
	}
	name := data.Get("name").(string)
	address := data.Get("address").(string)
	accountJSON, err := readAccount(ctx, req, name, address)
	if err != nil || accountJSON == nil {
		return nil, fmt.Errorf("error reading address")
	}
	if err := req.Storage.Delete(ctx, req.Path); err != nil {
		return nil, err
	}
	return nil, nil
}

func (b *PluginBackend) pathAccountsCreate(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	_, err := b.configured(ctx, req)
	if err != nil {
		return nil, err
	}
	name := data.Get("name").(string)

	walletJSON, err := readWallet(ctx, req, name)
	_, account, err := getWalletAndAccount(*walletJSON, walletJSON.Index)
	if err != nil {
		return nil, err
	}
	var whiteList []string
	if whiteListRaw, ok := data.GetOk("whitelist"); ok {
		whiteList = whiteListRaw.([]string)
	}
	var blackList []string
	if blackListRaw, ok := data.GetOk("blacklist"); ok {
		blackList = blackListRaw.([]string)
	}

	accountJSON := &AccountJSON{
		Index:     walletJSON.Index,
		Whitelist: whiteList,
		Blacklist: blackList,
	}

	walletJSON.Index = walletJSON.Index + 1
	b.updateWallet(ctx, req, name, walletJSON)
	path := QualifiedPath("wallets/" + name + "/accounts/" + account.Address.Hex())
	entry, err := logical.StorageEntryJSON(path, accountJSON)
	if err != nil {
		return nil, err
	}

	err = req.Storage.Put(ctx, entry)
	if err != nil {
		return nil, err
	}

	return &logical.Response{
		Data: map[string]interface{}{
			"address":   account.Address.Hex(),
			"whitelist": accountJSON.Whitelist,
			"blacklist": accountJSON.Blacklist,
			"index":     accountJSON.Index,
		},
	}, nil
}

func (b *PluginBackend) pathAccountsUpdate(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	_, err := b.configured(ctx, req)
	if err != nil {
		return nil, err
	}
	name := data.Get("name").(string)

	address := data.Get("address").(string)
	accountJSON, err := readAccount(ctx, req, name, address)
	if err != nil || accountJSON == nil {
		return nil, fmt.Errorf("error reading address")
	}

	walletJSON, err := readWallet(ctx, req, name)
	_, account, err := getWalletAndAccount(*walletJSON, accountJSON.Index)
	if err != nil {
		return nil, err
	}

	var whiteList []string
	if whiteListRaw, ok := data.GetOk("whitelist"); ok {
		whiteList = whiteListRaw.([]string)
	} else {
		whiteList = accountJSON.Whitelist
	}
	var blackList []string
	if blackListRaw, ok := data.GetOk("blacklist"); ok {
		blackList = blackListRaw.([]string)
	} else {
		blackList = accountJSON.Blacklist
	}

	accountJSON = &AccountJSON{
		Index:     accountJSON.Index,
		Whitelist: whiteList,
		Blacklist: blackList,
	}

	path := QualifiedPath("wallets/" + name + "/accounts/" + account.Address.Hex())
	entry, err := logical.StorageEntryJSON(path, accountJSON)
	if err != nil {
		return nil, err
	}

	err = req.Storage.Put(ctx, entry)
	if err != nil {
		return nil, err
	}

	return &logical.Response{
		Data: map[string]interface{}{
			"address":   account.Address.Hex(),
			"whitelist": accountJSON.Whitelist,
			"blacklist": accountJSON.Blacklist,
			"index":     accountJSON.Index,
		},
	}, nil
}

func (b *PluginBackend) pathDebit(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	var txDataToSign []byte
	config, err := b.configured(ctx, req)
	if err != nil {
		return nil, err
	}
	name := data.Get("name").(string)
	address := data.Get("address").(string)
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
		return nil, fmt.Errorf("cannot connect to " + config.getRPCURL())
	}

	walletJSON, err := readWallet(ctx, req, name)
	if err != nil {
		return nil, err
	}

	wallet, account, err := getWalletAndAccount(*walletJSON, accountJSON.Index)
	if err != nil {
		return nil, err
	}

	transactionParams, err := b.getData(client, account.Address, data)

	if err != nil {
		return nil, err
	}
	accountJSON.Whitelist = append(accountJSON.Whitelist, config.Whitelist...)
	accountJSON.Whitelist = append(accountJSON.Whitelist, walletJSON.Whitelist...)
	if len(accountJSON.Whitelist) > 0 && !util.Contains(accountJSON.Whitelist, transactionParams.Address.Hex()) {
		return nil, fmt.Errorf("%s violates the whitelist %+v", transactionParams.Address.Hex(), accountJSON.Whitelist)
	}
	err = config.BlackListed(transactionParams.Address)
	if err != nil {
		return nil, err
	}
	err = walletJSON.BlackListed(transactionParams.Address)
	if err != nil {
		return nil, err
	}
	err = accountJSON.BlackListed(transactionParams.Address)
	if err != nil {
		return nil, err
	}

	tx := types.NewTransaction(transactionParams.Nonce, *transactionParams.Address, transactionParams.Amount, transactionParams.GasLimit, transactionParams.GasPrice, txDataToSign)
	signedTx, err := wallet.SignTx(*account, tx, chainID)
	if err != nil {
		return nil, err
	}
	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return nil, err
	}

	var signedTxBuff bytes.Buffer
	signedTx.EncodeRLP(&signedTxBuff)

	return &logical.Response{
		Data: map[string]interface{}{
			"transaction_hash":   signedTx.Hash().Hex(),
			"signed_transaction": hexutil.Encode(signedTxBuff.Bytes()),
			"from":               account.Address.Hex(),
			"to":                 transactionParams.Address.String(),
			"amount":             transactionParams.Amount.String(),
			"nonce":              strconv.FormatUint(transactionParams.Nonce, 10),
			"gas_price":          transactionParams.GasPrice.String(),
			"gas_limit":          strconv.FormatUint(transactionParams.GasLimit, 10),
		},
	}, nil
}

func (b *PluginBackend) pathSignTx(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	var txDataToSign []byte
	config, err := b.configured(ctx, req)
	if err != nil {
		return nil, err
	}
	client, err := ethclient.Dial(config.getRPCURL())
	if err != nil {
		return nil, fmt.Errorf("cannot connect to " + config.getRPCURL())
	}

	name := data.Get("name").(string)
	address := data.Get("address").(string)
	accountJSON, err := readAccount(ctx, req, name, address)
	if err != nil || accountJSON == nil {
		return nil, fmt.Errorf("error reading address")
	}
	chainID := util.ValidNumber(config.ChainID)
	if chainID == nil {
		return nil, fmt.Errorf("invalid chain ID")
	}
	dataOrFile := data.Get("data").(string)
	encoding := data.Get("encoding").(string)
	if encoding == "hex" {
		txDataToSign, err = util.Decode([]byte(dataOrFile))
		if err != nil {
			return nil, err
		}
	} else if encoding == "utf8" {
		txDataToSign = []byte(dataOrFile)
	} else {
		return nil, fmt.Errorf("invalid encoding encountered - %s", encoding)
	}
	walletJSON, err := readWallet(ctx, req, name)
	if err != nil {
		return nil, err
	}

	wallet, account, err := getWalletAndAccount(*walletJSON, accountJSON.Index)
	if err != nil {
		return nil, err
	}
	transactionParams, err := b.getData(client, account.Address, data)
	if err != nil {
		return nil, err
	}

	accountJSON.Whitelist = append(accountJSON.Whitelist, config.Whitelist...)
	accountJSON.Whitelist = append(accountJSON.Whitelist, walletJSON.Whitelist...)
	if len(accountJSON.Whitelist) > 0 && !util.Contains(accountJSON.Whitelist, transactionParams.Address.Hex()) {
		return nil, fmt.Errorf("%s violates the whitelist %+v", transactionParams.Address.Hex(), accountJSON.Whitelist)
	}
	err = config.BlackListed(transactionParams.Address)
	if err != nil {
		return nil, err
	}
	err = walletJSON.BlackListed(transactionParams.Address)
	if err != nil {
		return nil, err
	}
	err = accountJSON.BlackListed(transactionParams.Address)
	if err != nil {
		return nil, err
	}

	tx := types.NewTransaction(transactionParams.Nonce, *transactionParams.Address, transactionParams.Amount, transactionParams.GasLimit, transactionParams.GasPrice, txDataToSign)

	signedTx, err := wallet.SignTx(*account, tx, chainID)
	if err != nil {
		return nil, err
	}
	var signedTxBuff bytes.Buffer
	signedTx.EncodeRLP(&signedTxBuff)

	return &logical.Response{
		Data: map[string]interface{}{
			"transaction_hash":   signedTx.Hash().Hex(),
			"signed_transaction": hexutil.Encode(signedTxBuff.Bytes()),
			"from":               account.Address.Hex(),
			"to":                 transactionParams.Address.String(),
			"amount":             transactionParams.Amount.String(),
			"nonce":              strconv.FormatUint(transactionParams.Nonce, 10),
			"gas_price":          transactionParams.GasPrice.String(),
			"gas_limit":          strconv.FormatUint(transactionParams.GasLimit, 10),
		},
	}, nil

}

func (b *PluginBackend) pathReadBalance(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	config, err := b.configured(ctx, req)
	if err != nil {
		return nil, err
	}

	address := data.Get("address").(string)

	client, err := ethclient.Dial(config.getRPCURL())
	if err != nil {
		return nil, err
	}
	balance, err := client.BalanceAt(context.Background(), common.HexToAddress(address), nil)
	if err != nil {
		return nil, err
	}

	return &logical.Response{
		Data: map[string]interface{}{
			"balance": balance.String(),
		},
	}, nil

}

func (b *PluginBackend) pathDeploy(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	config, err := b.configured(ctx, req)
	if err != nil {
		return nil, err
	}

	name := data.Get("name").(string)
	address := data.Get("address").(string)
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
		return nil, fmt.Errorf("cannot connect to " + config.getRPCURL())
	}

	walletJSON, err := readWallet(ctx, req, name)
	if err != nil {
		return nil, err
	}

	wallet, account, err := getWalletAndAccount(*walletJSON, accountJSON.Index)
	if err != nil {
		return nil, err
	}

	transactionParams, err := b.getBaseData(client, account.Address, data, Empty)

	if err != nil {
		return nil, err
	}

	abiData := data.Get("abi").(string)
	parsed, err := abi.JSON(strings.NewReader(abiData))
	if err != nil {
		return nil, err
	}
	binData := data.Get("bin").(string)
	if err != nil {
		return nil, err
	}
	binRaw := common.FromHex(binData)
	transactOpts, err := b.NewWalletTransactor(chainID, wallet, account)
	if err != nil {
		return nil, err
	}
	gasLimitIn := util.ValidNumber(data.Get("gas_limit").(string))
	gasLimit := gasLimitIn.Uint64()

	transactOpts.GasPrice = transactionParams.GasPrice
	transactOpts.Nonce = big.NewInt(int64(transactionParams.Nonce))
	transactOpts.Value = big.NewInt(0) // in wei

	gasLimit, err = util.EstimateGas(transactOpts, parsed, binRaw, client)
	if err != nil {
		return nil, err
	}
	transactOpts.GasLimit = gasLimit
	contractAddress, tx, _, err := bind.DeployContract(transactOpts, parsed, binRaw, client)
	if err != nil {
		return nil, err
	}
	//	b.LogTx(tx)
	var signedTxBuff bytes.Buffer
	tx.EncodeRLP(&signedTxBuff)

	return &logical.Response{
		Data: map[string]interface{}{
			"transaction_hash":   tx.Hash().Hex(),
			"signed_transaction": hexutil.Encode(signedTxBuff.Bytes()),
			"from":               account.Address.Hex(),
			"contract":           contractAddress.Hex(),
			"nonce":              transactOpts.Nonce.String(),
			"gas_price":          transactOpts.GasPrice.String(),
			"gas_limit":          strconv.FormatUint(gasLimit, 10),
		},
	}, nil
}
