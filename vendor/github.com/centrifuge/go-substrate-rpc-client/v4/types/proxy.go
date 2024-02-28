package types

import "github.com/centrifuge/go-substrate-rpc-client/v4/scale"

type ProxyDefinition struct {
	Delegate  AccountID
	ProxyType U8
	Delay     U32
}

func (p *ProxyDefinition) Decode(decoder scale.Decoder) error {
	if err := decoder.Decode(&p.Delegate); err != nil {
		return err
	}

	if err := decoder.Decode(&p.ProxyType); err != nil {
		return err
	}

	return decoder.Decode(&p.Delay)
}

func (p ProxyDefinition) Encode(encoder scale.Encoder) error {
	if err := encoder.Encode(p.Delegate); err != nil {
		return err
	}

	if err := encoder.Encode(p.ProxyType); err != nil {
		return err
	}

	return encoder.Encode(p.Delay)
}

type ProxyStorageEntry struct {
	ProxyDefinitions []ProxyDefinition
	Balance          U128
}

func (p *ProxyStorageEntry) Decode(decoder scale.Decoder) error {
	if err := decoder.Decode(&p.ProxyDefinitions); err != nil {
		return err
	}

	return decoder.Decode(&p.Balance)
}

func (p ProxyStorageEntry) Encode(encoder scale.Encoder) error {
	if err := encoder.Encode(p.ProxyDefinitions); err != nil {
		return err
	}

	return encoder.Encode(p.Balance)
}
