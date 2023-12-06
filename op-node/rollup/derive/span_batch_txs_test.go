package derive

import (
	"bytes"
	"math/big"
	"math/rand"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

type txTypeTest struct {
	name   string
	mkTx   func(rng *rand.Rand, signer types.Signer) *types.Transaction
	signer types.Signer
}

func TestSpanBatchTxsContractCreationBits(t *testing.T) {
	rng := rand.New(rand.NewSource(0x1234567))
	chainID := big.NewInt(rng.Int63n(1000))

	rawSpanBatch := RandomRawSpanBatch(rng, chainID)
	contractCreationBits := rawSpanBatch.txs.contractCreationBits
	totalBlockTxCount := rawSpanBatch.txs.totalBlockTxCount

	var sbt spanBatchTxs
	sbt.contractCreationBits = contractCreationBits
	sbt.totalBlockTxCount = totalBlockTxCount

	var buf bytes.Buffer
	err := sbt.encodeContractCreationBits(&buf)
	require.NoError(t, err)

	// contractCreationBit field is fixed length: single bit
	contractCreationBitBufferLen := totalBlockTxCount / 8
	if totalBlockTxCount%8 != 0 {
		contractCreationBitBufferLen++
	}
	require.Equal(t, buf.Len(), int(contractCreationBitBufferLen))

	result := buf.Bytes()
	sbt.contractCreationBits = nil

	r := bytes.NewReader(result)
	err = sbt.decodeContractCreationBits(r)
	require.NoError(t, err)

	require.Equal(t, contractCreationBits, sbt.contractCreationBits)
}

func TestSpanBatchTxsContractCreationCount(t *testing.T) {
	rng := rand.New(rand.NewSource(0x1337))
	chainID := big.NewInt(rng.Int63n(1000))

	rawSpanBatch := RandomRawSpanBatch(rng, chainID)

	contractCreationBits := rawSpanBatch.txs.contractCreationBits
	contractCreationCount, err := rawSpanBatch.txs.contractCreationCount()
	require.NoError(t, err)
	totalBlockTxCount := rawSpanBatch.txs.totalBlockTxCount

	var sbt spanBatchTxs
	sbt.contractCreationBits = contractCreationBits
	sbt.totalBlockTxCount = totalBlockTxCount

	var buf bytes.Buffer
	err = sbt.encodeContractCreationBits(&buf)
	require.NoError(t, err)

	result := buf.Bytes()
	sbt.contractCreationBits = nil

	r := bytes.NewReader(result)
	err = sbt.decodeContractCreationBits(r)
	require.NoError(t, err)

	contractCreationCount2, err := sbt.contractCreationCount()
	require.NoError(t, err)

	require.Equal(t, contractCreationCount, contractCreationCount2)
}

func TestSpanBatchTxsYParityBits(t *testing.T) {
	rng := rand.New(rand.NewSource(0x7331))
	chainID := big.NewInt(rng.Int63n(1000))

	rawSpanBatch := RandomRawSpanBatch(rng, chainID)
	yParityBits := rawSpanBatch.txs.yParityBits
	totalBlockTxCount := rawSpanBatch.txs.totalBlockTxCount

	var sbt spanBatchTxs
	sbt.yParityBits = yParityBits
	sbt.totalBlockTxCount = totalBlockTxCount

	var buf bytes.Buffer
	err := sbt.encodeYParityBits(&buf)
	require.NoError(t, err)

	// yParityBit field is fixed length: single bit
	yParityBitBufferLen := totalBlockTxCount / 8
	if totalBlockTxCount%8 != 0 {
		yParityBitBufferLen++
	}
	require.Equal(t, buf.Len(), int(yParityBitBufferLen))

	result := buf.Bytes()
	sbt.yParityBits = nil

	r := bytes.NewReader(result)
	err = sbt.decodeYParityBits(r)
	require.NoError(t, err)

	require.Equal(t, yParityBits, sbt.yParityBits)
}

func TestSpanBatchTxsProtectedBits(t *testing.T) {
	rng := rand.New(rand.NewSource(0x7331))
	chainID := big.NewInt(rng.Int63n(1000))

	rawSpanBatch := RandomRawSpanBatch(rng, chainID)
	protectedBits := rawSpanBatch.txs.protectedBits
	txTypes := rawSpanBatch.txs.txTypes
	totalBlockTxCount := rawSpanBatch.txs.totalBlockTxCount
	totalLegacyTxCount := rawSpanBatch.txs.totalLegacyTxCount

	var sbt spanBatchTxs
	sbt.protectedBits = protectedBits
	sbt.totalBlockTxCount = totalBlockTxCount
	sbt.txTypes = txTypes
	sbt.totalLegacyTxCount = totalLegacyTxCount

	var buf bytes.Buffer
	err := sbt.encodeProtectedBits(&buf)
	require.NoError(t, err)

	// protectedBit field is fixed length: single bit
	protectedBitBufferLen := totalLegacyTxCount / 8
	require.NoError(t, err)
	if totalLegacyTxCount%8 != 0 {
		protectedBitBufferLen++
	}
	require.Equal(t, buf.Len(), int(protectedBitBufferLen))

	result := buf.Bytes()
	sbt.protectedBits = nil

	r := bytes.NewReader(result)
	err = sbt.decodeProtectedBits(r)
	require.NoError(t, err)

	require.Equal(t, protectedBits, sbt.protectedBits)
}

func TestSpanBatchTxsTxSigs(t *testing.T) {
	rng := rand.New(rand.NewSource(0x73311337))
	chainID := big.NewInt(rng.Int63n(1000))

	rawSpanBatch := RandomRawSpanBatch(rng, chainID)
	txSigs := rawSpanBatch.txs.txSigs
	totalBlockTxCount := rawSpanBatch.txs.totalBlockTxCount

	var sbt spanBatchTxs
	sbt.totalBlockTxCount = totalBlockTxCount
	sbt.txSigs = txSigs

	var buf bytes.Buffer
	err := sbt.encodeTxSigsRS(&buf)
	require.NoError(t, err)

	// txSig field is fixed length: 32 byte + 32 byte = 64 byte
	require.Equal(t, buf.Len(), 64*int(totalBlockTxCount))

	result := buf.Bytes()
	sbt.txSigs = nil

	r := bytes.NewReader(result)
	err = sbt.decodeTxSigsRS(r)
	require.NoError(t, err)

	// v field is not set
	for i := 0; i < int(totalBlockTxCount); i++ {
		require.Equal(t, txSigs[i].r, sbt.txSigs[i].r)
		require.Equal(t, txSigs[i].s, sbt.txSigs[i].s)
	}
}

func TestSpanBatchTxsTxNonces(t *testing.T) {
	rng := rand.New(rand.NewSource(0x123456))
	chainID := big.NewInt(rng.Int63n(1000))

	rawSpanBatch := RandomRawSpanBatch(rng, chainID)
	txNonces := rawSpanBatch.txs.txNonces
	totalBlockTxCount := rawSpanBatch.txs.totalBlockTxCount

	var sbt spanBatchTxs
	sbt.totalBlockTxCount = totalBlockTxCount
	sbt.txNonces = txNonces

	var buf bytes.Buffer
	err := sbt.encodeTxNonces(&buf)
	require.NoError(t, err)

	result := buf.Bytes()
	sbt.txNonces = nil

	r := bytes.NewReader(result)
	err = sbt.decodeTxNonces(r)
	require.NoError(t, err)

	require.Equal(t, txNonces, sbt.txNonces)
}

func TestSpanBatchTxsTxGases(t *testing.T) {
	rng := rand.New(rand.NewSource(0x12345))
	chainID := big.NewInt(rng.Int63n(1000))

	rawSpanBatch := RandomRawSpanBatch(rng, chainID)
	txGases := rawSpanBatch.txs.txGases
	totalBlockTxCount := rawSpanBatch.txs.totalBlockTxCount

	var sbt spanBatchTxs
	sbt.totalBlockTxCount = totalBlockTxCount
	sbt.txGases = txGases

	var buf bytes.Buffer
	err := sbt.encodeTxGases(&buf)
	require.NoError(t, err)

	result := buf.Bytes()
	sbt.txGases = nil

	r := bytes.NewReader(result)
	err = sbt.decodeTxGases(r)
	require.NoError(t, err)

	require.Equal(t, txGases, sbt.txGases)
}

func TestSpanBatchTxsTxTos(t *testing.T) {
	rng := rand.New(rand.NewSource(0x54321))
	chainID := big.NewInt(rng.Int63n(1000))

	rawSpanBatch := RandomRawSpanBatch(rng, chainID)
	txTos := rawSpanBatch.txs.txTos
	contractCreationBits := rawSpanBatch.txs.contractCreationBits
	totalBlockTxCount := rawSpanBatch.txs.totalBlockTxCount

	var sbt spanBatchTxs
	sbt.txTos = txTos
	// creation bits and block tx count must be se to decode tos
	sbt.contractCreationBits = contractCreationBits
	sbt.totalBlockTxCount = totalBlockTxCount

	var buf bytes.Buffer
	err := sbt.encodeTxTos(&buf)
	require.NoError(t, err)

	// to field is fixed length: 20 bytes
	require.Equal(t, buf.Len(), 20*len(txTos))

	result := buf.Bytes()
	sbt.txTos = nil

	r := bytes.NewReader(result)
	err = sbt.decodeTxTos(r)
	require.NoError(t, err)

	require.Equal(t, txTos, sbt.txTos)
}

func TestSpanBatchTxsTxDatas(t *testing.T) {
	rng := rand.New(rand.NewSource(0x1234))
	chainID := big.NewInt(rng.Int63n(1000))

	rawSpanBatch := RandomRawSpanBatch(rng, chainID)
	txDatas := rawSpanBatch.txs.txDatas
	txTypes := rawSpanBatch.txs.txTypes
	totalBlockTxCount := rawSpanBatch.txs.totalBlockTxCount

	var sbt spanBatchTxs
	sbt.totalBlockTxCount = totalBlockTxCount

	sbt.txDatas = txDatas

	var buf bytes.Buffer
	err := sbt.encodeTxDatas(&buf)
	require.NoError(t, err)

	result := buf.Bytes()
	sbt.txDatas = nil
	sbt.txTypes = nil

	r := bytes.NewReader(result)
	err = sbt.decodeTxDatas(r)
	require.NoError(t, err)

	require.Equal(t, txDatas, sbt.txDatas)
	require.Equal(t, txTypes, sbt.txTypes)
}

func TestSpanBatchTxsRecoverV(t *testing.T) {
	rng := rand.New(rand.NewSource(0x123))

	chainID := big.NewInt(rng.Int63n(1000))
	londonSigner := types.NewLondonSigner(chainID)
	totalblockTxCount := 20 + rng.Intn(100)

	cases := []txTypeTest{
		{"unprotected legacy tx", testutils.RandomLegacyTx, types.HomesteadSigner{}},
		{"legacy tx", testutils.RandomLegacyTx, londonSigner},
		{"access list tx", testutils.RandomAccessListTx, londonSigner},
		{"dynamic fee tx", testutils.RandomDynamicFeeTx, londonSigner},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			var spanBatchTxs spanBatchTxs
			var txTypes []int
			var txSigs []spanBatchSignature
			var originalVs []uint64
			yParityBits := new(big.Int)
			protectedBits := new(big.Int)
			totalLegacyTxCount := 0
			for idx := 0; idx < totalblockTxCount; idx++ {
				tx := testCase.mkTx(rng, testCase.signer)
				txType := tx.Type()
				txTypes = append(txTypes, int(txType))
				var txSig spanBatchSignature
				v, r, s := tx.RawSignatureValues()
				if txType == types.LegacyTxType {
					protectedBit := uint(0)
					if tx.Protected() {
						protectedBit = uint(1)
					}
					protectedBits.SetBit(protectedBits, int(totalLegacyTxCount), protectedBit)
					totalLegacyTxCount++
				}
				// Do not fill in txSig.V
				txSig.r, _ = uint256.FromBig(r)
				txSig.s, _ = uint256.FromBig(s)
				txSigs = append(txSigs, txSig)
				originalVs = append(originalVs, v.Uint64())
				yParityBit, err := convertVToYParity(v.Uint64(), int(tx.Type()))
				require.NoError(t, err)
				yParityBits.SetBit(yParityBits, idx, yParityBit)
			}

			spanBatchTxs.yParityBits = yParityBits
			spanBatchTxs.txSigs = txSigs
			spanBatchTxs.txTypes = txTypes
			spanBatchTxs.protectedBits = protectedBits
			// recover txSig.v
			err := spanBatchTxs.recoverV(chainID)
			require.NoError(t, err)

			var recoveredVs []uint64
			for _, txSig := range spanBatchTxs.txSigs {
				recoveredVs = append(recoveredVs, txSig.v)
			}
			require.Equal(t, originalVs, recoveredVs, "recovered v mismatch")
		})
	}
}

func TestSpanBatchTxsRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(0x73311337))
	chainID := big.NewInt(rng.Int63n(1000))

	for i := 0; i < 4; i++ {
		rawSpanBatch := RandomRawSpanBatch(rng, chainID)
		sbt := rawSpanBatch.txs
		totalBlockTxCount := sbt.totalBlockTxCount

		var buf bytes.Buffer
		err := sbt.encode(&buf)
		require.NoError(t, err)

		result := buf.Bytes()
		r := bytes.NewReader(result)

		var sbt2 spanBatchTxs
		sbt2.totalBlockTxCount = totalBlockTxCount
		err = sbt2.decode(r)
		require.NoError(t, err)

		err = sbt2.recoverV(chainID)
		require.NoError(t, err)

		require.Equal(t, sbt, &sbt2)
	}
}

func TestSpanBatchTxsRoundTripFullTxs(t *testing.T) {
	rng := rand.New(rand.NewSource(0x13377331))
	chainID := big.NewInt(rng.Int63n(1000))
	londonSigner := types.NewLondonSigner(chainID)

	cases := []txTypeTest{
		{"unprotected legacy tx", testutils.RandomLegacyTx, types.HomesteadSigner{}},
		{"legacy tx", testutils.RandomLegacyTx, londonSigner},
		{"access list tx", testutils.RandomAccessListTx, londonSigner},
		{"dynamic fee tx", testutils.RandomDynamicFeeTx, londonSigner},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			for i := 0; i < 4; i++ {
				totalblockTxCounts := uint64(1 + rng.Int()&0xFF)
				var txs [][]byte
				for i := 0; i < int(totalblockTxCounts); i++ {
					tx := testCase.mkTx(rng, testCase.signer)
					rawTx, err := tx.MarshalBinary()
					require.NoError(t, err)
					txs = append(txs, rawTx)
				}
				sbt, err := newSpanBatchTxs(txs, chainID)
				require.NoError(t, err)

				txs2, err := sbt.fullTxs(chainID)
				require.NoError(t, err)

				require.Equal(t, txs, txs2)
			}
		})
	}
}

