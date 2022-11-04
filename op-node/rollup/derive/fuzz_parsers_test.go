package derive

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm/runtime"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/testutils"
)

var (
	pk, _                  = crypto.GenerateKey()
	addr                   = common.Address{0x42, 0xff}
	opts, _                = bind.NewKeyedTransactorWithChainID(pk, common.Big1)
	from                   = crypto.PubkeyToAddress(pk.PublicKey)
	portalContract, _      = bindings.NewOptimismPortal(addr, nil)
	l1BlockInfoContract, _ = bindings.NewL1Block(addr, nil)
)

func cap_byte_slice(b []byte, c int) []byte {
	if len(b) <= c {
		return b
	} else {
		return b[:c]
	}
}

func BytesToBigInt(b []byte) *big.Int {
	return new(big.Int).SetBytes(cap_byte_slice(b, 32))
}

// FuzzL1InfoRoundTrip checks that our encoder round trips properly
func FuzzL1InfoRoundTrip(f *testing.F) {
	f.Fuzz(func(t *testing.T, number, time uint64, baseFee, hash []byte, seqNumber uint64) {
		in := L1BlockInfo{
			Number:         number,
			Time:           time,
			BaseFee:        BytesToBigInt(baseFee),
			BlockHash:      common.BytesToHash(hash),
			SequenceNumber: seqNumber,
		}
		enc, err := in.MarshalBinary()
		if err != nil {
			t.Fatalf("Failed to marshal binary: %v", err)
		}
		var out L1BlockInfo
		err = out.UnmarshalBinary(enc)
		if err != nil {
			t.Fatalf("Failed to unmarshal binary: %v", err)
		}
		if !cmp.Equal(in, out, cmp.Comparer(testutils.BigEqual)) {
			t.Fatalf("The data did not round trip correctly. in: %v. out: %v", in, out)
		}

	})
}

// FuzzL1InfoAgainstContract checks the custom marshalling functions against the contract
// bindings to ensure that our functions are up to date and match the bindings.
func FuzzL1InfoAgainstContract(f *testing.F) {
	f.Fuzz(func(t *testing.T, number, time uint64, baseFee, hash []byte, seqNumber uint64, batcherHash []byte, l1FeeOverhead []byte, l1FeeScalar []byte) {
		expected := L1BlockInfo{
			Number:         number,
			Time:           time,
			BaseFee:        BytesToBigInt(baseFee),
			BlockHash:      common.BytesToHash(hash),
			SequenceNumber: seqNumber,
			BatcherAddr:    common.BytesToAddress(batcherHash),
			L1FeeOverhead:  eth.Bytes32(common.BytesToHash(l1FeeOverhead)),
			L1FeeScalar:    eth.Bytes32(common.BytesToHash(l1FeeScalar)),
		}

		// Setup opts
		opts.GasPrice = big.NewInt(100)
		opts.GasLimit = 100_000
		opts.NoSend = true
		opts.Nonce = common.Big0
		// Create the SetL1BlockValues transaction
		tx, err := l1BlockInfoContract.SetL1BlockValues(
			opts,
			number,
			time,
			BytesToBigInt(baseFee),
			common.BytesToHash(hash),
			seqNumber,
			common.BytesToAddress(batcherHash).Hash(),
			common.BytesToHash(l1FeeOverhead).Big(),
			common.BytesToHash(l1FeeScalar).Big(),
		)
		if err != nil {
			t.Fatalf("Failed to create the transaction: %v", err)
		}

		// Check that our encoder produces the same value and that we
		// can decode the contract values exactly
		enc, err := expected.MarshalBinary()
		if err != nil {
			t.Fatalf("Failed to marshal binary: %v", err)
		}
		if !bytes.Equal(enc, tx.Data()) {
			t.Logf("encoded  %x", enc)
			t.Logf("expected %x", tx.Data())
			t.Fatalf("Custom marshal does not match contract bindings")
		}

		var actual L1BlockInfo
		err = actual.UnmarshalBinary(tx.Data())
		if err != nil {
			t.Fatalf("Failed to unmarshal binary: %v", err)
		}

		if !cmp.Equal(expected, actual, cmp.Comparer(testutils.BigEqual)) {
			t.Fatalf("The data did not round trip correctly. expected: %v. actual: %v", expected, actual)
		}

	})
}

// Standard ABI types copied from golang ABI tests
var (
	Uint256Type, _ = abi.NewType("uint256", "", nil)
	Uint64Type, _  = abi.NewType("uint64", "", nil)
	BytesType, _   = abi.NewType("bytes", "", nil)
	BoolType, _    = abi.NewType("bool", "", nil)
	AddressType, _ = abi.NewType("address", "", nil)
)

