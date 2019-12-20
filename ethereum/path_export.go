// Copyright (C) OmiseGO - All Rights Reserved
// Unauthorized copying of this file, via any medium is strictly prohibited
// Proprietary and confidential
// Written by Jeff Ploughman <jeff@immutability.io>, October 2019

package ethereum

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/pborman/uuid"
	"github.com/tyler-smith/go-bip39"
	"github.com/omisego/immutability-eth-plugin/util"
)

// ExportPaths are the path handlers for Ethereum wallets
func ExportPaths(b *PluginBackend) []*framework.Path {
	return []*framework.Path{
		&framework.Path{
			Pattern:      QualifiedPath("export/" + framework.GenericNameRegex("name") + "/accounts/" + framework.GenericNameRegex("address")),
			HelpSynopsis: "Export a JSON keystore for an account. ",
			HelpDescription: `

Exports the JSON keystore for an account.

`,
			Fields: map[string]*framework.FieldSchema{
				"name":    &framework.FieldSchema{Type: framework.TypeString},
				"address": &framework.FieldSchema{Type: framework.TypeString},
				"passphrase": &framework.FieldSchema{
					Type:        framework.TypeString,
					Default:     Empty,
					Description: "The passphrase to use to encrypt the keystore. If not provided, one is generated and returned.",
				},
				"path": &framework.FieldSchema{
					Type:        framework.TypeString,
					Description: "Path to the keystore directory - not the parent directory.",
				},
			},
			ExistenceCheck: pathExistenceCheck,
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.CreateOperation: b.pathExportAccount,
			},
		},
	}
}

func (b *PluginBackend) pathExportAccount(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	_, err := b.configured(ctx, req)
	if err != nil {
		return nil, err
	}
	returnPassphrase := false
	id := uuid.NewRandom()
	name := data.Get("name").(string)
	address := data.Get("address").(string)
	walletJSON, err := readWallet(ctx, req, name)
	if err != nil {
		return nil, err
	}
	accountJSON, err := readAccount(ctx, req, name, address)
	if err != nil || accountJSON == nil {
		return nil, fmt.Errorf("error reading address")
	}

	wallet, account, err := getWalletAndAccount(*walletJSON, accountJSON.Index)
	if err != nil {
		return nil, err
	}
	key, err := wallet.PrivateKeyHex(*account)
	if err != nil {
		return nil, err
	}
	privateKey, err := crypto.HexToECDSA(key)
	if err != nil {
		return nil, err
	}

	keystorePath := data.Get("path").(string)
	defer util.ZeroKey(privateKey)

	passphrase := data.Get("passphrase").(string)
	if passphrase == Empty {
		entropy, err := bip39.NewEntropy(128)
		if err != nil {
			return nil, err
		}

		passphrase, err = bip39.NewMnemonic(entropy)
		returnPassphrase = true
	}

	if err != nil {
		return nil, err
	}

	jsonBytes, err := util.EncryptKey(privateKey, &account.Address, id, passphrase, keystore.StandardScryptN, keystore.StandardScryptP)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(keystorePath, util.KeyFileName(account.Address))

	util.WriteKeyFile(path, jsonBytes)
	if !returnPassphrase {
		passphrase = "*********"
	}
	return &logical.Response{
		Data: map[string]interface{}{
			"path":       path,
			"passphrase": passphrase,
		},
	}, nil
}
