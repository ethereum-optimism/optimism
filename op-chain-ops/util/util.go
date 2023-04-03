package util

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/urfave/cli/v2"
)

func ProgressLogger(n int, msg string) func(...any) {
	var i int

	return func(args ...any) {
		i++
		if i%n != 0 {
			return
		}
		log.Info(msg, append([]any{"count", i}, args...)...)
	}
}

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
	l1RpcURL := ctx.String("l1-rpc-url")
	l1Client, err := ethclient.Dial(l1RpcURL)
	if err != nil {
		return nil, err
	}
	l1ChainID, err := l1Client.ChainID(context.Background())
	if err != nil {
		return nil, err
	}

	l2RpcURL := ctx.String("l2-rpc-url")
	l2Client, err := ethclient.Dial(l2RpcURL)
	if err != nil {
		return nil, err
	}
	l2ChainID, err := l2Client.ChainID(context.Background())
	if err != nil {
		return nil, err
	}

	l1RpcClient, err := rpc.DialContext(context.Background(), l1RpcURL)
	if err != nil {
		return nil, err
	}

	l2RpcClient, err := rpc.DialContext(context.Background(), l2RpcURL)
	if err != nil {
		return nil, err
	}

	l1GethClient := gethclient.New(l1RpcClient)
	l2GethClient := gethclient.New(l2RpcClient)

	log.Info(
		"Set up RPC clients",
		"l1-chain-id", l1ChainID,
		"l2-chain-id", l2ChainID,
	)

	return &Clients{
		L1Client:     l1Client,
		L2Client:     l2Client,
		L1RpcClient:  l1RpcClient,
		L2RpcClient:  l2RpcClient,
		L1GethClient: l1GethClient,
		L2GethClient: l2GethClient,
	}, nil
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