func TestSpanBatchTxsRecoverVInvalidTxType(t *testing.T) {
	rng := rand.New(rand.NewSource(0x321))
	chainID := big.NewInt(rng.Int63n(1000))

	var sbt spanBatchTxs

	sbt.txTypes = []int{types.DepositTxType}
	sbt.txSigs = []spanBatchSignature{{v: 0, r: nil, s: nil}}
	sbt.yParityBits = new(big.Int)
	sbt.protectedBits = new(big.Int)

	err := sbt.recoverV(chainID)
	require.ErrorContains(t, err, "invalid tx type")
}

func TestSpanBatchTxsFullTxNotEnoughTxTos(t *testing.T) {
	rng := rand.New(rand.NewSource(0x13572468))
	chainID := big.NewInt(rng.Int63n(1000))
	londonSigner := types.NewLondonSigner(chainID)

	cases := []txTypeTest{
		{"unprotected legacy tx", testutils.RandomLegacyTx, types.HomesteadSigner{}},
		{"legacy tx", testutils.RandomLegacyTx, londonSigner},
		{"access list tx", testutils.RandomAccessListTx, londonSigner},
		{"dynamic fee tx", testutils.RandomDynamicFeeTx, londonSigner},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			totalblockTxCounts := uint64(1 + rng.Int()&0xFF)
			var txs [][]byte
			for i := 0; i < int(totalblockTxCounts); i++ {
				tx := testCase.mkTx(rng, testCase.signer)
				rawTx, err := tx.MarshalBinary()
				require.NoError(t, err)
				txs = append(txs, rawTx)
			}
			sbt, err := newSpanBatchTxs(txs, chainID)
			require.NoError(t, err)

			// drop single to field
			sbt.txTos = sbt.txTos[:len(sbt.txTos)-2]

			_, err = sbt.fullTxs(chainID)
			require.EqualError(t, err, "tx to not enough")
		})
	}
}

func TestSpanBatchTxsMaxContractCreationBitsLength(t *testing.T) {
	var sbt spanBatchTxs
	sbt.totalBlockTxCount = 0xFFFFFFFFFFFFFFFF

	r := bytes.NewReader([]byte{})
	err := sbt.decodeContractCreationBits(r)
	require.ErrorIs(t, err, ErrTooBigSpanBatchSize)
}

func TestSpanBatchTxsMaxYParityBitsLength(t *testing.T) {
	var sb RawSpanBatch
	sb.blockCount = 0xFFFFFFFFFFFFFFFF

	r := bytes.NewReader([]byte{})
	err := sb.decodeOriginBits(r)
	require.ErrorIs(t, err, ErrTooBigSpanBatchSize)
}

func TestSpanBatchTxsMaxProtectedBitsLength(t *testing.T) {
	var sb RawSpanBatch
	sb.txs = &spanBatchTxs{}
	sb.txs.totalLegacyTxCount = 0xFFFFFFFFFFFFFFFF

	r := bytes.NewReader([]byte{})
	err := sb.txs.decodeProtectedBits(r)
	require.ErrorIs(t, err, ErrTooBigSpanBatchSize)
}
