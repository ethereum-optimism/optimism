package gen

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildScriptBinding(t *testing.T) {
	artifactPath := filepath.Join(t.TempDir(), "ExampleScript.json")
	writeExampleScriptArtifact(t, artifactPath)

	source, err := BuildScriptBinding(ScriptBindingConfig{
		ArtifactPath: artifactPath,
		FunctionName: "run",
		PackageName:  "examplescript",
	})
	require.NoError(t, err)

	sourceText := string(source)
	require.Contains(t, sourceText, "package examplescript")
	require.Contains(t, sourceText, `SourcePath            = "scripts/deploy/ExampleScript.s.sol"`)
	require.Contains(t, sourceText, `ForgeScriptPath       = "scripts/deploy/ExampleScript.s.sol:ExampleScript"`)
	require.Regexp(t, regexp.MustCompile("Guardian\\s+common\\.Address\\s+`abi:\"guardian\" json:\"guardian\" toml:\"guardian\"`"), sourceText)
	require.Regexp(t, regexp.MustCompile("Paused\\s+bool\\s+`abi:\"paused\" json:\"paused\" toml:\"paused\"`"), sourceText)
	require.Regexp(t, regexp.MustCompile("RecommendedProtocolVersion\\s+common\\.Hash\\s+`abi:\"recommendedProtocolVersion\" json:\"recommendedProtocolVersion\" toml:\"recommendedProtocolVersion\"`"), sourceText)
	require.Regexp(t, regexp.MustCompile("DisputeGameConfigs\\s+\\[\\]InputDisputeGameConfig\\s+`abi:\"disputeGameConfigs\" json:\"disputeGameConfigs\" toml:\"disputeGameConfigs\"`"), sourceText)
	require.Regexp(t, regexp.MustCompile("SuperchainConfigProxy\\s+common\\.Address\\s+`abi:\"superchainConfigProxy\" json:\"superchainConfigProxy\" toml:\"superchainConfigProxy\"`"), sourceText)
	require.Contains(t, sourceText, "type InputDisputeGameConfig struct")
	require.Regexp(t, regexp.MustCompile("GameType\\s+uint32\\s+`abi:\"gameType\" json:\"gameType\" toml:\"gameType\"`"), sourceText)
}

func TestGenerateScriptBindings(t *testing.T) {
	tempDir := t.TempDir()
	artifactPath := filepath.Join(tempDir, "artifacts", "ExampleScript.s.sol", "ExampleScript.json")
	writeExampleScriptArtifact(t, artifactPath)

	manifestPath := filepath.Join(tempDir, "bindings.yaml")
	require.NoError(t, os.WriteFile(manifestPath, []byte(`
artifact_base: artifacts
output_base: generated
manifest:
  package: exampletypes
  out: generated
  filename: manifest.gen.go
script_bindings:
  - name: ExampleScript
    artifact: ExampleScript.s.sol/ExampleScript.json
    package: examplescript
    out: examplescript
    filename: example_script.gen.go
`), 0o644))

	outPaths, err := GenerateScriptBindings(ScriptBindingsConfig{
		ConfigPath: manifestPath,
		BaseDir:    tempDir,
	})
	require.NoError(t, err)
	require.Equal(t, []string{
		filepath.Join(tempDir, "generated", "examplescript", "example_script.gen.go"),
		filepath.Join(tempDir, "generated", "manifest.gen.go"),
	}, outPaths)

	generated, err := os.ReadFile(outPaths[0])
	require.NoError(t, err)
	require.Contains(t, string(generated), "type Input struct")
	require.Contains(t, string(generated), "type InputDisputeGameConfig struct")

	manifest, err := os.ReadFile(outPaths[1])
	require.NoError(t, err)
	manifestText := string(manifest)
	require.Contains(t, manifestText, "package exampletypes")
	require.Contains(t, manifestText, "SchemaHash:")
	require.Contains(t, manifestText, `"ExampleScriptInput":`)
	require.Contains(t, manifestText, `{Name: "disputeGameConfigs", Type: "tuple[]", Required: true, Struct: "ExampleScriptInputDisputeGameConfig"}`)
	require.Contains(t, manifestText, `"ExampleScriptInputDisputeGameConfig":`)
	require.Contains(t, manifestText, `{Name: "gameType", Type: "uint32", Required: true}`)
}

func writeExampleScriptArtifact(t *testing.T, artifactPath string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(artifactPath), 0o755))
	require.NoError(t, os.WriteFile(artifactPath, []byte(`{
		"abi": [
			{
				"type": "function",
				"name": "run",
				"inputs": [
					{
						"name": "_input",
						"type": "tuple",
						"internalType": "struct ExampleScript.Input",
						"components": [
							{"name": "guardian", "type": "address", "internalType": "address"},
							{"name": "paused", "type": "bool", "internalType": "bool"},
							{"name": "recommendedProtocolVersion", "type": "bytes32", "internalType": "bytes32"},
							{
								"name": "disputeGameConfigs",
								"type": "tuple[]",
								"internalType": "struct ExampleScript.DisputeGameConfig[]",
								"components": [
									{"name": "gameType", "type": "uint32", "internalType": "uint32"},
									{"name": "permissioned", "type": "bool", "internalType": "bool"}
								]
							}
						]
					}
				],
				"outputs": [
					{
						"name": "output_",
						"type": "tuple",
						"internalType": "struct ExampleScript.Output",
						"components": [
							{"name": "superchainConfigProxy", "type": "address", "internalType": "contract ISuperchainConfig"}
						]
					}
				]
			}
		],
		"metadata": {
			"settings": {
				"compilationTarget": {
					"scripts/deploy/ExampleScript.s.sol": "ExampleScript"
				}
				}
			}
		}`), 0o644))
}
