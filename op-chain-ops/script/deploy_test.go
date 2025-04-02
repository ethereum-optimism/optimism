package script

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestForgeScriptImpl(t *testing.T) {
	t.Run("should return ABI from the artifact", func(t *testing.T) {
		abi := abi.ABI{
			Methods: map[string]abi.Method{
				"run": abi.NewMethod("Run", "run", abi.Function, "", false, false, abi.Arguments{}, abi.Arguments{}),
			},
		}
		artifact := &foundry.Artifact{
			ABI: abi,
		}
		script := &forgeScriptImpl{
			artifact: artifact,
		}

		require.Equal(t, abi, script.ABI())
	})

	t.Run("should return the specified name", func(t *testing.T) {
		script := &forgeScriptImpl{
			name: "MyScript",
		}

		require.Equal(t, "MyScript", script.Name())
	})

	t.Run("Call", func(t *testing.T) {
		artifact := &foundry.Artifact{}
		label := "MyScript"

		t.Run("should fail if script fails to deploy", func(t *testing.T) {
			backend := &mockForgeScriptBackend{
				deployError: fmt.Errorf("oh no"),
			}
			script := NewForgeScriptFromArtifact(artifact, label, backend)

			_, err := script.Call([]byte{})
			require.EqualError(t, err, "failed to deploy script MyScript: oh no")
		})

		t.Run("should fail if backend call fails", func(t *testing.T) {
			scriptAddress := common.BigToAddress(big.NewInt(1))
			callError := fmt.Errorf("oh no")
			backend := &mockForgeScriptBackend{
				deployResult: scriptAddress,
				callError:    callError,
			}
			script := NewForgeScriptFromArtifact(artifact, label, backend)

			input := []byte{1}
			result, err := script.Call(input)
			require.EqualError(t, err, "failed to call script MyScript using data 0x01: oh no")
			require.Nil(t, result)
			require.Equal(t, input, backend.calledWith)
			require.Equal(t, scriptAddress, backend.calledTo)
		})

		t.Run("should return the call result if backend call succeeds", func(t *testing.T) {
			scriptAddress := common.BigToAddress(big.NewInt(1))
			callResult := []byte{1, 0, 1}
			backend := &mockForgeScriptBackend{
				deployResult: scriptAddress,
				callResult:   callResult,
			}
			script := NewForgeScriptFromArtifact(artifact, label, backend)

			input := []byte{1}
			result, err := script.Call(input)
			require.NoError(t, err)
			require.Equal(t, callResult, result)
			require.Equal(t, input, backend.calledWith)
			require.Equal(t, scriptAddress, backend.calledTo)
		})

		t.Run("should destroy the script after the call", func(t *testing.T) {
			scriptAddress := common.BigToAddress(big.NewInt(1))
			backend := &mockForgeScriptBackend{
				deployResult: scriptAddress,
			}
			script := NewForgeScriptFromArtifact(artifact, label, backend)

			_, err := script.Call([]byte{})
			require.NoError(t, err)
			require.Equal(t, scriptAddress, backend.destroyedAddress)
		})
	})
}

func TestNewDeployScriptWithoutOutput(t *testing.T) {
	type ExampleInput struct {
		FieldA common.Address
		FieldB common.Address
	}

	t.Run("should fail if the script does not have a specified method", func(t *testing.T) {
		script := &mockForgeScript{
			abi: abi.ABI{
				Methods: map[string]abi.Method{},
			},
		}

		_, err := NewDeployScriptWithoutOutput[ExampleInput](script, "run")
		require.EqualError(t, err, "script MockScript does not have a method called run")
	})

	t.Run("should fail if the specified method does not have exactly one input", func(t *testing.T) {
		script := &mockForgeScript{
			abi: abi.ABI{
				Methods: map[string]abi.Method{
					"run": abi.NewMethod("Run", "run", abi.Function, "", false, false, []abi.Argument{}, []abi.Argument{}),
				},
			},
		}

		_, err := NewDeployScriptWithoutOutput[ExampleInput](script, "run")
		require.EqualError(t, err, "script MockScript does not have a method run that accepts an argument of type script.ExampleInput: ABI arguments don't match Go types: ABI has 0 arguments, Go has 1")
	})

	t.Run("should fail if the specified method does not have exactly one input whose type matches the input type", func(t *testing.T) {
		script := &mockForgeScript{
			abi: abi.ABI{
				Methods: map[string]abi.Method{
					"run": abi.NewMethod("Run", "run", abi.Function, "", false, false, []abi.Argument{
						{
							Name: "_input",
							Type: die(abi.NewType("uint256", "", []abi.ArgumentMarshaling{})),
						},
					}, []abi.Argument{}),
				},
			},
		}

		_, err := NewDeployScriptWithoutOutput[ExampleInput](script, "run")
		require.EqualError(t, err, "script MockScript does not have a method run that accepts an argument of type script.ExampleInput: ABI argument _input at index 0 doesn't match Go type: ABI type uint256 (represented by *big.Int) is not assignable to Go type script.ExampleInput")
	})

	t.Run("should not fail if the specified method has exactly one input whose type matches the input type", func(t *testing.T) {
		script := &mockForgeScript{
			abi: abi.ABI{
				Methods: map[string]abi.Method{
					"run": abi.NewMethod("Run", "run", abi.Function, "", false, false, []abi.Argument{
						{
							Name: "_input",
							Type: die(abi.NewType("tuple", "", []abi.ArgumentMarshaling{{Name: "fieldA", Type: "address"}, {Name: "fieldB", Type: "address"}})),
						},
					}, []abi.Argument{}),
				},
			},
		}

		deployScript, err := NewDeployScriptWithoutOutput[ExampleInput](script, "run")
		require.NoError(t, err)
		require.NotNil(t, deployScript)
	})
}

