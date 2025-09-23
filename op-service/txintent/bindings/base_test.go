package bindings

import (
	"encoding/hex"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

type TestGameSearchResult struct {
	Index     *big.Int
	Metadata  [32]byte
	Timestamp uint64
	RootClaim [32]byte
	ExtraData []byte
}

type TestProvenWithdrawalsResult struct {
	DisputeGameProxy common.Address
	Timestamp        uint64
}

type TestGameData struct {
	GameType  uint32
	RootClaim [32]byte
	Extradata []byte
}

type TestNestedDynamicStruct struct {
	A *big.Int
	B common.Address
	C []byte
	D struct {
		E *big.Int
		F []byte
	}
}

type TestDoubleNestedStruct struct {
	A *big.Int
	B common.Address
	C []byte
	D struct {
		E *big.Int
		F []byte
		G struct {
			H []byte
			I *big.Int
		}
	}
}

type TestStruct struct {
	B *big.Int
	C []byte
}

type TestDynamicSlice struct {
	A []TestStruct
}

type TestStruct2 struct {
	B common.Address
	C []byte
}

type TestDynamicArray struct {
	A [2]TestStruct2
}

type TestCustomTypeStruct struct {
	A eth.ETH
	B eth.ChainID
}

type TestDynamicIntStruct struct {
	A *Uint128
	B []byte
	C Int128
}

type TestIntStruct struct {
	A *Uint128
	B Int128
}

type TestContract struct {
	FinalizeWithdrawalTransaction func(tx struct {
		Nonce    *big.Int
		Sender   common.Address
		Target   common.Address
		Value    *big.Int
		GasLimit *big.Int
		Data     []byte
	}) TypedCall[any] `sol:"finalizeWithdrawalTransaction"`

	ProveWithdrawalTransaction func(tx struct {
		Nonce    *big.Int
		Sender   common.Address
		Target   common.Address
		Value    *big.Int
		GasLimit *big.Int
		Data     []byte
	}, disputeGameIndex *big.Int, outputRootProof struct {
		Version                  [32]byte
		StateRoot                [32]byte
		MessagePasserStorageRoot [32]byte
		LatestBlockhash          [32]byte
	}, withdrawalProof [][]byte) TypedCall[any] `sol:"proveWithdrawalTransaction"`

	FindLatestGames func(gameType uint32, start *big.Int, n *big.Int) TypedCall[[]TestGameSearchResult] `sol:"findLatestGames"`

	ProvenWithdrawals func(withdrawalHash [32]byte, submitter common.Address) TypedCall[TestProvenWithdrawalsResult] `sol:"provenWithdrawals"`

	GetRequiredBond func(position *Uint128) TypedCall[*big.Int] `sol:"getRequiredBond"`

	GameData func() TypedCall[TestGameData] `sol:"gameData"`

	TestFunc1 func() TypedCall[TestNestedDynamicStruct] `sol:"testfunc1"`

	TestFunc2 func() TypedCall[TestDynamicSlice] `sol:"testfunc2"`
	TestFunc3 func() TypedCall[[]TestStruct]     `sol:"testfunc3"`

	TestFunc4 func() TypedCall[TestDynamicArray] `sol:"testfunc4"`
	TestFunc5 func() TypedCall[[2]TestStruct2]   `sol:"testfunc5"`

	TestFunc6 func() TypedCall[TestDoubleNestedStruct] `sol:"testfunc6"`

	TestFunc7  func() TypedCall[Uint128]               `sol:"testfunc7"`
	TestFunc8  func() TypedCall[Int128]                `sol:"testfunc8"`
	TestFunc9  func(t TestIntStruct) TypedCall[any]    `sol:"testfunc9"`
	TestFunc10 func(t []TestIntStruct) TypedCall[any]  `sol:"testfunc10"`
	TestFunc11 func(t [3]TestIntStruct) TypedCall[any] `sol:"testfunc11"`

	TestFunc12 func(t [2]TestDynamicIntStruct) TypedCall[any] `sol:"testfunc12"`

	TestFunc13 func() TypedCall[*Uint128] `sol:"testfunc13"`
	TestFunc14 func() TypedCall[*Int128]  `sol:"testfunc14"`
}

func TestCustomIntConversion(t *testing.T) {
	type TestRecursivePointerStruct struct {
		A Uint128
		B ****Uint128
	}
	type ComplexStruct struct {
		A *Uint128
		B *big.Int
		C *TestStruct2
		D Int128
		E TestRecursivePointerStruct
	}
	v := Uint128(*big.NewInt(4321))
	a := &v
	b := &a
	c := &b
	d := &c
	arg := ComplexStruct{
		A: (*Uint128)(big.NewInt(1337)),
		B: big.NewInt(7331),
		C: &TestStruct2{
			B: common.Address{},
			C: []byte{0x12, 0x34},
		},
		D: Int128(*big.NewInt(-7331)),
		E: TestRecursivePointerStruct{
			A: Uint128(*big.NewInt(1234)),
			B: d,
		},
	}
	switch v := any(arg).(type) {
	default:
		converted, err := ReplaceCustomInts(v)
		require.NoError(t, err)
		w := reflect.ValueOf(converted)
		fieldA := w.FieldByName("A").Interface().(*big.Int)
		require.True(t, big.NewInt(1337).Cmp(fieldA) == 0, "A mismatch")
		fieldB := w.FieldByName("B").Interface().(*big.Int)
		require.True(t, big.NewInt(7331).Cmp(fieldB) == 0, "B mismatch")
		fieldC := w.FieldByName("C")
		require.True(t, fieldC.IsValid() && !fieldC.IsNil(), "C is nil")
		fieldCDeref := fieldC.Elem()
		fieldCB := fieldCDeref.FieldByName("B").Interface().([20]uint8)
		require.Equal(t, [20]uint8{}, fieldCB, "C.B mismatch")
		fieldCC := fieldCDeref.FieldByName("C").Interface().([]byte)
		require.Equal(t, []byte{0x12, 0x34}, fieldCC, "C.C mismatch")
		fieldD := w.FieldByName("D").Interface().(*big.Int)
		require.True(t, big.NewInt(-7331).Cmp(fieldD) == 0, "D mismatch")
		fieldE := w.FieldByName("E")
		require.True(t, fieldE.IsValid(), "E invalid")
		fieldEA := fieldE.FieldByName("A").Interface().(*big.Int)
		require.True(t, big.NewInt(1234).Cmp(fieldEA) == 0, "E.A mismatch")
		fieldEB := fieldE.FieldByName("B").Interface().(*big.Int)
		require.True(t, big.NewInt(4321).Cmp(fieldEB) == 0, "E.B mismatch")
	}
}

func TestEncodeStruct(t *testing.T) {
	testContract := NewBindings[TestContract]()

	call := testContract.FinalizeWithdrawalTransaction(
		struct {
			Nonce    *big.Int
			Sender   common.Address
			Target   common.Address
			Value    *big.Int
			GasLimit *big.Int
			Data     []byte
		}{
			Nonce:    new(big.Int).Lsh(big.NewInt(1), 240),
			Sender:   common.HexToAddress("0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65"),
			Target:   common.HexToAddress("0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65"),
			Value:    big.NewInt(500000000000),
			GasLimit: big.NewInt(21000),
			Data:     []byte(""),
		},
	)

	calldata, err := call.EncodeInputLambda()
	require.NoError(t, err)
	require.Equal(t, "8c3152e90000000000000000000000000000000000000000000000000000000000000020000100000000000000000000000000000000000000000000000000000000000000000000000000000000000015d34aaf54267db7d7c367839aaf71a00a2c6a6500000000000000000000000015d34aaf54267db7d7c367839aaf71a00a2c6a65000000000000000000000000000000000000000000000000000000746a528800000000000000000000000000000000000000000000000000000000000000520800000000000000000000000000000000000000000000000000000000000000c00000000000000000000000000000000000000000000000000000000000000000",
		hex.EncodeToString(calldata),
	)

	call = testContract.ProveWithdrawalTransaction(
		struct {
			Nonce    *big.Int
			Sender   common.Address
			Target   common.Address
			Value    *big.Int
			GasLimit *big.Int
			Data     []byte
		}{
			Nonce:    new(big.Int).Lsh(big.NewInt(1), 240),
			Sender:   common.HexToAddress("0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65"),
			Target:   common.HexToAddress("0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65"),
			Value:    big.NewInt(500000000000),
			GasLimit: big.NewInt(21000),
			Data:     []byte(""),
		},
		big.NewInt(1),
		struct {
			Version                  [32]byte
			StateRoot                [32]byte
			MessagePasserStorageRoot [32]byte
			LatestBlockhash          [32]byte
		}{
			Version:                  *(*[32]byte)(hexutil.MustDecode("0x0000000000000000000000000000000000000000000000000000000000000000")),
			StateRoot:                *(*[32]byte)(hexutil.MustDecode("0x73aa3ddeddee968a18a19312efccd06ebe116f86e3f23961cc83ef26346894ba")),
			MessagePasserStorageRoot: *(*[32]byte)(hexutil.MustDecode("0xe3f2a88ce530a8dab9f8cafac0ef934b1f126da1041d89e41cb84d46dfa5e841")),
			LatestBlockhash:          *(*[32]byte)(hexutil.MustDecode("0xf79e208e723e8ca525558786b4fc73c1c889e9eb0e25917ba5c5ec7640ffc257")),
		},
		[][]byte{
			hexutil.MustDecode("0xf8718080808080a08e9a5e2311b6926cff4a3b9b50fd0500e2d68f2d70c62f7b294aec18b62e94d980a08c82f7353a759f9fdf815a3065d8e8b1282d1383398e53f11f8f03bf64f50cfa808080a0f4984a11f61a2921456141df88de6e1a710d28681b91af794c5a721e47839cd78080808080"),
			hexutil.MustDecode("0xf8518080a0999c5deb49aff57f74c1a5871afb58461105ec7bf684c9716f8ee2c30221bfd78080808080808080a05219be3ea6e6c12cfa6927fd85a1548be9922594ebbd7d8ad717600fbd64f7fe8080808080"),
			hexutil.MustDecode("0xe2a0206c4fd0e580d501e7a56378cab19a4875bba79b4639cdbd1db734feb96f87dd01"),
		},
	)

	calldata, err = call.EncodeInputLambda()
	require.NoError(t, err)
	require.Equal(t, "4870496f00000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000073aa3ddeddee968a18a19312efccd06ebe116f86e3f23961cc83ef26346894bae3f2a88ce530a8dab9f8cafac0ef934b1f126da1041d89e41cb84d46dfa5e841f79e208e723e8ca525558786b4fc73c1c889e9eb0e25917ba5c5ec7640ffc25700000000000000000000000000000000000000000000000000000000000001c0000100000000000000000000000000000000000000000000000000000000000000000000000000000000000015d34aaf54267db7d7c367839aaf71a00a2c6a6500000000000000000000000015d34aaf54267db7d7c367839aaf71a00a2c6a65000000000000000000000000000000000000000000000000000000746a528800000000000000000000000000000000000000000000000000000000000000520800000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000030000000000000000000000000000000000000000000000000000000000000060000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000001800000000000000000000000000000000000000000000000000000000000000073f8718080808080a08e9a5e2311b6926cff4a3b9b50fd0500e2d68f2d70c62f7b294aec18b62e94d980a08c82f7353a759f9fdf815a3065d8e8b1282d1383398e53f11f8f03bf64f50cfa808080a0f4984a11f61a2921456141df88de6e1a710d28681b91af794c5a721e47839cd78080808080000000000000000000000000000000000000000000000000000000000000000000000000000000000000000053f8518080a0999c5deb49aff57f74c1a5871afb58461105ec7bf684c9716f8ee2c30221bfd78080808080808080a05219be3ea6e6c12cfa6927fd85a1548be9922594ebbd7d8ad717600fbd64f7fe8080808080000000000000000000000000000000000000000000000000000000000000000000000000000000000000000023e2a0206c4fd0e580d501e7a56378cab19a4875bba79b4639cdbd1db734feb96f87dd010000000000000000000000000000000000000000000000000000000000",
		hex.EncodeToString(calldata),
	)
}

func TestEncodeCustomIntStruct(t *testing.T) {
	testContract := NewBindings[TestContract]()

	{
		call := testContract.GetRequiredBond((*Uint128)(big.NewInt(1337)))
		calldata, err := call.EncodeInputLambda()
		require.NoError(t, err)
		require.Equal(t, "c395e1ca0000000000000000000000000000000000000000000000000000000000000539",
			hex.EncodeToString(calldata),
		)
	}
	{
		arg := TestIntStruct{
			A: (*Uint128)(big.NewInt(1337)),
			B: Int128(*big.NewInt(-7331)),
		}
		call := testContract.TestFunc9(arg)
		calldata, err := call.EncodeInputLambda()
		require.NoError(t, err)
		require.Equal(t, "d24d4e560000000000000000000000000000000000000000000000000000000000000539ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe35d",
			hex.EncodeToString(calldata),
		)
	}
	{
		arg := [3]TestIntStruct{
			{
				A: (*Uint128)(big.NewInt(1337)),
				B: Int128(*big.NewInt(-1)),
			},
			{
				A: (*Uint128)(big.NewInt(123456789123456789)),
				B: Int128(*big.NewInt(-123456789)),
			},
			{
				A: (*Uint128)(big.NewInt(13)),
				B: Int128(*big.NewInt(-37)),
			},
		}
		call := testContract.TestFunc11(arg)
		calldata, err := call.EncodeInputLambda()
		require.NoError(t, err)
		require.Equal(t, "a86faa120000000000000000000000000000000000000000000000000000000000000539ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00000000000000000000000000000000000000000000000001b69b4bacd05f15fffffffffffffffffffffffffffffffffffffffffffffffffffffffff8a432eb000000000000000000000000000000000000000000000000000000000000000dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffdb",
			hex.EncodeToString(calldata),
		)

		call = testContract.TestFunc10(arg[:])
		calldata, err = call.EncodeInputLambda()
		require.NoError(t, err)
		require.Equal(t, "b3e163fe000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000030000000000000000000000000000000000000000000000000000000000000539ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00000000000000000000000000000000000000000000000001b69b4bacd05f15fffffffffffffffffffffffffffffffffffffffffffffffffffffffff8a432eb000000000000000000000000000000000000000000000000000000000000000dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffdb",
			hex.EncodeToString(calldata),
		)
	}
}

func TestEncodeCustomIntStructWithDynamic(t *testing.T) {
	testContract := NewBindings[TestContract]()

	{
		arg := [2]TestDynamicIntStruct{
			{
				A: (*Uint128)(big.NewInt(1337)),
				B: []byte{0x13, 0x33, 0x37},
				C: Int128(*big.NewInt(-7331)),
			},
			{
				A: (*Uint128)(big.NewInt(13)),
				B: []byte{0x37, 0x33, 0x33, 0x31},
				C: Int128(*big.NewInt(-24)),
			},
		}
		call := testContract.TestFunc12(arg)
		calldata, err := call.EncodeInputLambda()
		require.NoError(t, err)
		require.Equal(t, "286d447b0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000e000000000000000000000000000000000000000000000000000000000000005390000000000000000000000000000000000000000000000000000000000000060ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe35d00000000000000000000000000000000000000000000000000000000000000031333370000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000d0000000000000000000000000000000000000000000000000000000000000060ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe800000000000000000000000000000000000000000000000000000000000000043733333100000000000000000000000000000000000000000000000000000000",
			hex.EncodeToString(calldata),
		)
	}
}

func TestDecodeCustomInt(t *testing.T) {
	testContract := NewBindings[TestContract]()
	{
		call := testContract.TestFunc7()
		data := hexutil.MustDecode("0x0000000000000000000000000000000000000000000000000000000000000539")
		value, err := call.DecodeOutput(data)
		require.NoError(t, err)
		require.True(t, big.NewInt(1337).Cmp(value.ToBig()) == 0)
	}
	{
		call := testContract.TestFunc8()
		data := hexutil.MustDecode("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe35d")
		value, err := call.DecodeOutput(data)
		require.NoError(t, err)
		require.True(t, big.NewInt(-7331).Cmp(value.ToBig()) == 0)
	}
	{
		call := testContract.TestFunc13()
		data := hexutil.MustDecode("0x0000000000000000000000000000000000000000000000000000000000000539")
		value, err := call.DecodeOutput(data)
		require.NoError(t, err)
		require.True(t, big.NewInt(1337).Cmp(value.ToBig()) == 0)
	}
	{
		call := testContract.TestFunc14()
		data := hexutil.MustDecode("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe35d")
		value, err := call.DecodeOutput(data)
		require.NoError(t, err)
		require.True(t, big.NewInt(-7331).Cmp(value.ToBig()) == 0)
	}
}

func TestDecodeArray(t *testing.T) {
	testContract := NewBindings[TestContract]()

	call := testContract.FindLatestGames(0, big.NewInt(0), big.NewInt(0))

	data := hexutil.MustDecode("0x00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000000000000100000000683ed2147c319523d93cee2cf01b19db5bdc88a8aff79bda00000000000000000000000000000000000000000000000000000000683ed2140fa71262076cb482e6f983cf3dd7eccb8f076d5c7aac1c5f8f5191eed2ad3bf600000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000003")
	games, err := call.DecodeOutput(data)
	require.NoError(t, err)

	require.Equal(t, 1, len(games))
	game := games[0]

	require.True(t, big.NewInt(0).Cmp(game.Index) == 0)
	require.Equal(t, *(*[32]byte)(hexutil.MustDecode("0x0000000100000000683ed2147c319523d93cee2cf01b19db5bdc88a8aff79bda")), game.Metadata)
	require.Equal(t, uint64(1748947476), game.Timestamp)
	require.Equal(t, *(*[32]byte)(hexutil.MustDecode("0x0fa71262076cb482e6f983cf3dd7eccb8f076d5c7aac1c5f8f5191eed2ad3bf6")), game.RootClaim)
	require.Equal(t, hexutil.MustDecode("0x0000000000000000000000000000000000000000000000000000000000000003"), game.ExtraData)
}

func TestDecodeStaticStruct(t *testing.T) {
	testContract := NewBindings[TestContract]()

	call := testContract.ProvenWithdrawals([32]byte{}, common.Address{})

	data := hexutil.MustDecode("0x00000000000000000000000046d257cf3803b353350ec1edc6aa106f355f3bd200000000000000000000000000000000000000000000000000000000683feed9")
	result, err := call.DecodeOutput(data)
	require.NoError(t, err)

	require.Equal(t, common.HexToAddress("0x46D257cf3803b353350ec1Edc6AA106f355F3bd2"), result.DisputeGameProxy)
	require.Equal(t, uint64(1749020377), result.Timestamp)
}

func TestDecodeDynamicStruct(t *testing.T) {
	testContract := NewBindings[TestContract]()

	call := testContract.GameData()

	data := hexutil.MustDecode("0x00000000000000000000000000000000000000000000000000000000000000fec0ced67668cc6e8e63517245aa7e34053a1332eb4303f3169b6051810e277036000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000015")
	result, err := call.DecodeOutput(data)
	require.NoError(t, err)

	require.Equal(t, uint32(254), result.GameType)
	require.Equal(t, *(*[32]byte)(hexutil.MustDecode("0xc0ced67668cc6e8e63517245aa7e34053a1332eb4303f3169b6051810e277036")), result.RootClaim)
	require.Equal(t, hexutil.MustDecode("0x0000000000000000000000000000000000000000000000000000000000000015"), result.Extradata)
}

func TestDecodeNestedDynamicStruct(t *testing.T) {
	testContract := NewBindings[TestContract]()

	call := testContract.TestFunc1()

	data := hexutil.MustDecode("0x000000000000000000000000000000000000000000000000000000000000007b000000000000000000000000abc123abc123abc123abc123abc123abc123abc1000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000c00000000000000000000000000000000000000000000000000000000000000004133773310000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001c800000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000004deadbeef00000000000000000000000000000000000000000000000000000000")
	result, err := call.DecodeOutput(data)
	require.NoError(t, err)

	require.True(t, new(big.Int).SetUint64(123).Cmp(result.A) == 0)
	require.Equal(t, common.HexToAddress("0xabc123abc123abc123abc123abc123abc123abc1"), result.B)
	require.Equal(t, hexutil.MustDecode("0x13377331"), result.C)
	require.True(t, new(big.Int).SetUint64(456).Cmp(result.D.E) == 0)
	require.Equal(t, hexutil.MustDecode("0xdeadbeef"), result.D.F)
}

func TestDecodeDynamicSlice(t *testing.T) {
	testContract := NewBindings[TestContract]()

	data := hexutil.MustDecode("0x00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000004deadbeef00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000004beefcafe00000000000000000000000000000000000000000000000000000000")

	{
		call := testContract.TestFunc2()
		result, err := call.DecodeOutput(data)
		require.NoError(t, err)
		require.Equal(t, 2, len(result.A))
		require.True(t, new(big.Int).SetUint64(1).Cmp(result.A[0].B) == 0)
		require.Equal(t, hexutil.MustDecode("0xdeadbeef"), result.A[0].C)
		require.True(t, new(big.Int).SetUint64(2).Cmp(result.A[1].B) == 0)
		require.Equal(t, hexutil.MustDecode("0xbeefcafe"), result.A[1].C)
	}
	{
		call := testContract.TestFunc3()
		result, err := call.DecodeOutput(data)
		require.NoError(t, err)
		require.Equal(t, 2, len(result))
		require.True(t, new(big.Int).SetUint64(1).Cmp(result[0].B) == 0)
		require.Equal(t, hexutil.MustDecode("0xdeadbeef"), result[0].C)
		require.True(t, new(big.Int).SetUint64(2).Cmp(result[1].B) == 0)
		require.Equal(t, hexutil.MustDecode("0xbeefcafe"), result[1].C)
	}
}

func TestDecodeDynamicArray(t *testing.T) {
	testContract := NewBindings[TestContract]()

	data := hexutil.MustDecode("0x0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000111111111111111111111111111111111111111100000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000003abcdef00000000000000000000000000000000000000000000000000000000000000000000000000000000002222222222222222222222222222222222222222000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000031234560000000000000000000000000000000000000000000000000000000000")

	{
		call := testContract.TestFunc4()
		result, err := call.DecodeOutput(data)
		require.NoError(t, err)
		require.Equal(t, common.HexToAddress("0x1111111111111111111111111111111111111111"), result.A[0].B)
		require.Equal(t, hexutil.MustDecode("0xabcdef"), result.A[0].C)
		require.Equal(t, common.HexToAddress("0x2222222222222222222222222222222222222222"), result.A[1].B)
		require.Equal(t, hexutil.MustDecode("0x123456"), result.A[1].C)
	}
	{
		call := testContract.TestFunc5()
		result, err := call.DecodeOutput(data)
		require.NoError(t, err)
		require.Equal(t, 2, len(result))
		require.Equal(t, common.HexToAddress("0x1111111111111111111111111111111111111111"), result[0].B)
		require.Equal(t, hexutil.MustDecode("0xabcdef"), result[0].C)
		require.Equal(t, common.HexToAddress("0x2222222222222222222222222222222222222222"), result[1].B)
		require.Equal(t, hexutil.MustDecode("0x123456"), result[1].C)
	}
}

func TestDoubleTestedStruct(t *testing.T) {
	testContract := NewBindings[TestContract]()

	call := testContract.TestFunc6()
	data := hexutil.MustDecode("0x00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000dead000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000000212340000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000000002abcd0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000000300000000000000000000000000000000000000000000000000000000000000015500000000000000000000000000000000000000000000000000000000000000")
	result, err := call.DecodeOutput(data)
	require.NoError(t, err)

	require.True(t, new(big.Int).SetUint64(1).Cmp(result.A) == 0)
	require.Equal(t, common.HexToAddress("0x000000000000000000000000000000000000dEaD"), result.B)
	require.Equal(t, hexutil.MustDecode("0x1234"), result.C)
	require.True(t, new(big.Int).SetUint64(2).Cmp(result.D.E) == 0)
	require.Equal(t, hexutil.MustDecode("0xabcd"), result.D.F)
	require.Equal(t, hexutil.MustDecode("0x55"), result.D.G.H)
	require.True(t, new(big.Int).SetUint64(3).Cmp(result.D.G.I) == 0)
}
