package noopsigner

import (
	"github.com/ethereum-optimism/optimism/op-service/signer"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work"
)

type NoopSignedBlock struct {
	work.Block
}

func (s *NoopSignedBlock) VerifySignature(_ signer.Authenticator) error {
	return nil
}
