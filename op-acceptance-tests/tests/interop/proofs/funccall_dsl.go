package proofs

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3"
)

type funcCallDSL struct {
	t       devtest.T
	client  apis.EthClient
	fn      *w3.Func
	args    []any
	returns []any
}

func newFuncCallDSL(t devtest.T, client apis.EthClient, fn *w3.Func) *funcCallDSL {
	return &funcCallDSL{t: t, client: client, fn: fn}
}

func (c *funcCallDSL) WithArgs(args ...any) *funcCallDSL {
	c.args = args
	return c
}

func (c *funcCallDSL) WithReturns(returns ...any) *funcCallDSL {
	c.returns = returns
	return c
}

func (c *funcCallDSL) Call(target common.Address) {
	callArgs, err := c.fn.EncodeArgs(c.args...)
	c.t.Require().NoError(err)
	callResult, err := c.client.Call(c.t.Ctx(), ethereum.CallMsg{To: &target, Data: callArgs})
	c.t.Require().NoError(err)
	c.t.Require().NoError(c.fn.DecodeReturns(callResult, c.returns...))
}
