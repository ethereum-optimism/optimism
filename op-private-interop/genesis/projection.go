// Package genesis defines the deterministic genesis projection for private interop.
//
// The source of a projection is a STOCK op-deployer genesis: a custom-gas-token chain with interop
// active at genesis. No private marker, dev-feature bit, or bespoke predeploy identifies the source;
// the validator recognises it by the state that shape leaves behind, and rejects everything else.
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
	"github.com/ethereum/go-ethereum/crypto"
	gethparams "github.com/ethereum/go-ethereum/params"
)

var (
	errNilGenesis         = errors.New("private-chain genesis is nil")
	errMissingChainConfig = errors.New("private-chain genesis has no chain config")

	// ErrNotCustomGasToken rejects an ordinary ETH genesis.
	ErrNotCustomGasToken = errors.New("genesis is not a private chain: custom gas token is disabled")
	// ErrInteropInactive rejects a genesis whose interop feature set is not active at genesis. The
	// projection is only defined over a Lagoon-at-genesis source: an activation block on the
	// projection would run the stock network-upgrade bundle and replace the replay messenger.
	ErrInteropInactive = errors.New("genesis is not a private chain: interop is not active at genesis")
	// ErrMessengerNotStock rejects a source whose L2ToL2CrossDomainMessenger implementation is not
	// the stock one the projection was built against: a genesis from a different contract release,
	// or a genesis that has already been projected.
	ErrMessengerNotStock = errors.New("genesis is not a private chain: L2ToL2CrossDomainMessenger implementation is not the stock release")
	// ErrAlreadyProjected rejects a public projection offered as a source.
	ErrAlreadyProjected = errors.New("genesis is already a public projection: projection predeploys are active")

	implementationSlot   = common.HexToHash("0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc")
	adminSlot            = common.HexToHash("0xb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d6103")
	customGasTokenSlot   = common.HexToHash("0x4ad9936a67aeb1898ef7b848aecdf71a1f8999fbf63ff2f5b5691cb14bedfe4d")
	devFeatureBitmapSlot = common.HexToHash("0xc8bc8f9195cfb2d040744aac63412d02ffc186ea9bd519039edc4666ee9032bc")
	proxyAdminWord       = common.BytesToHash(predeploys.ProxyAdminAddr.Bytes())

	// l1BlockInteropFeatureSlot is L1Block.isFeatureEnabled[INTEROP]: `isFeatureEnabled` is a
	// mapping(bytes32 => bool) at storage slot 9 in both L1Block and L1BlockCGT, so the slot is
	// keccak256(abi.encode(bytes32("INTEROP"), uint256(9))). Swapping the implementation preserves
	// it: the two contracts share the layout.
	l1BlockInteropFeatureSlot = l1BlockFeatureSlot("INTEROP")
	trueWord                  = common.BigToHash(big.NewInt(1))
)

// StockL2ToL2CrossDomainMessengerCodeHash is the keccak256 of the stock L2ToL2CrossDomainMessenger
// implementation the projection replaces. It pins the contract release the projection was built
// against: a source whose messenger differs is from another release (or is already projected), and
// projecting it would pair a replay messenger with predeploys it was never tested with.
//
// The value comes from the stock op-deployer genesis fixture in testdata (see projection_test.go).
var StockL2ToL2CrossDomainMessengerCodeHash = common.HexToHash("0x6c9a755164bb4bf014b4b99358425e4480f3b022b7586b939a86f135c019acce")

//go:embed bytecode/*.hex
var bytecodes embed.FS

var publicProjectionCode = map[common.Address][]byte{
	codeNamespace(predeploys.L1BlockAddr):                    mustBytecode("L1Block"),
	codeNamespace(predeploys.L2ToL1MessagePasserAddr):        mustBytecode("L2ToL1MessagePasser"),
	codeNamespace(predeploys.L2toL2CrossDomainMessengerAddr): mustBytecode("L2ToL2CrossDomainMessengerReplay"),
	codeNamespace(predeploys.ClaimRegistryAddr):              mustBytecode("ClaimRegistry"),
	codeNamespace(predeploys.EventReplayerAddr):              mustBytecode("EventReplayer"),
}

