package privateinterop

// Acceptance tests of the sound proof profile, sp1-private-projection-v1 (spec-sound-profile §H WP5).
//
// Every test here runs the SP1 verifier in admission with mock envelopes (kind 0x02), which the
// projection config accepts only because mock_proofs is set, the test gate is compiled into these go
// test binaries, and the chain ID is a devstack chain. A mock envelope proves nothing. What these
// tests establish is the plumbing: the batcher asks the producer for an envelope over the exact
// public values admission will compute, admission in op-node recomputes all 21 words from its own
// derivation state and rejects any difference, and claim-follow advances private safety only
// through admitted spans. Real Groth16 proving is covered only by TestPrivateSoundProfileGroth16,
// which is skipped unless PRIVATE_INTEROP_SP1_PROVER selects a real prover.

import (
	"math/big"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/shared/rustbin"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-private-interop/codec"
	"github.com/ethereum-optimism/optimism/op-private-interop/projection"
	"github.com/ethereum-optimism/optimism/op-private-interop/render"
	"github.com/ethereum-optimism/optimism/op-private-interop/wire"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
)

// privateProjectionExecutor locates (or, under RUST_JIT_BUILD=1, builds) the producer the batcher
// invokes as its proof command.
func privateProjectionExecutor(t *testing.T) string {
	t.Helper()
	command, err := (rustbin.Spec{SrcDir: "rust", Package: "kona-sp1-super-range-executor", Binary: "kona-sp1-private-projection-executor"}).EnsureExists(t.Context(), testlog.Logger(t, log.LevelInfo))
	if err != nil {
		t.Fatalf("private projection executor unavailable: %v", err)
	}
	return command
}

// soundProfilePair builds a two-L2 world whose chain B is a private pair running
// sp1-private-projection-v1 with mock envelopes.
func soundProfilePair(gt *testing.T, extra ...sysgo.PrivateInteropOption) (devtest.T, *presets.TwoL2SupernodeInterop) {
	gt.Helper()
	command := privateProjectionExecutor(gt)
	return soundProfilePairWith(gt, sysgo.WithPrivateInteropSP1Mock(command), true, extra...)
}

func soundProfilePairWith(gt *testing.T, profile sysgo.PrivateInteropOption, mock bool, extra ...sysgo.PrivateInteropOption) (devtest.T, *presets.TwoL2SupernodeInterop) {
	gt.Helper()
	t := devtest.SerialT(gt)
	opts := append([]sysgo.PrivateInteropOption{profile}, extra...)
	sys := presets.NewTwoL2SupernodeLightSequencerInterop(t, 0,
		presets.WithDeployerOptions(sysgo.WithSequencingWindow(10)),
		presets.WithPrivateInteropChain(opts...))
	cfg := sys.PrivateInterop.RenderingRollupConfig.PrivateProjection
	t.Require().NotNil(cfg, "the rendering must carry a private_projection config")
	t.Require().Equal(projection.SP1PrivateProjectionV1, cfg.Verifier)
	t.Require().Equal(mock, cfg.MockProofs, "mock_proofs")
	t.Require().NotEqual(common.Hash{}, cfg.ProgramVKey)
	t.Require().NotEqual(common.Hash{}, cfg.PrivateConfigHash)
	t.Require().NotEqual(common.Hash{}, cfg.DependencySetHash)
	prover := "real Groth16"
	if mock {
		prover = "mock (SP1 mock prover over the local ELF)"
		if cfg.ProgramVKey == sysgo.PrivateInteropNativeMockVKey {
			prover = "native-mock (no ELF: native relation, mock envelope)"
		}
	}
	t.Logger().Info("Sound profile pair", "prover", prover, "program_vkey", cfg.ProgramVKey,
		"private_config_hash", cfg.PrivateConfigHash, "dependency_set_hash", cfg.DependencySetHash)
	return t, sys
}

// publishedSpan is one admitted span as the projection carries it.
type publishedSpan struct {
	Claim    *codec.RangeClaim
	Envelope *projection.Envelope
	// Records[i] is the recordOutput root of projection block Claim.FirstBlock+i.
	Records []common.Hash
}

func (s *publishedSpan) word(k int) common.Hash {
	return common.BytesToHash(s.Envelope.PublicValues[32*k : 32*k+32])
}

