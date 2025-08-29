package fjord

import (
	"context"
	"crypto/rand"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	txib "github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum-optimism/optimism/op-service/txplan"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

var (
	rip7212Precompile = common.HexToAddress("0x0000000000000000000000000000000000000100")
	invalid7212Data   = []byte{0x00}
	valid7212Data     = common.FromHex("4cee90eb86eaa050036147a12d49004b6b9c72bd725d39d4785011fe190f0b4da73bd4903f0ce3b639bbbf6e8e80d16931ff4bcf5993d58468e8fb19086e8cac36dbcd03009df8c59286b162af3bd7fcc0450c9aa81be5d10d312af6c66b1d604aebd3099c618202fcfe16ae7770b0c49ab5eadf74b754204a3bb6060e44eff37618b065f9832de4ca6ca971a7a1adc826d0f7c00181a5fb2ddf79ae00b4e10e")
)

func TestCheckFjordScript(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)
	require := t.Require()
	ctx := t.Ctx()

	err := dsl.RequiresL2Fork(ctx, sys, 0, rollup.Fjord)
	require.NoError(err)

	wallet := sys.FunderL2.NewFundedEOA(eth.OneThirdEther)

	checkRIP7212(t, ctx, sys)
	checkGasPriceOracle(t, ctx, sys)
	checkFastLZTransactions(t, ctx, sys, wallet)
}

func checkRIP7212(t devtest.T, ctx context.Context, sys *presets.Minimal) {
	require := t.Require()
	l2Client := sys.L2EL.Escape().EthClient()

	// Test invalid signature
	response, err := l2Client.Call(ctx, ethereum.CallMsg{
		To:   &rip7212Precompile,
		Data: invalid7212Data,
	}, rpc.LatestBlockNumber)
	require.NoError(err)
	require.Empty(response)

	// Test valid signature
	response, err = l2Client.Call(ctx, ethereum.CallMsg{
		To:   &rip7212Precompile,
		Data: valid7212Data,
	}, rpc.LatestBlockNumber)
	require.NoError(err)
	expected := common.LeftPadBytes([]byte{1}, 32)
	require.Equal(expected, response)
}

func checkGasPriceOracle(t devtest.T, ctx context.Context, sys *presets.Minimal) {
	require := t.Require()

	l2Client := sys.L2EL.Escape().EthClient()
	gpo := txib.NewGasPriceOracle(
		txib.WithClient(l2Client),
		txib.WithTo(predeploys.GasPriceOracleAddr),
		txib.WithTest(t),
	)

	isFjord, err := contractio.Read(gpo.IsFjord(), ctx)
	require.NoError(err)
	require.True(isFjord)
}

