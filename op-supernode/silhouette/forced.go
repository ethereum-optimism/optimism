package silhouette

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// The forced-extension convention, executable.
//
// When P's L1 origin advances seq_window_size past the last proven block's origin without a new
// proof batch, stock derivation force-generates empty blocks. That is DESIGNED liveness (PLAN.md,
// DR-2): a dead prover can never stall the dependency set's cross-safe frontier. It also means the
// verifier must be able to say exactly which blocks those are and exactly what they hash to,
// because three implementations have to agree on it — this one, the prover, and the superroot
// program — and because the shim EL's fail-stop guard serves proven-or-forced facts and nothing
// else. Only this one is on this branch; the other two are prover-side, on the shelf.
//
// The normative text is g-decisions.md G2 D2, corrected by G2 D7. This file is the executable form
// of it, and the tests in forced_test.go are the executable spec a prover is matched against.
//
// The one thing to hold on to while reading: a forced block is not an EVM block. It carries one
// transaction that nothing ever executed, so it has a transaction and no receipt. State does not
// move (identity-STF: the state root and message-passer root carry forward), but the header still
// progresses. Everything here is a pure function of the last proven block's wire facts, the frozen
// SystemConfig, and the L1 header chain — no state, no execution, no value that is not either on
// the wire or in configuration. That is what G2 D7 bought by pinning the base fee.

// forcedExtraData is the parent-carried eip-1559 encoding a forced block repeats verbatim.
//
// It is NOT the "silhouette-v1" ASCII marker PLAN.md's fabrication class 2 asks for: post-Holocene
// extraData is the consensus-critical carrier of the eip-1559 parameters, strictly length- and
// version-checked, and a 13-byte marker would both fail validation and silently reset the chain's
// fee parameters on the following block (op-node reconstructs the SystemConfig by reading
// eip1559Params back out of the parent header). See G2 D3 — escalated to Karl, not improvised here.

// L1Headers is the L1 access forced-block computation needs: headers by number, on the canonical
// chain this node follows. Nothing here reads receipts — a forced block has no deposits to derive
// (DR-2) and the SystemConfig is frozen, so the L1 walk is headers-only, which is the same property
// that makes a prover's L1 walk cheap.
type L1Headers interface {
	L1BlockRefByNumber(ctx context.Context, num uint64) (eth.L1BlockRef, error)
	InfoByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, error)
}

// ForcedParams is everything the convention reads that is not the parent block or L1.
type ForcedParams struct {
	Rollup *rollup.Config
	// L1Chain is the L1 chain config, needed to build the L1-info transaction's blob-base-fee field.
	L1Chain *params.ChainConfig
	// SysCfg is P's FROZEN genesis SystemConfig (DR-2). Freezing it is what lets gasLimit and
	// eip1559Params be treated as constants rather than as values a ConfigUpdate log could move
	// mid-extension.
	SysCfg eth.SystemConfig
}

