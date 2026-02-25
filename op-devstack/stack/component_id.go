package stack

import (
	"bytes"
	"fmt"
	"log/slog"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/holiman/uint256"
)

// ComponentKind identifies the type of component.
// This is used in serialization to make each ID unique and type-safe.
type ComponentKind string

// All component kinds. These values are used in serialization and must remain stable.
const (
	KindL1ELNode          ComponentKind = "L1ELNode"
	KindL1CLNode          ComponentKind = "L1CLNode"
	KindL1Network         ComponentKind = "L1Network"
	KindL2ELNode          ComponentKind = "L2ELNode"
	KindL2CLNode          ComponentKind = "L2CLNode"
	KindL2Network         ComponentKind = "L2Network"
	KindL2Batcher         ComponentKind = "L2Batcher"
	KindL2Proposer        ComponentKind = "L2Proposer"
	KindL2Challenger      ComponentKind = "L2Challenger"
	KindRollupBoostNode   ComponentKind = "RollupBoostNode"
	KindOPRBuilderNode    ComponentKind = "OPRBuilderNode"
	KindFaucet            ComponentKind = "Faucet"
	KindSyncTester        ComponentKind = "SyncTester"
	KindSupervisor        ComponentKind = "Supervisor"
	KindConductor         ComponentKind = "Conductor"
	KindCluster           ComponentKind = "Cluster"
	KindSuperchain        ComponentKind = "Superchain"
	KindSupernode         ComponentKind = "Supernode"
	KindTestSequencer     ComponentKind = "TestSequencer"
	KindFlashblocksClient ComponentKind = "FlashblocksWSClient"
)

var hydrationComponentKindOrder = []ComponentKind{
	KindSuperchain,
	KindCluster,
	KindL1Network,
	KindL2Network,
	KindL1ELNode,
	KindL1CLNode,
	KindL2ELNode,
	KindOPRBuilderNode,
	KindRollupBoostNode,
	KindL2CLNode,
	KindSupervisor,
	KindTestSequencer,
	KindL2Batcher,
	KindL2Challenger,
	KindL2Proposer,
}

// HydrationComponentKindOrder returns the deterministic kind ordering used by orchestrator hydration.
func HydrationComponentKindOrder() []ComponentKind {
	out := make([]ComponentKind, len(hydrationComponentKindOrder))
	copy(out, hydrationComponentKindOrder)
	return out
}

// IDShape defines which fields an ID uses.
type IDShape uint8

const (
	// IDShapeKeyAndChain indicates the ID has both a key and chainID (e.g., L2Batcher-mynode-420)
	IDShapeKeyAndChain IDShape = iota
	// IDShapeChainOnly indicates the ID has only a chainID (e.g., L1Network-1)
	IDShapeChainOnly
	// IDShapeKeyOnly indicates the ID has only a key (e.g., Supervisor-mysupervisor)
	IDShapeKeyOnly
)

// ComponentID is the unified identifier for all components.
// It contains all possible fields; the shape determines which are used.
type ComponentID struct {
	kind    ComponentKind
	shape   IDShape
	key     string
	chainID eth.ChainID
}

// NewComponentID creates a new ComponentID with key and chainID.
func NewComponentID(kind ComponentKind, key string, chainID eth.ChainID) ComponentID {
	return ComponentID{
		kind:    kind,
		shape:   IDShapeKeyAndChain,
		key:     key,
		chainID: chainID,
	}
}

// NewComponentIDChainOnly creates a new ComponentID with only a chainID.
func NewComponentIDChainOnly(kind ComponentKind, chainID eth.ChainID) ComponentID {
	return ComponentID{
		kind:    kind,
		shape:   IDShapeChainOnly,
		chainID: chainID,
	}
}

// NewComponentIDKeyOnly creates a new ComponentID with only a key.
func NewComponentIDKeyOnly(kind ComponentKind, key string) ComponentID {
	return ComponentID{
		kind:  kind,
		shape: IDShapeKeyOnly,
		key:   key,
	}
}

func (id ComponentID) Kind() ComponentKind {
	return id.kind
}

func (id ComponentID) Shape() IDShape {
	return id.shape
}

// HasChainID returns true if this ID has a chain ID component.
// This is true for IDShapeKeyAndChain and IDShapeChainOnly shapes.
func (id ComponentID) HasChainID() bool {
	return id.shape == IDShapeKeyAndChain || id.shape == IDShapeChainOnly
}

