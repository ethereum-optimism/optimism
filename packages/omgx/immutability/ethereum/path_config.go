// Copyright (C) OmiseGO - All Rights Reserved
// Unauthorized copying of this file, via any medium is strictly prohibited
// Proprietary and confidential
// Written by Jeff Ploughman <jeff@immutability.io>, October 2019

package ethereum

import (
	"context"
	"fmt"

	"github.com/omgnetwork/immutability-eth-plugin/util"

	"github.com/ethereum/go-ethereum/common"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/helper/cidrutil"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	// EthereumMainnet Chain ID
	EthereumMainnet string = "1"
	// Morden Chain ID
	Morden string = "2"
	// Ropsten Chain ID
	Ropsten string = "3"
	// Rinkeby Chain ID
	Rinkeby string = "4"
	// RootstockMainnet Chain ID
	RootstockMainnet string = "30"
	// RootstockTestnet Chain ID
	RootstockTestnet string = "31"
	// Kovan Chain ID
	Kovan string = "42"
	// EthereumClassicMainnet Chain ID
	EthereumClassicMainnet string = "61"
	// EthereumClassicTestnet Chain ID
	EthereumClassicTestnet string = "62"
	// GethPrivateChains Chain ID
	GethPrivateChains string = "1337"
	// InfuraMainnet is the default for EthereumMainnet
	InfuraMainnet string = "https://mainnet.infura.io"
	// InfuraRopsten is the default for Ropsten
	InfuraRopsten string = "https://ropsten.infura.io"
	// InfuraKovan is the default for Kovan
	InfuraKovan string = "https://kovan.infura.io"
	// InfuraRinkeby is the default for Rinkeby
	InfuraRinkeby string = "https://rinkeby.infura.io"
	// Local is the default for localhost
	Local string = "http://localhost:8545"
)

// ConfigJSON contains the configuration for each mount
type ConfigJSON struct {
	BoundCIDRList []string `json:"bound_cidr_list_list" structs:"bound_cidr_list" mapstructure:"bound_cidr_list"`
	Whitelist     []string `json:"whitelist"`
	Blacklist     []string `json:"blacklist"`
	RPC           string   `json:"rpc_url"`
	ChainID       string   `json:"chain_id"`
	RPCl2         string   `json:"rpc_l2_url"`
	ChainIDl2     string   `json:"chain_l2_id"`
}

// BlackListed returns an error if the address is blacklisted
func (config *ConfigJSON) BlackListed(toAddress *common.Address) error {
	if util.Contains(config.Blacklist, toAddress.Hex()) {
		return fmt.Errorf("%s is blacklisted globally", toAddress.Hex())
	}
	return nil
}

// ConfigPaths implements the Ethereum config paths
func ConfigPaths(b *PluginBackend) []*framework.Path {
	return []*framework.Path{
		{
			Pattern: QualifiedPath("config"),
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.CreateOperation: b.pathWriteConfig,
				logical.UpdateOperation: b.pathWriteConfig,
				logical.ReadOperation:   b.pathReadConfig,
			},
			HelpSynopsis: "Configure the Vault Ethereum plugin.",
			HelpDescription: `
			Configure the Vault Ethereum plugin.
			`,
			Fields: map[string]*framework.FieldSchema{
				"chain_id": {
					Type: framework.TypeString,
					Description: `Ethereum network - can be one of the following values:

					1 - Ethereum mainnet
					2 - Morden (disused), Expanse mainnet
					3 - Ropsten
					4 - Rinkeby
					30 - Rootstock mainnet
					31 - Rootstock testnet
					42 - Kovan
					61 - Ethereum Classic mainnet
					62 - Ethereum Classic testnet
					1337 - Geth private chains (default)`,
				},
				"rpc_url": {
					Type:        framework.TypeString,
					Description: `The RPC address of the Ethereum network.`,
				},
				"chain_l2_id": {
					Type: framework.TypeString,
					Description: `Ethereum L2 network - can be one of the following values:

					1 - Ethereum mainnet
					2 - Morden (disused), Expanse mainnet
					3 - Ropsten
					4 - Rinkeby
					30 - Rootstock mainnet
					31 - Rootstock testnet
					42 - Kovan
					61 - Ethereum Classic mainnet
					62 - Ethereum Classic testnet
					1337 - Geth private chains (default)`,
				},
				"rpc_l2_url": {
					Type:        framework.TypeString,
					Description: `The RPC address of the L2 Ethereum network.`,
				},
				"whitelist": {
					Type:        framework.TypeCommaStringSlice,
					Description: "The list of accounts that any account can send ETH to.",
				},
				"blacklist": {
					Type:        framework.TypeCommaStringSlice,
					Description: "The list of accounts that any account can't send ETH to.",
				},
				"bound_cidr_list": {
					Type: framework.TypeCommaStringSlice,
					Description: `Comma separated string or list of CIDR blocks. If set, specifies the blocks of
IPs which can perform the login operation.`,
				},
			},
		},
	}
}

