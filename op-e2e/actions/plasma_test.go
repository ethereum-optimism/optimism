package actions

import (
	"context"
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/node/safedb"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	plasma "github.com/ethereum-optimism/optimism/op-plasma"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// L2PlasmaDA is a test harness for manipulating plasma DA state.
type L2PlasmaDA struct {
	log        log.Logger
	storage    *plasma.DAErrFaker
	daMgr      *plasma.DA
	plasmaCfg  plasma.Config
	contract   *bindings.DataAvailabilityChallenge
	batcher    *L2Batcher
	sequencer  *L2Sequencer
	engine     *L2Engine
	engCl      *sources.EngineClient
	sd         *e2eutils.SetupData
	dp         *e2eutils.DeployParams
	miner      *L1Miner
	alice      *CrossLayerUser
	lastComm   []byte
	lastCommBn uint64
}

func NewL2AltDA(log log.Logger, p *e2eutils.TestParams, t Testing) *L2PlasmaDA {
	dp := e2eutils.MakeDeployParams(t, p)
	sd := e2eutils.Setup(t, dp, defaultAlloc)

	require.True(t, sd.RollupCfg.UsePlasma)

	miner := NewL1Miner(t, log, sd.L1Cfg)
	l1Client := miner.EthClient()

	jwtPath := e2eutils.WriteDefaultJWT(t)
	engine := NewL2Engine(t, log, sd.L2Cfg, sd.RollupCfg.Genesis.L1, jwtPath)
	engCl := engine.EngineClient(t, sd.RollupCfg)

	storage := &plasma.DAErrFaker{Client: plasma.NewMockDAClient(log)}

	l1F, err := sources.NewL1Client(miner.RPCClient(), log, nil, sources.L1ClientDefaultConfig(sd.RollupCfg, false, sources.RPCKindBasic))
	require.NoError(t, err)

	plasmaCfg, err := sd.RollupCfg.PlasmaConfig()
	require.NoError(t, err)

	daMgr := plasma.NewPlasmaDAWithStorage(log, plasmaCfg, storage, l1F, &plasma.NoopMetrics{})

	sequencer := NewL2Sequencer(t, log, l1F, nil, daMgr, engCl, sd.RollupCfg, 0)
	miner.ActL1SetFeeRecipient(common.Address{'A'})
	sequencer.ActL2PipelineFull(t)

	daMgr.OnFinalizedHeadSignal(func(ctx context.Context, ref eth.L1BlockRef) {
		sequencer.derivation.Finalize(ref)
	})

	batcher := NewL2Batcher(log, sd.RollupCfg, PlasmaBatcherCfg(dp, storage), sequencer.RollupClient(), l1Client, engine.EthClient(), engCl)

	addresses := e2eutils.CollectAddresses(sd, dp)
	cl := engine.EthClient()
	l2UserEnv := &BasicUserEnv[*L2Bindings]{
		EthCl:          cl,
		Signer:         types.LatestSigner(sd.L2Cfg.Config),
		AddressCorpora: addresses,
		Bindings:       NewL2Bindings(t, cl, engine.GethClient()),
	}
	alice := NewCrossLayerUser(log, dp.Secrets.Alice, rand.New(rand.NewSource(0xa57b)))
	alice.L2.SetUserEnv(l2UserEnv)

	contract, err := bindings.NewDataAvailabilityChallenge(sd.RollupCfg.DAChallengeAddress, l1Client)
	require.NoError(t, err)

	challengeWindow, err := contract.ChallengeWindow(nil)
	require.NoError(t, err)
	require.Equal(t, plasmaCfg.ChallengeWindow, challengeWindow.Uint64())

	resolveWindow, err := contract.ResolveWindow(nil)
	require.NoError(t, err)
	require.Equal(t, plasmaCfg.ResolveWindow, resolveWindow.Uint64())

	return &L2PlasmaDA{
		log:       log,
		storage:   storage,
		daMgr:     daMgr,
		plasmaCfg: plasmaCfg,
		contract:  contract,
		batcher:   batcher,
		sequencer: sequencer,
		engine:    engine,
		engCl:     engCl,
		sd:        sd,
		dp:        dp,
		miner:     miner,
		alice:     alice,
	}
}

func (a *L2PlasmaDA) StorageClient() *plasma.DAErrFaker {
	return a.storage
}

func (a *L2PlasmaDA) NewVerifier(t Testing) *L2Verifier {
	jwtPath := e2eutils.WriteDefaultJWT(t)
	engine := NewL2Engine(t, a.log, a.sd.L2Cfg, a.sd.RollupCfg.Genesis.L1, jwtPath)
	engCl := engine.EngineClient(t, a.sd.RollupCfg)
	l1F, err := sources.NewL1Client(a.miner.RPCClient(), a.log, nil, sources.L1ClientDefaultConfig(a.sd.RollupCfg, false, sources.RPCKindBasic))
	require.NoError(t, err)

	daMgr := plasma.NewPlasmaDAWithStorage(a.log, a.plasmaCfg, a.storage, l1F, &plasma.NoopMetrics{})

	verifier := NewL2Verifier(t, a.log, l1F, nil, daMgr, engCl, a.sd.RollupCfg, &sync.Config{}, safedb.Disabled)
	daMgr.OnFinalizedHeadSignal(func(ctx context.Context, ref eth.L1BlockRef) {
		verifier.derivation.Finalize(ref)
	})

	return verifier
}

func (a *L2PlasmaDA) ActSequencerIncludeTx(t Testing) {
	a.alice.L2.ActResetTxOpts(t)
	a.alice.L2.ActSetTxToAddr(&a.dp.Addresses.Bob)(t)
	a.alice.L2.ActMakeTx(t)

	a.sequencer.ActL2PipelineFull(t)

	a.sequencer.ActL2StartBlock(t)
	a.engine.ActL2IncludeTx(a.alice.Address())(t)
	a.sequencer.ActL2EndBlock(t)
}

func (a *L2PlasmaDA) ActNewL2Tx(t Testing) {
	a.ActSequencerIncludeTx(t)

	a.batcher.ActL2BatchBuffer(t)
	a.batcher.ActL2ChannelClose(t)
	a.batcher.ActL2BatchSubmit(t, func(tx *types.DynamicFeeTx) {
		a.lastComm = tx.Data
	})

	a.miner.ActL1StartBlock(3)(t)
	a.miner.ActL1IncludeTx(a.dp.Addresses.Batcher)(t)
	a.miner.ActL1EndBlock(t)

	a.lastCommBn = a.miner.l1Chain.CurrentBlock().Number.Uint64()
}

func (a *L2PlasmaDA) ActDeleteLastInput(t Testing) {
	a.storage.Client.DeleteData(a.lastComm)
}

func (a *L2PlasmaDA) ActChallengeLastInput(t Testing) {
	a.ActChallengeInput(t, a.lastComm, a.lastCommBn)

	a.log.Info("challenged last input", "block", a.lastCommBn)
}

func (a *L2PlasmaDA) ActChallengeInput(t Testing, comm []byte, bn uint64) {
	bondValue, err := a.contract.BondSize(&bind.CallOpts{})
	require.NoError(t, err)

	txOpts, err := bind.NewKeyedTransactorWithChainID(a.dp.Secrets.Alice, a.sd.L1Cfg.Config.ChainID)
	require.NoError(t, err)

	txOpts.Value = bondValue
	_, err = a.contract.Deposit(txOpts)
	require.NoError(t, err)

	a.miner.ActL1StartBlock(3)(t)
	a.miner.ActL1IncludeTx(a.alice.Address())(t)
	a.miner.ActL1EndBlock(t)

	txOpts, err = bind.NewKeyedTransactorWithChainID(a.dp.Secrets.Alice, a.sd.L1Cfg.Config.ChainID)
	require.NoError(t, err)

	_, err = a.contract.Challenge(txOpts, big.NewInt(int64(bn)), comm)
	require.NoError(t, err)

	a.miner.ActL1StartBlock(3)(t)
	a.miner.ActL1IncludeTx(a.alice.Address())(t)
	a.miner.ActL1EndBlock(t)
}

func (a *L2PlasmaDA) ActExpireLastInput(t Testing) {
	reorgWindow := a.plasmaCfg.ResolveWindow + a.plasmaCfg.ChallengeWindow
	for a.miner.l1Chain.CurrentBlock().Number.Uint64() <= a.lastCommBn+reorgWindow {
		a.miner.ActL1StartBlock(3)(t)
		a.miner.ActL1EndBlock(t)
	}
}

func (a *L2PlasmaDA) ActResolveLastChallenge(t Testing) {
	input, err := a.storage.GetInput(t.Ctx(), a.lastComm)
	require.NoError(t, err)

	txOpts, err := bind.NewKeyedTransactorWithChainID(a.dp.Secrets.Alice, a.sd.L1Cfg.Config.ChainID)

	_, err = a.contract.Resolve(txOpts, big.NewInt(int64(a.lastCommBn)), a.lastComm, input)
	require.NoError(t, err)

	a.miner.ActL1StartBlock(3)(t)
	a.miner.ActL1IncludeTx(a.alice.Address())(t)
	a.miner.ActL1EndBlock(t)
}

func (a *L2PlasmaDA) ActL1Blocks(t Testing, n uint64) {
	for i := uint64(0); i < n; i++ {
		a.miner.ActL1StartBlock(3)(t)
		a.miner.ActL1EndBlock(t)
	}
}

func (a *L2PlasmaDA) GetLastTxBlock(t Testing) *types.Block {
	rcpt, err := a.engine.EthClient().TransactionReceipt(t.Ctx(), a.alice.L2.lastTxHash)
	require.NoError(t, err)
	blk, err := a.engine.EthClient().BlockByHash(t.Ctx(), rcpt.BlockHash)
	require.NoError(t, err)
	return blk
}

// Commitment is challenged but never resolved, chain reorgs when challenge window expires.
func TestPlasma_ChallengeExpired(gt *testing.T) {
	t := NewDefaultTesting(gt)

	p := &e2eutils.TestParams{
		MaxSequencerDrift:   2,
		SequencerWindowSize: 4,
		ChannelTimeout:      4,
		L1BlockTime:         3,
	}

	log := testlog.Logger(t, log.LevelDebug)

	harness := NewL2AltDA(log, p, t)

	// generate enough initial l1 blocks to have a finalized head.
	harness.ActL1Blocks(t, 5)

	// Include a new l2 transaction, submitting an input commitment to the l1.
	harness.ActNewL2Tx(t)

	// Challenge the input commitment on the l1 challenge contract.
	harness.ActChallengeLastInput(t)

	blk := harness.GetLastTxBlock(t)

	// catch up the sequencer derivation pipeline with the new l1 blocks.
	harness.sequencer.ActL2PipelineFull(t)

	// create enough l1 blocks to expire the resolve window.
	harness.ActExpireLastInput(t)

	// catch up the sequencer derivation pipeline with the new l1 blocks.
	harness.sequencer.ActL2PipelineFull(t)

	// make sure that the finalized head was correctly updated on the engine.
	l2Finalized, err := harness.engCl.L2BlockRefByLabel(t.Ctx(), eth.Finalized)
	require.NoError(t, err)
	require.Equal(t, uint64(8), l2Finalized.Number)

	newBlk, err := harness.engine.EthClient().BlockByNumber(t.Ctx(), blk.Number())
	require.NoError(t, err)

	// reorg happened even though data was available
	require.NotEqual(t, blk.Hash(), newBlk.Hash())

	// now delete the data from the storage service so it is not available at all
	// to the verifier derivation pipeline.
	harness.ActDeleteLastInput(t)

	syncStatus := harness.sequencer.SyncStatus()

	// verifier is able to sync with expired missing data
	verifier := harness.NewVerifier(t)
	verifier.ActL2PipelineFull(t)

	verifSyncStatus := verifier.SyncStatus()

	require.Equal(t, syncStatus.FinalizedL2, verifSyncStatus.FinalizedL2)
}

// Commitment is challenged after sequencer derived the chain but data disappears. A verifier
// derivation pipeline stalls until the challenge is resolved and then resumes with data from the contract.
func TestPlasma_ChallengeResolved(gt *testing.T) {
	t := NewDefaultTesting(gt)
	p := &e2eutils.TestParams{
		MaxSequencerDrift:   2,
		SequencerWindowSize: 4,
		ChannelTimeout:      4,
		L1BlockTime:         3,
	}
	log := testlog.Logger(t, log.LvlDebug)
	harness := NewL2AltDA(log, p, t)

	// include a new l2 transaction, submitting an input commitment to the l1.
	harness.ActNewL2Tx(t)

	// generate 3 l1 blocks.
	harness.ActL1Blocks(t, 3)

	// challenge the input commitment for that l2 transaction on the l1 challenge contract.
	harness.ActChallengeLastInput(t)

	// catch up sequencer derivatio pipeline.
	// this syncs the latest event within the AltDA manager.
	harness.sequencer.ActL2PipelineFull(t)

	// resolve the challenge on the l1 challenge contract.
	harness.ActResolveLastChallenge(t)

	// catch up the sequencer derivation pipeline with the new l1 blocks.
	// this syncs the resolved status and input data within the AltDA manager.
	harness.sequencer.ActL2PipelineFull(t)

	// delete the data from the storage service so it is not available at all
	// to the verifier derivation pipeline.
	harness.ActDeleteLastInput(t)

	syncStatus := harness.sequencer.SyncStatus()

	// new verifier is able to sync and resolve the input from calldata
	verifier := harness.NewVerifier(t)
	verifier.ActL2PipelineFull(t)

	verifSyncStatus := verifier.SyncStatus()

	require.Equal(t, syncStatus.SafeL2, verifSyncStatus.SafeL2)
}

// DA storage service goes offline while sequencer keeps making blocks. When storage comes back online, it should be able to catch up.
func TestAltDA_StorageError(gt *testing.T) {
	t := NewDefaultTesting(gt)
	p := &e2eutils.TestParams{
		MaxSequencerDrift:   2,
		SequencerWindowSize: 4,
		ChannelTimeout:      4,
		L1BlockTime:         3,
	}
	log := testlog.Logger(t, log.LvlDebug)
	harness := NewL2AltDA(log, p, t)

	// include a new l2 transaction, submitting an input commitment to the l1.
	harness.ActNewL2Tx(t)

	txBlk := harness.GetLastTxBlock(t)

	// mock a storage client error when trying to get the pre-image.
	// this simulates the storage service going offline for example.
	harness.storage.ActGetPreImageFail()

	// try to derive the l2 chain from the submitted inputs commitments.
	// the storage call will fail the first time then succeed.
	harness.sequencer.ActL2PipelineFull(t)

	// sequencer derivation was able to sync to latest l1 origin
	syncStatus := harness.sequencer.SyncStatus()
	require.Equal(t, uint64(1), syncStatus.SafeL2.Number)
	require.Equal(t, txBlk.Hash(), syncStatus.SafeL2.Hash)
}

// Commitment is challenged but with a wrong block number.
func TestAltDA_ChallengeBadBlockNumber(gt *testing.T) {
	t := NewDefaultTesting(gt)
	p := &e2eutils.TestParams{
		MaxSequencerDrift:   2,
		SequencerWindowSize: 4,
		ChannelTimeout:      4,
		L1BlockTime:         3,
	}
	log := testlog.Logger(t, log.LvlDebug)
	harness := NewL2AltDA(log, p, t)

	// generate 3 blocks of l1 chain
	harness.ActL1Blocks(t, 3)

	// include a new transaction on l2
	harness.ActNewL2Tx(t)

	// move the l1 chain so the challenge window expires
	harness.ActExpireLastInput(t)

	// catch up derivation
	harness.sequencer.ActL2PipelineFull(t)

	// challenge the input but with a wrong block number
	// in the current challenge window
	harness.ActChallengeInput(t, harness.lastComm, 14)

	// catch up derivation
	harness.sequencer.ActL2PipelineFull(t)

	// da mgr should not have save the challenge
	found := harness.daMgr.State().IsTracking(harness.lastComm, 14)
	require.False(t, found)
}
