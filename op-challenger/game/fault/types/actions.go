package types

import "github.com/ethereum/go-ethereum/common"

type ActionType string

func (a ActionType) String() string {
	return string(a)
}

const (
	ActionTypeMove ActionType = "move"
	ActionTypeStep ActionType = "step"
)

type Action struct {
	Type      ActionType
	ParentIdx int
	IsAttack  bool

	// Moves
	Value common.Hash

	// Steps
	PreState   []byte
	ProofData  []byte
	OracleData *PreimageOracleData
}
