package op_e2e

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

// TestMissingGasLimit tests that op-geth cannot build a block without gas limit while optimism is active in the chain config.
func TestMissingGasLimit(t *testing.T) {
	cfg := DefaultSystemConfig(t)
	cfg.DeployConfig.FundDevAccounts = false
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	opGeth, err := NewOpGeth(t, ctx, &cfg)
	require.NoError(t, err)
	defer opGeth.Close()

	attrs, err := opGeth.CreatePayloadAttributes()
	require.NoError(t, err)
	// Remove the GasLimit from the otherwise valid attributes
	attrs.GasLimit = nil

	res, err := opGeth.StartBlockBuilding(ctx, attrs)
	require.ErrorIs(t, err, eth.InputError{})
	require.Equal(t, eth.InvalidPayloadAttributes, err.(eth.InputError).Code)
	require.Nil(t, res)
}

// TestInvalidDepositInFCU runs an invalid deposit through a FCU/GetPayload/NewPayload/FCU set of calls.
// This tests that deposits must always allow the block to be built even if they are invalid.
func TestInvalidDepositInFCU(t *testing.T) {
	cfg := DefaultSystemConfig(t)
	cfg.DeployConfig.FundDevAccounts = false
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	opGeth, err := NewOpGeth(t, ctx, &cfg)
	require.NoError(t, err)
	defer opGeth.Close()

	// Create a deposit from alice that will always fail (not enough funds)
	fromAddr := cfg.Secrets.Addresses().Alice
	balance, err := opGeth.L2Client.BalanceAt(ctx, fromAddr, nil)
	require.Nil(t, err)
	require.Equal(t, 0, balance.Cmp(common.Big0))

	badDepositTx := types.NewTx(&types.DepositTx{
		SourceHash:          opGeth.L1Head.Hash(),
		From:                fromAddr,
		To:                  &fromAddr, // send it to ourselves
		Value:               big.NewInt(params.Ether),
		Gas:                 25000,
		IsSystemTransaction: false,
	})

	// We are inserting a block with an invalid deposit.
	// The invalid deposit should still remain in the block.
	_, err = opGeth.AddL2Block(ctx, badDepositTx)
	require.NoError(t, err)

	// Deposit tx was included, but Alice still shouldn't have any ETH
	balance, err = opGeth.L2Client.BalanceAt(ctx, fromAddr, nil)
	require.Nil(t, err)
	require.Equal(t, 0, balance.Cmp(common.Big0))
}
