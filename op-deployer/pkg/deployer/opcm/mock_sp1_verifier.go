package opcm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-chain-ops/solc"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
)

// The MockSP1Verifier contract lives under test/. op-deployer/scripts/build-embedded-contracts.sh
// compiles it into the embedded artifacts; other contracts builds that skip test/ lack it.
const (
	MockSP1VerifierArtifact = "MockSP1Verifier.sol"
	MockSP1VerifierContract = "MockSP1Verifier"
)

// DeployMockSP1Verifier deploys the accept-all MockSP1Verifier into a script host and returns its
// address and init code. Development environments pass it as DeployImplementationsInput.SP1Verifier.
func DeployMockSP1Verifier(host *script.Host, deployer common.Address) (common.Address, []byte, error) {
	artifact, err := host.Artifacts().ReadArtifact(MockSP1VerifierArtifact, MockSP1VerifierContract)
	if err != nil {
		return common.Address{}, nil, fmt.Errorf("failed to read MockSP1Verifier artifact: %w", err)
	}
	initCode := artifact.Bytecode.Object
	verifier, err := host.Create(deployer, initCode)
	if err != nil {
		return common.Address{}, nil, fmt.Errorf("failed to deploy MockSP1Verifier: %w", err)
	}
	if verifier == (common.Address{}) {
		return common.Address{}, nil, fmt.Errorf("MockSP1Verifier deployment produced no contract address")
	}
	return verifier, initCode, nil
}

// DeployMockSP1VerifierGenesis puts the accept-all MockSP1Verifier into the genesis state of env, at
// the next CREATE address of its sender, and returns its address.
func DeployMockSP1VerifierGenesis(env *ForgeEnv) (common.Address, error) {
	if env.Genesis == nil {
		return common.Address{}, errors.New("cannot deploy genesis MockSP1Verifier without the genesis state")
	}
	if env.Artifacts == nil {
		return common.Address{}, errors.New("cannot deploy MockSP1Verifier without the contract artifacts")
	}
	artifact, err := (&foundry.ArtifactsFS{FS: env.Artifacts}).ReadArtifact(MockSP1VerifierArtifact, MockSP1VerifierContract)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to read MockSP1Verifier artifact: %w", err)
	}
	// The genesis gets the runtime code without running the init code, so refuse what the artifact
	// shows the init code doing: a constructor, immutables or state variables. A missing constructor
	// is the zero Method, whose String is empty; its Type is abi.Constructor all the same. The ABI
	// omits an inherited constructor, so MockSP1Verifier may inherit only interfaces.
	if artifact.ABI.Constructor.String() != "" || hasImmutables(artifact) || len(artifact.StorageLayout.Storage) != 0 {
		return common.Address{}, errors.New("MockSP1Verifier needs its init code: it has a constructor, immutables or storage")
	}
	return env.Genesis.Create(artifact.DeployedBytecode.Object), nil
}

// hasImmutables reports whether the runtime code of artifact has immutables to fill. Invalid
// references count as immutables, which refuses the code.
func hasImmutables(artifact *foundry.Artifact) bool {
	var refs solc.ImmutableReferences
	raw := artifact.DeployedBytecode.ImmutableReferences
	return len(raw) != 0 && (json.Unmarshal(raw, &refs) != nil || len(refs) != 0)
}

// DeployMockSP1VerifierLive sends a deployment of the accept-all MockSP1Verifier from the deployer
// of env to its live L1, and returns its address. Later scripts of env fork at or after its block.
func DeployMockSP1VerifierLive(env *ForgeEnv) (common.Address, error) {
	if env.Artifacts == nil {
		return common.Address{}, errors.New("cannot deploy MockSP1Verifier without the contract artifacts")
	}
	artifact, err := (&foundry.ArtifactsFS{FS: env.Artifacts}).ReadArtifact(MockSP1VerifierArtifact, MockSP1VerifierContract)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to read MockSP1Verifier artifact: %w", err)
	}
	ctx := env.Context
	if ctx == nil {
		ctx = context.Background()
	}
	key, client, chainID, err := dialDeployer(ctx, env)
	if err != nil {
		return common.Address{}, err
	}
	defer client.Close()
	from := crypto.PubkeyToAddress(key.PublicKey)
	logger := env.logger()

	_, nonce, err := waitForSettledNonce(ctx, logger, client, from, env.sentBlock)
	if err != nil {
		return common.Address{}, err
	}
	deploy := broadcaster.Broadcast{
		From:  from,
		Input: hexutil.Bytes(artifact.Bytecode.Object),
		Value: (*hexutil.U256)(uint256.NewInt(0)),
		Type:  broadcaster.BroadcastCreate,
		Nonce: nonce,
	}
	sentBlock, err := sendReplay(ctx, logger, client, key, chainID, nonce, []broadcaster.Broadcast{deploy}, env.ReceiptQueryInterval)
	env.sentBlock = max(env.sentBlock, sentBlock)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to deploy MockSP1Verifier: %w", err)
	}
	// The signer of sendReplay only signs at nonce.
	verifier := crypto.CreateAddress(from, nonce)
	code, err := client.CodeAt(ctx, verifier, new(big.Int).SetUint64(sentBlock))
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to read MockSP1Verifier code: %w", err)
	}
	if len(code) == 0 {
		return common.Address{}, fmt.Errorf("MockSP1Verifier deployment left no code at %s", verifier)
	}
	return verifier, nil
}
