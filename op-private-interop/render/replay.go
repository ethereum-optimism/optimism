package render

import (
	"crypto/ecdsa"
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-private-interop/codec"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
)

// ExportGasPerMessageByte covers both calldata (up to 16 gas/byte) and LOG data (8 gas/byte),
// with a conservative four-gas margin for hashing and memory expansion. GasLimitExport is the
// fixed base; this coefficient makes the complete limit a deterministic function of the payload.
const ExportGasPerMessageByte uint64 = 28

// ReplayTxBuilder turns replay actions into the standard batcher's signed transactions.
//
// It is an interface for two reasons. The contracts it calls are still being finalised by the
// sibling lane, so the encodings in abi.go are provisional; and the standard batcher key lives behind
// whatever signer the deployment uses, which a pure package must not know about. Everything about
// WHICH transactions exist and in what order is decided by RenderBlock and is not negotiable here.
//
// # Determinism contract
//
// An implementation MUST be a pure function of (the nonce given to Reset, the sequence of calls
// made since). No clock, no fee oracle, no random nonce: the rendering's transactions are consensus
// data, and a builder that prices a transaction from a live gas oracle produces a different chain
// every time it runs. Fees come from the frozen configuration the rendering's genesis already
// encodes.
type ReplayTxBuilder interface {
	// Reset positions the builder at the batcher EOA's nonce for the FIRST transaction of a range.
	//
	// The builder is nonce-sequential from there. Making the starting point an explicit input, rather
	// than something the builder discovers from a node, is what keeps a range's bytes reproducible:
	// re-running a range from the same starting nonce reproduces it exactly.
	Reset(nonce uint64)
	// Nonce reports the nonce the next transaction will use.
	Nonce() uint64
	// ReplayTx builds one replay transaction: an export re-emission, an import execution, or a
	// generic log re-emission.
	ReplayTx(act ReplayAction) (*types.Transaction, error)
	// ClaimTx builds the transaction posting a range's OWN claim to the ClaimRegistry. It is the
	// FIRST transaction of the FIRST block of the range it describes. The registry is log-less, so
	// leading placement shifts nothing: every message's rendering log index still equals its
	// RenderedLogs rank, in range-opening blocks as everywhere else.
	ClaimTx(claim *codec.RangeClaim) (*types.Transaction, error)
}

// GasPolicy is the frozen pricing the public projection's transactions use.
//
// Frozen, not observed: these are configuration, identical on every run, and the public projection
// has no fee market to observe — it has no mempool and no sequencer, and the batcher is the only sender.
type GasPolicy struct {
	// GasLimitExport, GasLimitImport, GasLimitEvent and GasLimitClaim are per-kind gas limits. A
	// single limit would have to be the maximum of all four, and the public projection pays for what it
	// declares.
	GasLimitExport uint64
	GasLimitImport uint64
	GasLimitEvent  uint64
	GasLimitClaim  uint64
}

// DefaultGasPolicy is a starting point, not a measurement. The numbers are deliberately generous:
// the batcher is the only sender on a chain with no fee competition, and an under-provisioned
// replay transaction is a stuck public projection. These values should be replaced with measured
// costs once replay-transaction execution is exercised on a derived chain.
func DefaultGasPolicy() GasPolicy {
	return GasPolicy{
		GasLimitExport: 500_000,
		GasLimitImport: 500_000,
		GasLimitEvent:  500_000,
		GasLimitClaim:  500_000,
	}
}

// SignerFn signs a public-projection transaction. It must be deterministic; go-ethereum's secp256k1 signing
// is (RFC 6979), so a plain private-key signer qualifies and a remote signer that adds entropy does
// not.
type SignerFn func(tx *types.Transaction) (*types.Transaction, error)

// PrivateKeySigner is the deterministic signer used in tests and local-key deployments.
func PrivateKeySigner(key *ecdsa.PrivateKey, chainID *big.Int) SignerFn {
	signer := types.LatestSignerForChainID(chainID)
	return func(tx *types.Transaction) (*types.Transaction, error) {
		return types.SignTx(tx, signer, key)
	}
}

// BatcherTxBuilder is the default ReplayTxBuilder: zero-priced EIP-1559 transactions from the
// standard SystemConfig batcher account, nonced from an explicit starting point.
type BatcherTxBuilder struct {
	chainID *big.Int
	gas     GasPolicy
	sign    SignerFn
	nonce   uint64
	// eventReplayer is the genesis-assigned EventReplayer address. It is a field rather than a read
	// of the package variable because it is per-deployment configuration: the contract is deployed
	// into the rendering's genesis, so no address can be a constant here.
	eventReplayer common.Address
	// registry is the genesis-assigned ClaimRegistry address, for the same reason.
	registry common.Address
	// messenger is where the export replay implementation is installed. It defaults to the
	// L2ToL2CrossDomainMessenger predeploy, which is the ONLY address a valid export can render at:
	// a re-emitted SentMessage must carry the emitter every stock consumer expects, and one at any
	// other address is a publicly consumable message with a broken identity. It is a field rather
	// than the constant so that a deployment which moved it can say so, and so that a zero value is
	// caught here rather than by a reverting transaction on a live rendering.
	messenger common.Address
}

var _ ReplayTxBuilder = (*BatcherTxBuilder)(nil)

// NewBatcherTxBuilder builds the default replay-transaction builder.
func NewBatcherTxBuilder(chainID *big.Int, gas GasPolicy, sign SignerFn) *BatcherTxBuilder {
	// eventReplayer and registry stay ZERO: they are per-deployment genesis addresses with no
	// defensible default, and every ReplayTx/ClaimTx path refuses a zero one. See abi.go.
	return &BatcherTxBuilder{
		chainID:   new(big.Int).Set(chainID),
		gas:       gas,
		sign:      sign,
		messenger: predeploys.L2toL2CrossDomainMessengerAddr,
	}
}

