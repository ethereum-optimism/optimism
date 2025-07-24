package noopsigner

import (
	"github.com/HashKeyChain/verse/op-service/signer"
	"github.com/HashKeyChain/verse/op-test-sequencer/sequencer/backend/work"
)

type NoopSignedBlock struct {
	work.Block
}

func (s *NoopSignedBlock) VerifySignature(_ signer.Authenticator) error {
	return nil
}