func checkFastLZTransactions(t devtest.T, ctx context.Context, sys *presets.Minimal, wallet *dsl.EOA) {
	require := t.Require()

	l2Client := sys.L2EL.Escape().EthClient()
	gasPriceOracle := txib.NewGasPriceOracle(
		txib.WithClient(l2Client),
		txib.WithTo(predeploys.GasPriceOracleAddr),
		txib.WithTest(t),
	)

	testCases := []struct {
		name string
		data []byte
	}{
		{"empty", nil},
		{"all-zero-256", make([]byte, 256)},
		{"all-42-256", func() []byte {
			data := make([]byte, 256)
			for i := range data {
				data[i] = 0x42
			}
			return data
		}()},
		{"random-256", func() []byte {
			data := make([]byte, 256)
			_, _ = rand.Read(data)
			return data
		}()},
	}

	for _, tc := range testCases {
		walletAddr := wallet.Address()
		var receipt *types.Receipt
		var signedTx *types.Transaction

		if len(tc.data) == 0 {
			plannedTx := wallet.Transfer(walletAddr, eth.ZeroWei)
			var err error
			receipt, err = plannedTx.Included.Eval(ctx)
			require.NoError(err)
			require.NotNil(receipt)

			_, txs, err := l2Client.InfoAndTxsByHash(ctx, receipt.BlockHash)
			require.NoError(err)

			for _, tx := range txs {
				if tx.Hash() == receipt.TxHash {
					signedTx = tx
					break
				}
			}
			require.NotNil(signedTx)
		} else {
			opt := txplan.Combine(
				wallet.Plan(),
				txplan.WithTo(&walletAddr),
				txplan.WithValue(eth.ZeroWei),
				txplan.WithData(tc.data),
			)
			plannedTx := txplan.NewPlannedTx(opt)
			var err error
			receipt, err = plannedTx.Included.Eval(ctx)
			require.NoError(err)
			require.NotNil(receipt)

			_, txs, err := l2Client.InfoAndTxsByHash(ctx, receipt.BlockHash)
			require.NoError(err)

			for _, tx := range txs {
				if tx.Hash() == receipt.TxHash {
					signedTx = tx
					break
				}
			}
			require.NotNil(signedTx)
		}

		require.Equal(uint64(1), receipt.Status)

		blockNumber := receipt.BlockNumber

		unsignedTx := types.NewTx(&types.DynamicFeeTx{
			Nonce:     signedTx.Nonce(),
			To:        signedTx.To(),
			Value:     signedTx.Value(),
			Gas:       signedTx.Gas(),
			GasFeeCap: signedTx.GasFeeCap(),
			GasTipCap: signedTx.GasTipCap(),
			Data:      signedTx.Data(),
		})

		txUnsigned, err := unsignedTx.MarshalBinary()
		require.NoError(err)
		txSigned, err := signedTx.MarshalBinary()
		require.NoError(err)

		gpoFee, err := contractio.Read(gasPriceOracle.GetL1Fee(txUnsigned), ctx)
		require.NoError(err)

		fastLzSize := uint64(types.FlzCompressLen(txUnsigned) + 68)
		gethGPOFee, err := fjordL1Cost(l2Client, blockNumber, fastLzSize)
		require.NoError(err)
		require.Equal(gethGPOFee.Uint64(), gpoFee.ToBig().Uint64())

		signedFastLzSize := uint64(types.FlzCompressLen(txSigned))
		gethFee, err := fjordL1Cost(l2Client, blockNumber, signedFastLzSize)
		require.NoError(err)
		require.NotNil(receipt.L1Fee)
		require.Equal(gethFee.Uint64(), receipt.L1Fee.Uint64())

		upperBound, err := contractio.Read(gasPriceOracle.GetL1FeeUpperBound(big.NewInt(int64(len(txUnsigned)))), ctx)
		require.NoError(err)
		txLenGPO := len(txUnsigned) + 68
		flzUpperBound := uint64(txLenGPO + txLenGPO/255 + 16)
		upperBoundCost, err := fjordL1Cost(l2Client, blockNumber, flzUpperBound)
		require.NoError(err)
		require.Equal(upperBoundCost.Uint64(), upperBound.ToBig().Uint64())

		_, err = contractio.Read(gasPriceOracle.BaseFeeScalar(), ctx)
		require.NoError(err)
		_, err = contractio.Read(gasPriceOracle.BlobBaseFeeScalar(), ctx)
		require.NoError(err)
	}
}

func fjordL1Cost(client apis.EthClient, blockNumber *big.Int, fastLzSize uint64) (*big.Int, error) {
	l1Block := txib.NewL1Block(
		txib.WithClient(client),
		txib.WithTo(predeploys.L1BlockAddr),
	)

	baseFeeScalar, err := contractio.Read(l1Block.BasefeeScalar(), context.Background())
	if err != nil {
		return nil, err
	}
	l1BaseFee, err := contractio.Read(l1Block.Basefee(), context.Background())
	if err != nil {
		return nil, err
	}
	blobBaseFeeScalar, err := contractio.Read(l1Block.BlobBaseFeeScalar(), context.Background())
	if err != nil {
		return nil, err
	}
	blobBaseFee, err := contractio.Read(l1Block.BlobBaseFee(), context.Background())
	if err != nil {
		return nil, err
	}

	costFunc := types.NewL1CostFuncFjord(
		l1BaseFee,
		blobBaseFee,
		new(big.Int).SetUint64(uint64(baseFeeScalar)),
		new(big.Int).SetUint64(uint64(blobBaseFeeScalar)))

	fee, _ := costFunc(types.RollupCostData{FastLzSize: fastLzSize})
	return fee, nil
}
