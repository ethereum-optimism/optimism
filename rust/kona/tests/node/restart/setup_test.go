package node_restart

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log/logcli"
	node_utils "github.com/ethereum-optimism/optimism/rust/kona/tests/node/utils"
)

var sharedRestartRuntime *sysgo.MixedSingleChainRuntime

type packageInitResult struct {
	code int
}

func TestMain(m *testing.M) {
	// go test -list executes TestMain; when only listing, print the names and
	// exit before booting the shared devstack runtime below.
	for _, arg := range os.Args[1:] {
		if arg == "-test.list" || strings.HasPrefix(arg, "-test.list=") {
			os.Exit(m.Run())
		}
	}

	logger := logcli.NewLogger(os.Stderr, logcli.DefaultCLIConfig())
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
			KonaSequencerNodesWithReth: 1,
			KonaNodesWithReth:          1,
		}, sysgo.WithL2BlockTimes(map[eth.ChainID]uint64{
			sysgo.DefaultL2AID: 1,
		}))
		code = m.Run()
	}()

	pkg.Close()
	os.Exit(code)
}
