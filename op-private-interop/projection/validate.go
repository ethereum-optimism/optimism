// Package projection validates complete public projection ranges before execution.
// Admission is pure; the separate ContextCollector resolves its explicit canonical inputs.
package projection

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"slices"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-private-interop/codec"
	"github.com/ethereum-optimism/optimism/op-private-interop/wire"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	// InsecureStub accepts any proof bytes. Test-gated (§B.5); retired from production.
	InsecureStub = "insecure-stub-v1"
	// ExecutionMock accepts the forgeable native-execution envelope. Test-gated (§B.5).
	ExecutionMock = "execution-mock-v1"
	// SP1PrivateProjectionV1 is the production profile: an SP1 Groth16 proof (circuit v6.1.0) of
	// the private-projection relation over PublicValuesV1.
	SP1PrivateProjectionV1 = "sp1-private-projection-v1"
	MaxRangeBlocks         = 65536
	MaxTxGas               = wire.MaxTxGas
)

// Config is consensus configuration, not an operator-selected verifier override.
type Config struct {
	Verifier          string      `json:"verifier"`
	AllowEvents       bool        `json:"allow_events,omitempty"`
	GenesisOutputRoot common.Hash `json:"genesis_output_root"`
	ProgramVKey       common.Hash `json:"program_vkey"`
	PrivateConfigHash common.Hash `json:"private_config_hash"`
	DependencySetHash common.Hash `json:"dependency_set_hash"`
	MockProofs        bool        `json:"mock_proofs,omitempty"`
}

// Check applies the chain-independent field rules of §B.1. CheckChain adds the §B.5 test gate.
func (c *Config) Check() error {
	if c == nil {
		return fmt.Errorf("unsupported projection verifier")
	}
	switch c.Verifier {
	case SP1PrivateProjectionV1:
		switch {
		case c.GenesisOutputRoot == (common.Hash{}):
			return fmt.Errorf("projection genesis output root is zero")
		case c.ProgramVKey == (common.Hash{}) || !isBN254Scalar(c.ProgramVKey):
			return fmt.Errorf("projection program vkey is zero or not a BN254 scalar")
		case c.PrivateConfigHash == (common.Hash{}):
			return fmt.Errorf("projection private config hash is zero")
		case c.DependencySetHash == (common.Hash{}):
			return fmt.Errorf("projection dependency set hash is zero")
		case c.AllowEvents:
			return fmt.Errorf("sp1-private-projection-v1 does not support event replays")
		}
	case InsecureStub, ExecutionMock:
		switch {
		case c.GenesisOutputRoot == (common.Hash{}):
			return fmt.Errorf("projection genesis output root is zero")
		case c.DependencySetHash == (common.Hash{}):
			return fmt.Errorf("projection dependency set hash is zero")
		case c.ProgramVKey != (common.Hash{}) || c.PrivateConfigHash != (common.Hash{}) || c.MockProofs:
			return fmt.Errorf("%s requires zero program vkey and private config hash and no mock proofs", c.Verifier)
		}
	default:
		return fmt.Errorf("unsupported projection verifier")
	}
	return nil
}

// CheckChain is Check plus the §B.5 gate: a test-gated mode (stub, execution mock, SP1 with mock
// envelopes) passes only in a binary with the test verifiers compiled in (build tag
// private_interop_test_verifiers, or a go test binary) and only on an allowlisted chain ID.
func (c *Config) CheckChain(chainID *big.Int) error {
	return c.checkChain(chainID, testVerifiersCompiledOrTesting())
}

func (c *Config) checkChain(chainID *big.Int, compiledOrTesting bool) error {
	if err := c.Check(); err != nil {
		return err
	}
	if c.testGated() && !gateAllows(compiledOrTesting, chainID) {
		return fmt.Errorf("projection verifier mode %q (mock proofs %v) is test-gated and not enabled for chain %v", c.Verifier, c.MockProofs, chainID)
	}
	return nil
}

// Range is the decoded full span, including any already-canonical overlap.
type Range interface {
	GetBlockCount() int
	GetBlockTimestamp(int) uint64
	GetBlockEpochNum(int) uint64
	GetBlockTransactions(int) []hexutil.Bytes
}

// Context is the derivation-side view admission binds the claim to. GenesisHash is the
// projection's genesis L2 hash; L1Head is the hash of the L1 block whose number is the span's last
// epoch, taken from the node's own L1 window.
type Context struct {
	ChainID                               *big.Int
	GenesisNumber, GenesisTime, BlockTime uint64
	GenesisHash, ParentHash, L1Head       common.Hash
	Continuation                          Continuation
}

