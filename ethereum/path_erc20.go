// Copyright (C) Immutability, LLC - All Rights Reserved
// Unauthorized copying of this file, via any medium is strictly prohibited
// Proprietary and confidential
// Written by Jeff Ploughman <jeff@immutability.io>, August 2019

package ethereum

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/omisego/immutability-eth-plugin/contracts/erc20"
	"github.com/omisego/immutability-eth-plugin/util"
)

const erc20Contract string = "erc-20"

// contract ERC20Interface {
//     string public constant name = "";
//     string public constant symbol = "";
//     uint8 public constant decimals = 0;

//     function totalSupply() public view returns (uint);
//     function balanceOf(address tokenOwner) public view returns (uint balance);
//     function allowance(address tokenOwner, address spender) public view returns (uint remaining);
//     function transfer(address to, uint tokens) public returns (bool success);
//     function approve(address spender, uint tokens) public returns (bool success);
//     function transferFrom(address from, address to, uint tokens) public returns (bool success);

//     event Transfer(address indexed from, address indexed to, uint tokens);
//     event Approval(address indexed tokenOwner, address indexed spender, uint tokens);
// }

// ERC20Paths are the path handlers for Ethereum wallets
func ERC20Paths(b *PluginBackend) []*framework.Path {
	return []*framework.Path{
		&framework.Path{
			Pattern:      ContractPath(erc20Contract, "balanceOf"),
			HelpSynopsis: "Return the balance for an address's ERC-20 holdings",
			HelpDescription: `

Return the balance for an address's ERC-20 holdings.

`,
			Fields: map[string]*framework.FieldSchema{
				"name":    &framework.FieldSchema{Type: framework.TypeString},
				"address": &framework.FieldSchema{Type: framework.TypeString},
				"contract": &framework.FieldSchema{
					Type:        framework.TypeString,
					Description: "The address of the ERC-20 token.",
				},
			},
			ExistenceCheck: pathExistenceCheck,
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.ReadOperation: b.pathERC20BalanceOf,
			},
		},
		&framework.Path{
			Pattern:      ContractPath(erc20Contract, "totalSupply"),
			HelpSynopsis: "Return the balance for an address's ERC-20 holdings",
			HelpDescription: `

Return the total supply for a ERC-20 token.

`,
			Fields: map[string]*framework.FieldSchema{
				"name":    &framework.FieldSchema{Type: framework.TypeString},
				"address": &framework.FieldSchema{Type: framework.TypeString},
				"contract": &framework.FieldSchema{
					Type:        framework.TypeString,
					Description: "The address of the ERC-20 token.",
				},
			},
			ExistenceCheck: pathExistenceCheck,
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.ReadOperation: b.pathERC20TotalSupply,
			},
		},

		&framework.Path{
			Pattern:      ContractPath(erc20Contract, "transfer"),
			HelpSynopsis: "Transfer some ERC-20 holdings to another address",
			HelpDescription: `

Transfer some ERC-20 holdings to another address.

`,
			Fields: map[string]*framework.FieldSchema{
				"name":    &framework.FieldSchema{Type: framework.TypeString},
				"address": &framework.FieldSchema{Type: framework.TypeString},
				"contract": &framework.FieldSchema{
					Type:        framework.TypeString,
					Description: "The address of the ERC-20 token.",
				},
				"to": &framework.FieldSchema{
					Type:        framework.TypeString,
					Description: "The address of the wallet to send tokens to.",
				},
				"tokens": &framework.FieldSchema{
					Type:        framework.TypeString,
					Default:     "0",
					Description: "The number of tokens to transfer.",
				},
			},
			ExistenceCheck: pathExistenceCheck,
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.CreateOperation: b.pathERC20Transfer,
				logical.UpdateOperation: b.pathERC20Transfer,
			},
		},
		&framework.Path{
			Pattern:      ContractPath(erc20Contract, "transferFrom"),
			HelpSynopsis: "Transfer some ERC-20 holdings from another address to this address",
			HelpDescription: `

Transfer some ERC-20 holdings from another address to this address.

`,
			Fields: map[string]*framework.FieldSchema{
				"name":    &framework.FieldSchema{Type: framework.TypeString},
				"address": &framework.FieldSchema{Type: framework.TypeString},
				"contract": &framework.FieldSchema{
					Type:        framework.TypeString,
					Description: "The address of the ERC-20 token.",
				},
				"from": &framework.FieldSchema{
					Type:        framework.TypeString,
					Description: "The address of the wallet to send tokens from.",
				},
				"tokens": &framework.FieldSchema{
					Type:        framework.TypeString,
					Default:     "0",
					Description: "The number of tokens to transfer.",
				},
			},
			ExistenceCheck: pathExistenceCheck,
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.CreateOperation: b.pathERC20TransferFrom,
				logical.UpdateOperation: b.pathERC20TransferFrom,
			},
		},
		&framework.Path{
			Pattern:      ContractPath(erc20Contract, "approve"),
			HelpSynopsis: "Allow spender to withdraw from your account",
			HelpDescription: `

Allow spender to withdraw from your account, multiple times, up to the tokens amount.
If this function is called again it overwrites the current allowance with _value.

`,
			Fields: map[string]*framework.FieldSchema{
				"name":    &framework.FieldSchema{Type: framework.TypeString},
				"address": &framework.FieldSchema{Type: framework.TypeString},
				"contract": &framework.FieldSchema{
					Type:        framework.TypeString,
					Description: "The address of the ERC-20 token.",
				},
				"spender": &framework.FieldSchema{
					Type:        framework.TypeString,
					Description: "The address of the spender.",
				},
				"tokens": &framework.FieldSchema{
					Type:        framework.TypeString,
					Default:     "0",
					Description: "The number of tokens to transfer.",
				},
			},
			ExistenceCheck: pathExistenceCheck,
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.CreateOperation: b.pathERC20Approve,
				logical.UpdateOperation: b.pathERC20Approve,
			},
		},
	}
}

