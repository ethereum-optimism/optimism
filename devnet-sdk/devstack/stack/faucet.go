package stack

// FaucetID identifies a Faucet by name and chainID, is type-safe, and can be value-copied and used as map key.
type FaucetID idWithChain

const FaucetKind Kind = "Faucet"

func (id FaucetID) String() string {
	return idWithChain(id).string(FaucetKind)
}

func (id FaucetID) MarshalText() ([]byte, error) {
	return idWithChain(id).marshalText(FaucetKind)
}

func (id *FaucetID) UnmarshalText(data []byte) error {
	return (*idWithChain)(id).unmarshalText(FaucetKind, data)
}

func SortFaucetIDs(ids []FaucetID) []FaucetID {
	return copyAndSort(ids, func(a, b FaucetID) bool {
		return lessIDWithChain(idWithChain(a), idWithChain(b))
	})
}

type Faucet interface {
	Common
	ID() FaucetID
	// NewUser creates a new pre-funded user account
	NewUser() User
}
