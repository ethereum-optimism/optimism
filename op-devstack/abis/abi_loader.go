package abis

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

type ArtifactData struct {
	Bytecode []byte
	ABI      abi.ABI
}

type ArtifactBytecode struct {
	Object string `json:"object"`
}

type ArtifactJSON struct {
	Bytecode ArtifactBytecode `json:"bytecode"`
	ABI      json.RawMessage  `json:"abi"`
}

func GetArtifactData(t devtest.CommonT, contractName string) *ArtifactData {
	req := require.New(t)
	artifactPath := getArtifactPath(t, contractName)

	fileData, err := os.ReadFile(artifactPath)
	req.NoErrorf(err, "Failed to read %s artifact file", contractName)

	var artifactJSON ArtifactJSON
	err = json.Unmarshal(fileData, &artifactJSON)
	req.NoErrorf(err, "Failed to parse %s artifact JSON", contractName)

	req.NotEmptyf(artifactJSON.Bytecode.Object, "Bytecode object not found in %s artifact", contractName)

	bytecode := common.FromHex(artifactJSON.Bytecode.Object)

	parsedABI, err := abi.JSON(bytes.NewReader(artifactJSON.ABI))
	req.NoError(err, "Failed to parse ABI")

	return &ArtifactData{
		Bytecode: bytecode,
		ABI:      parsedABI,
	}
}

func getArtifactPath(t devtest.CommonT, contractName string) string {
	req := require.New(t)
	wd, err := os.Getwd()
	req.NoError(err, "Failed to get current working directory")

	monorepoRoot, err := opservice.FindMonorepoRoot(wd)
	req.NoError(err, "Failed to find monorepo root")

	contractsBedrock := filepath.Join(monorepoRoot, "packages", "contracts-bedrock")
	return filepath.Join(contractsBedrock, "forge-artifacts", contractName+".sol", contractName+".json")
}
