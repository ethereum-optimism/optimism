package engine

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

func TestFinalizedHead_SuperAuthorityMetrics(t *testing.T) {
	t.Parallel()

	m := metrics.NewMetrics("supernode", prometheus.Labels{"virtual_node_chain_id": "901"})
	eng := &testutils.MockEngine{}
	t.Cleanup(func() { eng.AssertExpectations(t) })
	sa := &mockSuperAuthority{finalizedL2HeadSource: rollup.VerifierHeadVerified}
	ec := NewEngineController(context.Background(), eng, testlog.Logger(t, 0), m,
		&rollup.Config{BlockTime: 2}, &sync.Config{}, &testutils.MockL1Source{}, &testutils.MockEmitter{}, sa)
	local := eth.L2BlockRef{Hash: common.Hash{0xff}, Number: 100, Time: 200}
	ec.SetFinalizedHead(local)

	checkMetrics := func(ref eth.L2BlockRef) {
		t.Helper()
		compareRefMetrics(t, m.Registry(), fmt.Sprintf(`
# HELP op_node_supernode_refs_number Gauge representing the different L1/L2 reference block numbers
# TYPE op_node_supernode_refs_number gauge
op_node_supernode_refs_number{layer="l1_origin",type="l2_finalized",virtual_node_chain_id="901"} 0
op_node_supernode_refs_number{layer="l1_origin",type="l2_super_authority_finalized",virtual_node_chain_id="901"} %d
op_node_supernode_refs_number{layer="l2",type="l2_finalized",virtual_node_chain_id="901"} 100
op_node_supernode_refs_number{layer="l2",type="l2_super_authority_finalized",virtual_node_chain_id="901"} %d
# HELP op_node_supernode_refs_time Gauge representing the different L1/L2 reference block timestamps
# TYPE op_node_supernode_refs_time gauge
op_node_supernode_refs_time{layer="l2",type="l2_finalized",virtual_node_chain_id="901"} 200
op_node_supernode_refs_time{layer="l2",type="l2_super_authority_finalized",virtual_node_chain_id="901"} %d
# HELP op_node_supernode_refs_hash Gauge representing the different L1/L2 reference block hashes truncated to float values
# TYPE op_node_supernode_refs_hash gauge
op_node_supernode_refs_hash{layer="l1_origin",type="l2_finalized",virtual_node_chain_id="901"} 0
op_node_supernode_refs_hash{layer="l1_origin",type="l2_super_authority_finalized",virtual_node_chain_id="901"} 0
op_node_supernode_refs_hash{layer="l2",type="l2_finalized",virtual_node_chain_id="901"} 255
op_node_supernode_refs_hash{layer="l2",type="l2_super_authority_finalized",virtual_node_chain_id="901"} %d
`, ref.L1Origin.Number, ref.Number, ref.Time, ref.Hash[0]), "op_node_supernode_refs_number", "op_node_supernode_refs_time", "op_node_supernode_refs_hash")
		require.Equal(t, local, ec.localFinalizedHead)
		require.Equal(t, local, ec.deprecatedFinalizedHead)
		require.Equal(t, ref, ec.superAuthorityFinalizedHead)
	}

	first := eth.L2BlockRef{Hash: common.Hash{0xaa}, Number: 50, Time: 100, L1Origin: eth.BlockID{Number: 10}}
	sa.finalizedL2Head = first.ID()
	eng.ExpectL2BlockRefByHash(first.Hash, first, nil)
	require.Equal(t, first, ec.FinalizedHead())
	checkMetrics(first)

	advanced := eth.L2BlockRef{Hash: common.Hash{0xbb}, Number: 60, Time: 120, L1Origin: eth.BlockID{Number: 11}}
	sa.finalizedL2Head = advanced.ID()
	eng.ExpectL2BlockRefByHash(advanced.Hash, advanced, nil)
	require.Equal(t, advanced, ec.FinalizedHead())
	checkMetrics(advanced)

	eng.ExpectL2BlockRefByHash(advanced.Hash, advanced, nil)
	require.Equal(t, advanced, ec.FinalizedHead())
	checkMetrics(advanced)

	sa.holdPreviousFinalized = true
	require.Equal(t, advanced, ec.FinalizedHead())
	checkMetrics(advanced)

	sa.holdPreviousFinalized = false
	sa.finalizedL2Head = eth.BlockID{Hash: common.Hash{0xcc}, Number: 70}
	eng.ExpectL2BlockRefByHash(sa.finalizedL2Head.Hash, eth.L2BlockRef{}, errors.New("EL unavailable"))
	require.Equal(t, advanced, ec.FinalizedHead())
	checkMetrics(advanced)

	sa.finalizedL2Head = first.ID()
	eng.ExpectL2BlockRefByHash(first.Hash, first, nil)
	require.Equal(t, advanced, ec.FinalizedHead())
	checkMetrics(advanced)

	sa.finalizedL2HeadSource = rollup.VerifierHeadAnchor
	sa.finalizedTimestamp = 160
	anchor := eth.L2BlockRef{Hash: common.Hash{0xdd}, Number: 80, Time: 160, L1Origin: eth.BlockID{Number: 12}}
	eng.ExpectL2BlockRefByNumber(anchor.Number, anchor, nil)
	require.Equal(t, anchor, ec.FinalizedHead())
	checkMetrics(anchor)
}

