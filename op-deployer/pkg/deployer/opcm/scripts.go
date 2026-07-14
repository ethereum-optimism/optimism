package opcm

// Scripts contains all the deployment scripts for ease of passing them around. Build the
// engine-backed bundle with scriptbackend.NewEngineScripts.
type Scripts struct {
	DeployAlphabetVM      DeployAlphabetVMScript
	DeployAltDA           DeployAltDAScript
	DeployDisputeGame     DeployDisputeGameScript
	DeployImplementations DeployImplementationsScript
	DeployMIPS            DeployMIPSScript
	DeploySuperchain      DeploySuperchainScript
	DeployOPChain         DeployOPChainScript
}
