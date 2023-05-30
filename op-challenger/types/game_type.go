package types

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/enum"
)

// GameType is the type of dispute game
type GameType uint8

// DefaultGameType returns the default dispute game type.
func DefaultGameType() GameType {
	return AttestationDisputeGameType
}

// String returns the string value of a dispute game type.
func (g GameType) String() string {
	return DisputeGameTypes[g]
}

const (
	// AttestationDisputeGameType is the uint8 enum value for the attestation dispute game
	AttestationDisputeGameType GameType = iota
	// FaultDisputeGameType is the uint8 enum value for the fault dispute game
	FaultDisputeGameType
	// ValidityDisputeGameType is the uint8 enum value for the validity dispute game
	ValidityDisputeGameType
)

// DisputeGameTypes is a list of dispute game types.
var DisputeGameTypes = []string{"attestation", "fault", "validity"}

// Valid returns true if the game type is within the valid range.
func (g GameType) Valid() bool {
	return g >= AttestationDisputeGameType && g <= ValidityDisputeGameType
}

// DisputeGameType is a custom flag type for dispute game type.
type DisputeGameType struct {
	Enum     []enum.Stringered
	selected GameType
}

// NewDisputeGameType returns a new dispute game type.
func NewDisputeGameType() *DisputeGameType {
	return &DisputeGameType{
		Enum:     enum.StringeredList(DisputeGameTypes),
		selected: DefaultGameType(),
	}
}

// Set sets the dispute game type.
func (d *DisputeGameType) Set(value string) error {
	for i, enum := range d.Enum {
		if enum.String() == value {
			d.selected = GameType(i)
			return nil
		}
	}

	return fmt.Errorf("allowed values are %s", enum.EnumString(d.Enum))
}

// String returns the selected dispute game type.
func (d DisputeGameType) String() string {
	return d.selected.String()
}

// Type maps the [DisputeGameType] string value to a [GameType] enum value.
func (d DisputeGameType) Type() GameType {
	return d.selected
}
