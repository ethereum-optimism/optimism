package op_e2e

import (
	"context"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

// TestRemoteStaticCall tests the the remote static call precompile
func TestRemoteStaticCall(t *testing.T) {
	InitParallel(t)

	cfg := DefaultSystemConfig(t)

	sys, err := cfg.Start()
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	log := testlog.Logger(t, log.LvlInfo)
	log.Info("genesis", "l2", sys.RollupConfig.Genesis.L2, "l1", sys.RollupConfig.Genesis.L1, "l2_time", sys.RollupConfig.Genesis.L2Time)

	l1Client := sys.Clients["l1"]
	l2Client := sys.Clients["sequencer"]

	opts, err := bind.NewKeyedTransactorWithChainID(sys.cfg.Secrets.Alice, cfg.L1ChainIDBig())
	require.Nil(t, err)

	// Deploy WETH9
	weth9Address, tx, WETH9, err := bindings.DeployWETH9(opts, l1Client)
	require.NoError(t, err)
	_, err = waitForTransaction(tx.Hash(), l1Client, 3*time.Duration(cfg.DeployConfig.L1BlockTime)*time.Second)
	require.NoError(t, err, "Waiting for deposit tx on L1")

	// Get some WETH
	opts.Value = big.NewInt(params.Ether)
	tx, err = WETH9.Fallback(opts, []byte{})
	require.NoError(t, err)
	_, err = waitForTransaction(tx.Hash(), l1Client, 3*time.Duration(cfg.DeployConfig.L1BlockTime)*time.Second)
	require.NoError(t, err)
	opts.Value = nil
	wethBalance, err := WETH9.BalanceOf(&bind.CallOpts{}, opts.From)
	require.NoError(t, err)
	require.Equal(t, big.NewInt(params.Ether), wethBalance)

	l1Block, err := l1Client.BlockNumber(context.Background())
	require.NoError(t, err)
	l1BlockBigInt := big.NewInt(int64(l1Block))

	WETH9Abi, err := abi.JSON(strings.NewReader(bindings.WETH9ABI))
	require.NoError(t, err)
	weth_balanceOf_calldata, err := WETH9Abi.Pack("balanceOf", opts.From)
	require.NoError(t, err)

	// Check that `Call` on L1 with the calldata returns correct information
	l1CallResult, err := l1Client.CallContract(context.Background(), ethereum.CallMsg{To: &weth9Address, Data: weth_balanceOf_calldata}, l1BlockBigInt)
	require.NoError(t, err)
	var l1_decoded_balance *big.Int
	err = WETH9Abi.UnpackIntoInterface(&l1_decoded_balance, "balanceOf", l1CallResult)
	require.NoError(t, err)
	require.Equal(t, wethBalance, l1_decoded_balance)

	// Spin until the L2 has been updated with the L1 block
	l1BlockContract, err := bindings.NewL1Block(predeploys.L1BlockAddr, l2Client)
	require.NoError(t, err)
	l2Updated := false

	l2Block, err := l2Client.BlockNumber(context.Background())
	require.NoError(t, err)
	l2BlockBigInt := big.NewInt(int64(l2Block))

	for !l2Updated {
		l1BlockOnL2, err := l1BlockContract.Number(&bind.CallOpts{})
		require.NoError(t, err)
		l1BlockOnL2AsInt := big.NewInt(int64(l1BlockOnL2))
		if l1BlockOnL2AsInt.Cmp(l1BlockBigInt) >= 0 {
			l2Updated = true
			l2Block, err := l2Client.BlockNumber(context.Background())
			require.NoError(t, err)
			l2BlockBigInt = big.NewInt(int64(l2Block))
		}
		time.Sleep(3 * time.Second)
	}

	const definition = `[{
        "name": "encode_address_bytes",
        "type": "function",
        "inputs": [{
            "name": "addr",
            "type": "address"
        }, {
            "name": "b",
            "type": "bytes"
        }],
        "outputs": []
    }]`

	// Create a new ABI object
	encode_abi, err := abi.JSON(strings.NewReader(definition))
	require.NoError(t, err)

	remote_static_call_data, err := encode_abi.Pack("encode_address_bytes", weth9Address, weth_balanceOf_calldata)
	require.NoError(t, err)
	// Slice off the first 4 bytes of the selector, since we just want the abi.encode(...)
	remote_static_call_data = remote_static_call_data[4:]

	remoteStaticCallAddr := common.HexToAddress("0x0000000000000000000000000000000000000013")

	// Send a `eth_call` to the L2 remote static call precompile contract
	remote_static_call_result, err := l2Client.CallContract(context.Background(), ethereum.CallMsg{
		From:     opts.From,
		To:       &remoteStaticCallAddr,
		Gas:      1000000000,
		GasPrice: big.NewInt(0),
		Value:    big.NewInt(0),
		Data:     remote_static_call_data,
	}, l2BlockBigInt)
	require.NoError(t, err)

	var alice_balance *big.Int
	err = WETH9Abi.UnpackIntoInterface(&alice_balance, "balanceOf", remote_static_call_result)
	require.NoError(t, err)
	require.Equal(t, wethBalance, alice_balance)
}
