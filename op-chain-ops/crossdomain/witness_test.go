package crossdomain

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestRead(t *testing.T) {
	witnesses, addresses, err := ReadWitnessData("testdata/witness.txt")
	require.NoError(t, err)

	require.Equal(t, []*SentMessage{
		{
			Who: common.HexToAddress("0x4200000000000000000000000000000000000007"),
			Msg: common.FromHex(
				"0xcbd4ece900000000000000000000000099c9fc46f92e8a1c0dec1b1747d01090" +
					"3e884be100000000000000000000000042000000000000000000000000000000" +
					"0000001000000000000000000000000000000000000000000000000000000000" +
					"0000008000000000000000000000000000000000000000000000000000000000" +
					"00019bd000000000000000000000000000000000000000000000000000000000" +
					"000000e4a9f9e675000000000000000000000000d533a949740bb3306d119cc7" +
					"77fa900ba034cd520000000000000000000000000994206dfe8de6ec6920ff4d" +
					"779b0d950605fb53000000000000000000000000e3a44dd2a8c108be56a78635" +
					"121ec914074da16d000000000000000000000000e3a44dd2a8c108be56a78635" +
					"121ec914074da16d0000000000000000000000000000000000000000000001b0" +
					"ac98ab3858d75478000000000000000000000000000000000000000000000000" +
					"00000000000000c0000000000000000000000000000000000000000000000000" +
					"0000000000000000000000000000000000000000000000000000000000000000" +
					"00000000",
			),
		},
		{
			Who: common.HexToAddress("0x8b1d477410344785ff1df52500032e6d5f532ee4"),
			Msg: common.FromHex("0x042069"),
		},
	}, witnesses)

	require.Equal(t, OVMETHAddresses{
		common.HexToAddress("0x6340d44c5174588B312F545eEC4a42f8a514eF50"): true,
	}, addresses)
}

// TestDecodeWitnessCallData tests that the witness data is parsed correctly
// from an input bytes slice.
func TestDecodeWitnessCallData(t *testing.T) {
	tests := []struct {
		name string
		err  bool
		msg  []byte
		want []byte
	}{
		{
			name: "too-small",
			err:  true,
			msg:  common.FromHex("0x0000"),
		},
		{
			name: "unknown-selector",
			err:  true,
			msg:  common.FromHex("0x00000000"),
		},
		{
			name: "wrong-selector",
			err:  true,
			// 0x54fd4d50 is the selector for `version()`
			msg: common.FromHex("0x54fd4d50"),
		},
		{
			name: "invalid-calldata-only-selector",
			err:  true,
			// 0xcafa81dc is the selector for `passMessageToL1(bytes)`
			msg: common.FromHex("0xcafa81dc"),
		},
		{
			name: "invalid-calldata-invalid-bytes",
			err:  true,
			// 0xcafa81dc is the selector for passMessageToL1(bytes)
			msg: common.FromHex("0xcafa81dc0000"),
		},
		{
			name: "valid-calldata",
			msg: common.FromHex(
				"0xcafa81dc" +
					"0000000000000000000000000000000000000000000000000000000000000020" +
					"0000000000000000000000000000000000000000000000000000000000000002" +
					"1234000000000000000000000000000000000000000000000000000000000000",
			),
			want: common.FromHex("0x1234"),
		},
	}

	for _, tt := range tests {
		test := tt
		t.Run(test.name, func(t *testing.T) {
			if test.err {
				_, err := decodeWitnessCalldata(test.msg)
				require.Error(t, err)
			} else {
				want, err := decodeWitnessCalldata(test.msg)
				require.NoError(t, err)
				require.Equal(t, test.want, want)
			}
		})
	}
}