func TestFinalizedHead_NoSuperAuthorityMetricBeforeResolution(t *testing.T) {
	t.Parallel()

	for _, mode := range []string{"standalone", "pre-activation", "hold-previous", "lookup-failure", "ahead-of-local"} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()
			m := metrics.NewMetrics("test", nil)
			eng := &testutils.MockEngine{}
			t.Cleanup(func() { eng.AssertExpectations(t) })
			local := eth.L2BlockRef{Hash: common.Hash{0xff}, Number: 100, Time: 200}
			var authority rollup.SuperAuthority
			want := local
			if mode != "standalone" {
				sa := &mockSuperAuthority{finalizedL2HeadSource: rollup.VerifierHeadPreActivation}
				authority = sa
				switch mode {
				case "hold-previous":
					sa.holdPreviousFinalized = true
					want = eth.L2BlockRef{}
				case "lookup-failure":
					sa.finalizedL2HeadSource = rollup.VerifierHeadVerified
					sa.finalizedL2Head = eth.BlockID{Hash: common.Hash{0xaa}, Number: 50}
					eng.ExpectL2BlockRefByHash(sa.finalizedL2Head.Hash, eth.L2BlockRef{}, errors.New("EL unavailable"))
					want = eth.L2BlockRef{}
				case "ahead-of-local":
					sa.finalizedL2HeadSource = rollup.VerifierHeadVerified
					sa.finalizedL2Head = eth.BlockID{Hash: common.Hash{0xaa}, Number: 101}
				}
			}
			ec := NewEngineController(context.Background(), eng, testlog.Logger(t, 0), m,
				&rollup.Config{BlockTime: 2}, &sync.Config{}, &testutils.MockL1Source{}, &testutils.MockEmitter{}, authority)
			ec.SetFinalizedHead(local)
			require.Equal(t, want, ec.FinalizedHead())
			require.Equal(t, local, ec.localFinalizedHead)
			require.Equal(t, local, ec.deprecatedFinalizedHead)
			require.Empty(t, ec.superAuthorityFinalizedHead)
			compareRefMetrics(t, m.Registry(), `
# HELP op_node_test_refs_number Gauge representing the different L1/L2 reference block numbers
# TYPE op_node_test_refs_number gauge
op_node_test_refs_number{layer="l1_origin",type="l2_finalized"} 0
op_node_test_refs_number{layer="l2",type="l2_finalized"} 100
`, "op_node_test_refs_number")
		})
	}
}

func compareRefMetrics(t *testing.T, registry *prometheus.Registry, expected string, names ...string) {
	t.Helper()
	families, err := registry.Gather()
	require.NoError(t, err)
	var actual strings.Builder
	for _, name := range names {
		for _, family := range families {
			if family.GetName() == name {
				_, err := expfmt.MetricFamilyToText(&actual, family)
				require.NoError(t, err)
			}
		}
	}
	require.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(actual.String()))
}