// claimInBlock returns the postClaim a projection block opens with, if any. Transaction 0 is the
// L1-info deposit; a span's claim is the first sequencer transaction of its first block.
func claimInBlock(txs types.Transactions) *codec.RangeClaim {
	for _, tx := range txs {
		if tx.Type() == types.DepositTxType {
			continue
		}
		if tx.To() == nil || *tx.To() != predeploys.ClaimRegistryAddr {
			return nil
		}
		c, err := wire.DecodeClaim(tx.Data())
		if err != nil {
			return nil
		}
		return c
	}
	return nil
}

// recordInBlock returns the block's recordOutput root.
func recordInBlock(t devtest.T, txs types.Transactions) common.Hash {
	for _, tx := range txs {
		if tx.Type() == types.DepositTxType || tx.To() == nil || *tx.To() != predeploys.ClaimRegistryAddr {
			continue
		}
		if root, err := wire.DecodeOutput(tx.Data()); err == nil {
			return root
		}
	}
	t.Require().FailNow("projection block carries no recordOutput")
	return common.Hash{}
}

// spanCovering walks the canonical projection back from height to the span that published it.
func spanCovering(t devtest.T, sys *presets.TwoL2SupernodeInterop, height uint64) *publishedSpan {
	require := t.Require()
	client := sys.L2BSupernodeEL.Escape().L2EthClient()
	for n := height; n > 0; n-- {
		_, txs, err := client.InfoAndTxsByNumber(t.Ctx(), n)
		require.NoError(err, "reading projection block %d", n)
		c := claimInBlock(txs)
		if c == nil {
			continue
		}
		require.LessOrEqual(c.FirstBlock, height)
		require.GreaterOrEqual(c.LastBlock, height, "the nearest claim at or below %d does not cover it", height)
		env, err := projection.DecodeEnvelope(c.Proof)
		require.NoError(err, "the admitted claim's proof slot must be a strict sp1 envelope")
		span := &publishedSpan{Claim: c, Envelope: env}
		for b := c.FirstBlock; b <= c.LastBlock; b++ {
			_, txs, err := client.InfoAndTxsByNumber(t.Ctx(), b)
			require.NoError(err)
			span.Records = append(span.Records, recordInBlock(t, txs))
		}
		return span
	}
	require.FailNow("no span covers the height", "height %d", height)
	return nil
}

// privateOutputRoot is the OutputV0 root of the canonical PRIVATE block at n.
func privateOutputRoot(t devtest.T, sys *presets.TwoL2SupernodeInterop, n uint64) common.Hash {
	info, err := sys.L2ELB.Escape().L2EthClient().InfoByNumber(t.Ctx(), n)
	t.Require().NoError(err)
	t.Require().NotNil(info.WithdrawalsRoot(), "Isthmus private blocks carry the message-passer root")
	return common.Hash(eth.OutputRoot(&eth.OutputV0{
		StateRoot:                eth.Bytes32(info.Root()),
		MessagePasserStorageRoot: eth.Bytes32(*info.WithdrawalsRoot()),
		BlockHash:                info.Hash(),
	}))
}

// privateMessageLeaves renders the canonical PRIVATE block at n and returns its message leaves: the
// messages a complete projection must carry.
func privateMessageLeaves(t devtest.T, sys *presets.TwoL2SupernodeInterop, n uint64) []common.Hash {
	ref := sys.L2ELB.BlockRefByNumber(n)
	_, receipts, err := sys.L2ELB.Escape().L2EthClient().FetchReceipts(t.Ctx(), ref.Hash)
	t.Require().NoError(err)
	var logs []*types.Log
	for _, r := range receipts.Geth() {
		logs = append(logs, r.Logs...)
	}
	leaves, err := render.MessageLeaves(n, render.RenderedLogs(logs, render.NewEmitterSet()))
	t.Require().NoError(err)
	return leaves
}

