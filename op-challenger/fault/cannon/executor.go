package cannon

import (
	"fmt"

	"github.com/ethereum/go-ethereum/log"
)

type executor struct {
	logger log.Logger
}

func newExecutor(logger log.Logger) Executor {
	return &executor{
		logger: logger,
	}
}

func (e *executor) GenerateProof(dir string, i uint64) error {
	return fmt.Errorf("please execute cannon with --proof-at %v --proof-fmt %v/%v/%%d.json", i, dir, proofsDir)
}
