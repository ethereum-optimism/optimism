package deployer

import (
	"flag"
	"testing"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

const (
	testL1RPCUrl = "http://localhost:8545"
	testPrivKey  = "0000000000000000000000000000000000000000000000000000000000000001"
)

// newPrepareCtx builds a CLI context with the prepare flags applied and the
// given private key + a test L1 RPC URL set.
func newPrepareCtx(t *testing.T, privKey string) *cli.Context {
	t.Helper()

	app := cli.NewApp()
	flagSet := flag.NewFlagSet("test-prepare", flag.ContinueOnError)
	for _, f := range PrepareFlags {
		require.NoError(t, f.Apply(flagSet))
	}
	require.NoError(t, flagSet.Set(PrivateKeyFlagName, privKey))
	require.NoError(t, flagSet.Set(L1RPCURLFlagName, testL1RPCUrl))

	return cli.NewContext(app, flagSet, nil)
}

func TestNewPrepareConfig_FlagsPassed(t *testing.T) {
	cfg := newPrepareConfig(newPrepareCtx(t, testPrivKey), log.NewLogger(log.DiscardHandler()))
	require.Equal(t, testPrivKey, cfg.PrivateKey)
	require.Equal(t, testL1RPCUrl, cfg.L1RPCUrl)
}

func TestMakePredictionInput(t *testing.T) {
	opcmAddr := common.HexToAddress("0xaaaa000000000000000000000000000000000001")
	superchainConfig := common.HexToAddress("0xbbbb000000000000000000000000000000000002")
	salt := common.HexToHash("0xcccc000000000000000000000000000000000000000000000000000000000003")
	chainID := common.HexToHash("0x000000000000000000000000000000000000000000000000000000000000000a")

	intent := &state.Intent{
		OPCMAddress:           &opcmAddr,
		SuperchainConfigProxy: &superchainConfig,
	}
	st := &state.State{Create2Salt: salt}
	chain := &state.ChainIntent{ID: chainID} // GasLimit unset -> defaulted

	dci, err := makePredictionInput(intent, st, chain)
	require.NoError(t, err)

	// Committed values are passed through verbatim so the prediction matches the
	// eventual broadcast.
	require.Equal(t, opcmAddr, dci.Opcm)
	require.Equal(t, superchainConfig, dci.SuperchainConfig)
	require.Equal(t, salt.String(), dci.SaltMixer)
	require.Equal(t, chainID.Big(), dci.L2ChainId)
	require.Equal(t, standard.GasLimit, dci.GasLimit)

	// Roles are non-zero placeholders (they don't affect the predicted addresses,
	// but DeployOPChain.checkInput requires them set).
	for _, role := range []common.Address{
		dci.OpChainProxyAdminOwner, dci.SystemConfigOwner, dci.Batcher,
		dci.UnsafeBlockSigner, dci.Proposer, dci.Challenger,
	} {
		require.Equal(t, placeholderRole, role)
		require.NotEqual(t, common.Address{}, role)
	}
}

func TestMakePredictionInput_MissingRequiredAddresses(t *testing.T) {
	opcmAddr := common.HexToAddress("0xaaaa000000000000000000000000000000000001")
	superchainConfig := common.HexToAddress("0xbbbb000000000000000000000000000000000002")
	st := &state.State{Create2Salt: common.HexToHash("0x03")}
	chain := &state.ChainIntent{ID: common.HexToHash("0x0a")}

	_, err := makePredictionInput(&state.Intent{SuperchainConfigProxy: &superchainConfig}, st, chain)
	require.ErrorContains(t, err, "opcmAddress must be set")

	_, err = makePredictionInput(&state.Intent{OPCMAddress: &opcmAddr}, st, chain)
	require.ErrorContains(t, err, "superchainConfigProxy must be set")
}

func TestPrepareConfigCheck(t *testing.T) {
	valid := PrepareConfig{
		Workdir:    "/tmp",
		Logger:     log.NewLogger(log.DiscardHandler()),
		PrivateKey: testPrivKey,
		L1RPCUrl:   testL1RPCUrl,
	}
	require.NoError(t, valid.Check())

	missingKey := valid
	missingKey.PrivateKey = ""
	require.ErrorContains(t, missingKey.Check(), "private key must be specified")

	invalidKey := valid
	invalidKey.PrivateKey = "not-a-valid-key"
	require.ErrorContains(t, invalidKey.Check(), "failed to parse private key")

	missingL1RPC := valid
	missingL1RPC.L1RPCUrl = ""
	require.ErrorContains(t, missingL1RPC.Check(), "l1 RPC URL must be specified")

	validPrestate := valid
	validPrestate.Prestate = "0xabc123"
	require.NoError(t, validPrestate.Check())
	require.Equal(t, common.HexToHash("0xabc123"), validPrestate.prestate)

	zeroPrestate := valid
	zeroPrestate.Prestate = "0x0"
	require.ErrorContains(t, zeroPrestate.Check(), "prestate must be a non-zero hash")
}

func TestResolvePrestate(t *testing.T) {
	flagPrestate := common.HexToHash("0x1111")
	chainOverride := common.HexToHash("0x2222")
	globalOverride := common.HexToHash("0x3333")

	chain := func(override *common.Hash) *state.ChainIntent {
		c := &state.ChainIntent{ID: common.HexToHash("0x0a")}
		if override != nil {
			c.DeployOverrides = map[string]any{faultGameAbsolutePrestateKey: override.Hex()}
		}
		return c
	}
	intent := func(override *common.Hash) *state.Intent {
		i := &state.Intent{}
		if override != nil {
			i.GlobalDeployOverrides = map[string]any{faultGameAbsolutePrestateKey: override.Hex()}
		}
		return i
	}

	t.Run("flag takes precedence over overrides", func(t *testing.T) {
		got, err := resolvePrestate(flagPrestate, intent(&globalOverride), chain(&chainOverride))
		require.NoError(t, err)
		require.Equal(t, flagPrestate, got)
	})

	t.Run("chain override takes precedence over global", func(t *testing.T) {
		got, err := resolvePrestate(common.Hash{}, intent(&globalOverride), chain(&chainOverride))
		require.NoError(t, err)
		require.Equal(t, chainOverride, got)
	})

	t.Run("falls back to global override", func(t *testing.T) {
		got, err := resolvePrestate(common.Hash{}, intent(&globalOverride), chain(nil))
		require.NoError(t, err)
		require.Equal(t, globalOverride, got)
	})

	t.Run("returns zero when nothing is declared", func(t *testing.T) {
		got, err := resolvePrestate(common.Hash{}, intent(nil), chain(nil))
		require.NoError(t, err)
		require.Equal(t, common.Hash{}, got)
	})

	t.Run("rejects non-string override", func(t *testing.T) {
		i := &state.Intent{GlobalDeployOverrides: map[string]any{faultGameAbsolutePrestateKey: 123}}
		_, err := resolvePrestate(common.Hash{}, i, chain(nil))
		require.ErrorContains(t, err, "must be a hex string")
	})

	t.Run("rejects zero-hash override", func(t *testing.T) {
		i := &state.Intent{GlobalDeployOverrides: map[string]any{faultGameAbsolutePrestateKey: "0x0"}}
		_, err := resolvePrestate(common.Hash{}, i, chain(nil))
		require.ErrorContains(t, err, "must be a non-zero hash")
	})
}

func TestIsPermissionlessGameType(t *testing.T) {
	require.False(t, isPermissionlessGameType(1)) // PermissionedGameType
	require.False(t, isPermissionlessGameType(5)) // SuperPermissionedGameType
	require.True(t, isPermissionlessGameType(0))  // CannonGameType
	require.True(t, isPermissionlessGameType(8))  // CannonKonaGameType
}

func TestIsPermissionlessDeployment(t *testing.T) {
	chain := &state.ChainIntent{ID: common.HexToHash("0x0a")}

	t.Run("default is permissioned", func(t *testing.T) {
		got, err := isPermissionlessDeployment(&state.Intent{}, chain)
		require.NoError(t, err)
		require.False(t, got)
	})

	t.Run("global override to cannon is permissionless", func(t *testing.T) {
		intent := &state.Intent{GlobalDeployOverrides: map[string]any{"respectedGameType": uint32(0)}}
		got, err := isPermissionlessDeployment(intent, chain)
		require.NoError(t, err)
		require.True(t, got)
	})

	t.Run("chain override takes precedence over global", func(t *testing.T) {
		intent := &state.Intent{GlobalDeployOverrides: map[string]any{"respectedGameType": uint32(0)}}
		permissionedChain := &state.ChainIntent{
			ID:              common.HexToHash("0x0a"),
			DeployOverrides: map[string]any{"respectedGameType": uint32(1)},
		}
		got, err := isPermissionlessDeployment(intent, permissionedChain)
		require.NoError(t, err)
		require.False(t, got)
	})
}