// TestMessagePasserSafety ensures that the LegacyMessagePasser contract reverts when it is called
// with incorrect calldata. The function signature is correct but the calldata is not abi encoded
// correctly. It is expected the solidity reverts when it cannot abi decode the calldata correctly.
// Only a call to `passMessageToL1` with abi encoded `bytes` will result in the `successfulMessages`
// mapping being updated.
func TestMessagePasserSafety(t *testing.T) {
	testKey, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	testAddr := crypto.PubkeyToAddress(testKey.PublicKey)
	opts, err := bind.NewKeyedTransactorWithChainID(testKey, big.NewInt(1337))
	require.NoError(t, err)

	backend := backends.NewSimulatedBackend(
		core.GenesisAlloc{testAddr: {Balance: big.NewInt(10000000000000000)}},
		30_000_000,
	)
	defer backend.Close()

	// deploy the LegacyMessagePasser contract
	addr, tx, contract, err := bindings.DeployLegacyMessagePasser(opts, backend)
	require.NoError(t, err)

	backend.Commit()
	_, err = bind.WaitMined(context.Background(), backend, tx)
	require.NoError(t, err)

	// ensure that it deployed
	code, err := backend.CodeAt(context.Background(), addr, nil)
	require.NoError(t, err)
	require.True(t, len(code) > 0)

	// dummy message
	msg := []byte{0x00, 0x01, 0x02, 0x03}

	// call `passMessageToL1`
	msgTx, err := contract.PassMessageToL1(opts, msg)
	require.NoError(t, err)

	// ensure that the receipt is successful
	backend.Commit()
	msgReceipt, err := bind.WaitMined(context.Background(), backend, msgTx)
	require.NoError(t, err)
	require.Equal(t, msgReceipt.Status, types.ReceiptStatusSuccessful)

	// check for the data in the `successfulMessages` mapping
	data := make([]byte, len(msg)+len(testAddr))
	copy(data[:], msg)
	copy(data[len(msg):], testAddr.Bytes())
	digest := crypto.Keccak256Hash(data)
	contains, err := contract.SentMessages(&bind.CallOpts{}, digest)
	require.NoError(t, err)
	require.True(t, contains)

	// build a transaction with improperly formatted calldata
	nonce, err := backend.NonceAt(context.Background(), testAddr, nil)
	require.NoError(t, err)
	// append msg without abi encoding it
	selector := crypto.Keccak256([]byte("passMessageToL1(bytes)"))[0:4]
	require.Equal(t, selector, hexutil.MustDecode("0xcafa81dc"))
	calldata := append(selector, msg...)
	faultyTransaction, err := opts.Signer(testAddr, types.NewTx(&types.DynamicFeeTx{
		ChainID:   big.NewInt(1337),
		Nonce:     nonce,
		GasTipCap: msgTx.GasTipCap(),
		GasFeeCap: msgTx.GasFeeCap(),
		Gas:       msgTx.Gas() * 2,
		To:        msgTx.To(),
		Data:      calldata,
	}))
	require.NoError(t, err)
	err = backend.SendTransaction(context.Background(), faultyTransaction)
	require.NoError(t, err)

	// the transaction should revert
	backend.Commit()
	badReceipt, err := bind.WaitMined(context.Background(), backend, faultyTransaction)
	require.NoError(t, err)
	require.Equal(t, badReceipt.Status, types.ReceiptStatusFailed)

	// test the transaction calldata against the abi unpacking
	abi, err := bindings.LegacyMessagePasserMetaData.GetAbi()
	require.NoError(t, err)
	method, err := abi.MethodById(selector)
	require.NoError(t, err)
	require.Equal(t, method.Name, "passMessageToL1")

	// the faulty transaction has the correct 4 byte selector but doesn't
	// have abi encoded bytes following it
	require.Equal(t, faultyTransaction.Data()[:4], selector)
	_, err = method.Inputs.Unpack(faultyTransaction.Data()[4:])
	require.Error(t, err)

	// the original transaction has the correct 4 byte selector and abi encoded bytes
	_, err = method.Inputs.Unpack(msgTx.Data()[4:])
	require.NoError(t, err)
}
