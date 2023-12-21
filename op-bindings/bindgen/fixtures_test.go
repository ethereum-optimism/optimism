package main

// The Init bytecode used for these tests can either be sourced
// on-chain using the deployment tx of these contracts, or can be
// found in the bindings output from BindGen (../bindings/)
var removeDeploymentSaltTests = []struct {
	name           string
	deploymentData string
	deploymentSalt string
	expected       string
}{
	{
		"TestRemoveDeploymentSalt Case #1",
		Safe_v130InitBytecode,
		"0000000000000000000000000000000000000000000000000000000000000000",
		Safe_v130InitBytecodeNoSalt,
	},
	{
		"TestRemoveDeploymentSalt Case #2",
		Permit2InitBytecode,
		"0000000000000000000000000000000000000000d3af2663da51c10215000000",
		Permit2InitBytecodeNoSalt,
	},
	{
		"TestRemoveDeploymentSalt Case #3",
		EntryPointInitBytecode,
		"0000000000000000000000000000000000000000000000000000000000000000",
		EntryPointInitBytecodeNoSalt,
	},
}

var removeDeploymentSaltTestsFailures = []struct {
	name           string
	deploymentData string
	deploymentSalt string
	expectedError  string
}{
	{
		"TestRemoveDeploymentSalt Failure Case #1 Invalid Regex",
		"0x1234abc",
		"[invalid-regex",
		"failed to compile regular expression: error parsing regexp: missing closing ]: `[invalid-regex)`",
	},
	{
		"TestRemoveDeploymentSalt Failure Case #2 Salt Not Found",
		"0x1234abc",
		"4567",
		"expected salt: 4567 to be at the beginning of the contract initialization code: 0x1234abc, but it wasn't",
	},
}
