package helpers

import (
	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

type TestFixture struct {
	Name           string        `toml:"name"`
	ExpectedStatus uint8         `toml:"expected-status"`
	Inputs         FixtureInputs `toml:"inputs"`
}

type FaultProofProgramL2Source struct {
	Node        *helpers.L2Verifier
	Engine      *helpers.L2Engine
	ChainConfig *params.ChainConfig
}

type FixtureInputs struct {
	L2BlockNumber uint64      `toml:"l2-block-number"`
	L2Claim       common.Hash `toml:"l2-claim"`
	L2Head        common.Hash `toml:"l2-head"`
	L2OutputRoot  common.Hash `toml:"l2-output-root"`
	L2ChainID     eth.ChainID `toml:"l2-chain-id"`
	L1Head        common.Hash `toml:"l1-head"`

	// AgreedPrestate and ClaimTimestamp address the transition for the interop client program,
	// which is timestamp- rather than block-addressed: AgreedPrestate is the encoded PreState (a
	// super root, or a transition state part-way through a timestamp) and ClaimTimestamp is the
	// super-root timestamp the claim commits to. L2Head/L2OutputRoot/L2BlockNumber address the
	// same transition for the single-chain programs (now only the kona-sp1 range guest).
	AgreedPrestate []byte `toml:"agreed-prestate"`
	ClaimTimestamp uint64 `toml:"claim-timestamp"`

	L2Sources []*FaultProofProgramL2Source

	// L2RPCTracker is an optional observer for L2 JSON-RPC calls made by the host.
	// It is not serialized as part of the test fixture inputs.
	L2RPCTracker *L2RPCTracker `toml:"-"`

	// CorruptClaim, when set, instructs the SP1 range executor to tamper the claimed output root in
	// the generated witness so the guest rejects it (the invalid-claim soundness path). Ignored by
	// the native fault-proof program. Not part of the serialized fixture.
	CorruptClaim bool `toml:"-"`

	// SP1NativeCore, when set, instructs the SP1 range executor to generate the witness and then run
	// the shared range-program core natively instead of executing the SP1 ELF. Ignored by the native
	// fault-proof program. Not part of the serialized fixture.
	SP1NativeCore bool `toml:"-"`
}
