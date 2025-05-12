package presets

import (
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
)

type Backend string

const (
	SysGo  Backend = "sysgo"
	SysExt Backend = "sysext"
)

var globalBackend Backend = SysGo

func detectBackend(logger devtest.Logger) {
	kind, ok := os.LookupEnv("DEVSTACK_ORCHESTRATOR")
	if !ok {
		logger.Warn("Selecting sysgo as default backend")
		globalBackend = SysGo
		return
	}
	backend := Backend(kind)
	switch backend {
	case SysGo:
		globalBackend = SysGo
	case SysExt:
		globalBackend = SysExt
	default:
		panic(fmt.Sprintf("Unknown backend: %s", backend))
	}
	logger.Info("Detected", "backend", backend)
}