// Statement commits to the public records, not to proof-dependent signatures.
// Claim.Proof is always nil. Continuation binds the surviving checkpoint and
// canonical recovery inputs; a real proof must verify their private execution.
// PublicValues(statement) is the exact sp1-private-projection-v1 public input.
type Statement struct {
	ChainID        common.Hash
	ParentHash     common.Hash
	ProjectionHash common.Hash
	Continuation   Continuation
	Claim          codec.RangeClaim

	ProjectionConfigHash common.Hash
	PrivateConfigHash    common.Hash
	OutputsRoot          common.Hash
	MessagesRoot         common.Hash
	TerminalOutput       common.Hash
}

// ProofVerifier implementations must be pure: no I/O, mutation of inputs, clock,
// randomness, or mutable state. Production selection is fixed by consensus config.
type ProofVerifier interface{ Verify(Statement, []byte) error }

// StubVerifier deliberately provides no cryptographic execution guarantee.
type StubVerifier struct{}

func (StubVerifier) Verify(Statement, []byte) error { return nil }

// VerifierFor returns the consensus verifier for c on the projection chain chainID, after
// CheckChain. Mock SP1 envelopes are accepted only when mock_proofs is set and the §B.5 gate is open.
func VerifierFor(c *Config, chainID *big.Int) (ProofVerifier, error) {
	return verifierFor(c, chainID, testVerifiersCompiledOrTesting())
}

func verifierFor(c *Config, chainID *big.Int, compiledOrTesting bool) (ProofVerifier, error) {
	if err := c.checkChain(chainID, compiledOrTesting); err != nil {
		return nil, err
	}
	switch c.Verifier {
	case SP1PrivateProjectionV1:
		return SP1Verifier{cfg: *c, allowMock: c.MockProofs && gateAllows(compiledOrTesting, chainID)}, nil
	case ExecutionMock:
		return ExecutionMockVerifier{}, nil
	case InsecureStub:
		return StubVerifier{}, nil
	}
	return nil, fmt.Errorf("unsupported projection verifier")
}