// checkSoundSpan asserts the public-values words the spec calls out (15, 18, 19, 20) against the
// canonical chains, plus the words that identify the span and its program.
func checkSoundSpan(t devtest.T, sys *presets.TwoL2SupernodeInterop, span *publishedSpan) (messages int) {
	require := t.Require()
	cfg := sys.PrivateInterop.RenderingRollupConfig.PrivateProjection
	c := span.Claim
	if cfg.MockProofs {
		require.Equal(projection.EnvelopeKindMock, span.Envelope.Kind, "the envelope must be an SP1 mock envelope (kind 0x02)")
		require.Len(span.Envelope.Proof, projection.MockProofLength)
		require.Equal(cfg.ProgramVKey, common.BytesToHash(span.Envelope.Proof[:32]), "mock proof word 0 is the pinned program vkey")
	} else {
		require.Equal(projection.EnvelopeKindGroth16, span.Envelope.Kind, "the envelope must be an SP1 Groth16 envelope (kind 0x01)")
		require.Len(span.Envelope.Proof, 356)
	}

	require.Equal(projection.PublicValuesMagic, span.word(projection.WordMagic))
	require.Equal(cfg.PrivateConfigHash, span.word(projection.WordPrivateConfigHash))
	require.Equal(cfg.DependencySetHash, span.word(projection.WordDepSetHash))
	require.Equal(c.RollupConfigHash, span.word(projection.WordProjectionConfigHash))
	require.Equal(c.FirstBlock, u64Word(span.word(projection.WordFirstBlock)))
	require.Equal(c.LastBlock, u64Word(span.word(projection.WordLastBlock)))

	// Word 15: the canonical L1 origin of the span's terminal block.
	terminal := sys.L2BSupernodeEL.BlockRefByNumber(c.LastBlock)
	require.Equal(terminal.L1Origin.Hash, span.word(projection.WordL1Head), "l1Head is the terminal block's L1 origin")
	require.Equal(terminal.L1Origin.Hash, c.L1Head)
	require.Equal(terminal.L1Origin.Hash, sys.L1EL.BlockRefByNumber(terminal.L1Origin.Number).Hash, "the L1 origin is canonical")

	// Word 18: every recorded output, each equal to the private chain's actual output.
	var outputLeaves, messageLeaves []common.Hash
	for i, root := range span.Records {
		n := c.FirstBlock + uint64(i)
		require.Equal(privateOutputRoot(t, sys, n), root, "the record at %d is the private block's OutputV0", n)
		outputLeaves = append(outputLeaves, projection.OutputLeaf(n, root))
		messageLeaves = append(messageLeaves, privateMessageLeaves(t, sys, n)...)
	}
	require.Equal(projection.CommitmentRoot(projection.OutputsDomain, outputLeaves), span.word(projection.WordOutputsRoot))
	// Word 19: every message the private blocks emitted, rendered, in order.
	require.Equal(projection.CommitmentRoot(projection.MessagesDomain, messageLeaves), span.word(projection.WordMessagesRoot),
		"messagesRoot must commit exactly the private blocks' rendered messages")
	// Word 20: the last record.
	require.Equal(span.Records[len(span.Records)-1], span.word(projection.WordTerminalOutput))
	t.Logger().Info("Checked an admitted sound-profile span", "first", c.FirstBlock, "last", c.LastBlock,
		"messages", len(messageLeaves), "l1_head", c.L1Head)
	return len(messageLeaves)
}

// u64Word reads a uint256 public-values word as a uint64 (max uint64 if it does not fit).
func u64Word(w common.Hash) uint64 {
	n := new(big.Int).SetBytes(w[:])
	if !n.IsUint64() {
		return ^uint64(0)
	}
	return bigs.Uint64Strict(n)
}

// TestPrivateSoundProfileMockProofsPublish runs the sound profile end to end with mock envelopes:
// batcher, SP1 admission in the projection's op-node, and claim-follow.
func TestPrivateSoundProfileMockProofsPublish(gt *testing.T) {
	testSoundProfilePublish(gt)
}