func TestNewDeployScriptWithOutput(t *testing.T) {
	type ExampleInput struct {
		FieldA common.Address
		FieldB common.Address
	}

	type ExampleOutput struct {
		FieldC common.Address
		FieldD common.Address
	}

	t.Run("should fail if the script does not have a specified method", func(t *testing.T) {
		script := &mockForgeScript{
			abi: abi.ABI{
				Methods: map[string]abi.Method{},
			},
		}

		_, err := NewDeployScriptWithOutput[ExampleInput, ExampleOutput](script, "run")
		require.EqualError(t, err, "script MockScript does not have a method called run")
	})

	t.Run("should fail if the specified method does not have exactly one input", func(t *testing.T) {
		script := &mockForgeScript{
			abi: abi.ABI{
				Methods: map[string]abi.Method{
					"run": abi.NewMethod("Run", "run", abi.Function, "", false, false, []abi.Argument{}, []abi.Argument{}),
				},
			},
		}

		_, err := NewDeployScriptWithOutput[ExampleInput, ExampleOutput](script, "run")
		require.EqualError(t, err, "script MockScript does not have a method run that accepts an argument of type script.ExampleInput: ABI arguments don't match Go types: ABI has 0 arguments, Go has 1")
	})

	t.Run("should fail if the specified method does not have exactly one input whose type matches the input type", func(t *testing.T) {
		script := &mockForgeScript{
			abi: abi.ABI{
				Methods: map[string]abi.Method{
					"run": abi.NewMethod("Run", "run", abi.Function, "", false, false, []abi.Argument{
						{
							Name: "_input",
							Type: die(abi.NewType("uint256", "", []abi.ArgumentMarshaling{})),
						},
					}, []abi.Argument{}),
				},
			},
		}

		_, err := NewDeployScriptWithOutput[ExampleInput, ExampleOutput](script, "run")
		require.EqualError(t, err, "script MockScript does not have a method run that accepts an argument of type script.ExampleInput: ABI argument _input at index 0 doesn't match Go type: ABI type uint256 (represented by *big.Int) is not assignable to Go type script.ExampleInput")
	})

	t.Run("should fail if the specified method does not have exactly one output", func(t *testing.T) {
		script := &mockForgeScript{
			abi: abi.ABI{
				Methods: map[string]abi.Method{
					"run": abi.NewMethod("Run", "run", abi.Function, "", false, false, []abi.Argument{
						{
							Name: "_input",
							Type: die(abi.NewType("tuple", "", []abi.ArgumentMarshaling{{Name: "fieldA", Type: "address"}, {Name: "fieldB", Type: "address"}})),
						},
					}, []abi.Argument{}),
				},
			},
		}

		_, err := NewDeployScriptWithOutput[ExampleInput, ExampleOutput](script, "run")
		require.EqualError(t, err, "script MockScript does not have a method run that returns an argument of type script.ExampleOutput: ABI arguments don't match Go types: ABI has 0 arguments, Go has 1")
	})

	t.Run("should fail if the specified method does not have exactly one output whose type matches the output type", func(t *testing.T) {
		script := &mockForgeScript{
			abi: abi.ABI{
				Methods: map[string]abi.Method{
					"run": abi.NewMethod("Run", "run", abi.Function, "", false, false, []abi.Argument{
						{
							Name: "_input",
							Type: die(abi.NewType("tuple", "", []abi.ArgumentMarshaling{{Name: "fieldA", Type: "address"}, {Name: "fieldB", Type: "address"}})),
						},
					}, []abi.Argument{
						{
							Name: "output_",
							Type: die(abi.NewType("uint256", "", []abi.ArgumentMarshaling{})),
						},
					}),
				},
			},
		}

		_, err := NewDeployScriptWithOutput[ExampleInput, ExampleOutput](script, "run")
		require.EqualError(t, err, "script MockScript does not have a method run that returns an argument of type script.ExampleOutput: ABI argument output_ at index 0 doesn't match Go type: ABI type uint256 (represented by *big.Int) is not assignable to Go type script.ExampleOutput")
	})

	t.Run("should not fail if the specified method has exactly one input whose type matches the input type and exactly one output whose type matches the output type", func(t *testing.T) {
		script := &mockForgeScript{
			abi: abi.ABI{
				Methods: map[string]abi.Method{
					"run": abi.NewMethod("Run", "run", abi.Function, "", false, false, []abi.Argument{
						{
							Name: "_input",
							Type: die(abi.NewType("tuple", "", []abi.ArgumentMarshaling{{Name: "fieldA", Type: "address"}, {Name: "fieldB", Type: "address"}})),
						},
					}, []abi.Argument{
						{
							Name: "output_",
							Type: die(abi.NewType("tuple", "", []abi.ArgumentMarshaling{{Name: "fieldC", Type: "address"}, {Name: "fieldD", Type: "address"}})),
						},
					}),
				},
			},
		}

		deployScript, err := NewDeployScriptWithOutput[ExampleInput, ExampleOutput](script, "run")
		require.NoError(t, err)
		require.NotNil(t, deployScript)
	})
}

