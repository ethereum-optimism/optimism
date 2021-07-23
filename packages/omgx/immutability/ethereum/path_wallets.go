// Copyright (C) OmiseGO - All Rights Reserved
// Unauthorized copying of this file, via any medium is strictly prohibited
// Proprietary and confidential
// Written by Jeff Ploughman <jeff@immutability.io>, October 2019

package ethereum

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	hdwallet "github.com/immutability-io/go-ethereum-hdwallet"
	"github.com/tyler-smith/go-bip39"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/omgnetwork/immutability-eth-plugin/util"
)

const (
	// DerivationPath is the root in a BIP44 wallet
	DerivationPath string = "m/44'/60'/0'/0/%d"
	// Empty is the empty string
	Empty string = ""
	// Utf8Encoding is utf
	Utf8Encoding string = "utf8"
	// HexEncoding is hex
	HexEncoding string = "hex"
)

// WalletJSON is what we store for an Ethereum account
type WalletJSON struct {
	Index     int      `json:"index"`
	Mnemonic  string   `json:"mnemonic"`
	Whitelist []string `json:"whitelist"`
	Blacklist []string `json:"blacklist"`
}

// BlackListed returns an error if the address is blacklisted
func (wallet *WalletJSON) BlackListed(toAddress *common.Address) error {
	if util.Contains(wallet.Blacklist, toAddress.Hex()) {
		return fmt.Errorf("%s is blacklisted by this wallet", toAddress.Hex())
	}

	return nil
}

// TransactionParams are typical parameters for a transaction
type TransactionParams struct {
	Nonce    uint64          `json:"nonce"`
	Address  *common.Address `json:"address"`
	Amount   *big.Int        `json:"amount"`
	GasPrice *big.Int        `json:"gas_price"`
	GasLimit uint64          `json:"gas_limit"`
}

// WalletPaths are the path handlers for Ethereum wallets
func WalletPaths(b *PluginBackend) []*framework.Path {
	return []*framework.Path{
		{
			Pattern: QualifiedPath("wallets/?"),
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.ListOperation: b.pathWalletsList,
			},
			HelpSynopsis: "List all the Ethereum wallets at a path",
			HelpDescription: `
			All the Ethereum wallets will be listed.
			`,
		},
		{
			Pattern:      QualifiedPath("wallets/" + framework.GenericNameRegex("name")),
			HelpSynopsis: "Create an Ethereum wallet using a generated or provided passphrase.",
			HelpDescription: `

Creates (or updates) an Ethereum wallet: an wallet controlled by a private key. Also
The generator produces a high-entropy passphrase with the provided length and requirements.

`,
			Fields: map[string]*framework.FieldSchema{
				"name": {Type: framework.TypeString},
				"mnemonic": {
					Type:        framework.TypeString,
					Default:     Empty,
					Description: "The mnemonic to use to create the account. If not provided, one is generated.",
				},
				// whitelisting and blacklisting are not implemented in this release
				"whitelist": {
					Type:        framework.TypeCommaStringSlice,
					Description: "The list of wallets that this wallet can send transactions to.",
				},
				"blacklist": {
					Type:        framework.TypeCommaStringSlice,
					Description: "The list of wallets that this wallet can't send transactions to.",
				},
			},
			ExistenceCheck: pathExistenceCheck,
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.ReadOperation:   b.pathWalletsRead,
				logical.CreateOperation: b.pathWalletsCreate,
			},
		},
	}
}

func (b *PluginBackend) pathWalletsList(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	vals, err := req.Storage.List(ctx, QualifiedPath("wallets/"))
	if err != nil {
		return nil, err
	}
	return logical.ListResponse(vals), nil
}

func readWallet(ctx context.Context, req *logical.Request, name string) (*WalletJSON, error) {
	path := QualifiedPath(fmt.Sprintf("wallets/%s", name))
	entry, err := req.Storage.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, nil
	}

	var walletJSON WalletJSON
	err = entry.DecodeJSON(&walletJSON)

	if entry == nil {
		return nil, fmt.Errorf("failed to deserialize wallet at %s", path)
	}
	return &walletJSON, nil
}

func (b *PluginBackend) pathWalletsRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	_, err := b.configured(ctx, req)
	if err != nil {
		return nil, err
	}
	name := data.Get("name").(string)
	walletJSON, err := readWallet(ctx, req, name)
	if err != nil || walletJSON == nil {
		return nil, fmt.Errorf("Error reading wallet")
	}

	return &logical.Response{
		Data: map[string]interface{}{
			"index":     walletJSON.Index,
			"whitelist": walletJSON.Whitelist,
			"blacklist": walletJSON.Blacklist,
		},
	}, nil
}

func getWalletAndAccount(walletJSON WalletJSON, index int) (*hdwallet.Wallet, *accounts.Account, error) {
	wallet, err := hdwallet.NewFromMnemonic(walletJSON.Mnemonic)
	if err != nil {
		return nil, nil, err
	}
	derivationPath := fmt.Sprintf(DerivationPath, index)
	path := hdwallet.MustParseDerivationPath(derivationPath)
	account, err := wallet.Derive(path, true)
	if err != nil {
		return nil, nil, err
	}
	return wallet, &account, nil
}

