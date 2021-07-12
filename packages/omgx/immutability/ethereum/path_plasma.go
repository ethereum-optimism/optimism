// Copyright (C) Immutability, LLC - All Rights Reserved
// Unauthorized copying of this file, via any medium is strictly prohibited
// Proprietary and confidential
// Written by Jeff Ploughman <jeff@immutability.io>, August 2019

package ethereum

import (
	"bytes"
	"context"
	b64 "encoding/base64"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/omgnetwork/immutability-eth-plugin/contracts/plasma"
	"github.com/omgnetwork/immutability-eth-plugin/util"
)

const plasmaContract string = "plasma"

// PlasmaPaths are the path handlers for Ethereum wallets
func PlasmaPaths(b *PluginBackend) []*framework.Path {
	return []*framework.Path{

		{
			Pattern:         ContractPath(plasmaContract, "submitBlock"),
			HelpSynopsis:    "Submits the Merkle root of a Plasma block",
			HelpDescription: `Allows the authority to submit the Merkle root of a Plasma block.`,
			Fields: map[string]*framework.FieldSchema{
				"name":    {Type: framework.TypeString},
				"address": {Type: framework.TypeString},
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
				"block_root": {
					Type:        framework.TypeString,
					Description: "The Merkle root of a Plasma block.",
				},
			},
			ExistenceCheck: pathExistenceCheck,
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.CreateOperation: b.pathPlasmaSubmitBlock,
				logical.UpdateOperation: b.pathPlasmaSubmitBlock,
			},
		},
	}
}

func (b *PluginBackend) pathPlasmaSubmitBlock(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
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

	instance, err := plasma.NewPlasma(contractAddress, client)
	if err != nil {
		return nil, err
	}
	callOpts := &bind.CallOpts{}

	inputBlockRoot, ok := data.GetOk("block_root")
	if !ok {
		return nil, fmt.Errorf("invalid block root")
	}
	blockRoot, err := b64.StdEncoding.DecodeString(inputBlockRoot.(string))
	if err != nil {
		return nil, fmt.Errorf("invalid block root - not base64")
	}
	if len(blockRoot) != 32 {
		return nil, fmt.Errorf("invalid block root - not the right size")
	}
	blockRootSB := [32]byte{}
	copy(blockRootSB[:], blockRoot[0:32])

	transactOpts, err := b.NewWalletTransactor(chainID, wallet, account)
	if err != nil {
		return nil, err
	}
	//transactOpts needs gas etc. Use supplied gas_price
	gasPriceRaw := data.Get("gas_price").(string)
	if gasPriceRaw == "" {
		return nil, fmt.Errorf("invalid gas_price")
	}
	transactOpts.GasPrice = util.ValidNumber(gasPriceRaw)

	//transactOpts needs nonce. Use supplied nonce
	nonceRaw := data.Get("nonce").(string)
	if nonceRaw == "" {
		return nil, fmt.Errorf("invalid nonce")
	}
	transactOpts.Nonce = util.ValidNumber(nonceRaw)

	plasmaSession := &plasma.PlasmaSession{
		Contract:     instance,  // Generic contract caller binding to set the session for
		CallOpts:     *callOpts, // Call options to use throughout this session
		TransactOpts: *transactOpts,
	}

	tx, err := plasmaSession.SubmitBlock(blockRootSB)
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
