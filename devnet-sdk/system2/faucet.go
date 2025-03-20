package system2

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

type FaucetConfig struct {
	CommonConfig
	ID FaucetID
}

type presetFaucet struct {
	commonImpl
	id FaucetID
}

var _ Faucet = (*presetFaucet)(nil)

func NewFaucet(cfg FaucetConfig) Faucet {
	return &presetFaucet{
		id:         cfg.ID,
		commonImpl: newCommon(cfg.CommonConfig),
	}
}

func (p *presetFaucet) ID() FaucetID {
	return p.id
}

func (p *presetFaucet) NewUser() User {
	p.require().Fail("not implemented")
	return nil
}
