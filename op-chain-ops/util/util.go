package util

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/urfave/cli/v2"
)

var (
	// EIP1976ImplementationSlot
	EIP1967ImplementationSlot = common.HexToHash("0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc")
	// EIP1967AdminSlot
	EIP1967AdminSlot = common.HexToHash("0xb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d6103")
)

// clients represents a set of initialized RPC clients
type Clients struct {
	L1Client     *ethclient.Client
	L2Client     *ethclient.Client
	L1RpcClient  *rpc.Client
	L2RpcClient  *rpc.Client
	L1GethClient *gethclient.Client
	L2GethClient *gethclient.Client
}

// NewClients will create new RPC clients from a CLI context
func NewClients(ctx *cli.Context) (*Clients, error) {
	clients := Clients{}

	l1RpcURL := ctx.String("l1-rpc-url")
	if l1RpcURL != "" {
		l1Client, l1RpcClient, l1GethClient, err := newClients(l1RpcURL)
		if err != nil {
			return nil, err
		}
		clients.L1Client = l1Client
		clients.L1RpcClient = l1RpcClient
		clients.L1GethClient = l1GethClient
	}

	l2RpcURL := ctx.String("l2-rpc-url")
	if l2RpcURL != "" {
		l2Client, l2RpcClient, l2GethClient, err := newClients(l2RpcURL)
		if err != nil {
			return nil, err
		}
		clients.L2Client = l2Client
		clients.L2RpcClient = l2RpcClient
		clients.L2GethClient = l2GethClient
	}

	return &clients, nil
}

// newClients will create new clients from a given URL
func newClients(url string) (*ethclient.Client, *rpc.Client, *gethclient.Client, error) {
	ethClient, err := ethclient.Dial(url)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("cannot dial ethclient: %w", err)
	}
	rpcClient, err := rpc.DialContext(context.Background(), url)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("cannot dial rpc client: %w", err)
	}
	return ethClient, rpcClient, gethclient.New(rpcClient), nil
}

// ClientsFlags represent the flags associated with creating RPC clients.
var ClientsFlags = []cli.Flag{
	&cli.StringFlag{
		Name:     "l1-rpc-url",
		Required: true,
		Usage:    "L1 RPC URL",
		EnvVars:  []string{"L1_RPC_URL"},
	},
	&cli.StringFlag{
		Name:     "l2-rpc-url",
		Required: true,
		Usage:    "L2 RPC URL",
		EnvVars:  []string{"L2_RPC_URL"},
	},
}

// Addresses represents the address values of various contracts. The values can
// be easily populated via a [cli.Context].
type Addresses struct {
	AddressManager            common.Address
	OptimismPortal            common.Address
	L1StandardBridge          common.Address
	L1CrossDomainMessenger    common.Address
	CanonicalTransactionChain common.Address
	StateCommitmentChain      common.Address
}

// AddressesFlags represent the flags associated with address parsing.
var AddressesFlags = []cli.Flag{
	&cli.StringFlag{
		Name:    "address-manager-address",
		Usage:   "AddressManager address",
		EnvVars: []string{"ADDRESS_MANAGER_ADDRESS"},
	},
	&cli.StringFlag{
		Name:    "optimism-portal-address",
		Usage:   "OptimismPortal address",
		EnvVars: []string{"OPTIMISM_PORTAL_ADDRESS"},
	},
	&cli.StringFlag{
		Name:    "l1-standard-bridge-address",
		Usage:   "L1StandardBridge address",
		EnvVars: []string{"L1_STANDARD_BRIDGE_ADDRESS"},
	},
	&cli.StringFlag{
		Name:    "l1-crossdomain-messenger-address",
		Usage:   "L1CrossDomainMessenger address",
		EnvVars: []string{"L1_CROSSDOMAIN_MESSENGER_ADDRESS"},
	},
	&cli.StringFlag{
		Name:    "canonical-transaction-chain-address",
		Usage:   "CanonicalTransactionChain address",
		EnvVars: []string{"CANONICAL_TRANSACTION_CHAIN_ADDRESS"},
	},
	&cli.StringFlag{
		Name:    "state-commitment-chain-address",
		Usage:   "StateCommitmentChain address",
		EnvVars: []string{"STATE_COMMITMENT_CHAIN_ADDRESS"},
	},
}

// NewAddresses populates an Addresses struct given a [cli.Context].
// This is useful for writing scripts that interact with smart contracts.
func NewAddresses(ctx *cli.Context) (*Addresses, error) {
	var addresses Addresses
	var err error

	addresses.AddressManager, err = parseAddress(ctx, "address-manager-address")
	if err != nil {
		return nil, err
	}
	addresses.OptimismPortal, err = parseAddress(ctx, "optimism-portal-address")
	if err != nil {
		return nil, err
	}
	addresses.L1StandardBridge, err = parseAddress(ctx, "l1-standard-bridge-address")
	if err != nil {
		return nil, err
	}
	addresses.L1CrossDomainMessenger, err = parseAddress(ctx, "l1-crossdomain-messenger-address")
	if err != nil {
		return nil, err
	}
	addresses.CanonicalTransactionChain, err = parseAddress(ctx, "canonical-transaction-chain-address")
	if err != nil {
		return nil, err
	}
	addresses.StateCommitmentChain, err = parseAddress(ctx, "state-commitment-chain-address")
	if err != nil {
		return nil, err
	}
	return &addresses, nil
}

// parseAddress will parse a [common.Address] from a [cli.Context] and return
// an error if the configured address is not correct.
func parseAddress(ctx *cli.Context, name string) (common.Address, error) {
	value := ctx.String(name)
	if value == "" {
		return common.Address{}, nil
	}
	if !common.IsHexAddress(value) {
		return common.Address{}, fmt.Errorf("invalid address: %s", value)
	}
	return common.HexToAddress(value), nil
}
