package utils

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

const MultiStorageArtifact = "../contracts/artifacts/MultiStorage.sol/MultiStorage.json"

type MultiStorage struct {
	*Contract
	t devtest.T
}

func (c *MultiStorage) SetValues(user *dsl.EOA, a, b *big.Int) *types.Receipt {
	ctx := c.t.Ctx()
	callData, err := c.parsedABI.Pack("setValues", a, b)
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

func DeployMultiStorage(t devtest.T, user *dsl.EOA) (*MultiStorage, *types.Receipt) {
	parsedABI, bin := LoadArtifact(t, MultiStorageArtifact)
	contractAddress, receipt := DeployContract(t, user, bin)
	contract := NewContract(contractAddress, parsedABI)
	return &MultiStorage{contract, t}, receipt
}
