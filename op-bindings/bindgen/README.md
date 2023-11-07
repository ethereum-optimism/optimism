![BindGen Header Image](./bindgen_header.png)

A CLI for generating Go bindings from Forge artifacts and API clients such as Etherscan's

# Dependencies

- [Go](https://go.dev/dl/)
- [Foundry](https://getfoundry.sh/)
- [pnpm](https://pnpm.io/installation)

If you're running the CLI inside the Optimism monorepo, please make sure you've executed `pnpm i` to install all of the monorepo's dependencies.

# Makefile Commands

In [op-bindings Makefile](../Makefile), there are a few commands available that properly configure BindGen:

#### `bindgen`

Runs: `compile bindgen-generate-all`

This command will compile the Forge artifacts for the contract found in [contracts-bedrock](../../packages/contracts-bedrock/), then generates the bindings for locally and remotely sourced contracts.

- `ETHERSCAN_APIKEY` is required to be set, [here's a guide](https://docs.etherscan.io/getting-started/viewing-api-usage-statistics) on how to obtain a key
- `ETHERSCAN_OP_APIKEY` is required to be set since `--compare-deployment-bytecode` and `--compare-init-bytecode` flags are enabled.
    - The `--source-chainid` is configured to be `1` (Ethereum Mainnet) and the `--compare-chainid` is `10` (Optimism Mainnet).

```bash
ETHERSCAN_APIKEY=YOUR_KEY ETHERSCAN_OP_APIKEY=YOUR_KEY make bindgen
```

#### `bindgen-local`

Runs: `compile bindgen-generate-local`

This command will compile the Forge artifacts for the contract found in [contracts-bedrock](../../packages/contracts-bedrock/), then generates the bindings for locally sourced contracts.

```bash
make bindgen-local
```

#### `bindgen-remote`

Runs: `bindgen-generate-remote`

This command will generate the bindings for remotely sourced contracts.

- `ETHERSCAN_APIKEY` is required to be set, [here's a guide](https://docs.etherscan.io/getting-started/viewing-api-usage-statistics) on how to obtain a key
- `ETHERSCAN_OP_APIKEY` is required to be set since `--compare-deployment-bytecode` and `--compare-init-bytecode` flags are enabled.
    - The `--source-chainid` is configured to be `1` (Ethereum Mainnet) and the `--compare-chainid` is `10` (Optimism Mainnet).

```bash
ETHERSCAN_APIKEY=YOUR_KEY ETHERSCAN_OP_APIKEY=YOUR_KEY make bindgen-remote
```

# Using BindGen to Add Predeploys to L2 Genesis

This CLI util was originally built to generate the bindings for the Forge artifacts generated for [contracts-bedrock](../../packages/contracts-bedrock/), but has been extended to allow for bindings to be generated for deployed contracts. Furthermore, the generated bindings are used in the [L2 genesis generation script](../../op-chain-ops/genesis/layer_two.go) to add the contracts as predeploys for a new OP chain, this is an overview on how that's done:

The first step in adding a predeploy to L2 is to add the contract to your contracts list. [artifacts.json](../artifacts.json) is an implementation of this list and separates the list into `local` and `remote` contracts.

## Local Contracts

`local` contracts have their Forge artifacts available within the monorepo in the [contracts-bedrock](../../packages/contracts-bedrock/) package. When generating the Go bindings for these contracts, BindGen will look for each contract's Forge artifact under `contracts-bedrock/forge-artifacts` and use it as the source for the contract's ABI, initialization and deployment bytecode, etc.

To add a `local` contract, the Solidity file must be included under `contracts-bedrock/src`, so that Forge will compile an artifact file for it when [bindgen](#bindgen) or [bindgen-local](#bindgen-local) is ran. After that, running either `Makefile` command will generate a bindings file for your contract under [bindings](../bindings/).

### Predeploy Addresses File

Next you will need to add your contract's name and predeploy address in the [predeploys addresses file](../predeploys/addresses.go). Make sure to:

1. Add you contract's name equal to it's predeploy address as a const

```go
const (
	L2ToL1MessagePasser           = "0x4200000000000000000000000000000000000016"
	DeployerWhitelist             = "0x4200000000000000000000000000000000000002"
	WETH9                         = "0x4200000000000000000000000000000000000006"
    ...
    MyNewPredeploy                = "0xd9145CCE52D386f254917e481eB44e9943F39138"
)
```

2. Initialize a `common.Address` variable for your contract using it's name + `Addr`

```go
var (
	L2ToL1MessagePasserAddr           = common.HexToAddress(L2ToL1MessagePasser)
	DeployerWhitelistAddr             = common.HexToAddress(DeployerWhitelist)
	WETH9Addr                         = common.HexToAddress(WETH9)
    ...
    MyNewPredeployAddr                = common.HexToAddress(MyNewPredeploy)
)
```

3. If your contract is **not** behind a proxy, add it the `switch` statement

```go
func IsProxied(predeployAddr common.Address) bool {
	switch predeployAddr {
	case WETH9Addr:
	case GovernanceTokenAddr:
	...
	case MyNewPredeploy:
	default:
		return true
	}
	return false
}
```

4. Lastly, add it to the `Predeploys` slice

```go
func init() {
	Predeploys["L2ToL1MessagePasser"] = &L2ToL1MessagePasserAddr
	Predeploys["DeployerWhitelist"] = &DeployerWhitelistAddr
	Predeploys["WETH9"] = &WETH9Addr
    ...
	Predeploys["MyNewPredeploy"] = &MyNewPredeployAddr
}
```

### L2 Genesis Script

You will now need to add your contract to be included in the [L2 genesis generation script](../../op-chain-ops/genesis/layer_two.go). First, if your contract doesn't contain any `immutable` variables or doesn't depend on any chain specific properties such as the chain ID, add your contract's name to the first `case` in `BuildL2Genesis`'s `switch` statement

```go
for name, predeploy := range predeploys.Predeploys {
		addr := *predeploy

		codeAddr := addr
		switch name {
		case "SafeL2", "MultiSendCallOnly", "Multicall3", "Create2Deployer", "SafeSingletonFactory", "DeterministicDeploymentProxy", "MyNewPredeploy":
			db.CreateAccount(addr)
        case ...
```

`db.CreateAccount(addr)` will initialize the storage slot in the chain database for your address.

The script will then call `setupPredeploy` which will set the deployed bytecode for your contract at your given address.

### Updating Tests

After adding a new predeploy, there are two tests that need to be updated:

1. [check-l2](../../op-chain-ops/cmd/check-l2/main.go)

This script is intended to be ran when you have a devnet setup to verify the L2 was setup as intended.

Within the `checkPredeployConfig` function, add a `case` to the `switch` case for your predeploy. The simplest test case would be to just `checkPredeployBytecode` like so:

```go
case predeploys.MyNewPredeployAddr:
    bytecode, err := bindings.GetDeployedBytecode("MyNewPredeploy")
    if err != nil {
        return err
    }
    if err := checkPredeployBytecode(p, "MyNewPredeploy", client, bytecode); err != nil {
        return err
    }
```

`checkPredeployBytecode` will use `eth_getCode` to obtain the code stored at your predeploy's address and compare it against the expected deployed bytecode from BindGen's generated bindings.

2. [layer_two_test](../../op-chain-ops/genesis/layer_two_test.go)

Update the equality check for `len(gen.Alloc)` in both `TestBuildL2MainnetGenesis` and `TestBuildL2MainnetNoGovernanceGenesis` tests to the new size of the genesis `alloc` (that's now including your new predeploy).

## Remote Contracts

`remote` contracts don't have Forge artifacts, but instead have their details fetched from a remote `contractDataClient` such as Etherscan.

To add a `remote` contract, you must configure an object within the contracts list file to give BindGen all the info it needs to generate the Go bindings. [artifacts.json](../artifacts.json) has several configurations for various contracts that can be referenced, and for an explanation of what each config property does, see the [remote contract list file](#more-info-on-remote-contracts-contract-list-file) section.

Similarly to adding a `local` contract, after adding a config object to your contracts list file for your new remote contract, you must follow the same steps covered in the [Predeploy Addresses file](#predeploy-addresses-file) section. Next you will follow the [L2 Genesis Script](#l2-genesis-script) section, however different steps are required here if your contract contains `immutable` variables, otherwise the steps are identical. Lastly follow the [Update Tests](#updating-tests) section.

### Adding a Predeploy with Immutable Variables

If your contract has [manuallyResolveImmutables](#manuallyResolveImmutables) enabled, then BindGen did **not** include the deployed bytecode for your contract as part of the generated bindings file. This is because `immutable` values are set upon execution of a contract's initialization bytecode, and have the possibility to be dependent on properties that may not be the same on every chain or need to be set to a specific value.

As an example of this scenario, Uniswap's `Permit2` contract uses the [EIP712](https://github.com/Uniswap/permit2/blob/cc56ad0f3439c502c246fc5cfcc3db92bb8b7219/src/EIP712.sol) contract which has an `immutable` that's dependant on `block.chainid`

```solidity
contract EIP712 is IEIP712 {
    // Cache the domain separator as an immutable value, but also store the chain id that it
    // corresponds to, in order to invalidate the cached domain separator if the chain id changes.
    bytes32 private immutable _CACHED_DOMAIN_SEPARATOR;
    uint256 private immutable _CACHED_CHAIN_ID;

    bytes32 private constant _HASHED_NAME = keccak256("Permit2");
    bytes32 private constant _TYPE_HASH =
        keccak256("EIP712Domain(string name,uint256 chainId,address verifyingContract)");

    constructor() {
        _CACHED_CHAIN_ID = block.chainid;
        _CACHED_DOMAIN_SEPARATOR = _buildDomainSeparator(_TYPE_HASH, _HASHED_NAME);
    }
    ...
```
So, to make sure our deployed bytecode has this value set correctly, we must be able to deploy `Permit2` to a chain that has the same chain ID of our soon-to-be L2 chain. The [L2 Genesis Script](#l2-genesis-script) will handle spinning up a simulated backend for us to deploy to, but we need to configure it to do so for our contract.

#### L2 Genesis Script with Immutable Variables

Instead of following the steps in the above [L2 Genesis Script](#l2-genesis-script) section, we will be adding our contract's name to the **second** `case` in `BuildL2Genesis`'s `switch` statement

```go
for name, predeploy := range predeploys.Predeploys {
    addr := *predeploy

    codeAddr := addr
    switch name {
    case ...
    case "Permit2", "EntryPoint", "MultiSend", "MyNewPredeploy":
        deployerAddress, err := bindings.GetDeployerAddress(name)
        if err != nil {
            return nil, err
        }
        deployerAddressPtr := common.BytesToAddress(deployerAddress)
        predeploys := map[string]*common.Address{
            "DeterministicDeploymentProxy": &deployerAddressPtr,
        }
        backend, err := deployer.NewL2BackendWithChainIDAndPredeploys(
            new(big.Int).SetUint64(config.L2ChainID),
            predeploys,
        )
        if err != nil {
            return nil, err
        }
        deployedBin, err := deployer.DeployWithDeterministicDeployer(backend, name)
        if err != nil {
            return nil, err
        }
        deployResults[name] = deployedBin
        db.CreateAccount(addr)
```

This `case` assumes your contract was deployed using [Arachnid's Deterministic Deployment Proxy](https://github.com/Arachnid/deterministic-deployment-proxy), and will create a simulated L2 backend using `deployer.NewL2BackendWithChainIDAndPredeploys` (your L2's chain ID will be sourced from your provided [DeployConfig](../../op-chain-ops/genesis/config.go)) with the proxy deployer set at it's expected address (`0x4e59b44847b379578588920cA78FbF26c0B4956C`) as a predeploy. `deployer.DeployWithDeterministicDeployer` will then use the proxy deployer and your contract's initialization bytecode to deploy your contract to the simulated backend, setting the `immutable` variables. This functions will also validate that the resulting address of the simulated deployment matches the computed `CREATE2` address using the [create2DeployerAddress](#create2deployeraddress) and [deploymentsalt](#deploymentsalt) values you provided in the contracts list file, and your contract's initialization bytecode.

This code has a lot of built-in assumptions, but satisfies the requirements needed to predeploy `Permit2`, EIP-4337's `EntryPoint`, and Safe's `MultiSend` contracts. Please use this implementation as reference code if you need to add additional custom logic to handle a contract with `immutable` variables that has other requirements.

Lastly, `db.CreateAccount(addr)` is called initializing the storage slot in the chain database for your address, and the script will call `setupPredeploy` which will set the deployed bytecode for your contract at your given address for your L2.

#### Updating Tests for Immutables

We don't have to update [check-l2](../../op-chain-ops/cmd/check-l2/main.go), because we don't have static deployed bytecode to use for the equality check, but you will need to adjust the expected length of the genesis `alloc` in the [layer_two_test](../../op-chain-ops/genesis/layer_two_test.go) as mentioned in the previous [Updating Tests](#updating-tests) section.

If you want to sanity check the `CREATE2` address generation logic `deployer.DeployWithDeterministicDeployer` uses, you can add your contract details to [deployer_test.go](../../op-chain-ops/deployer/deployer_test.go).

# CLI Commands

## `generate`

This is the main command of BindGen and it expects one of three subcommands:

### Subcommands

Command  | Description                                                              | Flags                            | Usage
-------- | ------------------------------------------------------------------------ | -------------------------------- | ---------------------------------
`all`    | Generates bindings for both local and remote contracts.                  | Combines local and remote flags. | `bindgen generate all [flags]`
`local`  | Generates bindings for contracts with locally available Forge artifacts. | [Local Flags](#local-flags)      | `bindgen generate local [flags]`
`remote` | Generates bindings for contracts from a remote source.                   | [Remote Flags](#remote-flags)    | `bindgen generate remote [flags]`

# Flags

## Global Flags

Flag            | Type   | Description                                                                    | Required
--------------- | ------ | ------------------------------------------------------------------------------ | --------
`metadata-out`  | String | Output directory for contract metadata files                                   | Yes
`go-package`    | String | Go package name for generated bindings                                         | Yes
`monorepo-base` | String | Path to the base of the monorepo                                               | Yes
`log.level`     | String | Log level (`none`, `debug`, `info`, `warn`, `error`, `crit`) (Default: `info`) | No

## Local Flags

Flag               | Type   | Description                                                        | Required
------------------ | ------ | ------------------------------------------------------------------ | --------
`local-contracts`  | String | Path to file with a list of local contracts to create bindings for | Yes
`forge-artifacts`  | String | Path to the directory with compiled Forge artifacts                | Yes
`source-maps-list` | String | Comma-separated list of contracts to generate source-maps for      | No

## Remote Flags

Flag                          | Type   | Description                                                                                                                                                         | Required
----------------------------- | ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------
`client`                      | String | Name of the remote client to connect to for contract data queries (currently `etherscan` is the only available option)                                                     | Yes
`remote-contracts`            | String | Path to file with a list of remote contracts to create bindings for ([More info](#more-info-on-remote-contracts-contract-list-file))                                                                                                | Yes
`source-chainid`              | Int    | Chain ID of the network `client` will connect to for fetching contract data such as bytecode and deployment transaction (currently only `1` and `10` are supported) | Yes
`source-apikey`               | String | API key `client` will use for auth when making requests to the source chain data source                                                                             | Yes
`compare-chainid`             | Int    | Chain ID of the network `client` will connect to for bytecode comparisons (currently only `1` and `10` are supported)                                               | No
`compare-apikey`              | String | API key `client` will use for auth when making requests to the compare chain data source                                                                            | No
`api-max-retries`             | Int    | Max retries for fetching data via the `client` if the request fails (Default: `3`)                                                                                  | No
`api-retry-delay`             | Int    | Delay in seconds between retries (Default: `2`)                                                                                                                     | No
`compare-deployment-bytecode` | Bool   | Signals BindGen to compare deployment bytecode from the source chain vs the compare chain for each contract (Default: `false`)                                      | No
`compare-init-bytecode`       | Bool   | Signals BindGen to compare initialization bytecode from the source chain vs the compare chain for each contract (Default: `false`)                                  | No

### More Info on `remote-contracts` Contract List File

[artifacts.json](../artifacts.json) is an implementation of both a local and remote contract list file. Below is a snippet of that file that we'll breakdown:

```json
{
    "local": [...],
    "remote": [
        {
            "name": "Create2Deployer",
            "verified": true,
            "manuallyResolveImmutables": false,
            "create2ProxyDeployed": false,
            "deployments": {
                "1": "0xF49600926c7109BD66Ab97a2c036bf696e58Dbc2",
                "10": "0x13b0D85CcB8bf860b6b79AF3029fCA081AE9beF2"
            },
            "useDeploymentBytecodeFromChainId": 1,
            "useInitBytecodeFromChainId": 1
        },
        {
            "name": "SafeL2",
            "verified": true,
            "manuallyResolveImmutables": false,
            "create2ProxyDeployed": true,
            "DeploymentSalt": "0000000000000000000000000000000000000000000000000000000000000000",
            "deployments": {
                "1": "0x3E5c63644E683549055b9Be8653de26E0B4CD36E",
                "10": "0xfb1bffC9d739B8D520DaF37dF666da4C687191EA"
            }
        },
        {
            "name": "SafeSingletonFactory",
            "verified": false,
            "manuallyResolveImmutables": false,
            "create2ProxyDeployed": false,
            "abi": "[{\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"fallback\",\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"salt\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"creationCode\",\"type\":\"bytes\"}]}]",
            "deployments": {
                "1": "0x914d7Fec6aaC8cd542e72Bca78B30650d45643d7",
                "10": "0x914d7Fec6aaC8cd542e72Bca78B30650d45643d7"
            }
        },
        {
            "name": "MultiSend",
            "verified": true,
            "manuallyResolveImmutables": true,
            "create2ProxyDeployed": true,
            "DeploymentSalt": "0000000000000000000000000000000000000000000000000000000000000000",
            "create2DeployerAddress": "0x914d7fec6aac8cd542e72bca78b30650d45643d7",
            "deployments": {
                "1": "0xA238CBeb142c10Ef7Ad8442C6D1f9E89e07e7761",
                "10": "0x998739BFdAAdde7C933B942a68053933098f9EDa"
            }
        },
    ]
}
```

#### `name`

This property is the name of the contract and will be used as the name of the generated Go bindings file.

#### `verified`

- `true`

    This signals BindGen that the contract is verified on the contract data source the `client` is connected to for the source chain (and the compare chain if bytecode comparison is enabled). This means BindGen is able to fetch the verified ABI and the deployment transaction hash for the contract from the contract data source.

- `false`

    This signals BindGen that the contract is **not** verified on the contract data source the `client` is connected to for the source chain (and the compare chain if bytecode comparison is enabled). This means BindGen is expecting the ABI and the deployment transaction hash for the contract to be provided in the contract list file like so:
    ```json
    {
        "name": "SafeSingletonFactory",
        "verified": false,
        "manuallyResolveImmutables": false,
        "Create2ProxyDeployed": false,
        "deploymentTxHashes": {
            "1": "0x69c275b5304db980105b7a6d731f9e1157a3fe29e7ff6ff95235297df53e9928"
        },
        "abi": "[{\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"fallback\",\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"salt\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"creationCode\",\"type\":\"bytes\"}]}]",
        "deployments": {...}
    }
    ```
    At the very least, the deployment transaction hash on the source chain **must** be provided. If

#### `manuallyResolveImmutables`

- `true`

    This signals BindGen that the contract has `immutable` variables and therefore the deployed bytecode is not relevant and will need to obtained through other means. The initialization bytecode will be recorded in the generated metadata file to make obtaining the correct deployed bytecode a bit easier.

    - **Note** BindGen does not handle the generation of the deployed bytecode for contracts containing `immutable` variables.
    - **Note** When BindGen is used to generate bindings that will be used as apart of the L2 genesis generation script found in [op-chain-ops](../../op-chain-ops/), [create2DeployerAddress](#create2DeployerAddress) and [DeploymentSalt](#deploymentSalt) are required to be set so that the correct deployment of the contract can be made on a simulated backend to obtain the correct initialization bytecode

- `false`

    This signals BindGen that the contract does **not** have `immutable` variables and therefore the deployed bytecode will be recorded in the generated metadata file.

#### `create2ProxyDeployed`

- `true`

    This signals BindGen that the contract was deployed via a `CREATE2` proxy deployer and may need to parse out a `CREATE2` salt (specified by [DeploymentSalt](#expected-salt)) from the deployment transaction input data to obtain the initialization bytecode.
    ```json
    {
        "name": "MultiSend",
        "verified": true,
        "manuallyResolveImmutables": true,
        "Create2ProxyDeployed": true,
        "DeploymentSalt": "0000000000000000000000000000000000000000000000000000000000000000",
        "create2DeployerAddress": "0x914d7fec6aac8cd542e72bca78b30650d45643d7",
        "deployments": {...}
    }
    ```

- `false`

    This signals BindGen that the contract has a standard deployment and the deployment transaction input data can be used as the initialization bytecode.

#### `abi`

This property sets the ABI to be used for Go binding generation for the contract. Ideally this is only used when a contract is **not** verified on the contract data source, and the ABI cannot be obtained by making a request for it.

#### `deploymentSalt`

This property specified the `CREATE2` salt used to deploy the contract. If provided and `!= ""`, the salt will be parsed out of the deployment transaction input data to obtain the correct contract initialization bytecode. The value of `DeploymentSalt` will be recorded in the generated metadata file as `{{.Name}}DeploymentSalt`.

#### `create2DeployerAddress`

This property specifies the address of the `CREATE2` proxy deployer and is recorded in the generated metadata file if provided.

- **Note** This property **must** be specified if the generated bindings for a contract that `hasImmutable == true && create2ProxyDeployed == true` are intended to be used by the L2 genesis generation script found in [op-chain-ops](../../op-chain-ops/). This is to allow for the correct deployment of the contract to a simulated backend to correctly compute the values for `immutable` variables.
- **Note** Is **not** required if `hasImmutable == false`

#### `deployments`

This is an object where the keys are chain IDs and the values are the address the contract was deployed to on that specific chain. An address for the chain ID used as `--source-chainid` **must** be provided, while an address for the `--compare-chainid` is optional. The address corresponding to the `--source-chainid` is used to obtain the contract's ABI, deployment transaction hash, and deployed bytecode. If an address corresponding to the `--compare-chainid` is provided and the flag(s) `--compare-deployment-bytecode` or `compare-init-bytecode` are provided, the compare address is used to obtain the deployment transaction hash so that deployment and initialization bytecode can fetched for comparison against the source's bytecode.

- **Note** If an address is not provided for the chain ID specified by `--compare-chainid`, then bytecode verification will be skipped for the contract.

#### `useDeploymentBytecodeFromChainId`

This property signals BindGen to use the deployment bytecode from a specific chain if the bytecode differs between the source and compare chains. The value of this property **must** be either the value of `--source-chainid` or `--compare-chainid`, or an error will be returned.

#### `useInitBytecodeFromChainId`

This property signals BindGen to use the initialization bytecode from a specific chain if the bytecode differs between the source and compare chains. The value of this property **must** be either the value of `--source-chainid` or `--compare-chainid`, or an error will be returned.