func (b *PluginBackend) pathERC20BalanceOf(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	config, err := b.configured(ctx, req)
	if err != nil {
		return nil, err
	}
	ownerAddress := common.HexToAddress(data.Get("address").(string))

	client, err := ethclient.Dial(config.getRPCURL())
	if err != nil {
		return nil, err
	}

	contractAddress := common.HexToAddress(data.Get("contract").(string))
	instance, err := erc20.NewErc20(contractAddress, client)
	if err != nil {
		return nil, err
	}
	callOpts := &bind.CallOpts{}
	erc20CallerSession := &erc20.Erc20CallerSession{
		Contract: &instance.Erc20Caller, // Generic contract caller binding to set the session for
		CallOpts: *callOpts,             // Call options to use throughout this session
	}

	bal, err := erc20CallerSession.BalanceOf(ownerAddress)
	if err != nil {
		return nil, err
	}

	tokenName, err := erc20CallerSession.Name()
	if err != nil {
		return nil, err
	}

	symbol, err := erc20CallerSession.Symbol()
	if err != nil {
		return nil, err
	}

	decimals, err := erc20CallerSession.Decimals()
	if err != nil {
		return nil, err
	}

	fbal := new(big.Float)
	fbal.SetString(bal.String())
	value := new(big.Float).Quo(fbal, big.NewFloat(math.Pow10(int(decimals))))

	return &logical.Response{
		Data: map[string]interface{}{
			"contract": contractAddress.Hex(),
			"symbol":   symbol,
			"name":     tokenName,
			"balance":  value,
		},
	}, nil

}

