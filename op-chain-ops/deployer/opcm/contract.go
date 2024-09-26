package opcm

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Contract struct {
	addr   common.Address
	client *ethclient.Client
}

func NewContract(addr common.Address, client *ethclient.Client) *Contract {
	return &Contract{addr: addr, client: client}
}

func (c *Contract) SuperchainConfig(ctx context.Context) (common.Address, error) {
	return c.getAddress(ctx, "superchainConfig")
}

func (c *Contract) ProtocolVersions(ctx context.Context) (common.Address, error) {
	return c.getAddress(ctx, "protocolVersions")
}

func (c *Contract) getAddress(ctx context.Context, name string) (common.Address, error) {
	method := abi.NewMethod(
		name,
		name,
		abi.Function,
		"view",
		true,
		false,
		abi.Arguments{},
		abi.Arguments{
			abi.Argument{
				Name:    "address",
				Type:    mustType("address"),
				Indexed: false,
			},
		},
	)

	calldata, err := method.Inputs.Pack()
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to pack inputs: %w", err)
	}

	msg := ethereum.CallMsg{
		To:   &c.addr,
		Data: append(bytes.Clone(method.ID), calldata...),
	}
	result, err := c.client.CallContract(ctx, msg, nil)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to call contract: %w", err)
	}

	out, err := method.Outputs.Unpack(result)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to unpack result: %w", err)
	}
	if len(out) != 1 {
		return common.Address{}, fmt.Errorf("unexpected output length: %d", len(out))
	}
	addr, ok := out[0].(common.Address)
	if !ok {
		return common.Address{}, fmt.Errorf("unexpected type: %T", out[0])
	}
	return addr, nil
}

func mustType(t string) abi.Type {
	typ, err := abi.NewType(t, "", nil)
	if err != nil {
		panic(err)
	}
	return typ
}
