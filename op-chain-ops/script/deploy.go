package script

import (
	"fmt"
	"reflect"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// ForgeScript is a generic script instance
type ForgeScript interface {
	// Underlying script ABI
	ABI() abi.ABI

	// Script name (mostly for logging purposes)
	Name() string

	// Sends the input as a payload to the script contract
	Call(input []byte) (result []byte, err error)
}

// DeployScriptWithoutOutput is a specific ForgeScript that accepts typed input
// and calls a specific script entrypoint (the run method by convention)
//
// The method is assumed to return nothing (empty bytes) and the output is discarded
type DeployScriptWithoutOutput[I any] interface {
	ForgeScript
	Run(input I) (err error)
}

// DeployScriptWithOutput is a specific ForgeScript that accepts typed input
// and calls a specific script entrypoint (the run method by convention)
//
// The method is assumed to return a single value of type O
type DeployScriptWithOutput[I any, O any] interface {
	ForgeScript
	Run(input I) (output O, err error)
}

// We make sure that our implementations match the interfaces above
var (
	_ DeployScriptWithoutOutput[any]   = (*deployScriptWithoutOutputImpl[any])(nil)
	_ DeployScriptWithOutput[any, any] = (*deployScriptWithOutputImpl[any, any])(nil)
)

// NewDeployScriptWithoutOutput creates an instance of DeployScriptWithoutOutput[I], a void-returning deploy script
func NewDeployScriptWithoutOutput[I any](script ForgeScript, methodName string) (DeployScriptWithoutOutput[I], error) {
	return newDeployScriptWithoutOutput[I](script, methodName)
}

// NewDeployScriptWithOutput creates an instance of DeployScriptWithoutOutput[I, O], a result-returning deploy script
func NewDeployScriptWithOutput[I any, O any](script ForgeScript, methodName string) (DeployScriptWithOutput[I, O], error) {
	return newDeployScriptWithOutput[I, O](script, methodName)
}

// newDeployScriptWithoutOutput creates an instance of deployScriptWithoutOutputImpl[I]
//
// It is used internally to maximize code reuse:
// - its return value is returned from NewDeployScriptWithoutOutput (but returned as an interface, not leaking the implementation details)
// - its return values is used internally in newDeployScriptWithOutput that relies on the implementation details
func newDeployScriptWithoutOutput[I any](script ForgeScript, methodName string) (*deployScriptWithoutOutputImpl[I], error) {
	// Just to keep things DRY a bit
	scriptName := script.Name()

	// Make sure the method exists on the ABI
	method, ok := script.ABI().Methods[methodName]
	if !ok {
		return nil, fmt.Errorf("script %s does not have a method called %s", scriptName, methodName)
	}

	// Now make sure the ABI has exactly one argument of the correct type
	inputType := reflect.TypeOf(*new(I))
	err := matchArguments(method.Inputs, inputType)
	if err != nil {
		return nil, fmt.Errorf("script %s does not have a method %s that accepts an argument of type %v: %w", scriptName, methodName, inputType, err)
	}

	// Then after all that we're good to create the script
	return &deployScriptWithoutOutputImpl[I]{
		script: script,
		method: method,
	}, nil
}

// newDeployScriptWithOutput creates an instance of deployScriptWithOutputImpl[I, O]
//
// Although we don't need to reuse it similar to newDeployScriptWithoutOutput, it is nice to keep things symmetrical and predictable
// so its pattern copies the one of newDeployScriptWithoutOutput
func newDeployScriptWithOutput[I any, O any](script ForgeScript, methodName string) (*deployScriptWithOutputImpl[I, O], error) {
	// First validate the input by creating an instance of deployScriptWithoutOutputImpl[I]
	deployScriptWithoutOutputImpl, err := newDeployScriptWithoutOutput[I](script, methodName)
	if err != nil {
		return nil, err
	}

	// Now make sure the return value matches the ABI
	outputType := reflect.TypeOf(*new(O))
	err = matchArguments(deployScriptWithoutOutputImpl.method.Outputs, outputType)
	if err != nil {
		return nil, fmt.Errorf("script %s does not have a method %s that returns an argument of type %v: %w", script.Name(), methodName, outputType, err)
	}

	// Then after all that we're good to create the script
	return &deployScriptWithOutputImpl[I, O]{
		deployScriptWithoutOutputImpl: *deployScriptWithoutOutputImpl,
	}, nil
}

// deployScriptWithoutOutputImpl[I] implements DeployScriptWithoutOutput[I]
type deployScriptWithoutOutputImpl[I any] struct {
	script ForgeScript
	method abi.Method
}

// ABI implements ForgeScript.
func (d *deployScriptWithoutOutputImpl[I]) ABI() abi.ABI {
	return d.script.ABI()
}

// Call implements ForgeScript.
func (d *deployScriptWithoutOutputImpl[I]) Call(input []byte) (result []byte, err error) {
	return d.script.Call(input)
}

// Name implements ForgeScript.
func (d *deployScriptWithoutOutputImpl[I]) Name() string {
	return d.script.Name()
}

// run is a helper function that encodes the input (that represents arguments to an ABI method) and returns the raw result
//
// It exists so that deployScriptWithoutOutputImpl and deployScriptWithOutputImpl can share the input encoding logic
func (d *deployScriptWithoutOutputImpl[I]) run(input I) (result []byte, err error) {
	// Just to keep things DRY a tiny bit
	scriptName := d.Name()
	methodName := d.method.RawName

	packed, err := d.ABI().Pack(methodName, input)
	if err != nil {
		return nil, fmt.Errorf("failed to encode input for %s method of script %s using:\n\n%v\n\n: %w", methodName, scriptName, input, err)
	}

	result, err = d.Call(packed)
	if err != nil {
		return nil, fmt.Errorf("failed to run %s method of script %s using:\n\n%v\n\n: %w", methodName, scriptName, input, err)
	}

	return result, nil
}

// Run implements DeployScriptWithoutOutput[I].
func (d *deployScriptWithoutOutputImpl[I]) Run(input I) (err error) {
	_, err = d.run(input)

	return err
}

// deployScriptWithOutputImpl[I, O] implements DeployScriptWithOutput[I, O]
//
// It embeds deployScriptWithoutOutputImpl[I] to be able to reuse the run method
// and not have to worry about input encoding
type deployScriptWithOutputImpl[I any, O any] struct {
	deployScriptWithoutOutputImpl[I]
}

// Run implements DeployScriptWithOutput.
func (d *deployScriptWithOutputImpl[I, O]) Run(input I) (output O, err error) {
	// Just to keep things DRY a tiny bit
	scriptName := d.Name()
	methodName := d.method.RawName

	// We use the run to get the raw output of the contract call
	result, err := d.deployScriptWithoutOutputImpl.run(input)
	if err != nil {
		return output, err
	}

	// We then decode the raw output to an anonymous struct
	unpacked, err := d.ABI().Unpack(methodName, result)
	if err != nil {
		return output, fmt.Errorf("failed to decode output for %s method of script %s using data 0x%s: %w", methodName, scriptName, common.Bytes2Hex(result), err)
	}

	// And finally we convert the anonymous struct into our typed output
	return *abi.ConvertType(unpacked[0], new(O)).(*O), nil
}