func (b *PluginBackend) pathERC20Transfer(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	var tokens *big.Int
	config, err := b.configured(ctx, req)
	if err != nil {
		return nil, err
	}
	address := data.Get("address").(string)
	name := data.Get("name").(string)
	tokenAddress := common.HexToAddress(data.Get("contract").(string))
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

	instance, err := erc20.NewErc20(tokenAddress, client)
	if err != nil {
		return nil, err
	}
	callOpts := &bind.CallOpts{}

	erc20CallerSession := &erc20.Erc20CallerSession{
		Contract: &instance.Erc20Caller, // Generic contract caller binding to set the session for
		CallOpts: *callOpts,             // Call options to use throughout this session
	}

	tokenName, err := erc20CallerSession.Name()
	if err != nil {
		return nil, err
	}

	symbol, err := erc20CallerSession.Symbol()
	if err != nil {
		return nil, err
	}

	decimals, err := erc20CallerSession.Decimals()
	if err != nil {
		return nil, err
	}

	transactionParams, err := b.getBaseData(client, account.Address, data, "to")
	if err != nil {
		return nil, err
	}
	_, ok := data.GetOk("tokens")
	if ok {
		tokens = util.ValidNumber(data.Get("tokens").(string))
		if tokens == nil {
			return nil, fmt.Errorf("number of tokens are required")
		}
	} else {
		tokens = util.ValidNumber("0")
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

	tokenAmount := util.TokenAmount(tokens.Int64(), decimals)
	transactOpts, err := b.NewWalletTransactor(chainID, wallet, account)
	if err != nil {
		return nil, err
	}

	//transactOpts needs gas etc.
	tokenSession := &erc20.Erc20Session{
		Contract:     instance,  // Generic contract caller binding to set the session for
		CallOpts:     *callOpts, // Call options to use throughout this session
		TransactOpts: *transactOpts,
	}

	tx, err := tokenSession.Transfer(*transactionParams.Address, tokenAmount)
	if err != nil {
		return nil, err
	}

	var signedTxBuff bytes.Buffer
	tx.EncodeRLP(&signedTxBuff)
	return &logical.Response{
		Data: map[string]interface{}{
			"contract":           tokenAddress.Hex(),
			"symbol":             symbol,
			"name":               tokenName,
			"transaction_hash":   tx.Hash().Hex(),
			"signed_transaction": hexutil.Encode(signedTxBuff.Bytes()),
			"from":               account.Address.Hex(),
			"to":                 transactionParams.Address.String(),
			"amount":             tokenAmount.String(),
			"nonce":              tx.Nonce(),
			"gas_price":          tx.GasPrice(),
			"gas_limit":          tx.Gas(),
		},
	}, nil

}

func (b *PluginBackend) pathERC20TotalSupply(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	config, err := b.configured(ctx, req)
	if err != nil {
		return nil, err
	}

	client, err := ethclient.Dial(config.getRPCURL())
	if err != nil {
		return nil, err
	}

	contractAddress := common.HexToAddress(data.Get("contract").(string))
	instance, err := erc20.NewErc20(contractAddress, client)
	if err != nil {
		return nil, err
	}
	callOpts := &bind.CallOpts{}
	erc20CallerSession := &erc20.Erc20CallerSession{
		Contract: &instance.Erc20Caller, // Generic contract caller binding to set the session for
		CallOpts: *callOpts,             // Call options to use throughout this session
	}

	totalSupply, err := erc20CallerSession.TotalSupply()
	if err != nil {
		return nil, err
	}

	tokenName, err := erc20CallerSession.Name()
	if err != nil {
		return nil, err
	}

	symbol, err := erc20CallerSession.Symbol()
	if err != nil {
		return nil, err
	}

	decimals, err := erc20CallerSession.Decimals()
	if err != nil {
		return nil, err
	}

	fbal := new(big.Float)
	fbal.SetString(totalSupply.String())
	value := new(big.Float).Quo(fbal, big.NewFloat(math.Pow10(int(decimals))))

	return &logical.Response{
		Data: map[string]interface{}{
			"contract":     contractAddress.Hex(),
			"symbol":       symbol,
			"name":         tokenName,
			"total_supply": fmt.Sprintf("%.0f", value),
		},
	}, nil

}

func (b *PluginBackend) pathERC20Approve(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	var tokens *big.Int
	config, err := b.configured(ctx, req)
	if err != nil {
		return nil, err
	}
	address := data.Get("address").(string)
	name := data.Get("name").(string)
	tokenAddress := common.HexToAddress(data.Get("contract").(string))
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

	instance, err := erc20.NewErc20(tokenAddress, client)
	if err != nil {
		return nil, err
	}
	callOpts := &bind.CallOpts{}

	erc20CallerSession := &erc20.Erc20CallerSession{
		Contract: &instance.Erc20Caller, // Generic contract caller binding to set the session for
		CallOpts: *callOpts,             // Call options to use throughout this session
	}

	tokenName, err := erc20CallerSession.Name()
	if err != nil {
		return nil, err
	}

	symbol, err := erc20CallerSession.Symbol()
	if err != nil {
		return nil, err
	}

	decimals, err := erc20CallerSession.Decimals()
	if err != nil {
		return nil, err
	}

	transactionParams, err := b.getBaseData(client, account.Address, data, "spender")
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

	_, ok := data.GetOk("tokens")
	if ok {
		tokens = util.ValidNumber(data.Get("tokens").(string))
		if tokens == nil {
			return nil, fmt.Errorf("number of tokens are required")
		}
	} else {
		tokens = util.ValidNumber("0")
	}
	tokenAmount := util.TokenAmount(tokens.Int64(), decimals)
	transactOpts, err := b.NewWalletTransactor(chainID, wallet, account)
	if err != nil {
		return nil, err
	}

	//transactOpts needs gas etc.
	tokenSession := &erc20.Erc20Session{
		Contract:     instance,  // Generic contract caller binding to set the session for
		CallOpts:     *callOpts, // Call options to use throughout this session
		TransactOpts: *transactOpts,
	}

	tx, err := tokenSession.Approve(*transactionParams.Address, tokenAmount)
	if err != nil {
		return nil, err
	}

	var signedTxBuff bytes.Buffer
	tx.EncodeRLP(&signedTxBuff)
	return &logical.Response{
		Data: map[string]interface{}{
			"contract":           tokenAddress.Hex(),
			"symbol":             symbol,
			"name":               tokenName,
			"transaction_hash":   tx.Hash().Hex(),
			"signed_transaction": hexutil.Encode(signedTxBuff.Bytes()),
			"from":               account.Address.Hex(),
			"to":                 transactionParams.Address.String(),
			"amount":             tokenAmount.String(),
			"nonce":              tx.Nonce(),
			"gas_price":          tx.GasPrice(),
			"gas_limit":          tx.Gas(),
		},
	}, nil

}
func (b *PluginBackend) pathERC20TransferFrom(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	var tokens *big.Int
	config, err := b.configured(ctx, req)
	if err != nil {
		return nil, err
	}
	address := data.Get("address").(string)
	name := data.Get("name").(string)
	tokenAddress := common.HexToAddress(data.Get("contract").(string))
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

	instance, err := erc20.NewErc20(tokenAddress, client)
	if err != nil {
		return nil, err
	}
	callOpts := &bind.CallOpts{}

	erc20CallerSession := &erc20.Erc20CallerSession{
		Contract: &instance.Erc20Caller, // Generic contract caller binding to set the session for
		CallOpts: *callOpts,             // Call options to use throughout this session
	}

	tokenName, err := erc20CallerSession.Name()
	if err != nil {
		return nil, err
	}

	symbol, err := erc20CallerSession.Symbol()
	if err != nil {
		return nil, err
	}

	decimals, err := erc20CallerSession.Decimals()
	if err != nil {
		return nil, err
	}

	transactionParams, err := b.getBaseData(client, account.Address, data, "from")
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

	_, ok := data.GetOk("tokens")
	if ok {
		tokens = util.ValidNumber(data.Get("tokens").(string))
		if tokens == nil {
			return nil, fmt.Errorf("number of tokens are required")
		}
	} else {
		tokens = util.ValidNumber("0")
	}
	tokenAmount := util.TokenAmount(tokens.Int64(), decimals)
	transactOpts, err := b.NewWalletTransactor(chainID, wallet, account)
	if err != nil {
		return nil, err
	}

	//transactOpts needs gas etc.
	tokenSession := &erc20.Erc20Session{
		Contract:     instance,  // Generic contract caller binding to set the session for
		CallOpts:     *callOpts, // Call options to use throughout this session
		TransactOpts: *transactOpts,
	}

	tx, err := tokenSession.TransferFrom(*transactionParams.Address, account.Address, tokenAmount)
	if err != nil {
		return nil, err
	}

	var signedTxBuff bytes.Buffer
	tx.EncodeRLP(&signedTxBuff)
	return &logical.Response{
		Data: map[string]interface{}{
			"contract":           tokenAddress.Hex(),
			"symbol":             symbol,
			"name":               tokenName,
			"transaction_hash":   tx.Hash().Hex(),
			"signed_transaction": hexutil.Encode(signedTxBuff.Bytes()),
			"from":               account.Address.Hex(),
			"to":                 transactionParams.Address.String(),
			"amount":             tokenAmount.String(),
			"nonce":              tx.Nonce(),
			"gas_price":          tx.GasPrice(),
			"gas_limit":          tx.Gas(),
		},
	}, nil

}
