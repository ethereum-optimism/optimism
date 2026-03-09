package node_restart

import (
	"context"
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	node_utils "github.com/ethereum-optimism/optimism/rust/kona/tests/node/utils"
)

var sharedRestartRuntime *sysgo.MixedSingleChainRuntime

type packageInitResult struct {
	code int
}

func TestMain(m *testing.M) {
	logger := oplog.NewLogger(os.Stderr, oplog.DefaultCLIConfig())
	pkg := devtest.NewP(context.Background(), logger, func(_ bool) {
		panic(packageInitResult{code: 1})
	}, func() {
		panic(packageInitResult{code: 0})
	})

	devtest.RootContext = pkg.Ctx()

	code := 1
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				if result, ok := recovered.(packageInitResult); ok {
					code = result.code
					return
				}
				panic(recovered)
			}
		}()

		sharedRestartRuntime = node_utils.NewSharedMixedOpKonaRuntimeForConfig(pkg, node_utils.L2NodeConfig{
			KonaSequencerNodesWithGeth: 1,
			KonaNodesWithGeth:          1,
		})
		code = m.Run()
	}()

	pkg.Close()
	os.Exit(code)
}
