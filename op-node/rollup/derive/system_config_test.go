package derive

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

var (
	// ABI encoding helpers
	dynBytes, _ = abi.NewType("bytes", "", nil)
	address, _  = abi.NewType("address", "", nil)
	uint16T, _  = abi.NewType("uint16", "", nil)
	uint256T, _ = abi.NewType("uint256", "", nil)
	addressArgs = abi.Arguments{
		{Type: address},
	}
	bytesArgs = abi.Arguments{
		{Type: dynBytes},
	}
	twoUint256 = abi.Arguments{
		{Type: uint256T},
		{Type: uint256T},
	}
	oneUint16 = abi.Arguments{
		{Type: uint16T},
	}
	oneUint256 = abi.Arguments{
		{Type: uint256T},
	}
	eip1559Params     = []byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8}
	operatorFeeParams = []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x5, 0x0, 0x0, 0x0, 0x0, 0x0, 0x7, 0xd, 0x8}
	minBaseFee        = uint64(1e9)
)

// TestProcessSystemConfigUpdateLogEvent tests the parsing of an event and mutating the
// SystemConfig. The hook will build the ABI encoded data dynamically. All tests create
// a new SystemConfig and apply a log against it and then assert that the mutated system
// config is equal to the defined system config in the test.
func TestProcessSystemConfigUpdateLogEvent(t *testing.T) {
	tests := []struct {
		name   string
		log    *types.Log
		config eth.SystemConfig
		hook   func(*testing.T, *types.Log) *types.Log
		err    bool
		// forks (optional)
		ecotoneTime *uint64
		l1Time      uint64
	}{
		{
			// The log data is ignored by consensus and no modifications to the
			// system config occur.
			name: "UnsafeBlockSigner",
			log: &types.Log{
				Topics: []common.Hash{
					ConfigUpdateEventABIHash,
					ConfigUpdateEventVersion0,
					SystemConfigUpdateUnsafeBlockSigner,
				},
			},
			hook: func(t *testing.T, log *types.Log) *types.Log {
				addr := common.Address{}
				data, err := addressArgs.Pack(&addr)
				require.NoError(t, err)
				log.Data = data
				return log
			},
			config: eth.SystemConfig{},
			err:    false,
		},
		{
			// The batcher address should be updated.
			name: "Batcher",
			log: &types.Log{
				Topics: []common.Hash{
					ConfigUpdateEventABIHash,
					ConfigUpdateEventVersion0,
					SystemConfigUpdateBatcher,
				},
			},
			hook: func(t *testing.T, log *types.Log) *types.Log {
				addr := common.Address{19: 0xaa}
				addrData, err := addressArgs.Pack(&addr)
				require.NoError(t, err)
				data, err := bytesArgs.Pack(addrData)
				require.NoError(t, err)
				log.Data = data
				return log
			},
			config: eth.SystemConfig{
				BatcherAddr: common.Address{19: 0xaa},
			},
			err: false,
		},
		{
			// The overhead and the scalar should be updated.
			name: "GasConfig",
			log: &types.Log{
				Topics: []common.Hash{
					ConfigUpdateEventABIHash,
					ConfigUpdateEventVersion0,
					SystemConfigUpdateFeeScalars,
				},
			},
			hook: func(t *testing.T, log *types.Log) *types.Log {
				overhead := big.NewInt(0xff)
				scalar := big.NewInt(0xaa)
				numberData, err := twoUint256.Pack(overhead, scalar)
				require.NoError(t, err)
				data, err := bytesArgs.Pack(numberData)
				require.NoError(t, err)
				log.Data = data
				return log
			},
			config: eth.SystemConfig{
				Overhead: eth.Bytes32{31: 0xff},
				Scalar:   eth.Bytes32{31: 0xaa},
			},
			err: false,
		},
		{
			// The gas limit should be updated.
			name: "GasLimit",
			log: &types.Log{
				Topics: []common.Hash{
					ConfigUpdateEventABIHash,
					ConfigUpdateEventVersion0,
					SystemConfigUpdateGasLimit,
				},
			},
			hook: func(t *testing.T, log *types.Log) *types.Log {
				gasLimit := big.NewInt(0xbb)
				numberData, err := oneUint256.Pack(gasLimit)
				require.NoError(t, err)
				data, err := bytesArgs.Pack(numberData)
				require.NoError(t, err)
				log.Data = data
				return log
			},
			config: eth.SystemConfig{
				GasLimit: 0xbb,
			},
			err: false,
		},
		{
			// The ecotone scalars should be updated
			name: "GasConfigEcotone",
			log: &types.Log{
				Topics: []common.Hash{
					ConfigUpdateEventABIHash,
					ConfigUpdateEventVersion0,
					SystemConfigUpdateFeeScalars,
				},
			},
			hook: func(t *testing.T, log *types.Log) *types.Log {
				scalarData := common.Hash{0: 1, 24 + 3: 0xb3, 28 + 3: 0xbb}
				scalar := scalarData.Big()
				overhead := big.NewInt(0xff)
				numberData, err := twoUint256.Pack(overhead, scalar)
				require.NoError(t, err)
				data, err := bytesArgs.Pack(numberData)
				require.NoError(t, err)
				log.Data = data
				return log
			},
			config: eth.SystemConfig{
				Scalar: eth.Bytes32{0: 1, 24 + 3: 0xb3, 28 + 3: 0xbb},
			},
			err:         false,
			ecotoneTime: new(uint64), // activate ecotone
			l1Time:      200,
		},
		{
			name: "OneTopic",
			log: &types.Log{
				Topics: []common.Hash{
					ConfigUpdateEventABIHash,
				},
			},
			hook: func(t *testing.T, log *types.Log) *types.Log {
				return log
			},
			config: eth.SystemConfig{},
			err:    true,
		},
		{
			name: "EIP1559Params",
			log: &types.Log{
				Topics: []common.Hash{
					ConfigUpdateEventABIHash,
					ConfigUpdateEventVersion0,
					SystemConfigUpdateEIP1559Params,
				},
			},
			hook: func(t *testing.T, log *types.Log) *types.Log {
				numberData, err := oneUint256.Pack(new(big.Int).SetBytes(eip1559Params))
				require.NoError(t, err)
				data, err := bytesArgs.Pack(numberData)
				require.NoError(t, err)
				log.Data = data
				return log
			},
			config: eth.SystemConfig{
				EIP1559Params: eth.Bytes8(eip1559Params),
			},
			err: false,
		},
		{
			name: "EIP1559Params_ZeroDenominatorNonZeroElasticity",
			log: &types.Log{
				Topics: []common.Hash{
					ConfigUpdateEventABIHash,
					ConfigUpdateEventVersion0,
					SystemConfigUpdateEIP1559Params,
				},
			},
			hook: func(t *testing.T, log *types.Log) *types.Log {
				// denominator = 0, elasticity = 1 (invalid combination)
				params := []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1}
				numberData, err := oneUint256.Pack(new(big.Int).SetBytes(params))
				require.NoError(t, err)
				data, err := bytesArgs.Pack(numberData)
				require.NoError(t, err)
				log.Data = data
				return log
			},
			config: eth.SystemConfig{},
			err:    true,
		},
		{
			name: "EIP1559Params_NonZeroDenominatorZeroElasticity",
			log: &types.Log{
				Topics: []common.Hash{
					ConfigUpdateEventABIHash,
					ConfigUpdateEventVersion0,
					SystemConfigUpdateEIP1559Params,
				},
			},
			hook: func(t *testing.T, log *types.Log) *types.Log {
				// denominator = 1, elasticity = 0 (invalid combination)
				params := []byte{0x0, 0x0, 0x0, 0x1, 0x0, 0x0, 0x0, 0x0}
				numberData, err := oneUint256.Pack(new(big.Int).SetBytes(params))
				require.NoError(t, err)
				data, err := bytesArgs.Pack(numberData)
				require.NoError(t, err)
				log.Data = data
				return log
			},
			config: eth.SystemConfig{},
			err:    true,
		},
		{
			name: "EIP1559Params_BothZero",
			log: &types.Log{
				Topics: []common.Hash{
					ConfigUpdateEventABIHash,
					ConfigUpdateEventVersion0,
					SystemConfigUpdateEIP1559Params,
				},
			},
			hook: func(t *testing.T, log *types.Log) *types.Log {
				// denominator = 0, elasticity = 0 (valid - uses pre-Holocene constants)
				params := []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}
				numberData, err := oneUint256.Pack(new(big.Int).SetBytes(params))
				require.NoError(t, err)
				data, err := bytesArgs.Pack(numberData)
				require.NoError(t, err)
				log.Data = data
				return log
			},
			config: eth.SystemConfig{
				EIP1559Params: eth.Bytes8{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
			},
			err: false,
		},
		{
			name: "OperatorFeeParams",
			log: &types.Log{
				Topics: []common.Hash{
					ConfigUpdateEventABIHash,
					ConfigUpdateEventVersion0,
					SystemConfigUpdateOperatorFeeParams,
				},
			},
			hook: func(t *testing.T, log *types.Log) *types.Log {
				numberData, err := oneUint256.Pack(new(big.Int).SetBytes(operatorFeeParams))
				require.NoError(t, err)
				data, err := bytesArgs.Pack(numberData)
				require.NoError(t, err)
				log.Data = data
				return log
			},
			config: eth.SystemConfig{
				OperatorFeeParams: eth.Bytes32(operatorFeeParams),
			},
			err: false,
		},
		{
			name: "UpdateMinBaseFee",
			log: &types.Log{
				Topics: []common.Hash{
					ConfigUpdateEventABIHash,
					ConfigUpdateEventVersion0,
					SystemConfigUpdateMinBaseFee,
				},
			},
			hook: func(t *testing.T, log *types.Log) *types.Log {
				numberData, err := oneUint256.Pack(new(big.Int).SetUint64(minBaseFee))
				require.NoError(t, err)
				data, err := bytesArgs.Pack(numberData)
				require.NoError(t, err)
				log.Data = data
				return log
			},
			config: eth.SystemConfig{
				MinBaseFee: minBaseFee,
			},
			err: false,
		},
		{
			name: "DAFootprintGasScalar",
			log: &types.Log{
				Topics: []common.Hash{
					ConfigUpdateEventABIHash,
					ConfigUpdateEventVersion0,
					SystemConfigUpdateDAFootprintGasScalar,
				},
			},
			hook: func(t *testing.T, log *types.Log) *types.Log {
				numberData, err := oneUint16.Pack(uint16(100))
				require.NoError(t, err)
				data, err := bytesArgs.Pack(numberData)
				require.NoError(t, err)
				log.Data = data
				return log
			},
			config: eth.SystemConfig{
				DAFootprintGasScalar: 100,
			},
			err: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := eth.SystemConfig{}
			rollupCfg := rollup.Config{EcotoneTime: test.ecotoneTime}

			err := ProcessSystemConfigUpdateLogEvent(&config, test.hook(t, test.log), &rollupCfg, test.l1Time)
			if test.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, config, test.config)
		})
	}
}