// ValidateProjectionRange checks every block before invoking the proof verifier.
// It never releases a valid prefix of a structurally invalid range.
func ValidateProjectionRange(c *Config, ctx Context, span Range, verifier ProofVerifier) (*Statement, error) {
	if err := c.CheckChain(ctx.ChainID); err != nil {
		return nil, err
	}
	if verifier == nil || ctx.ChainID == nil || ctx.ChainID.Sign() <= 0 || ctx.ChainID.BitLen() > 256 || ctx.BlockTime == 0 || ctx.L1Head == (common.Hash{}) {
		return nil, fmt.Errorf("invalid projection context or verifier")
	}
	first, _, err := RangeBounds(ctx.GenesisNumber, ctx.GenesisTime, ctx.BlockTime, span)
	if err != nil {
		return nil, err
	}
	count, start := span.GetBlockCount(), span.GetBlockTimestamp(0)
	if ctx.Continuation.Anchor.Number >= first || ctx.Continuation.Anchor.Hash == (common.Hash{}) || ctx.Continuation.OutputRoot == (common.Hash{}) {
		return nil, fmt.Errorf("invalid authenticated private checkpoint")
	}
	configHash := ConfigHash(c, ctx)
	var leaves, outputLeaves, messageLeaves []common.Hash
	var terminalOutput common.Hash
	var claim *codec.RangeClaim
	var transcript bytes.Buffer
	put := func(n uint64) { var b [8]byte; binary.BigEndian.PutUint64(b[:], n); transcript.Write(b[:]) }
	signer := types.LatestSignerForChainID(ctx.ChainID)
	for i := 0; i < count; i++ {
		if span.GetBlockTimestamp(i) != start+uint64(i)*ctx.BlockTime {
			return nil, fmt.Errorf("noncontiguous projection timestamps")
		}
		blockStart := transcript.Len()
		put(first + uint64(i))
		put(span.GetBlockTimestamp(i))
		put(span.GetBlockEpochNum(i))
		txs := span.GetBlockTransactions(i)
		if i == 0 && len(txs) == 0 {
			return nil, fmt.Errorf("missing opening claim")
		}
		put(uint64(len(txs)))
		outputSeen := false
		outputPosition := 0
		// replayIndex is the rendered log index of the next replay: every replay emits exactly
		// one log (§E) and claim/output records emit none.
		var replayIndex uint32
		if i == 0 {
			outputPosition = 1
		}
		for j, raw := range txs {
			var tx types.Transaction
			if len(raw) > wire.MaxMessageBytes+4096 {
				return nil, fmt.Errorf("projection transaction too large")
			}
			if err := tx.UnmarshalBinary(raw); err != nil {
				return nil, fmt.Errorf("decode projection transaction: %w", err)
			}
			canonical, err := tx.MarshalBinary()
			if err != nil || !bytes.Equal(canonical, raw) {
				return nil, fmt.Errorf("noncanonical transaction")
			}
			if tx.Type() != types.DynamicFeeTxType || tx.To() == nil || tx.ChainId().Cmp(ctx.ChainID) != 0 || tx.Value().Sign() != 0 || tx.GasFeeCap().Sign() != 0 || tx.GasTipCap().Sign() != 0 || tx.Gas() == 0 || tx.Gas() > MaxTxGas {
				return nil, fmt.Errorf("invalid projection transaction envelope")
			}
			sender, err := types.Sender(signer, &tx)
			if err != nil {
				return nil, fmt.Errorf("invalid projection signature: %w", err)
			}
			data := tx.Data()
			var expected types.AccessList
			switch *tx.To() {
			case predeploys.ClaimRegistryAddr:
				if len(data) >= 4 && bytes.Equal(data[:4], wire.OutputSelector) {
					if outputSeen || j != outputPosition {
						return nil, fmt.Errorf("duplicate or misplaced private output")
					}
					root, err := wire.DecodeOutput(data)
					if err != nil {
						return nil, err
					}
					outputLeaves = append(outputLeaves, OutputLeaf(first+uint64(i), root))
					terminalOutput = root
					outputSeen = true
					break
				}
				if i != 0 || j != 0 {
					return nil, fmt.Errorf("duplicate or misplaced claim")
				}
				claim, err = wire.DecodeClaim(data)
				if err != nil {
					return nil, err
				}
				expected := ctx.Continuation
				if claim.AnchorBlock != expected.Anchor.Number || claim.AnchorOutputRoot != expected.OutputRoot || claim.RecoveryHash != expected.RecoveryHash || claim.ParentOutputRoot == (common.Hash{}) {
					return nil, fmt.Errorf("claim does not match canonical continuation")
				}
				if expected.Anchor.Number == first-1 {
					if expected.Anchor.Hash != ctx.ParentHash || expected.RecoveryHash != (common.Hash{}) || claim.ParentOutputRoot != expected.OutputRoot {
						return nil, fmt.Errorf("private parent differs from canonical checkpoint")
					}
				} else if expected.RecoveryHash == (common.Hash{}) {
					return nil, fmt.Errorf("missing canonical recovery inputs")
				}
				if claim.FirstBlock != first || claim.LastBlock != first+uint64(count-1) {
					return nil, fmt.Errorf("claim does not cover exactly the span")
				}
				if claim.L1Head != ctx.L1Head {
					return nil, fmt.Errorf("claim l1Head does not match the span's canonical L1 origin")
				}
				if claim.RollupConfigHash != configHash {
					return nil, fmt.Errorf("claim rollupConfigHash does not match the projection config")
				}
				if claim.DepSetHash != c.DependencySetHash {
					return nil, fmt.Errorf("claim depSetHash does not match the consensus dependency set")
				}
				normalized := *claim
				normalized.Proof = nil
				data, err = wire.EncodePostClaim(&normalized)
				if err != nil {
					return nil, err
				}
			case predeploys.L2toL2CrossDomainMessengerAddr:
				m, err := wire.DecodeReplaySentMessage(data)
				if err != nil {
					return nil, err
				}
				if m.Sender == predeploys.SuperchainETHBridgeAddr || m.Target == predeploys.SuperchainETHBridgeAddr {
					return nil, fmt.Errorf("private native bridge replay is forbidden")
				}
				messageLeaves = append(messageLeaves, MessageLeaf(first+uint64(i), replayIndex, MessageKindInit, ExportMessageHash(m)))
				replayIndex++
			case predeploys.CrossL2InboxAddr:
				m, err := wire.DecodeValidateMessage(data)
				if err != nil {
					return nil, err
				}
				expected = types.AccessList{{Address: predeploys.CrossL2InboxAddr, StorageKeys: messages.EncodeAccessList([]messages.Access{m.Access()})}}
				messageLeaves = append(messageLeaves, MessageLeaf(first+uint64(i), replayIndex, MessageKindExec, ImportMessageHash([192]byte(data[4:196]))))
				replayIndex++
			case predeploys.EventReplayerAddr:
				if !c.AllowEvents {
					return nil, fmt.Errorf("extra event replay is disabled")
				}
				if err := wire.CheckReplayEvent(data); err != nil {
					return nil, err
				}
				// A generic event consumes a rendered index but has no v1 message leaf; the
				// sp1 profile forbids allow_events.
				replayIndex++
			default:
				return nil, fmt.Errorf("unexpected projection destination")
			}
			if i == 0 && j == 0 && claim == nil {
				return nil, fmt.Errorf("opening transaction is not a claim")
			}
			al := tx.AccessList()
			if len(al) != len(expected) {
				return nil, fmt.Errorf("unexpected projection access list")
			}
			for k := range al {
				if al[k].Address != expected[k].Address || !slices.Equal(al[k].StorageKeys, expected[k].StorageKeys) {
					return nil, fmt.Errorf("incorrect import checksum access list")
				}
			}
			transcript.Write(sender.Bytes())
			put(tx.Nonce())
			put(tx.Gas())
			transcript.Write(tx.To().Bytes())
			put(uint64(len(data)))
			transcript.Write(data)
		}
		if !outputSeen {
			return nil, fmt.Errorf("missing private output record")
		}
		leaves = append(leaves, crypto.Keccak256Hash([]byte{0}, transcript.Bytes()[blockStart:]))
	}
	if claim == nil {
		return nil, fmt.Errorf("missing claim")
	}
	statement := &Statement{
		ChainID: common.BigToHash(ctx.ChainID), ParentHash: ctx.ParentHash, ProjectionHash: RecordsRoot(leaves),
		Continuation: ctx.Continuation, Claim: *claim,
		ProjectionConfigHash: configHash, PrivateConfigHash: c.PrivateConfigHash,
		OutputsRoot: CommitmentRoot(OutputsDomain, outputLeaves), MessagesRoot: CommitmentRoot(MessagesDomain, messageLeaves),
		TerminalOutput: terminalOutput,
	}
	statement.Claim.Proof = nil
	if err := verifier.Verify(*statement, claim.Proof); err != nil {
		return nil, fmt.Errorf("projection proof: %w", err)
	}
	return statement, nil
}