// ForcedExtension computes the forced blocks that stock derivation generates on top of `parent`,
// given that the pipeline has reached L1 block `pipelineOrigin`.
//
// This mirrors op-node's own generator (derive/base_batch_stage.go:162-206, and kona's twin in
// batch_validator.rs) rather than re-deriving the rule, because the stock pipeline running
// downstream of this source is the authority: whatever it generates is what the chain gets, and a
// prediction that disagreed with it would put facts in the store for blocks that never existed.
//
// parentOrigin is `parent`'s own L1 origin and parentSeqNumber its position within that epoch;
// together they are the state the generator carries between blocks.
func ForcedExtension(
	ctx context.Context,
	p ForcedParams,
	l1 L1Headers,
	parent Fact,
	pipelineOrigin uint64,
) ([]Fact, error) {
	if p.Rollup.BlockTime == 0 {
		return nil, fmt.Errorf("rollup config has a zero block time")
	}
	var out []Fact
	cur := parent
	// epoch walks forward from the parent's own origin. l1Blocks[0] in the stock generator is the
	// OLDEST buffered L1 block, which for a chain whose safe head sits at `parent` is exactly the
	// parent's origin — the generator advances it one block at a time below.
	epochNum := parent.L1Origin.Number

	for {
		epoch, err := l1.L1BlockRefByNumber(ctx, epochNum)
		if err != nil {
			return nil, fmt.Errorf("fetch forced-extension epoch %d: %w", epochNum, err)
		}
		// The window predicate. Under Holocene the stock generator hard-codes outOfData=true, so
		// `(expiry == origin && outOfData) || expiry < origin` collapses to `expiry <= origin`.
		if epoch.Number+p.Rollup.SeqWindowSize > pipelineOrigin {
			// The window has not expired for this epoch: no forced block is due, and none will be
			// until L1 advances further.
			return out, nil
		}
		nextEpoch, err := l1.L1BlockRefByNumber(ctx, epochNum+1)
		if err != nil {
			// The generator refuses to emit until the NEXT L1 block is buffered, because it needs
			// that block's timestamp to know how many blocks fill the current epoch. Not an error:
			// there is simply nothing to force yet.
			return out, nil
		}

		// firstOfEpoch OVERRIDES the timestamp bound: every epoch gets at least one L2 block, even
		// one whose timestamp lands at or past the next epoch's. Do not fold this into a closed-form
		// block count.
		firstOfEpoch := epochNum == cur.L1Origin.Number+1
		nextTimestamp := cur.Timestamp + p.Rollup.BlockTime

		if nextTimestamp >= nextEpoch.Time && !firstOfEpoch {
			// Every block this epoch can hold has been generated; advance exactly one epoch and
			// re-enter. This is why a forced extension walks L1 forward rather than parking.
			epochNum++
			continue
		}

		seqNumber := cur.SeqNumber + 1
		if firstOfEpoch {
			seqNumber = 0
		}
		blk, err := forcedBlock(ctx, p, l1, cur, epoch, seqNumber, nextTimestamp)
		if err != nil {
			return nil, err
		}
		out = append(out, blk)
		cur = blk
	}
}

// forcedBlock builds one forced block's header and reads its facts off it.
//
// Every field is either carried forward from the parent, read from the L1 origin header, taken from
// the frozen SystemConfig, or a protocol constant. The field list and its citations are G2 D2.4.
func forcedBlock(
	ctx context.Context,
	p ForcedParams,
	l1 L1Headers,
	parent Fact,
	epoch eth.L1BlockRef,
	seqNumber uint64,
	timestamp uint64,
) (Fact, error) {
	info, err := l1.InfoByHash(ctx, epoch.Hash)
	if err != nil {
		return Fact{}, fmt.Errorf("fetch L1 origin %s of forced block %d: %w", epoch, parent.Number+1, err)
	}

	// The single transaction: the stock L1-info deposit, origin-accurate, built by the unmodified
	// builder. Not dummy bytes — the stock CL parses it back for origin mapping and resets, so it
	// has to be the real thing.
	l1InfoBytes, err := derive.L1InfoDepositBytes(p.Rollup, p.L1Chain, p.SysCfg, seqNumber, info, timestamp)
	if err != nil {
		return Fact{}, fmt.Errorf("build L1-info tx for forced block %d: %w", parent.Number+1, err)
	}

	// The header goes through the SHARED assembler (rendered.go), the same one the shim EL renders
	// proven blocks with. Identity-STF enters as two inputs — the state root and the message-passer
	// root carry forward from the parent — and everything else is the ordinary stock progression.
	// Sharing the assembler is what keeps G2 D2.4's normative field list a single implementation.
	hdr, err := RenderHeader(p, HeaderInputs{
		Parent:                   parent,
		Number:                   parent.Number + 1,
		Timestamp:                timestamp,
		StateRoot:                parent.StateRoot,
		MessagePasserStorageRoot: parent.MessagePasserStorageRoot,
		Origin:                   info,
		Txs:                      [][]byte{l1InfoBytes},
	})
	if err != nil {
		return Fact{}, err
	}

	// A forced block's hash is the real hash of the deterministically built block: the convention
	// DEFINES the block, so there is nothing else it could be. A proven block's hash comes off the
	// wire instead, and its rendered header deliberately does not re-hash to it.
	hash := hdr.Hash()
	return Fact{
		Number:                   hdr.Number.Uint64(),
		Timestamp:                timestamp,
		Hash:                     hash,
		StateRoot:                parent.StateRoot,
		MessagePasserStorageRoot: parent.MessagePasserStorageRoot,
		OutputRoot: common.Hash(eth.OutputRoot(&eth.OutputV0{
			StateRoot:                eth.Bytes32(parent.StateRoot),
			MessagePasserStorageRoot: eth.Bytes32(parent.MessagePasserStorageRoot),
			BlockHash:                hash,
		})),
		L1Origin:  epoch.ID(),
		SeqNumber: seqNumber,
		Forced:    true,
		Header:    hdr,
		// A forced block's import list is empty AND KNOWN, at every wire version. It is not a wire
		// fact that happens to be absent — the convention DEFINES the block as carrying exactly one
		// transaction, the L1-info transaction, so "it consumed nothing" is something this node
		// computes rather than reads. Marking it unknown would make the forced extension — the
		// mechanism that exists so a dead prover cannot stall the depset — the one thing that stalls
		// it (G7G D4).
		ExecMsgsKnown: true,
	}, nil
}

