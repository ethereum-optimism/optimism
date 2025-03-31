package script

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

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
	_ ForgeScript = (*mockForgeScript)(nil)
)
