package op_e2e

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-e2e/config"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient/simulated"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

type testStateGetter struct {
	baseFee, blobBaseFee, overhead, scalar *big.Int
	baseFeeScalar, blobBaseFeeScalar       uint32
}

func (sg *testStateGetter) GetState(addr common.Address, slot common.Hash) common.Hash {
	buf := common.Hash{}
	switch slot {
	case types.L1BaseFeeSlot:
		sg.baseFee.FillBytes(buf[:])
	case types.OverheadSlot:
		sg.overhead.FillBytes(buf[:])
	case types.ScalarSlot:
		sg.scalar.FillBytes(buf[:])
	case types.L1BlobBaseFeeSlot:
		sg.blobBaseFee.FillBytes(buf[:])
	case types.L1FeeScalarsSlot:
		// fetch Ecotone fee sclars
		offset := 32 - types.BaseFeeScalarSlotOffset - 4 // todo maybe make scalarSelectSTartPublic
		binary.BigEndian.PutUint32(buf[offset:offset+4], sg.baseFeeScalar)
		binary.BigEndian.PutUint32(buf[offset+4:offset+8], sg.blobBaseFeeScalar)
	default:
		panic("unknown slot")
	}
	return buf
}

func FuzzFastLZ(f *testing.F) {
	l2Allocs := config.L2Allocs(genesis.L2AllocsFjord)
	backend := simulated.NewBackend(l2Allocs.Accounts)
	defer backend.Close()

	gpoCaller, err := bindings.NewGasPriceOracleCaller(predeploys.GasPriceOracleAddr, backend.Client())
	require.NoError(f, err)

	isFjord, err := gpoCaller.IsFjord(&bind.CallOpts{})
	require.NoError(f, err)
	require.True(f, isFjord)

	f.Fuzz(func(t *testing.T, fuzzedData []byte) {
		gpoValue, err := gpoCaller.GetL1GasUsed(&bind.CallOpts{}, fuzzedData)
		require.NoError(t, err)

		gpoWithoutPadding := gpoValue.Div(gpoValue, big.NewInt(16))
		gpoWithoutPadding.Sub(gpoWithoutPadding, big.NewInt(68))

		gethValue := types.FlzCompressLen(fuzzedData)

		require.Equal(t, uint64(gethValue), gpoWithoutPadding.Uint64())
	})
}

func FuzzFjordCostFunction(f *testing.F) {
	cfg := DefaultSystemConfig(f)
	s := hexutil.Uint64(0)
	cfg.DeployConfig.L2GenesisCanyonTimeOffset = &s
	cfg.DeployConfig.L2GenesisDeltaTimeOffset = &s
	cfg.DeployConfig.L2GenesisEcotoneTimeOffset = &s
	cfg.DeployConfig.L2GenesisFjordTimeOffset = &s

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	opGeth, err := NewOpGeth(f, ctx, &cfg)
	require.NoError(f, err)
	defer opGeth.Close()

	gpoCaller, err := bindings.NewGasPriceOracleCaller(predeploys.GasPriceOracleAddr, opGeth.L2Client)
	require.NoError(f, err)

	isFjord, err := gpoCaller.IsFjord(&bind.CallOpts{})
	require.NoError(f, err)
	require.True(f, isFjord)

	_, err = opGeth.AddL2Block(context.Background())
	require.NoError(f, err)

	baseFee, err := gpoCaller.L1BaseFee(&bind.CallOpts{})
	require.NoError(f, err)
	require.Greater(f, baseFee.Uint64(), uint64(0))

	blobBaseFee, err := gpoCaller.BlobBaseFee(&bind.CallOpts{})
	require.NoError(f, err)
	require.Greater(f, blobBaseFee.Uint64(), uint64(0))

	baseFeeScalar, err := gpoCaller.BaseFeeScalar(&bind.CallOpts{})
	require.NoError(f, err)
	require.Greater(f, baseFeeScalar, uint32(0))

	blobBaseFeeScalar, err := gpoCaller.BlobBaseFeeScalar(&bind.CallOpts{})
	require.NoError(f, err)
	require.Equal(f, blobBaseFeeScalar, uint32(0))

	// we can ignore the blobbasefee, as the scalar is set to zero.
	feeScaled := big.NewInt(16)
	feeScaled.Mul(feeScaled, baseFee)
	feeScaled.Mul(feeScaled, big.NewInt(int64(baseFeeScalar)))

	db := &testStateGetter{
		baseFee:           baseFee,
		blobBaseFee:       blobBaseFee,
		overhead:          big.NewInt(0), // not used for fjord
		scalar:            big.NewInt(0), // not used for fjord
		baseFeeScalar:     baseFeeScalar,
		blobBaseFeeScalar: blobBaseFeeScalar,
	}

	zeroTime := uint64(0)
	// create a config where ecotone/fjord upgrades are active
	config := &params.ChainConfig{
		Optimism:     params.OptimismTestConfig.Optimism,
		RegolithTime: &zeroTime,
		EcotoneTime:  &zeroTime,
		FjordTime:    &zeroTime,
	}
	require.True(f, config.IsOptimismEcotone(zeroTime))
	require.True(f, config.IsOptimismFjord(zeroTime))
	costFunc := types.NewL1CostFunc(config, db)

	f.Fuzz(func(t *testing.T, fuzzedData []byte) {
		flzSize := types.FlzCompressLen(fuzzedData)

		// Skip transactions that will be clamped to the minimum or less. These will fuzz to different values
		// due to the solidity l1BlockGenesis adding 68 extra bytes to account for the signature.
		estimatedSize := big.NewInt(int64(flzSize))
		estimatedSize.Mul(estimatedSize, types.L1CostFastlzCoef)
		estimatedSize.Add(estimatedSize, types.L1CostIntercept)

		if estimatedSize.Cmp(types.MinTransactionSizeScaled) < 0 {
			t.Skip()
			return
		}

		gpoValue, err := gpoCaller.GetL1Fee(&bind.CallOpts{}, fuzzedData)
		require.NoError(t, err)

		// remove the adjustment
		gpoValue.Mul(gpoValue, big.NewInt(1e12))
		gpoValue.Div(gpoValue, feeScaled)

		totat := new(big.Int).Mul(big.NewInt(68), big.NewInt(836_500))
		gpoValue.Sub(gpoValue, totat)

		gpoValue.Mul(gpoValue, feeScaled)
		gpoValue.Div(gpoValue, big.NewInt(1e12))

		costData := types.NewRollupCostData(fuzzedData)

		gethValue := costFunc(costData, zeroTime)

		require.Equal(t, gethValue.Uint64(), gpoValue.Uint64(), fmt.Sprintf("fuzzedData: %x", common.Bytes2Hex(fuzzedData)))
	})
}
