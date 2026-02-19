package utils

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

const SimpleStorageArtifact = "../contracts/artifacts/SimpleStorage.sol/SimpleStorage.json"

type SimpleStorage struct {
	*Contract
	t devtest.T
}

func (c *SimpleStorage) SetValue(user *dsl.EOA, value *big.Int) *types.Receipt {
	ctx := c.t.Ctx()
	callData, err := c.parsedABI.Pack("setValue", value)
	if err != nil {
		require.NoError(c.t, err, "failed to pack set call data")
	}

	callTx := txplan.NewPlannedTx(user.Plan(), txplan.WithTo(&c.Contract.address), txplan.WithData(callData))
	callRes, err := callTx.Included.Eval(ctx)
	if err != nil {
		require.NoError(c.t, err, "failed to create set tx")
	}

	if callRes.Status != types.ReceiptStatusSuccessful {
		require.NoError(c.t, err, "set transaction failed")
	}
	return callRes
}

func (c *SimpleStorage) PlanSetValue(user *dsl.EOA, value *big.Int) *txplan.PlannedTx {
	callData, err := c.parsedABI.Pack("setValue", value)
	if err != nil {
		require.NoError(c.t, err, "failed to pack set call data")
	}

	callTx := txplan.NewPlannedTx(user.Plan(), txplan.WithTo(&c.Contract.address), txplan.WithData(callData))
	return callTx
}

func DeploySimpleStorage(t devtest.T, user *dsl.EOA) (*SimpleStorage, *types.Receipt) {
	parsedABI, bin := LoadArtifact(t, SimpleStorageArtifact)
	contractAddress, receipt := DeployContract(t, user, bin)
	contract := NewContract(contractAddress, parsedABI)
	return &SimpleStorage{contract, t}, receipt
}
