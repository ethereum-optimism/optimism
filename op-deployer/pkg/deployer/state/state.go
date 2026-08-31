package state

import (
	"crypto/rand"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/core"

	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/ioutil"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
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

	// PreparedDeployment freezes inputs and predictions for chains awaiting deployment.
	// Values committed by other stages, such as Prestate and StartingAnchorRoot, remain in ChainState.
	PreparedDeployment *PreparedDeployment `json:"preparedDeployment,omitempty"`

	// AppliedIntent contains the chain intent that was last
	// successfully applied. It is diffed against new intent
	// in order to determine what deployment steps to take.
	// This field is nil for new deployments.
	AppliedIntent *Intent `json:"appliedIntent"`

	// InteropDepSet contains the interop dependency set generated from all intent chains during prepare.
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

	// SP1Verifier is the raw verifier selected when deploying the implementations bundle.
	SP1Verifier *common.Address `json:"sp1Verifier,omitempty"`

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
// during the prepare dry-run. States outside the prepare flow have no snapshot.
func (s *State) CheckL1PredictInputs(deployer common.Address, opcm common.Address) error {
	if s.PreparedDeployment == nil {
		return nil
	}
	if s.PreparedDeployment.Deployer != deployer {
		return fmt.Errorf("deployer address mismatch: expected %s, got %s", s.PreparedDeployment.Deployer.Hex(), deployer.Hex())
	}
	if s.PreparedDeployment.OPCM != opcm {
		return fmt.Errorf("opcm address mismatch: expected %s, got %s", s.PreparedDeployment.OPCM.Hex(), opcm.Hex())
	}
	return nil
}

// CheckNotPrepared returns an error if the state was produced by the prepare
// pipeline.
func (s *State) CheckNotPrepared() error {
	if s.PreparedDeployment != nil {
		return fmt.Errorf("state was produced by the prepare pipeline and cannot be applied")
	}
	return nil
}

// CheckNotApplied returns an error if the state was produced by the apply
// pipeline.
func (s *State) CheckNotApplied() error {
	if s.AppliedIntent != nil {
		return fmt.Errorf("state was produced by the apply pipeline and cannot be prepared")
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

// StartingAnchorProposal is the committed starting-anchor proposal for a permissionless chain.
type StartingAnchorProposal struct {
	Root             common.Hash    `json:"root"`
	L2SequenceNumber hexutil.Uint64 `json:"l2SequenceNumber"`
}

// PreparedArtifact binds an artifact locator to the bundle contents resolved during prepare.
type PreparedArtifact struct {
	Locator       *artifacts.Locator `json:"locator"`
	ContentDigest common.Hash        `json:"contentDigest"`
}

// PreparedDeployment freezes the inputs and predictions for chains awaiting deployment.
type PreparedDeployment struct {
	Intent      *Intent               `json:"intent"`
	Deployer    common.Address        `json:"deployer"`
	OPCM        common.Address        `json:"opcm"`
	L1Artifacts PreparedArtifact      `json:"l1Artifacts"`
	L2Artifacts PreparedArtifact      `json:"l2Artifacts"`
	Chains      []*PreparedChainState `json:"chains"`
}

// PreparedChainState freezes prediction output and timing for one undeployed chain.
type PreparedChainState struct {
	ID common.Hash `json:"id"`

	addresses.OpChainContracts

	// LegacyAltDAChallengeProxy and LegacyAltDAChallengeImpl are decode-only
	// compatibility fields for state files written before Alt-DA was removed.
	LegacyAltDAChallengeProxy *common.Address `json:"AltDAChallengeProxy,omitempty"`
	LegacyAltDAChallengeImpl  *common.Address `json:"AltDAChallengeImpl,omitempty"`

	StartBlock  *L1BlockRefJSON `json:"startBlock"`
	GenesisTime *hexutil.Uint64 `json:"genesisTime"`
}

// Chain returns the frozen state for a chain included in the prepared deployment.
func (p *PreparedDeployment) Chain(id common.Hash) (*PreparedChainState, error) {
	for _, chain := range p.Chains {
		if chain.ID == id {
			return chain, nil
		}
	}
	return nil, fmt.Errorf("prepared chain not found: %s", id.Hex())
}

// Clone returns a deep copy of the prepared deployment, detached from the receiver's pointers.
func (p *PreparedDeployment) Clone() (*PreparedDeployment, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("failed to encode prepared deployment: %w", err)
	}
	var clone PreparedDeployment
	if err := json.Unmarshal(data, &clone); err != nil {
		return nil, fmt.Errorf("failed to decode prepared deployment: %w", err)
	}
	return &clone, nil
}

// ContinuationState marks a chain recorded by the continuation workflow.
type ContinuationState struct{}

type ChainState struct {
	ID common.Hash `json:"id"`

	addresses.OpChainContracts

	// LegacyAltDAChallengeProxy and LegacyAltDAChallengeImpl are decode-only
	// compatibility fields for state files written before Alt-DA was removed.
	LegacyAltDAChallengeProxy *common.Address `json:"AltDAChallengeProxy,omitempty"`
	LegacyAltDAChallengeImpl  *common.Address `json:"AltDAChallengeImpl,omitempty"`

	// Deployed indicates whether the addresses in this chain have been deployed or are just addresses produced
	// by the prediction step of the prepare command.
	Deployed *bool `json:"deployed,omitempty"`

	// Prestate is the selected absolute prestate for permissionless games.
	Prestate common.Hash `json:"prestate,omitzero"`

	// StartingAnchorRoot is produced by the proposal-producing stage and consumed
	// when building the continuation deploy input.
	StartingAnchorRoot *StartingAnchorProposal `json:"startingAnchorRoot,omitempty"`

	// InitialGameType records the game type used by prepare to detect intent drift.
	// Legacy states without it must be prepared again before continuation.
	InitialGameType *uint32 `json:"initialGameType,omitempty"`

	AdditionalDisputeGames []AdditionalDisputeGameState `json:"additionalDisputeGames"`

	Allocs *GzipData[foundry.ForgeAllocs] `json:"allocs"`

	StartBlock *L1BlockRefJSON `json:"startBlock"`

	// GenesisTime is the L2 genesis timestamp pinned with StartBlock.
	// Nil leaves deploy config to derive it from L1StartingBlockTag.
	GenesisTime *hexutil.Uint64 `json:"genesisTime,omitempty"`

	Continuation *ContinuationState `json:"continuation,omitempty"`

	// GenesisBlockHash is the L2 genesis block hash computed from Allocs, the combined deploy
	// config, and the pinned StartBlock/GenesisTime. Used by post-deploy validation to confirm
	// on-chain seeding matches the predicted genesis.
	GenesisBlockHash *common.Hash `json:"genesisBlockHash,omitempty"`
}

func checkLegacyAltDAAddresses(id common.Hash, proxy, impl *common.Address) error {
	if (proxy != nil && *proxy != (common.Address{})) ||
		(impl != nil && *impl != (common.Address{})) {
		return fmt.Errorf("%w: chain %s contains legacy Alt-DA challenge addresses", ErrAltDANoLongerSupported, id)
	}
	return nil
}

func rejectLegacyAltDAAddresses(id common.Hash, proxy, impl **common.Address) error {
	if err := checkLegacyAltDAAddresses(id, *proxy, *impl); err != nil {
		return err
	}
	// Empty addresses were emitted by older op-deployer versions even when
	// Alt-DA was disabled. Accept them, but never write them back out.
	*proxy = nil
	*impl = nil
	return nil
}

// RejectUnsupportedAltDA rejects non-empty legacy Alt-DA state and clears
// empty decode-only compatibility fields.
func (s *State) RejectUnsupportedAltDA() error {
	if err := s.checkUnsupportedAltDA(); err != nil {
		return err
	}
	if s.AppliedIntent != nil {
		s.AppliedIntent.clearLegacyAltDACompatibilityFields()
	}
	if s.PreparedDeployment != nil && s.PreparedDeployment.Intent != nil {
		s.PreparedDeployment.Intent.clearLegacyAltDACompatibilityFields()
	}
	for _, chain := range s.Chains {
		if chain == nil {
			continue
		}
		if err := rejectLegacyAltDAAddresses(
			chain.ID,
			&chain.LegacyAltDAChallengeProxy,
			&chain.LegacyAltDAChallengeImpl,
		); err != nil {
			return err
		}
	}
	if s.PreparedDeployment != nil {
		for _, chain := range s.PreparedDeployment.Chains {
			if chain == nil {
				continue
			}
			if err := rejectLegacyAltDAAddresses(
				chain.ID,
				&chain.LegacyAltDAChallengeProxy,
				&chain.LegacyAltDAChallengeImpl,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *State) checkUnsupportedAltDA() error {
	if s.AppliedIntent != nil {
		if err := s.AppliedIntent.checkUnsupportedAltDA(); err != nil {
			return err
		}
	}
	if s.PreparedDeployment != nil && s.PreparedDeployment.Intent != nil {
		if err := s.PreparedDeployment.Intent.checkUnsupportedAltDA(); err != nil {
			return err
		}
	}
	for _, chain := range s.Chains {
		if chain == nil {
			continue
		}
		if err := checkLegacyAltDAAddresses(chain.ID, chain.LegacyAltDAChallengeProxy, chain.LegacyAltDAChallengeImpl); err != nil {
			return err
		}
	}
	if s.PreparedDeployment != nil {
		for _, chain := range s.PreparedDeployment.Chains {
			if chain == nil {
				continue
			}
			if err := checkLegacyAltDAAddresses(chain.ID, chain.LegacyAltDAChallengeProxy, chain.LegacyAltDAChallengeImpl); err != nil {
				return err
			}
		}
	}
	return nil
}

// ClearDerivedArtifacts clears every value derived from the chain's predicted L1
// addresses.
func (c *ChainState) ClearDerivedArtifacts() {
	c.Prestate = common.Hash{}
	c.StartingAnchorRoot = nil
	c.Allocs = nil
	c.GenesisBlockHash = nil
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
func (s *State) SetChainContracts(id common.Hash, contracts addresses.OpChainContracts, deployed bool) {
	for _, chain := range s.Chains {
		if chain.ID == id {
			chain.OpChainContracts = contracts
			chain.Deployed = &deployed
			return
		}
	}
	s.Chains = append(s.Chains, &ChainState{
		ID:               id,
		OpChainContracts: contracts,
		Deployed:         &deployed,
	})
}

// PinChainAnchor records a chain's L1 anchor block and derived L2 genesis time.
// Downstream stages must reuse this pair so generated artifacts agree and
// reruns remain idempotent. New entries are marked undeployed, while updates
// preserve fields written by other stages.
func (s *State) PinChainAnchor(id common.Hash, anchor *L1BlockRefJSON, genesisTime hexutil.Uint64) {
	for _, chain := range s.Chains {
		if chain.ID == id {
			chain.StartBlock = anchor
			chain.GenesisTime = &genesisTime
			return
		}
	}
	s.Chains = append(s.Chains, &ChainState{
		ID:          id,
		Deployed:    ptr.New(false),
		StartBlock:  anchor,
		GenesisTime: &genesisTime,
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
