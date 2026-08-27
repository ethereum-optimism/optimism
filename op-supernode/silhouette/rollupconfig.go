package silhouette

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	coreparams "github.com/ethereum-optimism/optimism/op-core/params"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// The silhouette rollup config is OUR artifact, not something read from the superchain registry: a
// silhouette chain is not in any registry, and its config differs from a stock chain in the ways
// listed below. Generating it here rather than hand-writing JSON is what keeps those differences
// visible and checked.
//
//  1. The sequencing window is FINITE and short (DR-2). Stock's 12-hour default is far too coarse
//     against a ten-minute proof cadence: the forced extension is meant to be a liveness backstop
//     that fires within an hour or two of a prover dying, not half a day later.
//  2. deposit_contract_address is the chain's stock OptimismPortal. Stock derivation reads its
//     TransactionDeposited events exactly as it does for an ordinary OP Stack chain.
//  3. Every fork is active at genesis. That is what removes activation blocks, which in turn is what
//     keeps a forced block single-transaction (an activation block would carry the fork's upgrade
//     transactions and an inflated gas limit) — see G2 D2.1.

// SilhouetteParams is the minimum a caller must decide to get a silhouette rollup config.
type SilhouetteParams struct {
	L2ChainID *big.Int
	L1ChainID *big.Int
	// L1Genesis is the L1 block the chain starts from, and L2Genesis its own genesis block.
	L1Genesis eth.BlockID
	L2Genesis eth.BlockID
	// L2Time is the L2 genesis timestamp.
	L2Time uint64
	// BlockTime is P's L2 block time in seconds.
	BlockTime uint64
	// SeqWindowSize is the FINITE sequencing window, in L1 blocks (DR-2).
	SeqWindowSize uint64
	// MaxSequencerDrift bounds how far an L2 block's timestamp may run ahead of its L1 origin.
	MaxSequencerDrift uint64
	// DepositContract is the chain's stock OptimismPortal address.
	DepositContract common.Address
	// BatchInbox is the ordinary rollup-config inbox. The silhouette op-batcher overrides its terminal
	// destination with the proof-batch inbox, while the field remains required by stock rollup config.
	BatchInbox common.Address
	// SystemConfigProxy is the L1 SystemConfig address. It is frozen: no ConfigUpdate event is ever
	// emitted, which is the precondition G2 D2 relies on to treat gasLimit and eip1559Params as
	// parent-header carry-forward.
	SystemConfigProxy common.Address
	// Genesis SystemConfig: the frozen values.
	GasLimit      uint64
	EIP1559Params eth.Bytes8
	MinBaseFee    uint64
	BatcherAddr   common.Address
	// BaseFeeScalar / BlobBaseFeeScalar are the Ecotone fee scalars. They are frozen like everything
	// else in the SystemConfig, and they are not optional: rollup.Config.Check refuses a genesis
	// system config with a zero scalar.
	BaseFeeScalar     uint32
	BlobBaseFeeScalar uint32
}

// EcotoneScalar packs the two Ecotone fee scalars into the SystemConfig scalar word: version byte 1,
// then the blob-base-fee and base-fee scalars in the low eight bytes.
func EcotoneScalar(baseFeeScalar, blobBaseFeeScalar uint32) eth.Bytes32 {
	var out eth.Bytes32
	out[0] = 1
	binary.BigEndian.PutUint32(out[24:28], blobBaseFeeScalar)
	binary.BigEndian.PutUint32(out[28:32], baseFeeScalar)
	return out
}

// DefaultSeqWindowSize is a finite window sized for a ten-minute proof cadence: 300 L1 blocks is one
// hour of Ethereum, roughly six proof intervals of margin. Long enough that an ordinary late batch
// never triggers the forced extension, short enough that a dead prover stops holding P's own progress
// hostage within the hour.
const DefaultSeqWindowSize = 300