func (config *ConfigJSON) getRPCURL() string {
	return config.RPC
}

func (b *PluginBackend) pathWriteConfig(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	rpcURL := data.Get("rpc_url").(string)
	if rpcURL == "" {
		return nil, fmt.Errorf("invalid rpc_url")
	}
	rpcURLl2 := data.Get("rpc_l2_url").(string)
	if rpcURLl2 == "" {
		return nil, fmt.Errorf("invalid rpc_l2_url")
	}

	chainID := data.Get("chain_id").(string)
	if chainID == "" {
		return nil, fmt.Errorf("invalid chain_id")
	}
	chainIDl2 := data.Get("chain_l2_id").(string)
	if chainIDl2 == "" {
		return nil, fmt.Errorf("invalid chain_l2_id")
	}
	var boundCIDRList []string
	if boundCIDRListRaw, ok := data.GetOk("bound_cidr_list"); ok {
		boundCIDRList = boundCIDRListRaw.([]string)
	}
	var whiteList []string
	if whiteListRaw, ok := data.GetOk("whitelist"); ok {
		whiteList = whiteListRaw.([]string)
	}
	var blackList []string
	if blackListRaw, ok := data.GetOk("blacklist"); ok {
		blackList = blackListRaw.([]string)
	}
	configBundle := ConfigJSON{
		BoundCIDRList: boundCIDRList,
		Whitelist:     whiteList,
		Blacklist:     blackList,
		ChainID:       chainID,
		RPC:           rpcURL,
		ChainIDl2:     chainIDl2,
		RPCl2:         rpcURLl2,
	}
	entry, err := logical.StorageEntryJSON("config", configBundle)

	if err != nil {
		return nil, err
	}

	if err := req.Storage.Put(ctx, entry); err != nil {
		return nil, err
	}
	// Return the secret
	return &logical.Response{
		Data: map[string]interface{}{
			"bound_cidr_list": configBundle.BoundCIDRList,
			"whitelist":       configBundle.Whitelist,
			"blacklist":       configBundle.Blacklist,
			"rpc_url":         configBundle.RPC,
			"chain_id":        configBundle.ChainID,
			"rpc_l2_url":      configBundle.RPCl2,
			"chain_l2_id":     configBundle.ChainIDl2,
		},
	}, nil
}

func (b *PluginBackend) pathReadConfig(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	configBundle, err := b.readConfig(ctx, req.Storage)
	if err != nil {
		return nil, err
	}

	if configBundle == nil {
		return nil, nil
	}

	// Return the secret
	return &logical.Response{
		Data: map[string]interface{}{
			"bound_cidr_list": configBundle.BoundCIDRList,
			"whitelist":       configBundle.Whitelist,
			"blacklist":       configBundle.Blacklist,
			"rpc_url":         configBundle.RPC,
			"chain_id":        configBundle.ChainID,
			"rpc_l2_url":      configBundle.RPCl2,
			"chain_l2_id":     configBundle.ChainIDl2,
		},
	}, nil
}

// Config returns the configuration for this PluginBackend.
func (b *PluginBackend) readConfig(ctx context.Context, s logical.Storage) (*ConfigJSON, error) {
	entry, err := s.Get(ctx, "config")
	if err != nil {
		return nil, err
	}

	if entry == nil {
		return nil, fmt.Errorf("the Ethereum backend is not configured properly")
	}

	var result ConfigJSON
	if entry != nil {
		if err := entry.DecodeJSON(&result); err != nil {
			return nil, fmt.Errorf("error reading configuration: %s", err)
		}
	}

	return &result, nil
}

func (b *PluginBackend) configured(ctx context.Context, req *logical.Request) (*ConfigJSON, error) {
	config, err := b.readConfig(ctx, req.Storage)
	if err != nil {
		return nil, err
	}
	if validConnection, err := b.validIPConstraints(config, req); !validConnection {
		return nil, err
	}

	return config, nil
}

func (b *PluginBackend) validIPConstraints(config *ConfigJSON, req *logical.Request) (bool, error) {
	if len(config.BoundCIDRList) != 0 {
		if req.Connection == nil || req.Connection.RemoteAddr == "" {
			return false, fmt.Errorf("failed to get connection information")
		}

		belongs, err := cidrutil.IPBelongsToCIDRBlocksSlice(req.Connection.RemoteAddr, config.BoundCIDRList)
		if err != nil {
			return false, errwrap.Wrapf("failed to verify the CIDR restrictions set on the role: {{err}}", err)
		}
		if !belongs {
			return false, fmt.Errorf("source address %q unauthorized through CIDR restrictions on the role", req.Connection.RemoteAddr)
		}
	}
	return true, nil
}