// forcedExtra is the header extraData: the stock Holocene/Jovian eip-1559 encoding of the frozen
// parameters, which under a frozen SystemConfig is byte-identical to the parent's.
//
// It goes through op-geth's own encoders rather than laying the bytes out here, so the encoding
// cannot drift from the one the execution layer validates against. A zero denominator is refused
// rather than defaulted: the execution layer would silently substitute the Canyon constants
// (miner/worker.go:377-381), which for a chain whose SystemConfig is supposed to be FROZEN is a
// configuration error worth failing on rather than a default worth applying.
func forcedExtra(p ForcedParams, timestamp uint64) ([]byte, error) {
	if !p.Rollup.IsHolocene(timestamp) {
		return nil, nil
	}
	denom, elasticity := eip1559.DecodeHolocene1559Params(p.SysCfg.EIP1559Params[:])
	// An all-zero SystemConfig value does NOT mean "no parameters": it means "use the chain config's
	// defaults", and the execution layer substitutes them when building the header
	// ($G/miner/worker.go:377-381). A live silhouette-shaped chain really does carry zeroes here —
	// Cove's chain P does — so a forced block has to make the same substitution or its extraData
	// disagrees with every other block on the chain.
	//
	// A MIXED pair is different: Holocene requires both zero or both non-zero, so one of each is a
	// corrupt config rather than a request for defaults, and it is refused.
	switch {
	case denom == 0 && elasticity == 0:
		denom, elasticity = chainDefault1559Params(p, timestamp)
	case denom == 0 || elasticity == 0:
		return nil, fmt.Errorf("frozen SystemConfig has eip1559 params %x: a denominator and elasticity "+
			"must be both zero (meaning chain defaults) or both non-zero", p.SysCfg.EIP1559Params)
	}
	if denom == 0 || elasticity == 0 {
		return nil, fmt.Errorf("chain config resolves eip1559 params to denominator %d elasticity %d; "+
			"a zero would make the next block's base fee a division by zero", denom, elasticity)
	}
	if p.Rollup.IsJovian(timestamp) {
		return eip1559.EncodeJovianExtraData(denom, elasticity, p.SysCfg.MinBaseFee), nil
	}
	return eip1559.EncodeHoloceneExtraData(denom, elasticity), nil
}

// chainDefault1559Params is the substitution the execution layer makes when the SystemConfig's
// eip-1559 parameters are all zero: the chain config's own denominator and elasticity, with Canyon's
// denominator taking over once Canyon is active.
func chainDefault1559Params(p ForcedParams, timestamp uint64) (denom, elasticity uint64) {
	opCfg := p.Rollup.ChainOpConfig
	denom = opCfg.EIP1559Denominator
	if p.Rollup.IsCanyon(timestamp) && opCfg.EIP1559DenominatorCanyon != nil {
		denom = *opCfg.EIP1559DenominatorCanyon
	}
	return denom, opCfg.EIP1559Elasticity
}
