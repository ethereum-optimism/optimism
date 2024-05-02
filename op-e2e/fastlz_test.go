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
	"github.com/ethereum-optimism/optimism/op-e2e/fastlz"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient/simulated"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

var (
	contract = &bind.MetaData{
		ABI: "[{\"type\":\"function\",\"name\":\"fastLz\",\"inputs\":[{\"name\":\"_data\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"pure\"}]",
	}

	fastLzBytecode = "0x608060405234801561001057600080fd5b506004361061002b5760003560e01c8063920a769114610030575b600080fd5b61004361003e366004610374565b610055565b60405190815260200160405180910390f35b600061006082610067565b5192915050565b60606101e0565b818153600101919050565b600082840393505b838110156100a25782810151828201511860001a1590930292600101610081565b9392505050565b825b602082106100d75782516100c0601f8361006e565b5260209290920191601f19909101906021016100ab565b81156100a25782516100ec600184038361006e565b520160010192915050565b60006001830392505b61010782106101385761012a8360ff1661012560fd6101258760081c60e0018961006e565b61006e565b935061010682039150610100565b600782106101655761015e8360ff16610125600785036101258760081c60e0018961006e565b90506100a2565b61017e8360ff166101258560081c8560051b018761006e565b949350505050565b80516101d890838303906101bc90600081901a600182901a60081b1760029190911a60101b17639e3779b90260131c611fff1690565b8060021b6040510182815160e01c1860e01b8151188152505050565b600101919050565b5060405161800038823961800081016020830180600d8551820103826002015b81811015610313576000805b50508051604051600082901a600183901a60081b1760029290921a60101b91909117639e3779b9810260111c617ffc16909101805160e081811c878603811890911b9091189091528401908183039084841061026857506102a3565b600184019350611fff821161029d578251600081901a600182901a60081b1760029190911a60101b17810361029d57506102a3565b5061020c565b8383106102b1575050610313565b600183039250858311156102cf576102cc87878886036100a9565b96505b6102e3600985016003850160038501610079565b91506102f08782846100f7565b9650506103088461030386848601610186565b610186565b915050809350610200565b5050617fe061032884848589518601036100a9565b03925050506020820180820383525b81811161034e57617fe08101518152602001610337565b5060008152602001604052919050565b634e487b7160e01b600052604160045260246000fd5b60006020828403121561038657600080fd5b813567ffffffffffffffff8082111561039e57600080fd5b818401915084601f8301126103b257600080fd5b8135818111156103c4576103c461035e565b604051601f8201601f19908116603f011681019083821181831017156103ec576103ec61035e565b8160405282815287602084870101111561040557600080fd5b82602086016020830137600092810160200192909252509594505050505056fea264697066735822122000646b2953fc4a6f501bd0456ac52203089443937719e16b3190b7979c39511264736f6c63430008190033"
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

		l1FeeSolidity, err := gpoCaller.GetL1Fee(&bind.CallOpts{}, fuzzedData)
		require.NoError(t, err)

		// remove the adjustment
		l1FeeSolidity.Mul(l1FeeSolidity, big.NewInt(1e12))
		l1FeeSolidity.Div(l1FeeSolidity, feeScaled)

		totat := new(big.Int).Mul(big.NewInt(68), big.NewInt(836_500))
		l1FeeSolidity.Sub(l1FeeSolidity, totat)

		l1FeeSolidity.Mul(l1FeeSolidity, feeScaled)
		l1FeeSolidity.Div(l1FeeSolidity, big.NewInt(1e12))

		costData := types.NewRollupCostData(fuzzedData)

		l1FeeGeth := costFunc(costData, zeroTime)

		require.Equal(t, l1FeeGeth.Uint64(), l1FeeSolidity.Uint64(), fmt.Sprintf("fuzzedData: %x", common.Bytes2Hex(fuzzedData)))
	})

}

func FuzzFastLzGethSolidity(f *testing.F) {
	a, err := contract.GetAbi()
	require.NoError(f, err)

	b := simulated.NewBackend(map[common.Address]core.GenesisAccount{
		predeploys.GasPriceOracleAddr: {
			Code: common.FromHex(fastLzBytecode),
		},
	})
	defer func() {
		require.NoError(f, b.Close())
	}()

	client := b.Client()

	f.Fuzz(func(t *testing.T, data []byte) {
		req, err := a.Pack("fastLz", data)
		require.NoError(t, err)

		response, err := client.CallContract(context.TODO(), ethereum.CallMsg{
			To:   &predeploys.GasPriceOracleAddr,
			Data: req,
		}, nil)
		require.NoError(t, err)

		result, err := a.Unpack("fastLz", response)
		require.NoError(t, err)

		gethCompressedLen := types.FlzCompressLen(data)
		require.Equal(t, result[0].(*big.Int).Uint64(), uint64(gethCompressedLen))
	})
}

func FuzzFastLzCgo(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) == 0 {
			t.Skip()
			return
		}

		// Our implementation in go-ethereum
		compressedLen := types.FlzCompressLen(data)

		out, err := fastlz.Compress(data)
		require.NoError(t, err)
		require.Equal(t, int(compressedLen), len(out))
	})
}
