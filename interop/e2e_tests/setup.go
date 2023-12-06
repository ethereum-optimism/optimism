package e2e_tests

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/indexer/node"
	"github.com/ethereum-optimism/optimism/interop"
	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

type E2ETestSuite struct {
	t *testing.T

	OpCfg *op_e2e.SystemConfig

	ChainIdA uint64
	OpSysA   *op_e2e.System
	PostieA  *interop.Postie

	ChainIdB uint64
	OpSysB   *op_e2e.System
	PostieB  *interop.Postie
}

func createE2ETestSuite(t *testing.T) E2ETestSuite {
	m := metrics.NewRegistry()

	// Rollup System Configuration. Unless specified,
	// omit logs emitted by the various components. Maybe
	// we can eventually dump these logs to a temp file
	log.Root().SetHandler(log.DiscardHandler())
	opCfg := op_e2e.DefaultSystemConfig(t)
	if len(os.Getenv("ENABLE_ROLLUP_LOGS")) == 0 {
		t.Log("set env 'ENABLE_ROLLUP_LOGS' to show rollup logs")
		for name, logger := range opCfg.Loggers {
			t.Logf("discarding logs for %s", name)
			logger.SetHandler(log.DiscardHandler())
		}
	}

	// NOTE: These two L2s will not settle on the same L1 local network which
	// is fine for now because this prototype does not care about L1 state and
	// L1 liquidity is still seperate per L2 chain.
	interopAtGenesis := hexutil.Uint64(0)
	opCfg.DeployConfig.L2GenesisInteropTimeOffset = &interopAtGenesis
	opCfg.DeployConfig.SuperchainPostie = &opCfg.Secrets.Addresses().Alice

	opCfg.DeployConfig.L2ChainID = 901
	opSysA, err := opCfg.Start(t)
	require.NoError(t, err)

	opCfg.DeployConfig.L2ChainID = 902
	opSysB, err := opCfg.Start(t)
	require.NoError(t, err)

	postieCfgA := interop.PostieConfig{
		Postie:           opCfg.Secrets.Alice,
		DestinationChain: opSysA.Clients["sequencer"],
		ConnectedChains:  []node.EthClient{node.FromRPCClient(opSysB.RawClients["sequencer"], node.NewMetrics(m, "b"))},
		UpdateInterval:   6 * time.Duration(opCfg.DeployConfig.L2BlockTime) * time.Second, // every ~6 blocks
	}
	postieCfgB := interop.PostieConfig{
		Postie:           opCfg.Secrets.Alice,
		DestinationChain: opSysB.Clients["sequencer"],
		ConnectedChains:  []node.EthClient{node.FromRPCClient(opSysA.RawClients["sequencer"], node.NewMetrics(m, "a"))},
		UpdateInterval:   6 * time.Duration(opCfg.DeployConfig.L2BlockTime) * time.Second, // every ~6 blocks
	}

	log := testlog.Logger(t, log.LvlInfo).New("role", "postie")

	postieA, err := interop.NewPostie(log.New("chain", "a"), postieCfgA)
	require.NoError(t, err)

	postieB, err := interop.NewPostie(log.New("chain", "b"), postieCfgB)
	require.NoError(t, err)

	require.NoError(t, postieA.Start(context.Background()))
	t.Cleanup(func() { require.NoError(t, postieA.Stop(context.Background())) })

	require.NoError(t, postieB.Start(context.Background()))
	t.Cleanup(func() { require.NoError(t, postieB.Stop(context.Background())) })

	return E2ETestSuite{
		t:     t,
		OpCfg: &opCfg,

		ChainIdA: 901,
		OpSysA:   opSysA,
		PostieA:  postieA,

		ChainIdB: 902,
		OpSysB:   opSysB,
		PostieB:  postieB,
	}
}
