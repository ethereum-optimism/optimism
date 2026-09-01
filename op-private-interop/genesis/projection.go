// Package genesis defines the deterministic genesis projection for private interop.
package genesis

import (
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum-optimism/optimism/op-core/devfeatures"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	gethparams "github.com/ethereum/go-ethereum/params"
)

var (
	errNilGenesis         = errors.New("private-chain genesis is nil")
	errMissingChainConfig = errors.New("private-chain genesis has no chain config")

	implementationSlot   = common.HexToHash("0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc")
	adminSlot            = common.HexToHash("0xb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d6103")
	customGasTokenSlot   = common.HexToHash("0x4ad9936a67aeb1898ef7b848aecdf71a1f8999fbf63ff2f5b5691cb14bedfe4d")
	devFeatureBitmapSlot = common.HexToHash("0xc8bc8f9195cfb2d040744aac63412d02ffc186ea9bd519039edc4666ee9032bc")
	proxyAdminWord       = common.BytesToHash(predeploys.ProxyAdminAddr.Bytes())
	maxUint128           = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(1))
)

//go:embed bytecode/*.hex
var bytecodes embed.FS

var publicProjectionCode = map[common.Address][]byte{
	codeNamespace(predeploys.L1BlockAddr):                    mustBytecode("L1Block"),
	codeNamespace(predeploys.L2ToL1MessagePasserAddr):        mustBytecode("L2ToL1MessagePasser"),
	codeNamespace(predeploys.L2toL2CrossDomainMessengerAddr): mustBytecode("L2ToL2CrossDomainMessengerReplay"),
	codeNamespace(predeploys.SuperchainETHBridgeAddr):        mustBytecode("SuperchainETHBridge"),
	codeNamespace(predeploys.ETHLiquidityAddr):               mustBytecode("ETHLiquidity"),
	codeNamespace(predeploys.ClaimRegistryAddr):              mustBytecode("ClaimRegistry"),
	codeNamespace(predeploys.EventReplayerAddr):              mustBytecode("EventReplayer"),
}

// ProjectGenesisFrom constructs the public-projection genesis from a private-chain genesis.
//
// The operation is pure: it does not mutate privateChainGenesis and performs no I/O. The embedded
// bytecode is part of the projection protocol, just like an execution client's hardfork code.
func ProjectGenesisFrom(privateChainGenesis *core.Genesis) (*core.Genesis, error) {
	if privateChainGenesis == nil {
		return nil, errNilGenesis
	}
	if privateChainGenesis.Config == nil {
		return nil, errMissingChainConfig
	}
	if err := validatePrivateChainGenesis(privateChainGenesis); err != nil {
		return nil, err
	}

	out := cloneGenesis(privateChainGenesis)
	out.GasLimit = gethparams.MaxGasLimit
	out.BaseFee = new(big.Int)

	// The public projection executes ordinary ETH semantics, not the private chain's custom gas
	// token semantics.
	deleteStorage(out.Alloc, predeploys.L1BlockAddr, customGasTokenSlot)
	clearStorageFlag(out.Alloc, predeploys.L2DevFeatureFlagsAddr, devFeatureBitmapSlot, devfeatures.PrivateInteropFlag)
	setImplementation(out.Alloc, predeploys.L1BlockAddr, publicProjectionCode[codeNamespace(predeploys.L1BlockAddr)])
	setImplementation(out.Alloc, predeploys.L2ToL1MessagePasserAddr, publicProjectionCode[codeNamespace(predeploys.L2ToL1MessagePasserAddr)])

	// The public projection carries the stock interop ETH path. Its liquidity is deliberately the
	// protocol's uint128 maximum, matching an ordinary non-CGT interop genesis.
	activateProxy(out.Alloc, predeploys.SuperchainETHBridgeAddr, publicProjectionCode[codeNamespace(predeploys.SuperchainETHBridgeAddr)])
	activateProxy(out.Alloc, predeploys.ETHLiquidityAddr, publicProjectionCode[codeNamespace(predeploys.ETHLiquidityAddr)])
	setBalance(out.Alloc, predeploys.ETHLiquidityAddr, maxUint128)

	// The custom-gas-token machinery and application mint bridge belong only to the private chain.
	deactivateProxy(out.Alloc, predeploys.NativeAssetLiquidityAddr)
	deactivateProxy(out.Alloc, predeploys.LiquidityControllerAddr)
	deactivateProxy(out.Alloc, predeploys.NativeMintBridgeAddr)

	// Public batches replay selected private-chain events and record the claim for every range.
	activateProxy(out.Alloc, predeploys.L2toL2CrossDomainMessengerAddr, publicProjectionCode[codeNamespace(predeploys.L2toL2CrossDomainMessengerAddr)])
	activateProxy(out.Alloc, predeploys.ClaimRegistryAddr, publicProjectionCode[codeNamespace(predeploys.ClaimRegistryAddr)])
	activateProxy(out.Alloc, predeploys.EventReplayerAddr, publicProjectionCode[codeNamespace(predeploys.EventReplayerAddr)])

	return out, nil
}