func testSoundProfilePublish(gt *testing.T, extra ...sysgo.PrivateInteropOption) {
	gt.Helper()
	t, sys := soundProfilePair(gt, extra...)
	require := t.Require()
	logger := t.Logger()

	sys.PrivateInterop.Invariant.RequireRenderingReached(3, 4*time.Minute)
	sys.PrivateInterop.Invariant.RequirePrivateSafeReached(1, 4*time.Minute)

	alice := sys.FunderA.NewFundedEOA(eth.OneTenthEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneTenthEther)
	eventLoggerA := alice.DeployEventLogger()
	eventLoggerB := bob.DeployEventLogger()
	eventLogger := bindings.NewBindings[bindings.EventLogger]()

	// Counterparty -> private: an import on the private chain, rendered as validateMessage.
	calldataIn, err := eventLogger.EmitLog([]eth.Bytes32{{0x01}}, []byte("into the private chain")).EncodeInputLambda()
	require.NoError(err)
	sendIn := txintent.NewIntent[*txintent.SendTrigger, *txintent.InteropOutput](alice.Plan())
	sendIn.Content.Set(&txintent.SendTrigger{Emitter: predeploys.L2toL2CrossDomainMessengerAddr, DestChainID: bob.ChainID(), Target: eventLoggerB, RelayedCalldata: calldataIn})
	_, err = sendIn.PlannedTx.Included.Eval(t.Ctx())
	require.NoError(err)
	sys.L2A.WaitForBlock()
	relayIn := txintent.NewIntent[*txintent.RelayTrigger, *txintent.InteropOutput](bob.Plan())
	relayIn.Content.DependOn(&sendIn.Result)
	relayIn.Content.Fn(txintent.RelayIndexed(predeploys.L2toL2CrossDomainMessengerAddr, &sendIn.Result, &sendIn.PlannedTx.Included, 0))
	relayInRec, err := relayIn.PlannedTx.Included.Eval(t.Ctx())
	require.NoError(err)
	importBlock := bigs.Uint64Strict(relayInRec.BlockNumber)

	// Private -> counterparty: an export, rendered as replaySentMessage and relayed on chain A.
	calldataOut, err := eventLogger.EmitLog([]eth.Bytes32{{0x02}}, []byte("out of the private chain")).EncodeInputLambda()
	require.NoError(err)
	sendOut := txintent.NewIntent[*txintent.SendTrigger, *txintent.InteropOutput](bob.Plan())
	sendOut.Content.Set(&txintent.SendTrigger{Emitter: predeploys.L2toL2CrossDomainMessengerAddr, DestChainID: alice.ChainID(), Target: eventLoggerA, RelayedCalldata: calldataOut})
	sendOutRec, err := sendOut.PlannedTx.Included.Eval(t.Ctx())
	require.NoError(err)
	exportBlock := bigs.Uint64Strict(sendOutRec.BlockNumber)
	relayOut := txintent.NewIntent[*txintent.RelayTrigger, *txintent.InteropOutput](alice.Plan())
	relayOut.Content.DependOn(&sendOut.Result)
	relayOut.Content.Fn(txintent.RelayIndexed(predeploys.L2toL2CrossDomainMessengerAddr, &sendOut.Result, &sendOut.PlannedTx.Included, 0))
	relayOutRec, err := relayOut.PlannedTx.Included.Eval(t.Ctx())
	require.NoError(err)
	require.Len(relayOutRec.Logs, 3, "the relay emits the executing message, the target's event and RelayedMessage")
	require.Equal(eventLoggerA, relayOutRec.Logs[1].Address)

	// Both private blocks become private-safe only through admitted, proven spans.
	top := max(importBlock, exportBlock)
	sys.PrivateInterop.Invariant.RequireRenderingReached(top, 5*time.Minute)
	sys.PrivateInterop.Invariant.RequirePrivateSafeReached(top, 5*time.Minute)

	// The export is delivered for good: the counterparty's cross-safe frontier passes the relay.
	dsl.CheckAll(t, sys.L2ASupernodeCL.ReachedWithProgressFn(
		safety.CrossSafe, safety.LocalUnsafe, bigs.Uint64Strict(relayOutRec.BlockNumber), 6*time.Minute, 90*time.Second))

	for _, height := range []uint64{importBlock, exportBlock} {
		span := spanCovering(t, sys, height)
		require.Positive(checkSoundSpan(t, sys, span), "the span covering %d carries the message", height)
	}
	// And the very first span, published from genesis.
	checkSoundSpan(t, sys, spanCovering(t, sys, 1))
	logger.Info("Sound profile published with mock envelopes", "import_block", importBlock, "export_block", exportBlock)
}

// forgeryHook flips one byte of public-values word 19 (messagesRoot) in every envelope the producer
// returns while armed, after the batcher's own envelope verification, and records the ranges it
// forged (read back from words 10 and 11 of the envelope itself).
type forgeryHook struct {
	armed  atomic.Bool
	mu     sync.Mutex
	forged [][2]uint64
}