// RecordsRoot authenticates every ordered block record, including intermediate
// private outputs. Leaves use H(0x00 || canonicalBlock); nodes H(0x01 || left ||
// right), duplicating the last node at odd levels. The leaf count is committed to
// separately so duplication cannot alias spans of different lengths.
func RecordsRoot(leaves []common.Hash) common.Hash {
	var count [8]byte
	binary.BigEndian.PutUint64(count[:], uint64(len(leaves)))
	level := slices.Clone(leaves)
	if len(level) == 0 {
		return common.Hash{}
	}
	for len(level) > 1 {
		next := make([]common.Hash, 0, (len(level)+1)/2)
		for i := 0; i < len(level); i += 2 {
			right := min(i+1, len(level)-1)
			next = append(next, crypto.Keccak256Hash([]byte{1}, level[i][:], level[right][:]))
		}
		level = next
	}
	return crypto.Keccak256Hash([]byte("optimism.private-projection.v2\x00"), count[:], level[0][:])
}

// RangeBounds rejects malformed geometry before derivation performs any parent
// lookup. In particular, genesis cannot underflow to an unresolvable parent.
func RangeBounds(genesisNumber, genesisTime, blockTime uint64, span Range) (uint64, uint64, error) {
	count := span.GetBlockCount()
	if count < 1 || count > MaxRangeBlocks || blockTime == 0 {
		return 0, 0, fmt.Errorf("invalid projection range length or block time")
	}
	start := span.GetBlockTimestamp(0)
	if start < genesisTime || (start-genesisTime)%blockTime != 0 {
		return 0, 0, fmt.Errorf("unaligned projection range")
	}
	offset := (start - genesisTime) / blockTime
	if offset == 0 || offset > math.MaxUint64-genesisNumber {
		return 0, 0, fmt.Errorf("invalid first projection height")
	}
	first := genesisNumber + offset
	if uint64(count-1) > math.MaxUint64-first || uint64(count-1) > (math.MaxUint64-start)/blockTime {
		return 0, 0, fmt.Errorf("projection range overflow")
	}
	for i := 0; i < count; i++ {
		if span.GetBlockTimestamp(i) != start+uint64(i)*blockTime {
			return 0, 0, fmt.Errorf("noncontiguous projection timestamps")
		}
	}
	return first, first + uint64(count-1), nil
}
