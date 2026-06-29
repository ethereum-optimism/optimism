package state

import (
	"crypto/rand"
	"fmt"

	"github.com/ethereum/go-ethereum/core"

	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/ioutil"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// State contains the data needed to recreate the deployment
// as it progresses and once it is fully applied.
type State struct {
	// Version versions the state so we can update it later.
	Version int `json:"version"`

	// OpDeployerVersion is the version of op-deployer that was used to create the state
	OpDeployerVersion string `json:"opDeployerVersion"`

	// Create2Salt is the salt used for CREATE2 deployments.
	Create2Salt common.Hash `json:"create2Salt"`

	// L1PredictSenderAddress is the address that performed the L1 deploy dry-run.
	// It is used to verify that the same deployer address is used for the relevant stages of the permissionless pipeline.
	L1PredictSenderAddress *common.Address `json:"predictSenderAddress,omitempty"`

	// L1PredictOPCMAddress is the OPCM address used for the L1 deploy dry-run.
	// It is used to verify that the same OPCM is used for the relevant stages of the permissionless pipeline.
	L1PredictOPCMAddress *common.Address `json:"predictOpcmAddress,omitempty"`

	// Prepared is set when this state was produced by the prepare pipeline.
	Prepared bool `json:"prepared,omitempty"`

	// AppliedIntent contains the chain intent that was last
	// successfully applied. It is diffed against new intent
	// in order to determine what deployment steps to take.
	// This field is nil for new deployments.
	AppliedIntent *Intent `json:"appliedIntent"`

	// InteropDepSet contains the interop dependency set render by the prestate SDK if interop is enabled
	InteropDepSet *depset.StaticConfigDependencySet `json:"interopDepSet,omitempty"`

	// SuperchainDeployment contains the addresses of the Superchain
	// deployment. It only contains the proxies because the implementations
	// can be looked up on chain.
	SuperchainDeployment *addresses.SuperchainContracts `json:"superchainContracts"`

	// SuperchainRoles contains the addresses of the Superchain roles.
	SuperchainRoles *addresses.SuperchainRoles `json:"superchainRoles"`

	// ImplementationsDeployment contains the addresses of the common implementation
	// contracts required for the Superchain to function.
	ImplementationsDeployment *addresses.ImplementationsContracts `json:"implementationsDeployment"`

	// Chains contains data about L2 chain deployments.
	Chains []*ChainState `json:"opChainDeployments"`

	// L1StateDump contains the complete L1 state dump of the deployment.
	L1StateDump *GzipData[foundry.ForgeAllocs] `json:"l1StateDump"`

	// L1DevGenesis contains the dev L1 genesis, and may be nil if this is not a genesis-strategy deployment.
	// Warning: the Allocs part of the genesis is not included. Instead, the stateHash attribute is set.
	// The allocs are included in L1StateDump.
	// The stateHash can be used for consistency checks and faster block-hash computation.
	L1DevGenesis *core.Genesis `json:"-"`

	// DeploymentCalldata contains the calldata of each transaction in the deployment. This is only
	// populated if apply is called with --deployment-target=calldata.
	DeploymentCalldata []broadcaster.CalldataDump
}

func (s *State) WriteToFile(path string) error {
	return jsonutil.WriteJSON(s, ioutil.ToAtomicFile(path, 0o755))
}

func (s *State) Chain(id common.Hash) (*ChainState, error) {
	for _, chain := range s.Chains {
		if chain.ID == id {
			return chain, nil
		}
	}
	return nil, fmt.Errorf("chain not found: %s", id.Hex())
}

// CheckL1PredictInputs verifies that the deployer and OPCM match the values pinned
// during the prepare dry-run, keeping the predicted L1 addresses valid across the
// relevant stages of the permissionless pipeline. A nil pinned value means nothing
// was pinned for that input, so any value is accepted (older pipeline).
func (s *State) CheckL1PredictInputs(deployer common.Address, opcm common.Address) error {
	if s.L1PredictSenderAddress != nil && *s.L1PredictSenderAddress != deployer {
		return fmt.Errorf("deployer address mismatch: expected %s, got %s", s.L1PredictSenderAddress.Hex(), deployer.Hex())
	}
	if s.L1PredictOPCMAddress != nil && *s.L1PredictOPCMAddress != opcm {
		return fmt.Errorf("opcm address mismatch: expected %s, got %s", s.L1PredictOPCMAddress.Hex(), opcm.Hex())
	}
	return nil
}

// CheckNotPrepared returns an error if the state was produced by the prepare
// pipeline.
func (s *State) CheckNotPrepared() error {
	if s.Prepared {
		return fmt.Errorf("state was produced by the prepare pipeline and cannot be applied")
	}
	return nil
}

// EnsureCreate2Salt generates a random CREATE2 salt if one has not been set yet.
// If a salt has been already set then it is preserved.
func (s *State) EnsureCreate2Salt() error {
	if s.Create2Salt != (common.Hash{}) {
		return nil
	}
	if _, err := rand.Read(s.Create2Salt[:]); err != nil {
		return fmt.Errorf("failed to generate CREATE2 salt: %w", err)
	}
	return nil
}

type AdditionalDisputeGameState struct {
	GameType      uint32
	GameAddress   common.Address
	VMAddress     common.Address
	OracleAddress common.Address
	VMType        VMType
}

type ChainState struct {
	ID common.Hash `json:"id"`

	addresses.OpChainContracts

	// Deployed indicates whether the addresses in this chain have been deployed or are just addresses produced
	// by the prediction step of the prepare command.
	Deployed *bool `json:"deployed,omitempty"`

	AdditionalDisputeGames []AdditionalDisputeGameState `json:"additionalDisputeGames"`

	Allocs *GzipData[foundry.ForgeAllocs] `json:"allocs"`

	StartBlock *L1BlockRefJSON `json:"startBlock"`
}

// IsChainDeployed reports whether the chain's addresses have been broadcast.
// States from older pipelines have no flag and are treated as deployed, any
// unknown chain is treated as not yet deployed.
func (s *State) IsChainDeployed(id common.Hash) bool {
	for _, chain := range s.Chains {
		if chain.ID == id {
			return chain.Deployed == nil || *chain.Deployed
		}
	}
	return false
}

// SetChainContracts records the L1 contract addresses for a chain. It creates
// the chain entry if it does not exist and otherwise updates it in place,
// preserving any other fields already set by other stages. deployed indicates whether the addresses
// have been already broadcast or are just predicted addresses from the prepare stage.
// A non-nil startBlock pins the chain's L1 anchor block and a nil startBlock leaves any
// existing StartBlock untouched so stages that don't set it (e.g. deploy) don't overwrite it.
func (s *State) SetChainContracts(id common.Hash, contracts addresses.OpChainContracts, startBlock *L1BlockRefJSON, deployed bool) {
	for _, chain := range s.Chains {
		if chain.ID == id {
			chain.OpChainContracts = contracts
			chain.Deployed = &deployed
			if startBlock != nil {
				chain.StartBlock = startBlock
			}
			return
		}
	}
	s.Chains = append(s.Chains, &ChainState{
		ID:               id,
		OpChainContracts: contracts,
		Deployed:         &deployed,
		StartBlock:       startBlock,
	})
}

type L1BlockRefJSON struct {
	Hash       common.Hash    `json:"hash"`
	ParentHash common.Hash    `json:"parentHash"`
	Number     hexutil.Uint64 `json:"number"`
	Time       hexutil.Uint64 `json:"timestamp"`
}

func (b *L1BlockRefJSON) ToBlockRef() *eth.BlockRef {
	return &eth.BlockRef{
		Hash:       b.Hash,
		Number:     uint64(b.Number),
		ParentHash: b.ParentHash,
		Time:       uint64(b.Time),
	}
}

func BlockRefJsonFromBlockRef(br *eth.BlockRef) *L1BlockRefJSON {
	return &L1BlockRefJSON{
		Hash:       br.Hash,
		Number:     hexutil.Uint64(br.Number),
		ParentHash: br.ParentHash,
		Time:       hexutil.Uint64(br.Time),
	}
}

func BlockRefJsonFromHeader(h *types.Header) *L1BlockRefJSON {
	return BlockRefJsonFromBlockRef(eth.BlockRefFromHeader(h))
}
