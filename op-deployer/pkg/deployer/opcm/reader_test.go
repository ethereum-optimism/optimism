package opcm

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-core/devfeatures"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3"
	"github.com/stretchr/testify/require"
)

type readerBackend struct {
	t         *testing.T
	responses map[string][]byte
}

func newReaderBackend(t *testing.T) *readerBackend {
	return &readerBackend{t: t, responses: make(map[string][]byte)}
}

func readerCallKey(contract common.Address, data []byte) string {
	return contract.Hex() + ":" + hex.EncodeToString(data)
}

func (b *readerBackend) set(contract common.Address, method *w3.Func, values ...any) {
	b.t.Helper()
	calldata, err := method.EncodeArgs()
	require.NoError(b.t, err)
	output, err := method.Returns.Pack(values...)
	require.NoError(b.t, err)
	b.responses[readerCallKey(contract, calldata)] = output
}

func (b *readerBackend) unset(contract common.Address, method *w3.Func) {
	b.t.Helper()
	calldata, err := method.EncodeArgs()
	require.NoError(b.t, err)
	delete(b.responses, readerCallKey(contract, calldata))
}

func (b *readerBackend) CallContract(
	_ context.Context,
	call ethereum.CallMsg,
	_ *big.Int,
) ([]byte, error) {
	if call.To == nil {
		return nil, fmt.Errorf("missing call recipient")
	}
	result, ok := b.responses[readerCallKey(*call.To, call.Data)]
	if !ok {
		return nil, fmt.Errorf("unexpected call to %s with calldata %s", *call.To, hex.EncodeToString(call.Data))
	}
	return bytes.Clone(result), nil
}

func TestReadSuperRootEnabled(t *testing.T) {
	opcmAddr := common.Address{0xcc}

	// TODO(#21662): devfeatures.IsDevFeatureEnabled hardcodes SuperRootGamesMigrationFlag to
	// true, so a cleared bitmap still reports super-root. Solidity does the same, so this
	// deliberately mirrors what DeployOPChain.s.sol observes.
	for _, bitmap := range []common.Hash{{}, devfeatures.SuperRootGamesMigrationFlag} {
		b := newReaderBackend(t)
		b.set(opcmAddr, DevFeatureBitmapMethod, bitmap)

		superRoot, err := ReadSuperRootEnabled(context.Background(), b, opcmAddr)
		require.NoError(t, err)
		require.True(t, superRoot)
	}

	t.Run("surfaces a failing read", func(t *testing.T) {
		_, err := ReadSuperRootEnabled(context.Background(), newReaderBackend(t), opcmAddr)
		require.ErrorContains(t, err, "devFeatureBitmap()")
		require.ErrorContains(t, err, opcmAddr.Hex())
	})
}

