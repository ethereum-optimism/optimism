package proofs

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ethereum-optimism/optimism/op-e2e/actions"
	"github.com/ethereum-optimism/optimism/op-program/client/claim"
	"github.com/ethereum-optimism/optimism/op-program/host/config"
	"github.com/ethereum/go-ethereum/common"
	"github.com/naoina/toml"
	"github.com/stretchr/testify/require"
)

var (
	dumpFixtures = false
	fixtureDir   string
)

func init() {
	fixtureDir = os.Getenv("OP_E2E_FPP_FIXTURE_DIR")
	if fixtureDir != "" {
		dumpFixtures = true
	}
}

type TestFixture struct {
	Name           string        `toml:"name"`
	ExpectedStatus uint8         `toml:"expected-status"`
	Inputs         FixtureInputs `toml:"inputs"`
}

type FixtureInputs struct {
	L2BlockNumber uint64      `toml:"l2-block-number"`
	L2Claim       common.Hash `toml:"l2-claim"`
	L2Head        common.Hash `toml:"l2-head"`
	L2OutputRoot  common.Hash `toml:"l2-output-root"`
	L2ChainID     uint64      `toml:"l2-chain-id"`
	L1Head        common.Hash `toml:"l1-head"`
}

// Dumps a `fp-tests` test fixture to disk if the `OP_E2E_DUMP_FIXTURES` environment variable is set.
//
// [fp-tests]: https://github.com/ethereum-optimism/fp-tests
func tryDumpTestFixture(t actions.Testing, result error, name string, env *L2FaultProofEnv, programCfg *config.Config) {
	if !dumpFixtures {
		return
	}

	name = convertToKebabCase(name)
	rollupCfg := env.sd.RollupCfg
	l2Genesis := env.sd.L2Cfg

	var expectedStatus uint8
	if result == nil {
		expectedStatus = 0
	} else if errors.Is(result, claim.ErrClaimNotValid) {
		expectedStatus = 1
	} else {
		expectedStatus = 2
	}

	fixture := TestFixture{
		Name:           name,
		ExpectedStatus: expectedStatus,
		Inputs: FixtureInputs{
			L2BlockNumber: programCfg.L2ClaimBlockNumber,
			L2Claim:       programCfg.L2Claim,
			L2Head:        programCfg.L2Head,
			L2OutputRoot:  programCfg.L2OutputRoot,
			L2ChainID:     env.sd.RollupCfg.L2ChainID.Uint64(),
			L1Head:        programCfg.L1Head,
		},
	}

	fixturePath := filepath.Join(fixtureDir, name)

	err := os.MkdirAll(filepath.Join(fixturePath), fs.ModePerm)
	require.NoError(t, err, "failed to create fixture dir")

	fixtureFilePath := filepath.Join(fixturePath, "fixture.toml")
	serFixture, err := toml.Marshal(fixture)
	require.NoError(t, err, "failed to serialize fixture")
	require.NoError(t, os.WriteFile(fixtureFilePath, serFixture, fs.ModePerm), "failed to write fixture")

	genesisPath := filepath.Join(fixturePath, "genesis.json")
	serGenesis, err := l2Genesis.MarshalJSON()
	require.NoError(t, err, "failed to serialize genesis")
	require.NoError(t, os.WriteFile(genesisPath, serGenesis, fs.ModePerm), "failed to write genesis")

	rollupPath := filepath.Join(fixturePath, "rollup.json")
	serRollup, err := json.Marshal(rollupCfg)
	require.NoError(t, err, "failed to serialize rollup")
	require.NoError(t, os.WriteFile(rollupPath, serRollup, fs.ModePerm), "failed to write rollup")

	// Copy the witness database into the fixture directory.
	cmd := exec.Command("cp", "-r", programCfg.DataDir, filepath.Join(fixturePath, "witness-db"))
	require.NoError(t, cmd.Run(), "Failed to copy witness DB")

	// Compress the genesis file.
	cmd = exec.Command("zstd", genesisPath)
	_ = cmd.Run()
	require.NoError(t, os.Remove(genesisPath), "Failed to remove uncompressed genesis file")

	// Compress the witness database.
	cmd = exec.Command(
		"tar",
		"--zstd",
		"-cf",
		filepath.Join(fixturePath, "witness-db.tar.zst"),
		filepath.Join(fixturePath, "witness-db"),
	)
	cmd.Dir = filepath.Join(fixturePath)
	require.NoError(t, cmd.Run(), "Failed to compress witness DB")
	require.NoError(t, os.RemoveAll(filepath.Join(fixturePath, "witness-db")), "Failed to remove uncompressed witness DB")
}

// Convert to lower kebab case for strings containing `/`
func convertToKebabCase(input string) string {
	if !strings.Contains(input, "/") {
		return input
	}

	// Replace non-alphanumeric characters with underscores
	re := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	snake := re.ReplaceAllString(input, "-")

	// Convert to lower case
	return strings.ToLower(snake)
}
