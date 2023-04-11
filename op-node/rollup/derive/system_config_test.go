package derive

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

var (
	// ABI encoding helpers
	dynBytes, _ = abi.NewType("bytes", "", nil)
	address, _  = abi.NewType("address", "", nil)
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
	oneUint256 = abi.Arguments{
		{Type: uint256T},
	}
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
	}{
		{
			// The log data is ignored by consensus and no modifications to the
			// system config occur.
			name: "SystemConfigUpdateUnsafeBlockSigner",
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
			name: "SystemConfigUpdateBatcher",
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
			name: "SystemConfigUpdateGasConfig",
			log: &types.Log{
				Topics: []common.Hash{
					ConfigUpdateEventABIHash,
					ConfigUpdateEventVersion0,
					SystemConfigUpdateGasConfig,
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
			name: "SystemConfigUpdateGasLimit",
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
			name: "SystemConfigOneTopic",
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
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			config := eth.SystemConfig{}

			err := ProcessSystemConfigUpdateLogEvent(&config, test.hook(t, test.log))
			if test.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, config, test.config)
		})
	}
}
