package cannon

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/testutil"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/versions"
	"github.com/ethereum-optimism/optimism/op-challenger/config"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	op_service "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/serialize"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm/memory"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/vm"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-service/ioutil"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

//go:embed test_data
var testData embed.FS

func PositionFromTraceIndex(provider *CannonTraceProvider, idx *big.Int) types.Position {
	return types.NewPosition(provider.gameDepth, idx)
}

func TestGet(t *testing.T) {
	dataDir, prestate := setupTestData(t)
	t.Run("ExistingProof", func(t *testing.T) {
		provider, generator := setupWithTestData(t, dataDir, prestate)
		value, err := provider.Get(context.Background(), PositionFromTraceIndex(provider, common.Big0))
		require.NoError(t, err)
		require.Equal(t, common.HexToHash("0x45fd9aa59768331c726e719e76aa343e73123af888804604785ae19506e65e87"), value)
		require.Empty(t, generator.generated)
	})

	t.Run("ErrorsTraceIndexOutOfBounds", func(t *testing.T) {
		provider, generator := setupWithTestData(t, dataDir, prestate)
		largePosition := PositionFromTraceIndex(provider, new(big.Int).Mul(new(big.Int).SetUint64(math.MaxUint64), big.NewInt(2)))
		_, err := provider.Get(context.Background(), largePosition)
		require.ErrorContains(t, err, "trace index out of bounds")
		require.Empty(t, generator.generated)
	})

	t.Run("ProofAfterEndOfTrace", func(t *testing.T) {
		provider, generator := setupWithTestData(t, dataDir, prestate)
		generator.finalState = &multithreaded.State{
			Memory: memory.NewMemory(),
			Step:   10,
			Exited: true,
		}
		value, err := provider.Get(context.Background(), PositionFromTraceIndex(provider, big.NewInt(7000)))
		require.NoError(t, err)
		require.Contains(t, generator.generated, 7000, "should have tried to generate the proof")
		_, stateHash := generator.finalState.EncodeWitness()
		require.Equal(t, stateHash, value)
	})

	t.Run("MissingPostHash", func(t *testing.T) {
		provider, generator := setupWithTestData(t, dataDir, prestate)
		_, err := provider.Get(context.Background(), PositionFromTraceIndex(provider, big.NewInt(1)))
		require.ErrorContains(t, err, "missing post hash")
		require.Empty(t, generator.generated)
	})

	t.Run("IgnoreUnknownFields", func(t *testing.T) {
		provider, generator := setupWithTestData(t, dataDir, prestate)
		value, err := provider.Get(context.Background(), PositionFromTraceIndex(provider, big.NewInt(2)))
		require.NoError(t, err)
		expected := common.HexToHash("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
		require.Equal(t, expected, value)
		require.Empty(t, generator.generated)
	})
}

func TestGetStepData(t *testing.T) {
	t.Run("ExistingProof", func(t *testing.T) {
		dataDir, prestate := setupTestData(t)
		provider, generator := setupWithTestData(t, dataDir, prestate)
		value, proof, data, err := provider.GetStepData(context.Background(), PositionFromTraceIndex(provider, new(big.Int)))
		require.NoError(t, err)
		expected := common.FromHex("b8f068de604c85ea0e2acd437cdb47add074a2d70b81d018390c504b71fe26f400000000000000000000000000000000000000000000000000000000000000000000000000")
		require.Equal(t, expected, value)
		expectedProof := common.FromHex("08028e3c0000000000000000000000003c01000a24210b7c00200008000000008fa40004")
		require.Equal(t, expectedProof, proof)
		// TODO: Need to add some oracle data
		require.Nil(t, data)
		require.Empty(t, generator.generated)
	})

	t.Run("ErrorsTraceIndexOutOfBounds", func(t *testing.T) {
		dataDir, prestate := setupTestData(t)
		provider, generator := setupWithTestData(t, dataDir, prestate)
		largePosition := PositionFromTraceIndex(provider, new(big.Int).Mul(new(big.Int).SetUint64(math.MaxUint64), big.NewInt(2)))
		_, _, _, err := provider.GetStepData(context.Background(), largePosition)
		require.ErrorContains(t, err, "trace index out of bounds")
		require.Empty(t, generator.generated)
	})

	t.Run("GenerateProof", func(t *testing.T) {
		dataDir, prestate := setupTestData(t)
		provider, generator := setupWithTestData(t, dataDir, prestate)
		generator.finalState = &multithreaded.State{
			Memory: memory.NewMemory(),
			Step:   10,
			Exited: true,
		}
		generator.proof = &utils.ProofData{
			ClaimValue:   common.Hash{0xaa},
			StateData:    []byte{0xbb},
			ProofData:    []byte{0xcc},
			OracleKey:    common.Hash{0xdd}.Bytes(),
			OracleValue:  []byte{0xdd},
			OracleOffset: 10,
		}
		preimage, proof, data, err := provider.GetStepData(context.Background(), PositionFromTraceIndex(provider, big.NewInt(4)))
		require.NoError(t, err)
		require.Contains(t, generator.generated, 4, "should have tried to generate the proof")

		require.EqualValues(t, generator.proof.StateData, preimage)
		require.EqualValues(t, generator.proof.ProofData, proof)
		expectedData := types.NewPreimageOracleData(generator.proof.OracleKey, generator.proof.OracleValue, generator.proof.OracleOffset)
		require.EqualValues(t, expectedData, data)
	})

	t.Run("ProofAfterEndOfTrace", func(t *testing.T) {
		dataDir, prestate := setupTestData(t)
		provider, generator := setupWithTestData(t, dataDir, prestate)
		generator.finalState = &multithreaded.State{
			Memory: memory.NewMemory(),
			Step:   10,
			Exited: true,
		}
		generator.proof = &utils.ProofData{
			ClaimValue:   common.Hash{0xaa},
			StateData:    []byte{0xbb},
			ProofData:    []byte{0xcc},
			OracleKey:    common.Hash{0xdd}.Bytes(),
			OracleValue:  []byte{0xdd},
			OracleOffset: 10,
		}
		preimage, proof, data, err := provider.GetStepData(context.Background(), PositionFromTraceIndex(provider, big.NewInt(7000)))
		require.NoError(t, err)
		require.Contains(t, generator.generated, 7000, "should have tried to generate the proof")

		witness, _ := generator.finalState.EncodeWitness()
		require.EqualValues(t, witness, preimage)
		require.Equal(t, []byte{}, proof)
		require.Nil(t, data)
	})

	t.Run("ReadLastStepFromDisk", func(t *testing.T) {
		dataDir, prestate := setupTestData(t)
		provider, initGenerator := setupWithTestData(t, dataDir, prestate)
		initGenerator.finalState = &multithreaded.State{
			Memory: memory.NewMemory(),
			Step:   10,
			Exited: true,
		}
		initGenerator.proof = &utils.ProofData{
			ClaimValue:   common.Hash{0xaa},
			StateData:    []byte{0xbb},
			ProofData:    []byte{0xcc},
			OracleKey:    common.Hash{0xdd}.Bytes(),
			OracleValue:  []byte{0xdd},
			OracleOffset: 10,
		}
		_, _, _, err := provider.GetStepData(context.Background(), PositionFromTraceIndex(provider, big.NewInt(7000)))
		require.NoError(t, err)
		require.Contains(t, initGenerator.generated, 7000, "should have tried to generate the proof")

		provider, generator := setupWithTestData(t, dataDir, prestate)
		generator.finalState = &multithreaded.State{
			Memory: memory.NewMemory(),
			Step:   10,
			Exited: true,
		}
		generator.proof = &utils.ProofData{
			ClaimValue: common.Hash{0xaa},
			StateData:  []byte{0xbb},
			ProofData:  []byte{0xcc},
		}
		preimage, proof, data, err := provider.GetStepData(context.Background(), PositionFromTraceIndex(provider, big.NewInt(7000)))
		require.NoError(t, err)
		require.Empty(t, generator.generated, "should not have to generate the proof again")

		encodedWitness, _ := initGenerator.finalState.EncodeWitness()
		require.EqualValues(t, encodedWitness, preimage)
		require.Empty(t, proof)
		require.Nil(t, data)
	})

	t.Run("MissingStateData", func(t *testing.T) {
		dataDir, prestate := setupTestData(t)
		provider, generator := setupWithTestData(t, dataDir, prestate)
		_, _, _, err := provider.GetStepData(context.Background(), PositionFromTraceIndex(provider, big.NewInt(1)))
		require.ErrorContains(t, err, "missing state data")
		require.Empty(t, generator.generated)
	})

	t.Run("IgnoreUnknownFields", func(t *testing.T) {
		dataDir, prestate := setupTestData(t)
		provider, generator := setupWithTestData(t, dataDir, prestate)
		value, proof, data, err := provider.GetStepData(context.Background(), PositionFromTraceIndex(provider, big.NewInt(2)))
		require.NoError(t, err)
		expected := common.FromHex("cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc")
		require.Equal(t, expected, value)
		expectedProof := common.FromHex("dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd")
		require.Equal(t, expectedProof, proof)
		require.Empty(t, generator.generated)
		require.Nil(t, data)
	})
}

func TestLastStepCacheAccuracy(t *testing.T) {
	dir := t.TempDir()
	monorepoRoot, err := op_service.FindMonorepoRoot(".")
	require.NoError(t, err)
	logger := testlog.Logger(t, log.LevelInfo)
	vmCfg := vm.Config{
		VmType:          gameTypes.CannonGameType,
		VmBin:           filepath.Join(monorepoRoot, "cannon/bin/cannon"),
		InfoFreq:        10_000,
		SnapshotFreq:    config.DefaultCannonSnapshotFreq,
		BinarySnapshots: true,
		Server:          "/usr/bin/true", // Preimages aren't required, just need something with a 0 exit code.
	}
	localInputs := utils.LocalGameInputs{
		L1Head:           common.Hash{0x01},
		L2Head:           common.Hash{0x02},
		L2OutputRoot:     common.Hash{0x03},
		L2Claim:          common.Hash{0x04},
		L2SequenceNumber: big.NewInt(5),
	}
	prestate := filepath.Join(dir, "prestate.bin.gz")

	// This requires cannon and its testdata to be built: cd cannon && make cannon elf
	state, _ := testutil.LoadELFProgram(t, filepath.Join(monorepoRoot, "cannon/testdata/go-1-24/bin/hello.64.elf"), multithreaded.CreateInitialState)
	versionedState, err := versions.NewFromState(versions.GetCurrentVersion(), state)
	require.NoError(t, err)
	err = serialize.Write(prestate, versionedState, os.FileMode(0o755))
	require.NoError(t, err, "failed to write prestate")

	newProvider := func() *CannonTraceProvider {
		return NewTraceProvider(
			logger,
			metrics.NoopMetrics.ToTypedVmMetrics("cannon"),
			vmCfg,
			vm.NewOpProgramServerExecutor(logger),
			nil,
			prestate,
			localInputs,
			t.TempDir(),
			types.Depth(30),
		)
	}
	cachedProvider := newProvider()
	t.Log("Priming cache so last step is cached")
	_, _, _, err = cachedProvider.GetStepData(t.Context(), types.RootPosition)
	require.NoError(t, err)

	verifyStepDataMatches := func(step uint64) {
		t.Logf("Generating step %d with cached provider with last step cached", step)
		pos := PositionFromTraceIndex(cachedProvider, new(big.Int).SetUint64(step))
		prestateData1, proofData1, _, err := cachedProvider.GetStepData(t.Context(), pos)
		require.NoError(t, err)

		t.Logf("Regenerating step %d with cached provider now exact step is cached", step)
		prestateData2, proofData2, _, err := cachedProvider.GetStepData(t.Context(), pos)
		require.NoError(t, err)
		require.EqualValues(t, prestateData1, prestateData2)
		require.EqualValues(t, proofData1, proofData2)

		t.Logf("Generating step %d with uncached provider", step)
		prestateDataUncached, proofDataUncached, _, err := newProvider().GetStepData(t.Context(), pos)
		require.NoError(t, err)
		require.EqualValues(t, prestateData1, prestateDataUncached)
		require.EqualValues(t, proofData1, proofDataUncached)
	}

	for step := cachedProvider.lastStep - 2; step <= cachedProvider.lastStep+2; step++ {
		verifyStepDataMatches(step)
	}

	// Verify it matches all the through the trace extension
	verifyStepDataMatches(math.MaxUint64)
}

func setupTestData(t *testing.T) (string, string) {
	srcDir := filepath.Join("test_data", "proofs")
	entries, err := testData.ReadDir(srcDir)
	require.NoError(t, err)
	dataDir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dataDir, utils.ProofsDir), 0o777))
	for _, entry := range entries {
		path := filepath.Join(srcDir, entry.Name())
		file, err := testData.ReadFile(path)
		require.NoErrorf(t, err, "reading %v", path)
		proofFile := filepath.Join(dataDir, utils.ProofsDir, entry.Name()+".gz")
		err = ioutil.WriteCompressedBytes(proofFile, file, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0o644)
		require.NoErrorf(t, err, "writing %v", path)
	}
	return dataDir, "state.json"
}