func TestReadImplementations(t *testing.T) {
	var (
		opcmAddr  = common.Address{0xcc}
		container = common.Address{0xc1}
		utils     = common.Address{0xc2}
		migrator  = common.Address{0xc3}
		validator = common.Address{0xc4}
		oracle    = common.Address{0xc5}
	)

	// Every stub address is distinct and non-zero, so a mis-mapped field fails the comparison
	// and an unmapped one stays zero.
	stub := opcmImplementations{
		SuperchainConfigImpl:             common.Address{0x01},
		L1ERC721BridgeImpl:               common.Address{0x02},
		OptimismPortalImpl:               common.Address{0x03},
		ETHLockboxImpl:                   common.Address{0x04},
		SystemConfigImpl:                 common.Address{0x05},
		OptimismMintableERC20FactoryImpl: common.Address{0x06},
		L1CrossDomainMessengerImpl:       common.Address{0x07},
		L1StandardBridgeImpl:             common.Address{0x08},
		DisputeGameFactoryImpl:           common.Address{0x09},
		AnchorStateRegistryImpl:          common.Address{0x0a},
		DelayedWETHImpl:                  common.Address{0x0b},
		MipsImpl:                         common.Address{0x0c},
		FaultDisputeGameImpl:             common.Address{0x0d},
		PermissionedDisputeGameImpl:      common.Address{0x0e},
		SuperFaultDisputeGameImpl:        common.Address{0x0f},
		SuperPermissionedDisputeGameImpl: common.Address{0x10},
		ZkDisputeGameImpl:                common.Address{0x11},
		StorageSetterImpl:                common.Address{0x12},
		SP1PlonkAdapterImpl:              common.Address{0x13},
	}

	newBackend := func(t *testing.T) *readerBackend {
		b := newReaderBackend(t)
		b.set(opcmAddr, ImplementationsMethod, stub)
		b.set(opcmAddr, contractsContainerMethod, container)
		b.set(opcmAddr, opcmUtilsMethod, utils)
		b.set(opcmAddr, opcmMigratorMethod, migrator)
		b.set(opcmAddr, opcmStandardValidatorMethod, validator)
		b.set(stub.MipsImpl, mipsOracleMethod, oracle)
		return b
	}

	t.Run("maps every implementation address", func(t *testing.T) {
		got, err := ReadImplementations(context.Background(), newBackend(t), opcmAddr)
		require.NoError(t, err)

		require.Equal(t, &addresses.ImplementationsContracts{
			OpcmStandardValidatorImpl:        validator,
			OpcmUtilsImpl:                    utils,
			OpcmMigratorImpl:                 migrator,
			OpcmV2Impl:                       opcmAddr,
			OpcmContainerImpl:                container,
			DelayedWethImpl:                  stub.DelayedWETHImpl,
			OptimismPortalImpl:               stub.OptimismPortalImpl,
			EthLockboxImpl:                   stub.ETHLockboxImpl,
			PreimageOracleImpl:               oracle,
			MipsImpl:                         stub.MipsImpl,
			SystemConfigImpl:                 stub.SystemConfigImpl,
			L1CrossDomainMessengerImpl:       stub.L1CrossDomainMessengerImpl,
			L1Erc721BridgeImpl:               stub.L1ERC721BridgeImpl,
			L1StandardBridgeImpl:             stub.L1StandardBridgeImpl,
			OptimismMintableErc20FactoryImpl: stub.OptimismMintableERC20FactoryImpl,
			DisputeGameFactoryImpl:           stub.DisputeGameFactoryImpl,
			AnchorStateRegistryImpl:          stub.AnchorStateRegistryImpl,
			FaultDisputeGameImpl:             stub.FaultDisputeGameImpl,
			PermissionedDisputeGameImpl:      stub.PermissionedDisputeGameImpl,
			ZkDisputeGameImpl:                stub.ZkDisputeGameImpl,
			StorageSetterImpl:                stub.StorageSetterImpl,
			SP1PlonkAdapterImpl:              stub.SP1PlonkAdapterImpl,
			SuperFaultDisputeGameImpl:        stub.SuperFaultDisputeGameImpl,
			SuperPermissionedDisputeGameImpl: stub.SuperPermissionedDisputeGameImpl,
		}, got)
	})

	t.Run("leaves no field unmapped", func(t *testing.T) {
		got, err := ReadImplementations(context.Background(), newBackend(t), opcmAddr)
		require.NoError(t, err)

		// A field added to ImplementationsContracts without a source here stays zero.
		fields := reflect.ValueOf(*got)
		for i := range fields.NumField() {
			name := fields.Type().Field(i).Name
			addr := fields.Field(i).Interface().(common.Address)
			require.NotEqual(t, common.Address{}, addr, "%s has no source in ReadImplementations", name)
			// SuperchainConfigImpl belongs to SuperchainContracts and must not leak in.
			require.NotEqual(t, stub.SuperchainConfigImpl, addr, "%s was mapped from superchainConfigImpl", name)
		}
	})

	t.Run("rejects a zero MIPS implementation", func(t *testing.T) {
		zeroMips := stub
		zeroMips.MipsImpl = common.Address{}
		b := newReaderBackend(t)
		b.set(opcmAddr, ImplementationsMethod, zeroMips)
		b.set(opcmAddr, contractsContainerMethod, container)
		b.set(opcmAddr, opcmUtilsMethod, utils)
		b.set(opcmAddr, opcmMigratorMethod, migrator)
		b.set(opcmAddr, opcmStandardValidatorMethod, validator)

		_, err := ReadImplementations(context.Background(), b, opcmAddr)
		require.ErrorContains(t, err, "zero MIPS implementation")
	})

	t.Run("names the failing getter", func(t *testing.T) {
		b := newBackend(t)
		b.unset(opcmAddr, opcmMigratorMethod)

		_, err := ReadImplementations(context.Background(), b, opcmAddr)
		require.ErrorContains(t, err, "opcmMigrator()")
		require.ErrorContains(t, err, opcmAddr.Hex())
	})

	t.Run("names the failing MIPS getter", func(t *testing.T) {
		b := newBackend(t)
		b.unset(stub.MipsImpl, mipsOracleMethod)

		_, err := ReadImplementations(context.Background(), b, opcmAddr)
		require.ErrorContains(t, err, "oracle()")
		require.ErrorContains(t, err, stub.MipsImpl.Hex())
	})
}