func TestUpdateSystemConfigWithL1Receipts_Atomicity(t *testing.T) {
	t.Run("applies all updates when all receipts well-formed", func(t *testing.T) {
		sysCfg := eth.SystemConfig{}
		l1Addr := common.Address{19: 0x42}
		cfg := rollup.Config{
			L1SystemConfigAddress: l1Addr,
		}
		// Build a well-formed Batcher update
		newBatcher := common.Address{19: 0xaa}
		addrData, err := addressArgs.Pack(&newBatcher)
		require.NoError(t, err)
		batcherData, err := bytesArgs.Pack(addrData)
		require.NoError(t, err)
		batcherLog := &types.Log{
			Address: l1Addr,
			Topics: []common.Hash{
				ConfigUpdateEventABIHash,
				ConfigUpdateEventVersion0,
				SystemConfigUpdateBatcher,
			},
			Data: batcherData,
		}
		// Build a well-formed GasLimit update
		gasLimit := big.NewInt(0xbb)
		gasDataEnc, err := oneUint256.Pack(gasLimit)
		require.NoError(t, err)
		gasData, err := bytesArgs.Pack(gasDataEnc)
		require.NoError(t, err)
		gasLog := &types.Log{
			Address: l1Addr,
			Topics: []common.Hash{
				ConfigUpdateEventABIHash,
				ConfigUpdateEventVersion0,
				SystemConfigUpdateGasLimit,
			},
			Data: gasData,
		}
		receipts := []*types.Receipt{
			{
				Status: types.ReceiptStatusSuccessful,
				Logs:   []*types.Log{batcherLog},
			},
			{
				Status: types.ReceiptStatusSuccessful,
				Logs:   []*types.Log{gasLog},
			},
		}
		err = UpdateSystemConfigWithL1Receipts(&sysCfg, receipts, &cfg, 0)
		require.NoError(t, err)
		require.Equal(t, newBatcher, sysCfg.BatcherAddr)
		require.Equal(t, uint64(0xbb), sysCfg.GasLimit)
	})

	t.Run("all valid updates apply, any invalid updates are not applied and return errors", func(t *testing.T) {
		// Start with a non-zero initial config so we can detect accidental partial updates
		initial := eth.SystemConfig{
			BatcherAddr: common.Address{19: 0x11},
			GasLimit:    0x1234,
		}
		sysCfg := initial
		l1Addr := common.Address{19: 0x43}
		cfg := rollup.Config{
			L1SystemConfigAddress: l1Addr,
		}
		// Well-formed Batcher update (would change value if applied)
		newBatcher := common.Address{19: 0xaa}
		addrData, err := addressArgs.Pack(&newBatcher)
		require.NoError(t, err)
		batcherData, err := bytesArgs.Pack(addrData)
		require.NoError(t, err)
		batcherLog := &types.Log{
			Address: l1Addr,
			Topics: []common.Hash{
				ConfigUpdateEventABIHash,
				ConfigUpdateEventVersion0,
				SystemConfigUpdateBatcher,
			},
			Data: batcherData,
		}
		// Malformed GasLimit update (invalid data to trigger parse failure)
		malformedGasLog := &types.Log{
			Address: l1Addr,
			Topics: []common.Hash{
				ConfigUpdateEventABIHash,
				ConfigUpdateEventVersion0,
				SystemConfigUpdateGasLimit,
			},
			Data: []byte{0x00}, // insufficient bytes for pointer/length -> parse error
		}
		// Future / unknown event type
		futureLogType := &types.Log{
			Address: l1Addr,
			Topics: []common.Hash{
				ConfigUpdateEventABIHash,
				ConfigUpdateEventVersion0,
				common.Hash{0: 'a', 31: 7}, // test assumes this is not a known event type
			},
		}
		// Future / unknown event version
		futureLogVersion := &types.Log{
			Address: l1Addr,
			Topics: []common.Hash{
				ConfigUpdateEventABIHash,
				common.Hash{31: 1}, // test assumes this is not a known event version
				SystemConfigUpdateBatcher,
			},
			Data: batcherData,
		}
		receipts := []*types.Receipt{
			{
				Status: types.ReceiptStatusSuccessful,
				Logs:   []*types.Log{batcherLog},
			},
			{
				Status: types.ReceiptStatusSuccessful,
				Logs:   []*types.Log{malformedGasLog},
			},
			{
				Status: types.ReceiptStatusSuccessful,
				Logs:   []*types.Log{futureLogType},
			},
			{
				Status: types.ReceiptStatusSuccessful,
				Logs:   []*types.Log{futureLogVersion},
			},
		}
		err = UpdateSystemConfigWithL1Receipts(&sysCfg, receipts, &cfg, 0)
		// Error should be returned due to malformed update, but valid updates should apply
		require.Error(t, err)
		// Confirm valid update applied
		require.Equal(t, newBatcher, sysCfg.BatcherAddr)
		// Confirm invalid update did not apply; GasLimit remains unchanged
		require.Equal(t, initial.GasLimit, sysCfg.GasLimit)
		// Confirm error contains expected messages
		require.ErrorContains(t, err, "invalid pointer field")
		require.ErrorIs(t, err, ErrParsingSystemConfig)
		require.ErrorIs(t, err, ErrUnknownEventType)
		require.ErrorIs(t, err, ErrUnknownEventVersion)
	})

	t.Run("applies multiple updates within a single receipt", func(t *testing.T) {
		sysCfg := eth.SystemConfig{}
		l1Addr := common.Address{19: 0x44}
		cfg := rollup.Config{
			L1SystemConfigAddress: l1Addr,
		}
		// Build well-formed Batcher update
		newBatcher := common.Address{19: 0xbb}
		addrData, err := addressArgs.Pack(&newBatcher)
		require.NoError(t, err)
		batcherData, err := bytesArgs.Pack(addrData)
		require.NoError(t, err)
		batcherLog := &types.Log{
			Address: l1Addr,
			Topics: []common.Hash{
				ConfigUpdateEventABIHash,
				ConfigUpdateEventVersion0,
				SystemConfigUpdateBatcher,
			},
			Data: batcherData,
		}
		// Build well-formed GasLimit update
		gasLimit := big.NewInt(0xcc)
		gasDataEnc, err := oneUint256.Pack(gasLimit)
		require.NoError(t, err)
		gasData, err := bytesArgs.Pack(gasDataEnc)
		require.NoError(t, err)
		gasLog := &types.Log{
			Address: l1Addr,
			Topics: []common.Hash{
				ConfigUpdateEventABIHash,
				ConfigUpdateEventVersion0,
				SystemConfigUpdateGasLimit,
			},
			Data: gasData,
		}
		receipts := []*types.Receipt{
			{
				Status: types.ReceiptStatusSuccessful,
				Logs:   []*types.Log{batcherLog, gasLog},
			},
		}
		err = UpdateSystemConfigWithL1Receipts(&sysCfg, receipts, &cfg, 0)
		require.NoError(t, err)
		require.Equal(t, newBatcher, sysCfg.BatcherAddr)
		require.Equal(t, uint64(0xcc), sysCfg.GasLimit)
	})

	t.Run("applies updates across multiple receipts within the same block", func(t *testing.T) {
		sysCfg := eth.SystemConfig{}
		l1Addr := common.Address{19: 0x45}
		cfg := rollup.Config{
			L1SystemConfigAddress: l1Addr,
		}
		blockHash := common.Hash{0: 0xaa}
		blockNumber := big.NewInt(12345)
		// Build well-formed Batcher update (tx 0)
		newBatcher := common.Address{19: 0xcc}
		addrData, err := addressArgs.Pack(&newBatcher)
		require.NoError(t, err)
		batcherData, err := bytesArgs.Pack(addrData)
		require.NoError(t, err)
		batcherLog := &types.Log{
			Address: l1Addr,
			Topics: []common.Hash{
				ConfigUpdateEventABIHash,
				ConfigUpdateEventVersion0,
				SystemConfigUpdateBatcher,
			},
			Data: batcherData,
		}
		// Build well-formed GasLimit update (tx 1)
		gasLimit := big.NewInt(0xdd)
		gasDataEnc, err := oneUint256.Pack(gasLimit)
		require.NoError(t, err)
		gasData, err := bytesArgs.Pack(gasDataEnc)
		require.NoError(t, err)
		gasLog := &types.Log{
			Address: l1Addr,
			Topics: []common.Hash{
				ConfigUpdateEventABIHash,
				ConfigUpdateEventVersion0,
				SystemConfigUpdateGasLimit,
			},
			Data: gasData,
		}
		// Build well-formed MinBaseFee update (tx 2)
		minBaseFeeEnc, err := oneUint256.Pack(new(big.Int).SetUint64(minBaseFee))
		require.NoError(t, err)
		minBaseFeeData, err := bytesArgs.Pack(minBaseFeeEnc)
		require.NoError(t, err)
		minBaseFeeLog := &types.Log{
			Address: l1Addr,
			Topics: []common.Hash{
				ConfigUpdateEventABIHash,
				ConfigUpdateEventVersion0,
				SystemConfigUpdateMinBaseFee,
			},
			Data: minBaseFeeData,
		}
		receipts := []*types.Receipt{
			{
				Status:           types.ReceiptStatusSuccessful,
				BlockHash:        blockHash,
				BlockNumber:      blockNumber,
				TransactionIndex: 0,
				Logs:             []*types.Log{batcherLog},
			},
			{
				Status:           types.ReceiptStatusSuccessful,
				BlockHash:        blockHash,
				BlockNumber:      blockNumber,
				TransactionIndex: 1,
				Logs:             []*types.Log{gasLog},
			},
			{
				Status:           types.ReceiptStatusSuccessful,
				BlockHash:        blockHash,
				BlockNumber:      blockNumber,
				TransactionIndex: 2,
				Logs:             []*types.Log{minBaseFeeLog},
			},
		}
		err = UpdateSystemConfigWithL1Receipts(&sysCfg, receipts, &cfg, 0)
		require.NoError(t, err)
		require.Equal(t, newBatcher, sysCfg.BatcherAddr)
		require.Equal(t, uint64(0xdd), sysCfg.GasLimit)
		require.Equal(t, minBaseFee, sysCfg.MinBaseFee)
	})
}