// ProjectRollupConfigFrom constructs the public-projection rollup config. The projected genesis
// supplies the only value that cannot be copied from the private chain: the L2 genesis hash.
func ProjectRollupConfigFrom(privateChainConfig *rollup.Config, publicProjectionGenesis *core.Genesis) (*rollup.Config, error) {
	if privateChainConfig == nil {
		return nil, errors.New("private-chain rollup config is nil")
	}
	if publicProjectionGenesis == nil {
		return nil, errors.New("public-projection genesis is nil")
	}

	out := *privateChainConfig
	out.Genesis = privateChainConfig.Genesis
	out.Genesis.SystemConfig = privateChainConfig.Genesis.SystemConfig
	out.Genesis.L2.Hash = publicProjectionGenesis.ToBlock().Hash()
	out.Genesis.SystemConfig.GasLimit = gethparams.MaxGasLimit
	out.Genesis.SystemConfig.Scalar = eth.EncodeScalar(eth.EcotoneScalars{})
	out.Genesis.SystemConfig.OperatorFeeParams = eth.EncodeOperatorFeeParams(eth.OperatorFeeParams{})
	out.Genesis.SystemConfig.MinBaseFee = 0
	if privateChainConfig.PrivateInterop != nil {
		meta := *privateChainConfig.PrivateInterop
		meta.ExtraEmitters = append([]common.Address(nil), privateChainConfig.PrivateInterop.ExtraEmitters...)
		out.PrivateInterop = &meta
	}
	return &out, nil
}

func validatePrivateChainGenesis(g *core.Genesis) error {
	bridge, ok := g.Alloc[predeploys.NativeMintBridgeAddr]
	if !ok || bridge.Storage[implementationSlot] == (common.Hash{}) {
		return errors.New("genesis is not a private chain: NativeMintBridge is inactive")
	}
	l1Block, ok := g.Alloc[predeploys.L1BlockAddr]
	if !ok || l1Block.Storage[customGasTokenSlot] == (common.Hash{}) {
		return errors.New("genesis is not a private chain: custom gas token is disabled")
	}
	featureFlags, ok := g.Alloc[predeploys.L2DevFeatureFlagsAddr]
	if !ok || !devfeatures.IsDevFeatureEnabled(featureFlags.Storage[devFeatureBitmapSlot], devfeatures.PrivateInteropFlag) {
		return errors.New("genesis is not a private chain: private interop feature is disabled")
	}
	return nil
}

func cloneGenesis(in *core.Genesis) *core.Genesis {
	out := *in
	config := *in.Config
	out.Config = &config
	out.ExtraData = append([]byte(nil), in.ExtraData...)
	if in.Difficulty != nil {
		out.Difficulty = new(big.Int).Set(in.Difficulty)
	}
	if in.BaseFee != nil {
		out.BaseFee = new(big.Int).Set(in.BaseFee)
	}
	out.Alloc = make(types.GenesisAlloc, len(in.Alloc))
	for addr, account := range in.Alloc {
		out.Alloc[addr] = cloneAccount(account)
	}
	return &out
}

func cloneAccount(in types.Account) types.Account {
	out := in
	out.Code = append([]byte(nil), in.Code...)
	if in.Balance != nil {
		out.Balance = new(big.Int).Set(in.Balance)
	}
	if in.Storage != nil {
		out.Storage = make(map[common.Hash]common.Hash, len(in.Storage))
		for key, value := range in.Storage {
			out.Storage[key] = value
		}
	}
	return out
}

func activateProxy(alloc types.GenesisAlloc, proxy common.Address, code []byte) {
	proxyAccount := accountAt(alloc, proxy)
	proxyAccount.Storage[implementationSlot] = common.BytesToHash(codeNamespace(proxy).Bytes())
	alloc[proxy] = proxyAccount
	setImplementation(alloc, proxy, code)
}

func setImplementation(alloc types.GenesisAlloc, proxy common.Address, code []byte) {
	alloc[codeNamespace(proxy)] = types.Account{
		Code:    append([]byte(nil), code...),
		Balance: new(big.Int),
		Storage: map[common.Hash]common.Hash{adminSlot: proxyAdminWord},
	}
}

func deactivateProxy(alloc types.GenesisAlloc, proxy common.Address) {
	account := accountAt(alloc, proxy)
	account.Balance = new(big.Int)
	account.Storage = map[common.Hash]common.Hash{adminSlot: proxyAdminWord}
	alloc[proxy] = account
	delete(alloc, codeNamespace(proxy))
}

func deleteStorage(alloc types.GenesisAlloc, addr common.Address, slot common.Hash) {
	account := accountAt(alloc, addr)
	delete(account.Storage, slot)
	alloc[addr] = account
}

func clearStorageFlag(alloc types.GenesisAlloc, addr common.Address, slot common.Hash, flag common.Hash) {
	account := accountAt(alloc, addr)
	value := account.Storage[slot]
	for i := range value {
		value[i] &^= flag[i]
	}
	account.Storage[slot] = value
	alloc[addr] = account
}

func setBalance(alloc types.GenesisAlloc, addr common.Address, balance *big.Int) {
	account := accountAt(alloc, addr)
	account.Balance = new(big.Int).Set(balance)
	alloc[addr] = account
}

func accountAt(alloc types.GenesisAlloc, addr common.Address) types.Account {
	account := alloc[addr]
	if account.Balance == nil {
		account.Balance = new(big.Int)
	}
	if account.Storage == nil {
		account.Storage = make(map[common.Hash]common.Hash)
	}
	return account
}

func codeNamespace(proxy common.Address) common.Address {
	out := common.HexToAddress("0xc0D3C0d3C0d3c0d3c0d3c0D3C0D3C0d3c0d30000")
	copy(out[18:], proxy[18:])
	return out
}

func mustBytecode(name string) []byte {
	data, err := bytecodes.ReadFile("bytecode/" + name + ".hex")
	if err != nil {
		panic(fmt.Errorf("reading embedded %s bytecode: %w", name, err))
	}
	code, err := hex.DecodeString(strings.TrimPrefix(strings.TrimSpace(string(data)), "0x"))
	if err != nil {
		panic(fmt.Errorf("decoding embedded %s bytecode: %w", name, err))
	}
	return code
}
