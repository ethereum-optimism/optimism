package stack

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/holiman/uint256"
)

// ComponentKind identifies the type of component.
// This is used in serialization to make each ID unique and type-safe.
type ComponentKind string

var _ slog.LogValuer = (*ComponentKind)(nil)

// ChainIDProvider presents a type that provides a relevant ChainID.
type ChainIDProvider interface {
	ChainID() eth.ChainID
}

// KindProvider presents a type that provides a relevant ComponentKind. E.g. KindL2Batcher.
type KindProvider interface {
	Kind() ComponentKind
}

// Keyed presents a type that provides a relevant string key. E.g. a named superchain.
type Keyed interface {
	Key() string
}

const maxIDLength = 100

var errInvalidID = errors.New("invalid ID")

func (k ComponentKind) LogValue() slog.Value {
	return slog.StringValue(string(k))
}

func (k ComponentKind) String() string {
	return string(k)
}

func (k ComponentKind) MarshalText() ([]byte, error) {
	return []byte(k), nil
}

func (k *ComponentKind) UnmarshalText(data []byte) error {
	*k = ComponentKind(data)
	return nil
}

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

// idMatcher wraps ComponentID to implement Matcher[E] for any component type.
type idMatcher[E Identifiable] struct {
	id ComponentID
}

func (m idMatcher[E]) Match(elems []E) []E {
	for i, elem := range elems {
		if elem.ID() == m.id {
			return elems[i : i+1]
		}
	}
	return nil
}

func (m idMatcher[E]) String() string {
	return m.id.String()
}

// ID returns the ComponentID this matcher wraps.
// This is used by shim.findMatch for direct registry lookup.
func (m idMatcher[E]) ID() ComponentID {
	return m.id
}

// ByID creates a matcher for a specific ComponentID.
// This allows using a ComponentID as a matcher for any component type.
func ByID[E Identifiable](id ComponentID) Matcher[E] {
	return idMatcher[E]{id: id}
}

// Convenience constructors for each component kind.

func NewL1ELNodeID(key string, chainID eth.ChainID) ComponentID {
	return NewComponentID(KindL1ELNode, key, chainID)
}

func NewL1CLNodeID(key string, chainID eth.ChainID) ComponentID {
	return NewComponentID(KindL1CLNode, key, chainID)
}

func NewL1NetworkID(chainID eth.ChainID) ComponentID {
	return NewComponentIDChainOnly(KindL1Network, chainID)
}

func NewL2ELNodeID(key string, chainID eth.ChainID) ComponentID {
	return NewComponentID(KindL2ELNode, key, chainID)
}

func NewL2CLNodeID(key string, chainID eth.ChainID) ComponentID {
	return NewComponentID(KindL2CLNode, key, chainID)
}

func NewL2NetworkID(chainID eth.ChainID) ComponentID {
	return NewComponentIDChainOnly(KindL2Network, chainID)
}

func NewL2BatcherID(key string, chainID eth.ChainID) ComponentID {
	return NewComponentID(KindL2Batcher, key, chainID)
}

func NewL2ProposerID(key string, chainID eth.ChainID) ComponentID {
	return NewComponentID(KindL2Proposer, key, chainID)
}

func NewL2ChallengerID(key string, chainID eth.ChainID) ComponentID {
	return NewComponentID(KindL2Challenger, key, chainID)
}

func NewRollupBoostNodeID(key string, chainID eth.ChainID) ComponentID {
	return NewComponentID(KindRollupBoostNode, key, chainID)
}

func NewOPRBuilderNodeID(key string, chainID eth.ChainID) ComponentID {
	return NewComponentID(KindOPRBuilderNode, key, chainID)
}

func NewFaucetID(key string, chainID eth.ChainID) ComponentID {
	return NewComponentID(KindFaucet, key, chainID)
}

func NewSyncTesterID(key string, chainID eth.ChainID) ComponentID {
	return NewComponentID(KindSyncTester, key, chainID)
}

func NewSupervisorID(key string) ComponentID {
	return NewComponentIDKeyOnly(KindSupervisor, key)
}

func NewConductorID(key string) ComponentID {
	return NewComponentIDKeyOnly(KindConductor, key)
}

func NewClusterID(key string) ComponentID {
	return NewComponentIDKeyOnly(KindCluster, key)
}

func NewSuperchainID(key string) ComponentID {
	return NewComponentIDKeyOnly(KindSuperchain, key)
}

func NewTestSequencerID(key string) ComponentID {
	return NewComponentIDKeyOnly(KindTestSequencer, key)
}

func NewFlashblocksWSClientID(key string, chainID eth.ChainID) ComponentID {
	return NewComponentID(KindFlashblocksClient, key, chainID)
}