func (id ComponentID) Key() string {
	return id.key
}

func (id ComponentID) ChainID() eth.ChainID {
	return id.chainID
}

func (id ComponentID) String() string {
	switch id.shape {
	case IDShapeKeyAndChain:
		return fmt.Sprintf("%s-%s-%s", id.kind, id.key, id.chainID)
	case IDShapeChainOnly:
		return fmt.Sprintf("%s-%s", id.kind, id.chainID)
	case IDShapeKeyOnly:
		return fmt.Sprintf("%s-%s", id.kind, id.key)
	default:
		return fmt.Sprintf("%s-<invalid>", id.kind)
	}
}

func (id ComponentID) LogValue() slog.Value {
	return slog.StringValue(id.String())
}

func (id ComponentID) MarshalText() ([]byte, error) {
	if id.shape == IDShapeKeyAndChain || id.shape == IDShapeKeyOnly {
		if len(id.key) > maxIDLength {
			return nil, errInvalidID
		}
	}
	return []byte(id.String()), nil
}

func (id *ComponentID) UnmarshalText(data []byte) error {
	return id.unmarshalTextWithKind(id.kind, data)
}

// unmarshalTextWithKind unmarshals the ID, validating that the kind matches.
func (id *ComponentID) unmarshalTextWithKind(expectedKind ComponentKind, data []byte) error {
	kindData, rest, ok := bytes.Cut(data, []byte("-"))
	if !ok {
		return fmt.Errorf("expected kind-prefix, but id has none: %q", data)
	}
	actualKind := ComponentKind(kindData)
	if actualKind != expectedKind {
		return fmt.Errorf("id %q has unexpected kind %q, expected %q", string(data), actualKind, expectedKind)
	}

	// Determine shape based on expected kind
	shape := kindToShape(expectedKind)
	id.kind = expectedKind
	id.shape = shape

	switch shape {
	case IDShapeKeyAndChain:
		keyData, chainData, ok := bytes.Cut(rest, []byte("-"))
		if !ok {
			return fmt.Errorf("expected chain separator, but found none: %q", string(data))
		}
		if len(keyData) > maxIDLength {
			return errInvalidID
		}
		var chainID eth.ChainID
		if err := chainID.UnmarshalText(chainData); err != nil {
			return fmt.Errorf("failed to unmarshal chain part: %w", err)
		}
		id.key = string(keyData)
		id.chainID = chainID
	case IDShapeChainOnly:
		var chainID eth.ChainID
		if err := chainID.UnmarshalText(rest); err != nil {
			return fmt.Errorf("failed to unmarshal chain part: %w", err)
		}
		id.chainID = chainID
		id.key = ""
	case IDShapeKeyOnly:
		if len(rest) > maxIDLength {
			return errInvalidID
		}
		id.key = string(rest)
		id.chainID = eth.ChainID(*uint256.NewInt(0))
	}
	return nil
}

// kindToShape returns the IDShape for a given ComponentKind.
func kindToShape(kind ComponentKind) IDShape {
	switch kind {
	case KindL1Network, KindL2Network:
		return IDShapeChainOnly
	case KindSupervisor, KindConductor, KindCluster, KindSuperchain, KindTestSequencer, KindSupernode:
		return IDShapeKeyOnly
	default:
		return IDShapeKeyAndChain
	}
}

// Less compares two ComponentIDs for sorting.
func (id ComponentID) Less(other ComponentID) bool {
	if id.kind != other.kind {
		return id.kind < other.kind
	}
	if id.key != other.key {
		return id.key < other.key
	}
	return id.chainID.Cmp(other.chainID) < 0
}

// KindMarker is implemented by marker types to associate them with their ComponentKind.
// This enables type-safe unmarshaling of ID[T] types.
type KindMarker interface {
	componentKind() ComponentKind
}

// ID is a type-safe wrapper around ComponentID.
// The type parameter T must implement KindMarker to enable unmarshaling.
// This prevents accidentally mixing up different ID types (e.g., L2BatcherID vs L2ELNodeID).
type ID[T KindMarker] struct {
	ComponentID
}