func (f *forgeryHook) mutate(envelope []byte) []byte {
	if !f.armed.Load() {
		return envelope
	}
	env, err := projection.DecodeEnvelope(envelope)
	if err != nil {
		return envelope
	}
	first := u64Word(common.BytesToHash(env.PublicValues[32*projection.WordFirstBlock : 32*projection.WordFirstBlock+32]))
	last := u64Word(common.BytesToHash(env.PublicValues[32*projection.WordLastBlock : 32*projection.WordLastBlock+32]))
	env.PublicValues[32*projection.WordMessagesRoot+31] ^= 0x01
	f.mu.Lock()
	f.forged = append(f.forged, [2]uint64{first, last})
	f.mu.Unlock()
	return projection.EncodeEnvelope(env)
}

// requireNoForgedSafeBlocks checks the canonical projection blocks from..to (inclusive; nothing if
// to < from): each is deposit-only, or carries the records of a span whose envelope passes
// checkSoundSpan. A forged word 19 fails that check, because it no longer equals the messages root
// of the private blocks.
func requireNoForgedSafeBlocks(t devtest.T, sys *presets.TwoL2SupernodeInterop, from, to uint64) {
	client := sys.L2BSupernodeEL.Escape().L2EthClient()
	checked := map[uint64]bool{}
	for n := from; n <= to; n++ {
		_, txs, err := client.InfoAndTxsByNumber(t.Ctx(), n)
		t.Require().NoError(err, "reading projection block %d", n)
		depositOnly := true
		for _, tx := range txs {
			if tx.Type() != types.DepositTxType {
				depositOnly = false
				break
			}
		}
		if depositOnly {
			continue
		}
		span := spanCovering(t, sys, n)
		if !checked[span.Claim.FirstBlock] {
			checkSoundSpan(t, sys, span)
			checked[span.Claim.FirstBlock] = true
		}
	}
	t.Logger().Info("No forged span is safe", "from", from, "to", to)
}

func (f *forgeryHook) first() ([2]uint64, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.forged) == 0 {
		return [2]uint64{}, false
	}
	return f.forged[0], true
}

// TestPrivateSoundProfileRejectsForgedEnvelope publishes a span whose envelope differs from the
// admission statement in word 19 only. The batcher's own post-proof admission preflight is skipped,
// so the forged span reaches L1; derivation must drop it. Private safety stays below it until the
// fault is removed, and then publication recovers.
func TestPrivateSoundProfileRejectsForgedEnvelope(gt *testing.T) {
	testSoundProfileRejectsForgedEnvelope(gt, &bss.PrivateInteropTestHooks{})
}

