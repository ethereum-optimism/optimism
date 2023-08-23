package cannon

import (
	"context"
	"embed"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/fault/types"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

//go:embed test_data
var testData embed.FS

func TestGet(t *testing.T) {
	dataDir, prestate := setupTestData(t)
	t.Run("ExistingProof", func(t *testing.T) {
		provider, generator := setupWithTestData(t, dataDir, prestate)
		value, err := provider.Get(context.Background(), 0)
		require.NoError(t, err)
		require.Equal(t, common.HexToHash("0x45fd9aa59768331c726e719e76aa343e73123af888804604785ae19506e65e87"), value)
		require.Empty(t, generator.generated)
	})

	t.Run("ProofAfterEndOfTrace", func(t *testing.T) {
		provider, generator := setupWithTestData(t, dataDir, prestate)
		generator.finalState = &mipsevm.State{
			Memory: &mipsevm.Memory{},
			Step:   10,
			Exited: true,
		}
		value, err := provider.Get(context.Background(), 7000)
		require.NoError(t, err)
		require.Contains(t, generator.generated, 7000, "should have tried to generate the proof")
		require.Equal(t, crypto.Keccak256Hash(generator.finalState.EncodeWitness()), value)
	})

	t.Run("MissingPostHash", func(t *testing.T) {
		provider, generator := setupWithTestData(t, dataDir, prestate)
		_, err := provider.Get(context.Background(), 1)
		require.ErrorContains(t, err, "missing post hash")
		require.Empty(t, generator.generated)
	})

	t.Run("IgnoreUnknownFields", func(t *testing.T) {
		provider, generator := setupWithTestData(t, dataDir, prestate)
		value, err := provider.Get(context.Background(), 2)
		require.NoError(t, err)
		expected := common.HexToHash("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
		require.Equal(t, expected, value)
		require.Empty(t, generator.generated)
	})
}

func TestGetStepData(t *testing.T) {
	dataDir, prestate := setupTestData(t)
	t.Run("ExistingProof", func(t *testing.T) {
		provider, generator := setupWithTestData(t, dataDir, prestate)
		value, proof, data, err := provider.GetStepData(context.Background(), 0)
		require.NoError(t, err)
		expected := common.Hex2Bytes("b8f068de604c85ea0e2acd437cdb47add074a2d70b81d018390c504b71fe26f400000000000000000000000000000000000000000000000000000000000000000000000000")
		require.Equal(t, expected, value)
		expectedProof := common.Hex2Bytes("08028e3c0000000000000000000000003c01000a24210b7c00200008000000008fa40004")
		require.Equal(t, expectedProof, proof)
		// TODO: Need to add some oracle data
		require.Nil(t, data)
		require.Empty(t, generator.generated)
	})

	t.Run("GenerateProof", func(t *testing.T) {
		provider, generator := setupWithTestData(t, dataDir, prestate)
		generator.finalState = &mipsevm.State{
			Memory: &mipsevm.Memory{},
			Step:   10,
			Exited: true,
		}
		generator.proof = &proofData{
			ClaimValue:   common.Hash{0xaa}.Bytes(),
			StateData:    []byte{0xbb},
			ProofData:    []byte{0xcc},
			OracleKey:    common.Hash{0xdd}.Bytes(),
			OracleValue:  []byte{0xdd},
			OracleOffset: 10,
		}
		preimage, proof, data, err := provider.GetStepData(context.Background(), 4)
		require.NoError(t, err)
		require.Contains(t, generator.generated, 4, "should have tried to generate the proof")

		require.EqualValues(t, generator.proof.StateData, preimage)
		require.EqualValues(t, generator.proof.ProofData, proof)
		expectedData := types.NewPreimageOracleData(generator.proof.OracleKey, generator.proof.OracleValue, generator.proof.OracleOffset)
		require.EqualValues(t, expectedData, data)
	})

	t.Run("ProofAfterEndOfTrace", func(t *testing.T) {
		provider, generator := setupWithTestData(t, dataDir, prestate)
		generator.finalState = &mipsevm.State{
			Memory: &mipsevm.Memory{},
			Step:   10,
			Exited: true,
		}
		generator.proof = &proofData{
			ClaimValue:   common.Hash{0xaa}.Bytes(),
			StateData:    []byte{0xbb},
			ProofData:    []byte{0xcc},
			OracleKey:    common.Hash{0xdd}.Bytes(),
			OracleValue:  []byte{0xdd},
			OracleOffset: 10,
		}
		preimage, proof, data, err := provider.GetStepData(context.Background(), 7000)
		require.NoError(t, err)
		require.Contains(t, generator.generated, 7000, "should have tried to generate the proof")

		witness := generator.finalState.EncodeWitness()
		require.EqualValues(t, witness, preimage)
		require.Equal(t, []byte{}, proof)
		require.Nil(t, data)
	})

	t.Run("MissingStateData", func(t *testing.T) {
		provider, generator := setupWithTestData(t, dataDir, prestate)
		_, _, _, err := provider.GetStepData(context.Background(), 1)
		require.ErrorContains(t, err, "missing state data")
		require.Empty(t, generator.generated)
	})

	t.Run("IgnoreUnknownFields", func(t *testing.T) {
		provider, generator := setupWithTestData(t, dataDir, prestate)
		value, proof, data, err := provider.GetStepData(context.Background(), 2)
		require.NoError(t, err)
		expected := common.Hex2Bytes("cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc")
		require.Equal(t, expected, value)
		expectedProof := common.Hex2Bytes("dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd")
		require.Equal(t, expectedProof, proof)
		require.Empty(t, generator.generated)
		require.Nil(t, data)
	})
}

func TestAbsolutePreState(t *testing.T) {
	dataDir := t.TempDir()

	prestate := "state.json"

	t.Run("StateUnavailable", func(t *testing.T) {
		provider, _ := setupWithTestData(t, "/dir/does/not/exist", prestate)
		_, err := provider.AbsolutePreState(context.Background())
		require.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("InvalidStateFile", func(t *testing.T) {
		setupPreState(t, dataDir, "invalid.json")
		provider, _ := setupWithTestData(t, dataDir, prestate)
		_, err := provider.AbsolutePreState(context.Background())
		require.ErrorContains(t, err, "invalid mipsevm state")
	})

	t.Run("ExpectedAbsolutePreState", func(t *testing.T) {
		setupPreState(t, dataDir, "state.json")
		provider, _ := setupWithTestData(t, dataDir, prestate)
		preState, err := provider.AbsolutePreState(context.Background())
		require.NoError(t, err)
		state := mipsevm.State{
			Memory:         mipsevm.NewMemory(),
			PreimageKey:    common.HexToHash("cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"),
			PreimageOffset: 0,
			PC:             0,
			NextPC:         1,
			LO:             0,
			HI:             0,
			Heap:           0,
			ExitCode:       0,
			Exited:         false,
			Step:           0,
			Registers:      [32]uint32{},
		}
		require.Equal(t, state.EncodeWitness(), preState)
	})
}

func TestUseGameSpecificSubdir(t *testing.T) {
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")
	setupPreState(t, tempDir, "state.json")
	logger := testlog.Logger(t, log.LvlInfo)
	cfg := &config.Config{
		CannonAbsolutePreState: filepath.Join(tempDir, "state.json"),
		Datadir:                dataDir,
	}
	gameDirName := "gameSubdir"
	localInputs := LocalGameInputs{}
	provider := NewTraceProviderFromInputs(logger, cfg, gameDirName, localInputs)
	require.Equal(t, filepath.Join(dataDir, gameDirName), provider.dir, "should use game specific subdir")
}

func TestCleanup(t *testing.T) {
	dataDir, prestate := setupTestData(t)
	provider, _ := setupWithTestData(t, dataDir, prestate)
	require.NoError(t, provider.Cleanup())
	_, err := os.Stat(dataDir)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func setupPreState(t *testing.T, dataDir string, filename string) {
	srcDir := filepath.Join("test_data")
	path := filepath.Join(srcDir, filename)
	file, err := testData.ReadFile(path)
	require.NoErrorf(t, err, "reading %v", path)
	err = os.WriteFile(filepath.Join(dataDir, "state.json"), file, 0o644)
	require.NoErrorf(t, err, "writing %v", path)
}

func setupTestData(t *testing.T) (string, string) {
	srcDir := filepath.Join("test_data", "proofs")
	entries, err := testData.ReadDir(srcDir)
	require.NoError(t, err)
	dataDir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dataDir, proofsDir), 0o777))
	for _, entry := range entries {
		path := filepath.Join(srcDir, entry.Name())
		file, err := testData.ReadFile(path)
		require.NoErrorf(t, err, "reading %v", path)
		err = os.WriteFile(filepath.Join(dataDir, proofsDir, entry.Name()), file, 0o644)
		require.NoErrorf(t, err, "writing %v", path)
	}
	return dataDir, "state.json"
}

func setupWithTestData(t *testing.T, dataDir string, prestate string) (*CannonTraceProvider, *stubGenerator) {
	generator := &stubGenerator{}
	return &CannonTraceProvider{
		logger:    testlog.Logger(t, log.LvlInfo),
		dir:       dataDir,
		generator: generator,
		prestate:  filepath.Join(dataDir, prestate),
	}, generator
}

type stubGenerator struct {
	generated  []int // Using int makes assertions easier
	finalState *mipsevm.State
	proof      *proofData
}

func (e *stubGenerator) GenerateProof(ctx context.Context, dir string, i uint64) error {
	e.generated = append(e.generated, int(i))
	if e.finalState != nil && e.finalState.Step <= i {
		// Requesting a trace index past the end of the trace
		data, err := json.Marshal(e.finalState)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dir, finalState), data, 0644)
	}
	if e.proof != nil {
		proofFile := filepath.Join(dir, proofsDir, fmt.Sprintf("%d.json", i))
		data, err := json.Marshal(e.proof)
		if err != nil {
			return err
		}
		return os.WriteFile(proofFile, data, 0644)
	}
	return nil
}
