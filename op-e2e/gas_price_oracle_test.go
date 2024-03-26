package op_e2e

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io/fs"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
)

func inputsToHex(inputs []interface{}) ([]byte, error) {
	resultBytes := []byte{}
	for _, input := range inputs {
		switch v := input.(type) {
		case int32:
			bytes := make([]byte, 4)
			binary.BigEndian.PutUint32(bytes, uint32(v))
			resultBytes = append(resultBytes, bytes...)
		case uint32:
			bytes := make([]byte, 4)
			binary.BigEndian.PutUint32(bytes, v)
			resultBytes = append(resultBytes, bytes...)
		case uint64:
			bytes := make([]byte, 8)
			binary.BigEndian.PutUint64(bytes, v)
			resultBytes = append(resultBytes, bytes...)
		default:
			return nil, fmt.Errorf("unsupported type: %T", v)
		}
	}
	// Print the hex-encoded string of 28 bytes
	return resultBytes, nil
}

func TestGasPriceOracle(t *testing.T) {

	var (
		sequenceNumber uint64 = 0
		blobFeeScalar  uint32 = 1_250_000
		baseFeeScalar  uint32 = 11_111
		costTxSizeCoef int32  = -88_664
		costFastlzCoef int32  = 1_031_462
		costIntercept  int32  = -27_321_890
	)

	inputs := []interface{}{baseFeeScalar, blobFeeScalar, sequenceNumber}
	payload, err := inputsToHex(inputs)
	assert.NoError(t, err)

	byteResult := append(make([]byte, 4), payload...)

	backend := backends.NewSimulatedBackend(map[common.Address]core.GenesisAccount{
		predeploys.GasPriceOracleAddr: {
			Code:    common.FromHex(bindings.GasPriceOracleDeployedBin),
			Balance: big.NewInt(0),
			Storage: map[common.Hash]common.Hash{
				common.HexToHash("0x0"): common.HexToHash("0x0101"), // isEcotone = true, isFjord = true
			},
		},
		predeploys.L1BlockAddr: {
			Code:    common.FromHex(bindings.L1BlockDeployedBin),
			Balance: big.NewInt(0),
			Storage: map[common.Hash]common.Hash{
				common.HexToHash("0x1"): common.HexToHash("0x01"),                         // l1BaseFee 1
				common.HexToHash("0x3"): common.HexToHash(hex.EncodeToString(byteResult)), // all other constants
				common.HexToHash("0x7"): common.HexToHash("0x01"),                         // l1BlobBaseFee 1
			},
		},
	}, math.MaxUint64)

	caller, err := bindings.NewGasPriceOracleCaller(predeploys.GasPriceOracleAddr, backend)
	assert.NoError(t, err)

	atLeastOnce := false
	err = filepath.WalkDir("../specs", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		used, err := caller.GetL1Fee(&bind.CallOpts{}, b)
		if err != nil {
			return err
		}

		l1BaseFee, err := caller.L1BaseFee(&bind.CallOpts{})

		if err != nil {
			return err
		}

		l1BaseFeeScaled := uint64(baseFeeScalar) * l1BaseFee.Uint64() * 16
		l1BlobBaseFee, err := caller.BlobBaseFee(&bind.CallOpts{})
		if err != nil {
			return err
		}

		l1BlobFeeScaled := uint64(blobFeeScalar) * l1BlobBaseFee.Uint64()
		l1FeeScaled := l1BaseFeeScaled + l1BlobFeeScaled
		fastLzLength := types.FlzCompressLen(b)
		txSize := len(b) + 68
		expected := ((int64(costIntercept) + int64(costFastlzCoef)*int64(fastLzLength+68) + int64(costTxSizeCoef)*int64(txSize)) * int64(l1FeeScaled)) / 1e12
		assert.Equal(t, uint64(expected), used.Uint64(), path)

		upperBound, err := caller.GetL1FeeUpperBound(&bind.CallOpts{}, big.NewInt(int64(len(b))))
		if err != nil {
			return err
		}
		flzUpperBound := txSize + txSize/255 + 16
		expectedFeeUpperBound := ((int64(costIntercept) + int64(costFastlzCoef)*int64(flzUpperBound) + int64(costTxSizeCoef)*int64(txSize)) * int64(l1FeeScaled)) / 1e12
		assert.Equal(t, uint64(expectedFeeUpperBound), upperBound.Uint64(), path)

		atLeastOnce = true
		return nil
	})
	assert.NoError(t, err)
	assert.True(t, atLeastOnce)
}