// Marker types for each component kind.
// Each marker implements KindMarker to associate it with the correct ComponentKind.
type (
	L1ELNodeMarker          struct{}
	L1CLNodeMarker          struct{}
	L1NetworkMarker         struct{}
	L2ELNodeMarker          struct{}
	L2CLNodeMarker          struct{}
	L2NetworkMarker         struct{}
	L2BatcherMarker         struct{}
	L2ProposerMarker        struct{}
	L2ChallengerMarker      struct{}
	RollupBoostNodeMarker   struct{}
	OPRBuilderNodeMarker    struct{}
	FaucetMarker            struct{}
	SyncTesterMarker        struct{}
	SupervisorMarker        struct{}
	ConductorMarker         struct{}
	ClusterMarker           struct{}
	SuperchainMarker        struct{}
	TestSequencerMarker     struct{}
	FlashblocksClientMarker struct{}
)

// KindMarker implementations for all marker types.
func (L1ELNodeMarker) componentKind() ComponentKind          { return KindL1ELNode }
func (L1CLNodeMarker) componentKind() ComponentKind          { return KindL1CLNode }
func (L1NetworkMarker) componentKind() ComponentKind         { return KindL1Network }
func (L2ELNodeMarker) componentKind() ComponentKind          { return KindL2ELNode }
func (L2CLNodeMarker) componentKind() ComponentKind          { return KindL2CLNode }
func (L2NetworkMarker) componentKind() ComponentKind         { return KindL2Network }
func (L2BatcherMarker) componentKind() ComponentKind         { return KindL2Batcher }
func (L2ProposerMarker) componentKind() ComponentKind        { return KindL2Proposer }
func (L2ChallengerMarker) componentKind() ComponentKind      { return KindL2Challenger }
func (RollupBoostNodeMarker) componentKind() ComponentKind   { return KindRollupBoostNode }
func (OPRBuilderNodeMarker) componentKind() ComponentKind    { return KindOPRBuilderNode }
func (FaucetMarker) componentKind() ComponentKind            { return KindFaucet }
func (SyncTesterMarker) componentKind() ComponentKind        { return KindSyncTester }
func (SupervisorMarker) componentKind() ComponentKind        { return KindSupervisor }
func (ConductorMarker) componentKind() ComponentKind         { return KindConductor }
func (ClusterMarker) componentKind() ComponentKind           { return KindCluster }
func (SuperchainMarker) componentKind() ComponentKind        { return KindSuperchain }
func (TestSequencerMarker) componentKind() ComponentKind     { return KindTestSequencer }
func (FlashblocksClientMarker) componentKind() ComponentKind { return KindFlashblocksClient }

// Type-safe ID type aliases using marker types.
// These maintain backward compatibility with existing code.
type (
	L1ELNodeID2          = ID[L1ELNodeMarker]
	L1CLNodeID2          = ID[L1CLNodeMarker]
	L1NetworkID2         = ID[L1NetworkMarker]
	L2ELNodeID2          = ID[L2ELNodeMarker]
	L2CLNodeID2          = ID[L2CLNodeMarker]
	L2NetworkID2         = ID[L2NetworkMarker]
	L2BatcherID2         = ID[L2BatcherMarker]
	L2ProposerID2        = ID[L2ProposerMarker]
	L2ChallengerID2      = ID[L2ChallengerMarker]
	RollupBoostNodeID2   = ID[RollupBoostNodeMarker]
	OPRBuilderNodeID2    = ID[OPRBuilderNodeMarker]
	FaucetID2            = ID[FaucetMarker]
	SyncTesterID2        = ID[SyncTesterMarker]
	SupervisorID2        = ID[SupervisorMarker]
	ConductorID2         = ID[ConductorMarker]
	ClusterID2           = ID[ClusterMarker]
	SuperchainID2        = ID[SuperchainMarker]
	TestSequencerID2     = ID[TestSequencerMarker]
	FlashblocksClientID2 = ID[FlashblocksClientMarker]
)

// Type-safe constructors for each ID type.

func NewL1ELNodeID2(key string, chainID eth.ChainID) L1ELNodeID2 {
	return L1ELNodeID2{NewComponentID(KindL1ELNode, key, chainID)}
}

func NewL1CLNodeID2(key string, chainID eth.ChainID) L1CLNodeID2 {
	return L1CLNodeID2{NewComponentID(KindL1CLNode, key, chainID)}
}

func NewL1NetworkID2(chainID eth.ChainID) L1NetworkID2 {
	return L1NetworkID2{NewComponentIDChainOnly(KindL1Network, chainID)}
}