func TestDeployScriptWithoutOutputImpl(t *testing.T) {
	makeDeployScript := func(t *testing.T) (*mockForgeScript, DeployScriptWithoutOutput[*big.Int]) {
		script := &mockForgeScript{
			abi: abi.ABI{
				Methods: map[string]abi.Method{
					"run": abi.NewMethod("Run", "run", abi.Function, "", false, false, []abi.Argument{
						{
							Name: "_input",
							Type: die(abi.NewType("uint256", "", []abi.ArgumentMarshaling{})),
						},
					}, []abi.Argument{}),
				},
			},
		}

		deployScript, err := NewDeployScriptWithoutOutput[*big.Int](script, "run")
		require.NoError(t, err)

		return script, deployScript
	}

	t.Run("should return an error if script.Call returns an error", func(t *testing.T) {
		script, deployScript := makeDeployScript(t)
		script.callError = fmt.Errorf("oh no")

		input := big.NewInt(1)
		err := deployScript.Run(input)
		require.Equal(t, die(script.abi.Pack("run", input)), script.callData)
		require.EqualError(t, err, "failed to run run method of script MockScript using:\n\n1\n\n: oh no")
	})

	t.Run("should not return an error if script.Call does not return an error", func(t *testing.T) {
		script, deployScript := makeDeployScript(t)
		script.callResult = []byte{1}

		input := big.NewInt(2)
		err := deployScript.Run(input)
		require.Equal(t, die(script.abi.Pack("run", input)), script.callData)
		require.NoError(t, err)
	})
}

func TestDeployScriptWithOutputImpl(t *testing.T) {
	makeDeployScript := func(t *testing.T) (*mockForgeScript, DeployScriptWithOutput[*big.Int, []*big.Int]) {
		script := &mockForgeScript{
			abi: abi.ABI{
				Methods: map[string]abi.Method{
					"run": abi.NewMethod("Run", "run", abi.Function, "", false, false, []abi.Argument{
						{
							Name: "_input",
							Type: die(abi.NewType("uint256", "", []abi.ArgumentMarshaling{})),
						},
					}, []abi.Argument{
						{
							Name: "output_",
							Type: die(abi.NewType("uint256[]", "", []abi.ArgumentMarshaling{})),
						},
					}),
				},
			},
		}

		deployScript, err := NewDeployScriptWithOutput[*big.Int, []*big.Int](script, "run")
		require.NoError(t, err)

		return script, deployScript
	}

	t.Run("should return an error if script.Call returns an error", func(t *testing.T) {
		script, deployScript := makeDeployScript(t)
		script.callError = fmt.Errorf("oh no")

		input := big.NewInt(1)
		output, err := deployScript.Run(input)
		require.Equal(t, die(script.abi.Pack("run", input)), script.callData)
		require.EqualError(t, err, "failed to run run method of script MockScript using:\n\n1\n\n: oh no")
		require.Nil(t, output)
	})

	t.Run("should not return an error if script.Call does not return an error", func(t *testing.T) {
		script, deployScript := makeDeployScript(t)

		expectedOutput := []*big.Int{big.NewInt(1), big.NewInt(2)}
		script.callResult = die(script.abi.Methods["run"].Outputs.Pack(expectedOutput))

		input := big.NewInt(2)
		output, err := deployScript.Run(input)
		require.Equal(t, die(script.abi.Pack("run", input)), script.callData)
		require.NoError(t, err)
		require.Equal(t, expectedOutput, output)
	})
}

