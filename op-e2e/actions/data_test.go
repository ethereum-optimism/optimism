package actions

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
)

type l1l2Data struct {
	t   Testing
	log log.Logger

	l2Seq    *L2Sequencer
	l2Eng    *L2Engine
	l2Cl     *ethclient.Client
	l2Signer types.Signer

	l1Miner  *L1Miner
	l1Cl     *ethclient.Client
	l1Signer types.Signer
}

type MalformedDataTestCase struct {
	name     string
	submit   func(d *l1l2Data) (expectedL2SafeHead common.Hash)
}

func TestMalformedData(t *testing.T) {

	batchSubmitSome := func(d *l1l2Data) (expectedL2SafeHead common.Hash) {
		// TODO: sequence some blocks
		// TODO: create batcher, submit all
	}

	batchSubmitRaw := func() (expectedL2SafeHead common.Hash) {

	}

	testCases := []MalformedDataTestCase{
		{name: "success", submit: batchSubmitSome},
		{name: "unknown derivation version"},
		{name: "invalid frame"},
		{name: "invalid compression"},
		{name: "invalid batch RLP parent hash"},
		{name: "invalid batch RLP epoch num"},
		{name: "invalid batch RLP timestamp"},
		{name: "invalid batch RLP transactions"},
		{name: "batch with empty tx entry"},
		{name: "batch RLP missing entry"},
		{name: "batch RLP extra entry"},
		{name: "batch RLP content too large"},
		{name: "unknown batch type"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.Run)
	}
}

func (tc *MalformedDataTestCase) Run(gt *testing.T) {
	t := NewDefaultTesting(gt)
	dp := e2eutils.MakeDeployParams(t, defaultRollupTestParams)
	sd := e2eutils.Setup(t, dp, defaultAlloc)
	log := testlog.Logger(t, log.LvlError)
	miner, seqEngine, sequencer := setupSequencerTest(t, sd, log)
	miner.ActL1SetFeeRecipient(common.Address{'A'})
	sequencer.ActL2PipelineFull(t)
	_, verifier := setupVerifier(t, sd, log, miner.L1Client(t, sd.RollupCfg))
	d := &l1l2Data{
		t:        t,
		log:      log,
		l2Seq:    sequencer,
		l2Eng:    seqEngine,
		l2Cl:     seqEngine.EthClient(),
		l2Signer: types.LatestSigner(sd.L2Cfg.Config),
		l1Miner:  miner,
		l1Cl:     miner.EthClient(),
		l1Signer: types.LatestSigner(sd.L1Cfg.Config),
	}

	// submit the batches to L1, and determine what to expect
	expectedL2SafeHead := tc.submit(d)

	// check that sequencer syncs
	sequencer.ActL1HeadSignal(t)
	sequencer.ActL2PipelineFull(t)
	require.Equal(t, expectedL2SafeHead, sequencer.L2Safe().Hash, "sequencer labels safe head correctly")

	// check that verifier syncs
	verifier.ActL1HeadSignal(t)
	verifier.ActL2PipelineFull(t)

	require.Equal(t, expectedL2SafeHead, verifier.L2Safe().Hash, "verifier syncs to expected safe head")
}