func NewL2ELNodeID2(key string, chainID eth.ChainID) L2ELNodeID2 {
	return L2ELNodeID2{NewComponentID(KindL2ELNode, key, chainID)}
}

func NewL2CLNodeID2(key string, chainID eth.ChainID) L2CLNodeID2 {
	return L2CLNodeID2{NewComponentID(KindL2CLNode, key, chainID)}
}

func NewL2NetworkID2(chainID eth.ChainID) L2NetworkID2 {
	return L2NetworkID2{NewComponentIDChainOnly(KindL2Network, chainID)}
}

func NewL2BatcherID2(key string, chainID eth.ChainID) L2BatcherID2 {
	return L2BatcherID2{NewComponentID(KindL2Batcher, key, chainID)}
}

func NewL2ProposerID2(key string, chainID eth.ChainID) L2ProposerID2 {
	return L2ProposerID2{NewComponentID(KindL2Proposer, key, chainID)}
}

func NewL2ChallengerID2(key string, chainID eth.ChainID) L2ChallengerID2 {
	return L2ChallengerID2{NewComponentID(KindL2Challenger, key, chainID)}
}

func NewRollupBoostNodeID2(key string, chainID eth.ChainID) RollupBoostNodeID2 {
	return RollupBoostNodeID2{NewComponentID(KindRollupBoostNode, key, chainID)}
}

func NewOPRBuilderNodeID2(key string, chainID eth.ChainID) OPRBuilderNodeID2 {
	return OPRBuilderNodeID2{NewComponentID(KindOPRBuilderNode, key, chainID)}
}

func NewFaucetID2(key string, chainID eth.ChainID) FaucetID2 {
	return FaucetID2{NewComponentID(KindFaucet, key, chainID)}
}

func NewSyncTesterID2(key string, chainID eth.ChainID) SyncTesterID2 {
	return SyncTesterID2{NewComponentID(KindSyncTester, key, chainID)}
}

func NewSupervisorID2(key string) SupervisorID2 {
	return SupervisorID2{NewComponentIDKeyOnly(KindSupervisor, key)}
}

func NewConductorID2(key string) ConductorID2 {
	return ConductorID2{NewComponentIDKeyOnly(KindConductor, key)}
}

func NewClusterID2(key string) ClusterID2 {
	return ClusterID2{NewComponentIDKeyOnly(KindCluster, key)}
}

func NewSuperchainID2(key string) SuperchainID2 {
	return SuperchainID2{NewComponentIDKeyOnly(KindSuperchain, key)}
}

func NewTestSequencerID2(key string) TestSequencerID2 {
	return TestSequencerID2{NewComponentIDKeyOnly(KindTestSequencer, key)}
}

func NewFlashblocksClientID2(key string, chainID eth.ChainID) FlashblocksClientID2 {
	return FlashblocksClientID2{NewComponentID(KindFlashblocksClient, key, chainID)}
}

// ID methods that delegate to ComponentID but preserve type safety.

// Kind returns the ComponentKind for this ID type.
// Unlike ComponentID.Kind(), this works even on zero values.
func (id ID[T]) Kind() ComponentKind {
	var marker T
	return marker.componentKind()
}

func (id ID[T]) String() string {
	return id.ComponentID.String()
}

func (id ID[T]) LogValue() slog.Value {
	return id.ComponentID.LogValue()
}

func (id ID[T]) MarshalText() ([]byte, error) {
	return id.ComponentID.MarshalText()
}

func (id *ID[T]) UnmarshalText(data []byte) error {
	var marker T
	return id.ComponentID.unmarshalTextWithKind(marker.componentKind(), data)
}

// Less compares two IDs of the same type for sorting.
func (id ID[T]) Less(other ID[T]) bool {
	return id.ComponentID.Less(other.ComponentID)
}

// SortIDs sorts a slice of IDs of any type.
func SortIDs[T KindMarker](ids []ID[T]) []ID[T] {
	return copyAndSort(ids, func(a, b ID[T]) bool {
		return a.Less(b)
	})
}

// AsComponentID returns the underlying ComponentID.
// This is useful when you need to work with IDs in a type-erased context.
func (id ID[T]) AsComponentID() ComponentID {
	return id.ComponentID
}

// Conversion helpers between old and new ID systems.
// These enable incremental migration from the old ID types to the new unified system.