func TestNewForgeScriptFromFile(t *testing.T) {
	t.Run("should deploy and execute an example script", func(t *testing.T) {
		type DeployScriptExampleInput struct {
			InputFieldA common.Address `abi:"fieldA"`
			InputFieldB common.Address `abi:"fieldB"`
		}

		type DeployScriptExampleOutput struct {
			OutputFieldA common.Address `abi:"fieldA"`
			OutputFieldB common.Address `abi:"fieldB"`
		}

		// First we'll setup the required dependencies
		logger, _ := testlog.CaptureLogger(t, log.LevelInfo)
		af := foundry.OpenArtifactsDir("./testdata/test-artifacts")
		host := NewHost(logger, af, nil, DefaultContext)

		// We'll use an example script that returns the input data, just mapped to a different struct
		script, err := NewForgeScriptFromFile(host, "DeployScriptExample.s.sol", "DeployScriptExample")
		require.NoError(t, err)

		// We wrap the script with a strongly-typed wrapper
		deploySuperchain, err := NewDeployScriptWithOutput[DeployScriptExampleInput, DeployScriptExampleOutput](script, "run")
		require.NoError(t, err)

		// Put some input & expected output together
		input := DeployScriptExampleInput{
			InputFieldA: common.BigToAddress(big.NewInt(7)),
			InputFieldB: common.BigToAddress(big.NewInt(6)),
		}
		expectedOutput := DeployScriptExampleOutput{
			OutputFieldA: input.InputFieldA,
			OutputFieldB: input.InputFieldB,
		}

		// And make sure that we get what we would expect
		output, err := deploySuperchain.Run(input)
		require.NoError(t, err)
		require.Equal(t, expectedOutput, output)

		// Now we make sure (and this depends on the contract logic) that reverts are handled
		zeroInput := DeployScriptExampleInput{
			InputFieldA: common.BigToAddress(big.NewInt(0)),
			InputFieldB: common.BigToAddress(big.NewInt(0)),
		}
		_, err = deploySuperchain.Run(zeroInput)
		require.ErrorContains(t, err, "failed to call script DeployScriptExample using data 0xfc61915400000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000: failed to call backend: execution reverted at")
	})
}

type mockForgeScriptBackend struct {
	callResult []byte
	callError  error

	calledTo   common.Address
	calledWith []byte

	deployResult common.Address
	deployError  error

	destroyedAddress common.Address
}

// Call implements ForgeScriptBackend.
func (m *mockForgeScriptBackend) Call(to common.Address, input []byte) (result []byte, err error) {
	m.calledTo = to
	m.calledWith = input

	return m.callResult, m.callError
}

// Deploy implements ForgeScriptBackend.
func (m *mockForgeScriptBackend) Deploy(artifact *foundry.Artifact, label string) (address common.Address, err error) {
	return m.deployResult, m.deployError
}

// Destroy implements ForgeScriptBackend.
func (m *mockForgeScriptBackend) Destroy(address common.Address) {
	m.destroyedAddress = address
}

type mockForgeScript struct {
	abi        abi.ABI
	callData   []byte
	callResult []byte
	callError  error
}

// ABI implements ForgeScript.
func (m *mockForgeScript) ABI() abi.ABI {
	return m.abi
}

// Call implements ForgeScript.
func (m *mockForgeScript) Call(input []byte) (result []byte, err error) {
	m.callData = input

	return m.callResult, m.callError
}

// Name implements ForgeScript.
func (m *mockForgeScript) Name() string {
	return "MockScript"
}

var (
	_ ForgeScript        = (*mockForgeScript)(nil)
	_ ForgeScriptBackend = (*mockForgeScriptBackend)(nil)
)