// SetEventReplayer sets the genesis-assigned EventReplayer address. Without it a ReplayEvent action
// cannot be built, which is deliberate: sending replayEvent to the zero address would produce a
// rendering block whose transaction reverts, and a reverting replay is a rendering that silently
// lost a log.
func (b *BatcherTxBuilder) SetEventReplayer(addr common.Address) { b.eventReplayer = addr }

// SetRegistry sets the genesis-assigned ClaimRegistry address.
func (b *BatcherTxBuilder) SetRegistry(addr common.Address) { b.registry = addr }

// SetReplayMessenger sets the address the export replay implementation is installed at. The
// constructor's default — the messenger predeploy — is the only value that renders valid exports;
// see the field.
func (b *BatcherTxBuilder) SetReplayMessenger(addr common.Address) { b.messenger = addr }

func (b *BatcherTxBuilder) Reset(nonce uint64) { b.nonce = nonce }
func (b *BatcherTxBuilder) Nonce() uint64      { return b.nonce }

func (b *BatcherTxBuilder) ReplayTx(act ReplayAction) (*types.Transaction, error) {
	switch act.Kind {
	case ReplayExport:
		if act.Export == nil {
			return nil, fmt.Errorf("export action at rendered index %d has no decoded SentMessage", act.RenderedLogIndex)
		}
		gasLimit, err := exportGasLimit(b.gas.GasLimitExport, len(act.Export.Message))
		if err != nil {
			return nil, fmt.Errorf("rendered index %d: %w", act.RenderedLogIndex, err)
		}
		data, err := EncodeReplaySentMessage(act.Export)
		if err != nil {
			return nil, fmt.Errorf("rendered index %d: %w", act.RenderedLogIndex, err)
		}
		if b.messenger == (common.Address{}) {
			return nil, fmt.Errorf("rendered index %d needs the replay messenger, whose address is not configured", act.RenderedLogIndex)
		}
		// To the messenger predeploy address, where the replay implementation is installed: the
		// re-emitted SentMessage must carry the emitter every stock consumer expects.
		return b.sendTx(b.messenger, data, nil, gasLimit)
	case ReplayEvent:
		if b.eventReplayer == (common.Address{}) {
			return nil, fmt.Errorf("rendered index %d needs EventReplayer, whose genesis address is not configured", act.RenderedLogIndex)
		}
		data, err := EncodeReplayEvent(act.Topics, act.Data)
		if err != nil {
			return nil, fmt.Errorf("rendered index %d: %w", act.RenderedLogIndex, err)
		}
		return b.sendTx(b.eventReplayer, data, nil, b.gas.GasLimitEvent)
	case ReplayImport:
		if act.Import == nil {
			return nil, fmt.Errorf("import action at rendered index %d has no decoded message", act.RenderedLogIndex)
		}
		// The checksum access list is NOT re-derived here. ExecTrigger is the in-tree encoder every
		// other executing-message sender already uses, and a second implementation of a checksum is
		// a second implementation that can drift.
		trigger := &txintent.ExecTrigger{Executor: predeploys.CrossL2InboxAddr, Msg: *act.Import}
		data, err := trigger.EncodeInput()
		if err != nil {
			return nil, fmt.Errorf("encoding validateMessage at rendered index %d: %w", act.RenderedLogIndex, err)
		}
		accessList, err := trigger.AccessList()
		if err != nil {
			return nil, fmt.Errorf("building access list at rendered index %d: %w", act.RenderedLogIndex, err)
		}
		return b.sendTx(predeploys.CrossL2InboxAddr, data, accessList, b.gas.GasLimitImport)
	default:
		return nil, fmt.Errorf("unknown replay kind %s", act.Kind)
	}
}

func exportGasLimit(base uint64, messageSize int) (uint64, error) {
	if messageSize > MaxRenderableMessageSize {
		return 0, fmt.Errorf(
			"SentMessage payload is %d bytes, exceeding the %d-byte rendering limit",
			messageSize, MaxRenderableMessageSize,
		)
	}
	extra := uint64(messageSize) * ExportGasPerMessageByte
	if base > math.MaxUint64-extra {
		return 0, fmt.Errorf("export gas limit overflows uint64")
	}
	return base + extra, nil
}

func (b *BatcherTxBuilder) ClaimTx(claim *codec.RangeClaim) (*types.Transaction, error) {
	if b.registry == (common.Address{}) {
		return nil, fmt.Errorf("the ClaimRegistry genesis address is not configured")
	}
	data, err := EncodePostClaim(claim)
	if err != nil {
		return nil, err
	}
	return b.sendTx(b.registry, data, nil, b.gas.GasLimitClaim)
}

func (b *BatcherTxBuilder) sendTx(to common.Address, data []byte, al types.AccessList, gasLimit uint64) (*types.Transaction, error) {
	if b.sign == nil {
		return nil, fmt.Errorf("no signer configured for the batcher EOA")
	}
	addr := to
	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:    new(big.Int).Set(b.chainID),
		Nonce:      b.nonce,
		GasTipCap:  new(big.Int),
		GasFeeCap:  new(big.Int),
		Gas:        gasLimit,
		To:         &addr,
		Value:      new(big.Int),
		Data:       data,
		AccessList: al,
	})
	signed, err := b.sign(tx)
	if err != nil {
		return nil, fmt.Errorf("signing rendering transaction with nonce %d: %w", b.nonce, err)
	}
	b.nonce++
	return signed, nil
}