func (b *PluginBackend) pathWalletsCreate(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	_, err := b.configured(ctx, req)
	if err != nil {
		return nil, err
	}
	name := data.Get("name").(string)
	var whiteList []string
	if whiteListRaw, ok := data.GetOk("whitelist"); ok {
		whiteList = whiteListRaw.([]string)
	}
	var blackList []string
	if blackListRaw, ok := data.GetOk("blacklist"); ok {
		blackList = blackListRaw.([]string)
	}
	mnemonic := data.Get("mnemonic").(string)
	if mnemonic == Empty {
		entropy, err := bip39.NewEntropy(128)
		if err != nil {
			return nil, err
		}

		mnemonic, err = bip39.NewMnemonic(entropy)
	}

	if err != nil {
		return nil, err
	}
	walletJSON := &WalletJSON{
		Index:     0,
		Mnemonic:  mnemonic,
		Whitelist: util.Dedup(whiteList),
		Blacklist: util.Dedup(blackList),
	}

	err = b.updateWallet(ctx, req, name, walletJSON)
	if err != nil {
		return nil, err
	}

	return &logical.Response{
		Data: map[string]interface{}{
			"index":     walletJSON.Index,
			"whitelist": walletJSON.Whitelist,
			"blacklist": walletJSON.Blacklist,
		},
	}, nil
}

func (b *PluginBackend) updateWallet(ctx context.Context, req *logical.Request, name string, walletJSON *WalletJSON) error {
	path := QualifiedPath(fmt.Sprintf("wallets/%s", name))

	entry, err := logical.StorageEntryJSON(path, walletJSON)
	if err != nil {
		return err
	}

	err = req.Storage.Put(ctx, entry)
	if err != nil {
		return err
	}
	return nil
}

func pathExistenceCheck(ctx context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
	out, err := req.Storage.Get(ctx, req.Path)
	if err != nil {
		return false, fmt.Errorf("existence check failed: %v", err)
	}

	return out != nil, nil
}

// returns (nonce, toAddress, amount, gasPrice, gasLimit, error)

func (b *PluginBackend) getData(client *ethclient.Client, fromAddress common.Address, data *framework.FieldData) (*TransactionParams, error) {
	transactionParams, err := b.getBaseData(client, fromAddress, data, "to")
	if err != nil {
		return nil, err
	}
	var gasLimitIn *big.Int

	gasLimitIn = util.ValidNumber(data.GetDefaultOrZero("gas_limit").(string))
	gasLimit := gasLimitIn.Uint64()

	return &TransactionParams{
		Nonce:    transactionParams.Nonce,
		Address:  transactionParams.Address,
		Amount:   transactionParams.Amount,
		GasPrice: transactionParams.GasPrice,
		GasLimit: gasLimit,
	}, nil
}

// NewWalletTransactor is used with Token contracts
func (b *PluginBackend) NewWalletTransactor(chainID *big.Int, wallet *hdwallet.Wallet, account *accounts.Account) (*bind.TransactOpts, error) {
	return &bind.TransactOpts{
		From: account.Address,
		Signer: func(signer types.Signer, address common.Address, tx *types.Transaction) (*types.Transaction, error) {
			if address != account.Address {
				return nil, errors.New("not authorized to sign this account")
			}
			signedTx, err := wallet.SignTx(*account, tx, chainID)
			if err != nil {
				return nil, err
			}

			return signedTx, nil
		},
	}, nil
}

func (b *PluginBackend) getBaseData(client *ethclient.Client, fromAddress common.Address, data *framework.FieldData, addressField string) (*TransactionParams, error) {
	var err error
	var address common.Address
	nonceData := "0"
	var nonce uint64
	var amount *big.Int
	var gasPriceIn *big.Int
	_, ok := data.GetOk("amount")
	if ok {
		amount = util.ValidNumber(data.Get("amount").(string))
		if amount == nil {
			return nil, fmt.Errorf("invalid amount")
		}
	} else {
		amount = util.ValidNumber("0")
	}

	_, ok = data.GetOk("nonce")
	if ok {
		nonceData = data.Get("nonce").(string)
		nonceIn := util.ValidNumber(nonceData)
		nonce = nonceIn.Uint64()
	} else {
		nonce, err = client.PendingNonceAt(context.Background(), fromAddress)
		if err != nil {
			return nil, err
		}
	}

	_, ok = data.GetOk("gas_price")
	if ok {
		gasPriceIn = util.ValidNumber(data.Get("gas_price").(string))
		if gasPriceIn == nil {
			return nil, fmt.Errorf("invalid gas price")
		}
	} else {
		gasPriceIn = util.ValidNumber("0")
	}

	if big.NewInt(0).Cmp(gasPriceIn) == 0 {
		gasPriceIn, err = client.SuggestGasPrice(context.Background())
		if err != nil {
			return nil, err
		}
	}

	if addressField != Empty {
		address = common.HexToAddress(data.Get(addressField).(string))
		return &TransactionParams{
			Nonce:    nonce,
			Address:  &address,
			Amount:   amount,
			GasPrice: gasPriceIn,
			GasLimit: 0,
		}, nil
	}
	return &TransactionParams{
		Nonce:    nonce,
		Address:  nil,
		Amount:   amount,
		GasPrice: gasPriceIn,
		GasLimit: 0,
	}, nil

}

// LogTx is for debugging
func (b *PluginBackend) LogTx(tx *types.Transaction) {
	b.Logger().Info(fmt.Sprintf("\nTX DATA: %s\nGAS: %d\nGAS PRICE: %d\nVALUE: %d\nNONCE: %d\nTO: %s\n", hexutil.Encode(tx.Data()), tx.Gas(), tx.GasPrice(), tx.Value(), tx.Nonce(), tx.To().Hex()))
}
