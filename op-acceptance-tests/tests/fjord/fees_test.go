package fjord

import (
	"context"
	"github.com/ethereum/go-ethereum/rpc"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"

	"github.com/ethereum-optimism/optimism/op-e2e/bindings"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
)

func TestFees(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)
	require := t.Require()
	ctx := t.Ctx()

	err := dsl.RequiresL2Fork(ctx, sys, 0, rollup.Fjord)
	require.NoError(err)
	operatorFee := dsl.NewOperatorFee(t, sys.L2Chain, sys.L1EL)
	operatorFee.SetOperatorFee(100000000, 500)
	operatorFee.WaitForL2SyncWithCurrentL1State()

	alice := sys.FunderL2.NewFundedEOA(eth.OneTenthEther)
	bob := sys.Wallet.NewEOA(sys.L2EL)

	fjordFees := dsl.NewFjordFees(t, sys.L2Chain)
	result := fjordFees.ValidateTransaction(alice, bob, big.NewInt(42000000000))

	validateGPO(t, ctx, sys, &result)
}

func validateGPO(t devtest.T, ctx context.Context, sys *presets.Minimal, result *dsl.FjordFeesValidationResult) {
	require := t.Require()

	l2Client := sys.L2EL.Escape().EthClient()
	contractBackend := &contractBackendAdapter{client: l2Client, ctx: ctx}
	gpoContract, err := bindings.NewGasPriceOracle(predeploys.GasPriceOracleAddr, contractBackend)
	require.NoError(err)

	gpoFjord, err := gpoContract.IsFjord(&bind.CallOpts{Context: ctx, BlockNumber: result.TransactionReceipt.BlockNumber})
	require.NoError(err)
	require.True(gpoFjord)

	txBytes := make([]byte, 100)

	gpoL1Fee, err := gpoContract.GetL1Fee(&bind.CallOpts{Context: ctx, BlockNumber: result.TransactionReceipt.BlockNumber}, txBytes)
	require.NoError(err)
	require.Equal(result.L1Fee, gpoL1Fee)
}

type contractBackendAdapter struct {
	client apis.EthClient
	ctx    context.Context
}

func (c *contractBackendAdapter) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	return c.client.Call(ctx, call, rpc.BlockNumber(blockNumber.Int64()))
}

func (c *contractBackendAdapter) CodeAt(ctx context.Context, account common.Address, blockNumber *big.Int) ([]byte, error) {
	blockRef, err := c.client.BlockRefByNumber(ctx, blockNumber.Uint64())
	if err != nil {
		return nil, err
	}
	return c.client.CodeAtHash(ctx, account, blockRef.Hash)
}

func (c *contractBackendAdapter) HeaderByNumber(ctx context.Context, number *big.Int) (*gethTypes.Header, error) {
	blockInfo, err := c.client.InfoByNumber(ctx, number.Uint64())
	if err != nil {
		return nil, err
	}
	return &gethTypes.Header{
		Number:   new(big.Int).SetUint64(number.Uint64()),
		Time:     blockInfo.Time(),
		BaseFee:  blockInfo.BaseFee(),
		GasLimit: blockInfo.GasLimit(),
		GasUsed:  blockInfo.GasUsed(),
	}, nil
}

func (c *contractBackendAdapter) PendingCallContract(ctx context.Context, call ethereum.CallMsg) ([]byte, error) {
	return c.client.Call(ctx, call, rpc.PendingBlockNumber)
}

func (c *contractBackendAdapter) PendingCodeAt(ctx context.Context, account common.Address) ([]byte, error) {
	latestInfo, err := c.client.InfoByLabel(ctx, "latest")
	if err != nil {
		return nil, err
	}
	return c.client.CodeAtHash(ctx, account, latestInfo.Hash())
}

func (c *contractBackendAdapter) EstimateGas(ctx context.Context, call ethereum.CallMsg) (uint64, error) {
	return c.client.EstimateGas(ctx, call)
}

func (c *contractBackendAdapter) SendTransaction(ctx context.Context, tx *gethTypes.Transaction) error {
	return c.client.SendTransaction(ctx, tx)
}

func (c *contractBackendAdapter) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	return big.NewInt(1000000000), nil // 1 gwei default
}

func (c *contractBackendAdapter) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	return big.NewInt(1000000000), nil // 1 gwei default
}

func (c *contractBackendAdapter) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	return c.client.PendingNonceAt(ctx, account)
}

func (c *contractBackendAdapter) FilterLogs(ctx context.Context, query ethereum.FilterQuery) ([]gethTypes.Log, error) {
	// Not needed for GPO validation, return empty
	return []gethTypes.Log{}, nil
}

func (c *contractBackendAdapter) SubscribeFilterLogs(ctx context.Context, query ethereum.FilterQuery, ch chan<- gethTypes.Log) (ethereum.Subscription, error) {
	// Not needed for GPO validation, return nil
	return nil, nil
}