// ProjectGenesisFrom constructs the public-projection genesis from a private-chain genesis.
//
// The operation is pure: it does not mutate privateChainGenesis and performs no I/O. The embedded
// bytecode is part of the projection protocol, just like an execution client's hardfork code.
//
// The source already carries the stock interop feature set (CrossL2Inbox, SuperchainETHBridge,
// ETHLiquidity with its uint128-max liquidity, L1Block.isFeatureEnabled[INTEROP]). The projection
// keeps all of that and changes exactly what makes the private chain private or custom-gas-token:
//
//   - execution semantics: the L1BlockCGT and L2ToL1MessagePasserCGT implementations become the
//     ETH ones and the custom-gas-token marker is cleared; LiquidityController and
//     NativeAssetLiquidity are deactivated;
//   - messaging: the stock L2ToL2CrossDomainMessenger implementation becomes the replay messenger,
//     and ClaimRegistry and EventReplayer are installed;
//   - block parameters: the gas limit is the maximum and the base fee is zero, because the batcher
//     is the projection's only sender and there is no fee market to observe.
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
	// token semantics. Every other L1Block storage word, the INTEROP feature bit included, is kept:
	// L1Block and L1BlockCGT share a storage layout.
	deleteStorage(out.Alloc, predeploys.L1BlockAddr, customGasTokenSlot)
	activateProxy(out.Alloc, predeploys.L1BlockAddr, publicProjectionCode[codeNamespace(predeploys.L1BlockAddr)])
	activateProxy(out.Alloc, predeploys.L2ToL1MessagePasserAddr, publicProjectionCode[codeNamespace(predeploys.L2ToL1MessagePasserAddr)])
	deactivateProxy(out.Alloc, predeploys.NativeAssetLiquidityAddr)
	deactivateProxy(out.Alloc, predeploys.LiquidityControllerAddr)

	// Public batches replay selected private-chain events and record the claim for every range.
	activateProxy(out.Alloc, predeploys.L2toL2CrossDomainMessengerAddr, publicProjectionCode[codeNamespace(predeploys.L2toL2CrossDomainMessengerAddr)])
	activateProxy(out.Alloc, predeploys.ClaimRegistryAddr, publicProjectionCode[codeNamespace(predeploys.ClaimRegistryAddr)])
	activateProxy(out.Alloc, predeploys.EventReplayerAddr, publicProjectionCode[codeNamespace(predeploys.EventReplayerAddr)])

	return out, nil
}

// ProjectRollupConfigFrom constructs the public-projection rollup config. The projected genesis
// supplies the only value that cannot be copied from the private chain: the L2 genesis hash. The
// genesis system config follows the projected block parameters. Everything else, the Lagoon
// activation time included, is the private chain's: both views activate interop at genesis.
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
	return &out, nil
}

// validatePrivateChainGenesis accepts exactly the stock op-deployer shape the projection is
// defined over, and names the first thing that is wrong otherwise.
func validatePrivateChainGenesis(g *core.Genesis) error {
	l1Block, ok := g.Alloc[predeploys.L1BlockAddr]
	if !ok || l1Block.Storage[customGasTokenSlot] == (common.Hash{}) {
		return ErrNotCustomGasToken
	}
	if l1Block.Storage[l1BlockInteropFeatureSlot] != trueWord {
		return ErrInteropInactive
	}
	featureFlags, ok := g.Alloc[predeploys.L2DevFeatureFlagsAddr]
	if !ok || !devfeatures.IsDevFeatureEnabled(featureFlags.Storage[devFeatureBitmapSlot], devfeatures.OptimismPortalInteropFlag) {
		return ErrInteropInactive
	}
	if implementationCodeHash(g.Alloc, predeploys.L2toL2CrossDomainMessengerAddr) != StockL2ToL2CrossDomainMessengerCodeHash {
		return ErrMessengerNotStock
	}
	for _, proxy := range []common.Address{predeploys.ClaimRegistryAddr, predeploys.EventReplayerAddr} {
		if g.Alloc[proxy].Storage[implementationSlot] != (common.Hash{}) {
			return ErrAlreadyProjected
		}
	}
	return nil
}

// implementationCodeHash follows a predeploy proxy's EIP-1967 implementation slot and hashes the
// code found there. An inactive proxy (no implementation) hashes to the empty-code hash, which no
// release matches.
func implementationCodeHash(alloc types.GenesisAlloc, proxy common.Address) common.Hash {
	impl := common.BytesToAddress(alloc[proxy].Storage[implementationSlot].Bytes())
	return crypto.Keccak256Hash(alloc[impl].Code)
}

func l1BlockFeatureSlot(feature string) common.Hash {
	var key [32]byte
	copy(key[:], feature)
	return crypto.Keccak256Hash(key[:], common.BigToHash(big.NewInt(9)).Bytes())
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

// activateProxy points a predeploy proxy at its code-namespace implementation and installs code
// there. A proxy whose implementation lived elsewhere (an L2ContractsManager deployment, say) is
// re-pointed; the old implementation account is left as it was, which is dead code and nothing
// else.
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