// testSoundProfileRejectsForgedEnvelope installs the forgery on top of base (whose other fields are
// kept, e.g. for diagnostics).
func testSoundProfileRejectsForgedEnvelope(gt *testing.T, base *bss.PrivateInteropTestHooks) {
	gt.Helper()
	forge := &forgeryHook{}
	hooks := base
	hooks.MutateProof, hooks.SkipAdmissionPreflight = forge.mutate, true
	t, sys := soundProfilePair(gt, sysgo.WithPrivateInteropTestHooks(hooks), sysgo.WithoutRenderingInvariantCheck())
	require := t.Require()

	// Let the pair publish normally first, so the forgery hits a continuation span.
	require.Eventually(func() bool {
		status, err := sys.L2BCL.Escape().RollupAPI().SyncStatus(t.Ctx())
		return err == nil && status.SafeL2.Number >= 2
	}, 4*time.Minute, time.Second, "private safety should advance before the forgery")
	forge.armed.Store(true)

	var forged [2]uint64
	require.Eventually(func() bool {
		var ok bool
		forged, ok = forge.first()
		return ok
	}, 4*time.Minute, time.Second, "the batcher must produce a span while the forgery is armed")
	t.Logger().Info("Forged an envelope", "first", forged[0], "last", forged[1])

	// Give the forged batch ample L1 inclusion time.
	l1Start := sys.L1EL.BlockRefByLabel(eth.Unsafe)
	sys.L1EL.WaitForBlockNumber(l1Start.Number + 4)
	// The projection's safe head must not include the forged span. Normally it stays below
	// forged[0]. It can still pass forged[0] without admitting anything: if the span reaches L1 late
	// in its epochs' 10-block sequencing window, derivation drops it and, once the window expires,
	// fills those heights with deposit-only blocks. So the invariant is on content, not height:
	// every safe projection block in the forged range is deposit-only or belongs to a span whose
	// envelope matches the canonical chains (a forged word 19 does not).
	projectionSafe := sys.L2BSupernodeEL.BlockRefByLabel(eth.Safe)
	requireNoForgedSafeBlocks(t, sys, forged[0], min(projectionSafe.Number, forged[1]))
	privateStatus, err := sys.L2BCL.Escape().RollupAPI().SyncStatus(t.Ctx())
	require.NoError(err)
	if projectionSafe.Number < forged[0] {
		require.Less(privateStatus.SafeL2.Number, forged[0], "private safety must not advance into the forged span")
	} else {
		t.Logger().Warn("The forged span arrived late: its heights became deposit-only after window expiry",
			"projection_safe", projectionSafe.Number, "private_safe", privateStatus.SafeL2.Number)
		require.LessOrEqual(privateStatus.SafeL2.Number, projectionSafe.Number,
			"private safety must not pass the projection's (checked) safe head")
	}

	// Remove the fault. Publication recovers: either a genuine re-publication of the range, or
	// sequencing-window expiry, deposit-only replacement and a recovery-mode span.
	forge.armed.Store(false)
	require.Eventually(func() bool {
		status, err := sys.L2BCL.Escape().RollupAPI().SyncStatus(t.Ctx())
		return err == nil && status.SafeL2.Number > forged[1]
	}, 6*time.Minute, time.Second, "private safety must advance past the forged range once genuine proofs resume")

	// No canonical projection block carries the forged envelope.
	client := sys.L2BSupernodeEL.Escape().L2EthClient()
	for n := forged[0]; n <= forged[1]; n++ {
		_, txs, err := client.InfoAndTxsByNumber(t.Ctx(), n)
		require.NoError(err)
		c := claimInBlock(txs)
		if c == nil {
			continue
		}
		env, err := projection.DecodeEnvelope(c.Proof)
		require.NoError(err)
		span := &publishedSpan{Claim: c, Envelope: env}
		for b := c.FirstBlock; b <= c.LastBlock; b++ {
			_, btxs, err := client.InfoAndTxsByNumber(t.Ctx(), b)
			require.NoError(err)
			span.Records = append(span.Records, recordInBlock(t, btxs))
		}
		checkSoundSpan(t, sys, span)
	}
	// And a fresh transaction becomes private-safe through the recovered publication.
	alice := sys.FunderB.NewFundedEOA(eth.OneHundredthEther)
	rec, err := alice.Transfer(common.Address{0xfe}, eth.OneGWei).Included.Eval(t.Ctx())
	require.NoError(err)
	sys.L2BCL.Reached(safety.CrossSafe, bigs.Uint64Strict(rec.BlockNumber), 180)
}

// TestPrivateSoundProfileGroth16 is the real-proof skeleton: the publish flow with a real SP1
// Groth16 prover and mock_proofs=false, so admission runs the pinned Groth16 verifier. It needs the
// SP1 toolchain, the private-projection ELF in $KONA_SP1_ELF_DIR and either
// substantial CPU/RAM (cpu) or prover-network credentials (network). CI provides none of these this
// round, so it is skipped there.
func TestPrivateSoundProfileGroth16(gt *testing.T) {
	prover := os.Getenv("PRIVATE_INTEROP_SP1_PROVER")
	if prover != "cpu" && prover != "network" {
		gt.Skip("real Groth16 proving is not exercised in CI: set PRIVATE_INTEROP_SP1_PROVER=cpu|network, " +
			"with the private-projection ELF built, to run this skeleton")
	}
	command := privateProjectionExecutor(gt)
	t, sys := soundProfilePairWith(gt, sysgo.WithPrivateInteropSP1Prover(command, prover), false)
	// A real proof per span takes minutes; one span is enough to show admission accepted it.
	sys.PrivateInterop.Invariant.RequireRenderingReached(1, 60*time.Minute)
	sys.PrivateInterop.Invariant.RequirePrivateSafeReached(1, 60*time.Minute)
	checkSoundSpan(t, sys, spanCovering(t, sys, 1))
}