func setupWithTestData(t *testing.T, dataDir string, prestate string) (*CannonTraceProvider, *stubGenerator) {
	generator := &stubGenerator{}
	return &CannonTraceProvider{
		logger:         testlog.Logger(t, log.LevelInfo),
		dir:            dataDir,
		generator:      generator,
		prestate:       filepath.Join(dataDir, prestate),
		gameDepth:      63,
		stateConverter: generator,
	}, generator
}

type stubGenerator struct {
	generated  []int // Using int makes assertions easier
	finalState *multithreaded.State
	proof      *utils.ProofData

	finalStatePath string
}

func (e *stubGenerator) ConvertStateToProof(ctx context.Context, statePath string) (*utils.ProofData, uint64, bool, error) {
	if statePath == e.finalStatePath {
		witness, hash := e.finalState.EncodeWitness()
		return &utils.ProofData{
			ClaimValue: hash,
			StateData:  witness,
			ProofData:  []byte{},
		}, e.finalState.Step, e.finalState.Exited, nil
	} else {
		return nil, 0, false, fmt.Errorf("loading unexpected state: %s, only support: %s", statePath, e.finalStatePath)
	}
}

func (e *stubGenerator) GenerateProof(ctx context.Context, dir string, i uint64) error {
	e.generated = append(e.generated, int(i))
	var proofFile string
	var data []byte
	var err error
	if e.finalState != nil && e.finalState.Step <= i {
		// Requesting a trace index past the end of the trace
		proofFile = vm.FinalStatePath(dir, false)
		e.finalStatePath = proofFile
		data, err = json.Marshal(e.finalState)
		if err != nil {
			return err
		}
		return ioutil.WriteCompressedBytes(proofFile, data, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0o644)
	}
	if e.proof != nil {
		proofFile = filepath.Join(dir, utils.ProofsDir, fmt.Sprintf("%d.json.gz", i))
		data, err = json.Marshal(e.proof)
		if err != nil {
			return err
		}
		return ioutil.WriteCompressedBytes(proofFile, data, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0o644)
	}
	return nil
}