// RollupConfigFor builds a silhouette chain's rollup config.
func RollupConfigFor(p SilhouetteParams) (*rollup.Config, error) {
	if p.L2ChainID == nil || p.L1ChainID == nil {
		return nil, errors.New("both chain IDs are required")
	}
	if p.BlockTime == 0 {
		return nil, errors.New("blockTime is required")
	}
	if p.DepositContract == (common.Address{}) {
		return nil, errors.New("depositContract is required: stock derivation reads deposit events from it")
	}
	if p.BaseFeeScalar == 0 {
		return nil, errors.New("baseFeeScalar is required: a zero genesis scalar is rejected outright")
	}
	if p.GasLimit == 0 {
		return nil, errors.New("gasLimit is required: it is frozen, so there is no default to fall back on")
	}
	denom, elasticity := uint64(p.EIP1559Params[0])<<24|uint64(p.EIP1559Params[1])<<16|
		uint64(p.EIP1559Params[2])<<8|uint64(p.EIP1559Params[3]),
		uint64(p.EIP1559Params[4])<<24|uint64(p.EIP1559Params[5])<<16|
			uint64(p.EIP1559Params[6])<<8|uint64(p.EIP1559Params[7])
	if denom == 0 || elasticity == 0 {
		return nil, fmt.Errorf("eip1559Params %x must encode a non-zero denominator and elasticity: "+
			"a forced block's extraData repeats them verbatim and a zero denominator makes the next "+
			"block's base fee a division by zero", p.EIP1559Params)
	}

	seqWindow := p.SeqWindowSize
	if seqWindow == 0 {
		seqWindow = DefaultSeqWindowSize
	}
	genesisActive := new(uint64) // every fork active from genesis

	cfg := &rollup.Config{
		Genesis: rollup.Genesis{
			L1:     p.L1Genesis,
			L2:     p.L2Genesis,
			L2Time: p.L2Time,
			SystemConfig: eth.SystemConfig{
				BatcherAddr:   p.BatcherAddr,
				GasLimit:      p.GasLimit,
				EIP1559Params: p.EIP1559Params,
				MinBaseFee:    p.MinBaseFee,
				Scalar:        EcotoneScalar(p.BaseFeeScalar, p.BlobBaseFeeScalar),
			},
		},
		BlockTime:         p.BlockTime,
		MaxSequencerDrift: p.MaxSequencerDrift,
		SeqWindowSize:     seqWindow,
		// The channel timeout is irrelevant in fact — a proof batch is one transaction, so a channel
		// never spans L1 blocks and never times out — but it must be set, and it must not be larger
		// than the sequencing window or a reset's lookback would exceed the window it is protecting.
		ChannelTimeoutBedrock:  300,
		L1ChainID:              p.L1ChainID,
		L2ChainID:              p.L2ChainID,
		BatchInboxAddress:      p.BatchInbox,
		DepositContractAddress: p.DepositContract,
		L1SystemConfigAddress:  p.SystemConfigProxy,
		RegolithTime:           genesisActive,
		CanyonTime:             genesisActive,
		DeltaTime:              genesisActive,
		EcotoneTime:            genesisActive,
		FjordTime:              genesisActive,
		GraniteTime:            genesisActive,
		HoloceneTime:           genesisActive,
		IsthmusTime:            genesisActive,
		JovianTime:             genesisActive,
		KarstTime:              genesisActive,
		LagoonTime:             genesisActive,
		ChainOpConfig: &coreparams.OptimismConfig{
			EIP1559Denominator:       uint64(denom),
			EIP1559Elasticity:        uint64(elasticity),
			EIP1559DenominatorCanyon: &denom,
		},
	}
	// Amsterdam is not reachable from a rollup config at all — it has no *_time field here — so G2 F3's
	// concern is structurally out of scope rather than something this generator has to suppress.
	// Karst and Lagoon ARE active at genesis, which matters twice over: it matches G1's
	// Lagoon-at-genesis reading of chain P, and it keeps their upgrade bundles (the only ones that
	// carry non-zero upgrade gas) out of every post-genesis block.

	// The silhouette invariants run FIRST: they are the specific, actionable failures, and letting
	// the generic stock Check speak first would report "missing scalar" for a config whose real
	// problem is a sequencing window three times too long.
	if err := checkSilhouetteInvariants(cfg); err != nil {
		return nil, err
	}
	if err := cfg.Check(); err != nil {
		return nil, fmt.Errorf("generated silhouette rollup config is invalid: %w", err)
	}
	return cfg, nil
}

// checkSilhouetteInvariants asserts the properties the rest of the stack relies on. They are checked
// rather than commented because each one, if violated, breaks something far away: a stock derivation
// rule, the forced-extension convention, or a prover's headers-only L1 walk.
func checkSilhouetteInvariants(cfg *rollup.Config) error {
	// FINITE and short (DR-2). An infinite or very long window means a dead prover stalls P for that
	// long; the whole forced-extension convention exists to bound it.
	if cfg.SeqWindowSize == 0 {
		return errors.New("sequencing window must be finite")
	}
	if cfg.SeqWindowSize > 2*3600/12 {
		return fmt.Errorf("sequencing window %d L1 blocks is longer than the ~1-2h DR-2 asks for",
			cfg.SeqWindowSize)
	}
	// A reset's lookback must stay inside the window, or the rewind in G2 D5 could leave an orphaned
	// carrier below the reset base.
	if cfg.ChannelTimeoutBedrock > cfg.SeqWindowSize {
		return fmt.Errorf("channel timeout %d exceeds the sequencing window %d",
			cfg.ChannelTimeoutBedrock, cfg.SeqWindowSize)
	}
	// Every fork active at genesis: this is what removes activation blocks, and an activation block
	// would break the single-transaction shape of a forced block (G2 D2.1.2).
	for name, t := range map[string]*uint64{
		"regolith": cfg.RegolithTime, "canyon": cfg.CanyonTime, "delta": cfg.DeltaTime,
		"ecotone": cfg.EcotoneTime, "fjord": cfg.FjordTime, "granite": cfg.GraniteTime,
		"holocene": cfg.HoloceneTime, "isthmus": cfg.IsthmusTime, "jovian": cfg.JovianTime,
		"karst": cfg.KarstTime, "lagoon": cfg.LagoonTime,
	} {
		if t == nil || *t != 0 {
			return fmt.Errorf("%s must be active from genesis", name)
		}
	}
	// Epochs must advance: an L2 block time at or above the L1 block time would make the greedy
	// origin rule adopt every L1 block, and a block time that does not divide the L1 one makes the
	// forced-extension fill counts uneven between epochs. Neither is fatal, but a silhouette chain
	// has no reason to want either.
	if cfg.BlockTime == 0 || cfg.BlockTime >= 12 {
		return fmt.Errorf("block time %d should be well below the L1 block time", cfg.BlockTime)
	}
	return nil
}