// ConvertL2BatcherID converts an old L2BatcherID to the new system.
func ConvertL2BatcherID(old L2BatcherID) L2BatcherID2 {
	return NewL2BatcherID2(old.Key(), old.ChainID())
}

// ConvertL2ELNodeID converts an old L2ELNodeID to the new system.
func ConvertL2ELNodeID(old L2ELNodeID) L2ELNodeID2 {
	return NewL2ELNodeID2(old.Key(), old.ChainID())
}

// ConvertL2CLNodeID converts an old L2CLNodeID to the new system.
func ConvertL2CLNodeID(old L2CLNodeID) L2CLNodeID2 {
	return NewL2CLNodeID2(old.Key(), old.ChainID())
}

// ConvertL1ELNodeID converts an old L1ELNodeID to the new system.
func ConvertL1ELNodeID(old L1ELNodeID) L1ELNodeID2 {
	return NewL1ELNodeID2(old.Key(), old.ChainID())
}

// ConvertL1CLNodeID converts an old L1CLNodeID to the new system.
func ConvertL1CLNodeID(old L1CLNodeID) L1CLNodeID2 {
	return NewL1CLNodeID2(old.Key(), old.ChainID())
}

// ConvertL1NetworkID converts an old L1NetworkID to the new system.
func ConvertL1NetworkID(old L1NetworkID) L1NetworkID2 {
	return NewL1NetworkID2(old.ChainID())
}

// ConvertL2NetworkID converts an old L2NetworkID to the new system.
func ConvertL2NetworkID(old L2NetworkID) L2NetworkID2 {
	return NewL2NetworkID2(old.ChainID())
}

// ConvertL2ProposerID converts an old L2ProposerID to the new system.
func ConvertL2ProposerID(old L2ProposerID) L2ProposerID2 {
	return NewL2ProposerID2(old.Key(), old.ChainID())
}

// ConvertL2ChallengerID converts an old L2ChallengerID to the new system.
func ConvertL2ChallengerID(old L2ChallengerID) L2ChallengerID2 {
	return NewL2ChallengerID2(old.Key(), old.ChainID())
}

// ConvertRollupBoostNodeID converts an old RollupBoostNodeID to the new system.
func ConvertRollupBoostNodeID(old RollupBoostNodeID) RollupBoostNodeID2 {
	return NewRollupBoostNodeID2(old.Key(), old.ChainID())
}

// ConvertOPRBuilderNodeID converts an old OPRBuilderNodeID to the new system.
func ConvertOPRBuilderNodeID(old OPRBuilderNodeID) OPRBuilderNodeID2 {
	return NewOPRBuilderNodeID2(old.Key(), old.ChainID())
}

// ConvertFaucetID converts an old FaucetID to the new system.
func ConvertFaucetID(old FaucetID) FaucetID2 {
	return NewFaucetID2(old.Key(), old.ChainID())
}

// ConvertSyncTesterID converts an old SyncTesterID to the new system.
func ConvertSyncTesterID(old SyncTesterID) SyncTesterID2 {
	return NewSyncTesterID2(old.Key(), old.ChainID())
}

// ConvertSupervisorID converts an old SupervisorID to the new system.
func ConvertSupervisorID(old SupervisorID) SupervisorID2 {
	return NewSupervisorID2(string(old))
}

// ConvertConductorID converts an old ConductorID to the new system.
func ConvertConductorID(old ConductorID) ConductorID2 {
	return NewConductorID2(string(old))
}

// ConvertClusterID converts an old ClusterID to the new system.
func ConvertClusterID(old ClusterID) ClusterID2 {
	return NewClusterID2(string(old))
}

// ConvertSuperchainID converts an old SuperchainID to the new system.
func ConvertSuperchainID(old SuperchainID) SuperchainID2 {
	return NewSuperchainID2(string(old))
}

// ConvertTestSequencerID converts an old TestSequencerID to the new system.
func ConvertTestSequencerID(old TestSequencerID) TestSequencerID2 {
	return NewTestSequencerID2(string(old))
}

// ConvertFlashblocksClientID converts an old FlashblocksWSClientID to the new system.
func ConvertFlashblocksClientID(old FlashblocksWSClientID) FlashblocksClientID2 {
	return NewFlashblocksClientID2(idWithChain(old).key, old.ChainID())
}
