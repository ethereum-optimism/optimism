package genesis

import (
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	// RequirePaidMessagesSlot matches the messenger's genesis-only policy slot.
	RequirePaidMessagesSlot = crypto.Keccak256Hash([]byte("privateinterop.requirePaidMessages"))
	policyMessengerCode     = mustBytecode("L2ToL2CrossDomainMessenger")
	policyBridgeCode        = mustBytecode("SuperchainETHBridge")
	// PolicyMessengerCodeHash pins the implementation that reads the private messaging policy.
	PolicyMessengerCodeHash = crypto.Keccak256Hash(policyMessengerCode)
	// PolicyBridgeCodeHash pins the implementation that enforces the native ETH route permissions.
	PolicyBridgeCodeHash = crypto.Keccak256Hash(policyBridgeCode)
	stockBridgeCodeHash  = common.HexToHash("0x90a1d78c6e6340d40c0c05a4a30b9bf326280fe95c7690ac964070b477d9fde4")
)

// ConfigurePrivateGenesis installs the ETH private-chain profile over a supported source genesis.
// It preserves ordinary deposits and balances, enables paid-only messenger operations, and starts
// the native bridge with empty send/relay permissions. The existing L2 ProxyAdmin owner controls
// those permissions. It does not configure or attest to the L1 backing pool.
//
// Only fresh, pinned source artifacts are accepted. This is an offline genesis transformation,
// not a live-state upgrade or a way to convert a CGT deployment to ETH.
func ConfigurePrivateGenesis(source *core.Genesis, cfg *rollup.Config) (*core.Genesis, *rollup.Config, error) {
	if source == nil || source.Config == nil || cfg == nil || cfg.L2ChainID == nil {
		return nil, nil, errors.New("source genesis and rollup chain configuration are required")
	}
	if source.Config.ChainID == nil || source.Config.ChainID.Cmp(cfg.L2ChainID) != 0 ||
		source.Timestamp != cfg.Genesis.L2Time || source.ToBlock().Hash() != cfg.Genesis.L2.Hash {
		return nil, nil, errors.New("source genesis does not match rollup configuration")
	}
	if err := validatePrivateChainGenesis(source); err != nil {
		return nil, nil, err
	}
	if isCustomGasToken(source) {
		return nil, nil, errors.New("private ETH profile requires an ETH source deployment, not CGT")
	}
	if source.Alloc[predeploys.L2toL2CrossDomainMessengerAddr].Storage[RequirePaidMessagesSlot] != (common.Hash{}) {
		return nil, nil, errors.New("private messenger policy is already configured")
	}
	bridgeHash := implementationCodeHash(source.Alloc, predeploys.SuperchainETHBridgeAddr)
	if bridgeHash != stockBridgeCodeHash && bridgeHash != PolicyBridgeCodeHash {
		return nil, nil, fmt.Errorf("unsupported source SuperchainETHBridge code hash %s", bridgeHash)
	}
	// Refuse unknown bridge state instead of silently retaining previously enabled routes.
	for slot, value := range source.Alloc[predeploys.SuperchainETHBridgeAddr].Storage {
		if slot != implementationSlot && slot != adminSlot && value != (common.Hash{}) {
			return nil, nil, fmt.Errorf("source bridge has nonempty policy storage at %s", slot)
		}
	}
	out := cloneGenesis(source)
	activateProxy(out.Alloc, predeploys.L2toL2CrossDomainMessengerAddr, policyMessengerCode)
	account := accountAt(out.Alloc, predeploys.L2toL2CrossDomainMessengerAddr)
	account.Storage[RequirePaidMessagesSlot] = trueWord
	out.Alloc[predeploys.L2toL2CrossDomainMessengerAddr] = account
	activateProxy(out.Alloc, predeploys.SuperchainETHBridgeAddr, policyBridgeCode)
	privateRollup := *cfg
	privateRollup.Genesis.L2.Hash = out.ToBlock().Hash()
	return out, &privateRollup, nil
}
