package script

import (
	"fmt"
	"reflect"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script/addresses"
	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// ForgeScriptBackend is a minimal interface suitable for deploying & interacting with
// foundry scripts on chain
type ForgeScriptBackend interface {
	// Call sends a transaction with a []byte data to a specified address
	Call(to common.Address, input []byte) (result []byte, err error)

	// Deploy deploys a contract based on an artifact, attaches metadata and labels the deployment
	Deploy(artifact *foundry.Artifact, label string) (address common.Address, err error)

	// Destroy wipes the specified address
	Destroy(address common.Address)
}

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
	_ ForgeScriptBackend               = (*forgeScriptBackendImpl)(nil)
	_ ForgeScript                      = (*forgeScriptImpl)(nil)
	_ DeployScriptWithoutOutput[any]   = (*deployScriptWithoutOutputImpl[any])(nil)
	_ DeployScriptWithOutput[any, any] = (*deployScriptWithOutputImpl[any, any])(nil)
)

// NewForgeScriptBackend creates an instance of ForgeScriptBackend
func NewForgeScriptBackend(host *Host) ForgeScriptBackend {
	return &forgeScriptBackendImpl{
		host: host,
	}
}

// NewForgeScriptFromFile loads a script artifact from the artifact filesystem on the host,
// creates a ForgeScriptBackend from the host and finally creates a ForgeScript instance based on these
func NewForgeScriptFromFile(host *Host, fileName string, name string) (ForgeScript, error) {
	artifact, err := host.Artifacts().ReadArtifact(fileName, name)
	if err != nil {
		return nil, fmt.Errorf("failed to load script %s from %s: %w", name, fileName, err)
	}

	backend := NewForgeScriptBackend(host)

	return NewForgeScriptFromArtifact(artifact, name, backend), nil
}

// NewForgeScriptFromArtifact creates a ForgeScript instance based on a foundry artifact,
// a name (used as a script label) and a ForgeScriptBackend instance
func NewForgeScriptFromArtifact(artifact *foundry.Artifact, name string, backend ForgeScriptBackend) ForgeScript {
	return &forgeScriptImpl{
		artifact: artifact,
		backend:  backend,
		name:     name,
	}
}

// NewDeployScriptWithoutOutputFromFile is a syntactic sugar around NewForgeScriptFromFile and NewDeployScriptWithoutOutput
func NewDeployScriptWithoutOutputFromFile[I any](host *Host, fileName string, name string) (DeployScriptWithoutOutput[I], error) {
	script, err := NewForgeScriptFromFile(host, fileName, name)
	if err != nil {
		return nil, err
	}

	return NewDeployScriptWithoutOutput[I](script, "run")
}

// NewDeployScriptWithOutputFromFile is a syntactic sugar around NewForgeScriptFromFile and NewDeployScriptWithOutput
func NewDeployScriptWithOutputFromFile[I any, O any](host *Host, fileName string, name string) (DeployScriptWithOutput[I, O], error) {
	script, err := NewForgeScriptFromFile(host, fileName, name)
	if err != nil {
		return nil, err
	}

	return NewDeployScriptWithOutput[I, O](script, "run")
}

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

// forgeScriptBackendImpl implements ForgeScriptBackend and encapsulates low-level deployment logic for scripts
//
// Its main purpose is to simplify testing and clearly separate the Host from the deploy scripts
type forgeScriptBackendImpl struct {
	host *Host
}

// Call sends a transaction to a contract
func (b *forgeScriptBackendImpl) Call(to common.Address, input []byte) (result []byte, err error) {
	result, _, err = b.host.Call(b.host.env.TxContext().Origin, to, input, DefaultFoundryGasLimit, uint256.NewInt(0))
	if err != nil {
		return nil, fmt.Errorf("failed to call backend: %w", err)
	}

	return result, nil
}

// Deploy deploys a contract specified by a foundry artifact
//
// It will name the contract using the provided label and save the contract source maps
// for ease of debugging
func (b *forgeScriptBackendImpl) Deploy(artifact *foundry.Artifact, label string) (address common.Address, err error) {
	deployer := addresses.ScriptDeployer
	deployNonce := b.host.state.GetNonce(deployer)

	// Compute address of script contract to be deployed
	address = crypto.CreateAddress(deployer, deployNonce)

	// Label the address using the contract name
	b.host.Label(address, label)

	b.host.AllowCheatcodes(address)    // before constructor execution, give our script cheatcode access
	b.host.state.MakeExcluded(address) // scripts are persistent across forks

	// disable contract size constraints
	b.host.EnforceMaxCodeSize(false)
	defer b.host.EnforceMaxCodeSize(true)

	// deploy the script
	deployedAddr, err := b.host.Create(deployer, artifact.Bytecode.Object)
	if err != nil {
		return address, fmt.Errorf("failed to deploy script %s: %w", label, err)
	}

	// make sure we deployed to the expected address
	if deployedAddr != address {
		return address, fmt.Errorf("deployed script %s to unexpected address %s, expected %s", label, deployedAddr, address)
	}

	// save the contract source map
	b.host.RememberArtifact(address, artifact, label)

	// and return the script address
	return address, nil
}

// Destroy will wipe an address
func (b *forgeScriptBackendImpl) Destroy(address common.Address) {
	b.host.Wipe(address)
}

// forgeScriptImpl is an implementation of ForgeScript
type forgeScriptImpl struct {
	artifact *foundry.Artifact
	backend  ForgeScriptBackend
	name     string
}

// Call deploys the script contract, sends transaction payload and destroys the contract
func (s *forgeScriptImpl) Call(input []byte) (output []byte, err error) {
	address, err := s.backend.Deploy(s.artifact, s.name)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy script %s: %w", s.name, err)
	}

	defer s.backend.Destroy(address)

	output, err = s.backend.Call(address, input)
	if err != nil {
		return nil, fmt.Errorf("failed to call script %s using data 0x%s: %w", s.name, common.Bytes2Hex(input), err)
	}

	return output, nil
}

func (d *forgeScriptImpl) Name() string {
	return d.name
}

func (d *forgeScriptImpl) ABI() abi.ABI {
	return d.artifact.ABI
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