// EncodeDepositOpaqueDataV0 performs ABI encoding to create the opaque data field of the deposit event.
func EncodeDepositOpaqueDataV0(t *testing.T, mint *big.Int, value *big.Int, gasLimit uint64, isCreation bool, data []byte) []byte {
	// in OptimismPortal.sol:
	// bytes memory opaqueData = abi.encodePacked(msg.value, _value, _gasLimit, _isCreation, _data);
	// Geth does not support abi.encodePacked, so we emulate it here by slicing of the padding from the individual elements
	// See https://github.com/ethereum/go-ethereum/issues/22257
	// And https://docs.soliditylang.org/en/v0.8.13/abi-spec.html#non-standard-packed-mode

	var out []byte

	v, err := abi.Arguments{{Name: "msg.value", Type: Uint256Type}}.Pack(mint)
	require.NoError(t, err)
	out = append(out, v...)

	v, err = abi.Arguments{{Name: "_value", Type: Uint256Type}}.Pack(value)
	require.NoError(t, err)
	out = append(out, v...)

	v, err = abi.Arguments{{Name: "_gasLimit", Type: Uint64Type}}.Pack(gasLimit)
	require.NoError(t, err)
	out = append(out, v[32-8:]...) // 8 bytes only with abi.encodePacked

	v, err = abi.Arguments{{Name: "_isCreation", Type: BoolType}}.Pack(isCreation)
	require.NoError(t, err)
	out = append(out, v[32-1:]...) // 1 byte only with abi.encodePacked

	// no slice header, just the raw data with abi.encodePacked
	out = append(out, data...)

	return out
}

// FuzzUnmarshallLogEvent runs a deposit event through the EVM and checks that output of the abigen parsing matches
// what was inputted and what we parsed during the UnmarshalDepositLogEvent function (which turns it into a deposit tx)
// The purpose is to check that we can never create a transaction that emits a log that we cannot parse as well
// as ensuring that our custom marshalling matches abigen.
func FuzzUnmarshallLogEvent(f *testing.F) {
	b := func(i int64) []byte {
		return big.NewInt(i).Bytes()
	}
	type setup struct {
		to         common.Address
		mint       int64
		value      int64
		gasLimit   uint64
		data       string
		isCreation bool
	}
	cases := []setup{
		{
			mint:     100,
			value:    50,
			gasLimit: 100000,
		},
	}
	for _, c := range cases {
		f.Add(c.to.Bytes(), b(c.mint), b(c.value), []byte(c.data), c.gasLimit, c.isCreation)
	}

	f.Fuzz(func(t *testing.T, _to, _mint, _value, data []byte, l2GasLimit uint64, isCreation bool) {
		to := common.BytesToAddress(_to)
		mint := BytesToBigInt(_mint)
		value := BytesToBigInt(_value)

		// Setup opts
		opts.Value = mint
		opts.GasPrice = common.Big2
		opts.GasLimit = 500_000
		opts.NoSend = true
		opts.Nonce = common.Big0
		// Create the deposit transaction
		tx, err := portalContract.DepositTransaction(opts, to, value, l2GasLimit, isCreation, data)
		if err != nil {
			t.Fatal(err)
		}

		state, err := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
		if err != nil {
			t.Fatal(err)
		}
		state.SetBalance(from, BytesToBigInt([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}))
		state.SetCode(addr, common.FromHex(bindings.OptimismPortalDeployedBin))
		_, err = state.Commit(false)
		if err != nil {
			t.Fatal(err)
		}

		cfg := runtime.Config{
			Origin:   from,
			Value:    tx.Value(),
			State:    state,
			GasLimit: opts.GasLimit,
		}

		_, _, err = runtime.Call(addr, tx.Data(), &cfg)
		logs := state.Logs()
		if err == nil && len(logs) != 1 {
			t.Fatal("No logs or error after execution")
		} else if err != nil {
			return
		}

		// Test that our custom parsing matches the ABI parsing
		depositEvent, err := portalContract.ParseTransactionDeposited(*(logs[0]))
		if err != nil {
			t.Fatalf("Could not parse log that was emitted by the deposit contract: %v", err)
		}
		depositEvent.Raw = types.Log{} // Clear out the log

		// Verify that is passes our custom unmarshalling logic
		dep, err := UnmarshalDepositLogEvent(logs[0])
		if err != nil {
			t.Fatalf("Could not unmarshal log that was emitted by the deposit contract: %v", err)
		}
		depMint := common.Big0
		if dep.Mint != nil {
			depMint = dep.Mint
		}
		opaqueData := EncodeDepositOpaqueDataV0(t, depMint, dep.Value, dep.Gas, dep.To == nil, dep.Data)

		reconstructed := &bindings.OptimismPortalTransactionDeposited{
			From:       dep.From,
			Version:    common.Big0,
			OpaqueData: opaqueData,
			Raw:        types.Log{},
		}
		if dep.To != nil {
			reconstructed.To = *dep.To
		}

		if !cmp.Equal(depositEvent, reconstructed, cmp.Comparer(testutils.BigEqual)) {
			t.Fatalf("The deposit tx did not match. tx: %v. actual: %v", reconstructed, depositEvent)
		}

		opaqueData = EncodeDepositOpaqueDataV0(t, mint, value, l2GasLimit, isCreation, data)

		inputArgs := &bindings.OptimismPortalTransactionDeposited{
			From:       from,
			To:         to,
			Version:    common.Big0,
			OpaqueData: opaqueData,
			Raw:        types.Log{},
		}
		if !cmp.Equal(depositEvent, inputArgs, cmp.Comparer(testutils.BigEqual)) {
			t.Fatalf("The input args did not match. input: %v. actual: %v", inputArgs, depositEvent)
		}
	})
}
