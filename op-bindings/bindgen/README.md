![BindGen Header Image](./bindgen_header.png)

A CLI for generating Go bindings from Forge artifacts and API clients such as Etherscan's

- [Dependencies](#dependencies)
- [Running BindGen](#running-bindgen)
  - [Using the Makefile Commands](#using-the-makefile-commands)
    - [`bindgen`](#bindgen)
      - [Required ENVs](#required-envs)
    - [`bindgen-local`](#bindgen-local)
    - [`bindgen-remote`](#bindgen-remote)
      - [Required ENVs](#required-envs-1)
  - [Using the CLI Directly](#using-the-cli-directly)
  - [CLI Flags](#cli-flags)
    - [Global Flags](#global-flags)
  - [Local Flags](#local-flags)
  - [Remote Flags](#remote-flags)
- [Using BindGen to Add New Preinstalls to L2 Genesis](#using-bindgen-to-add-new-preinstalls-to-l2-genesis)
  - [Anatomy of `artifacts.json`](#anatomy-of-artifactsjson)
    - [`"local"` Contracts](#local-contracts)
    - [`"remote"` Contracts](#remote-contracts)
    - [Adding A New `"remote"` Contract](#adding-a-new-remote-contract)
      - [Contracts that Don't Make Good Preinstalls](#contracts-that-dont-make-good-preinstalls)
    - [Adding the Contract to L2 Genesis](#adding-the-contract-to-l2-genesis)

# Dependencies

- [Go](https://go.dev/dl/)
- [Foundry](https://getfoundry.sh/)
- [pnpm](https://pnpm.io/installation)

If you're running the CLI inside the Optimism monorepo, please make sure you've executed `pnpm i` and `pnpm build` to install and setup all of the monorepo's dependencies.

# Running BindGen

BindGen can be run in one of two ways:

1. Using the provided [Makefile](../Makefile) which defaults some of the required flags
2. Executing the CLI directly with `go run`, or building a Go binary and executing it

Before executing BindGen, please review the [artifacts.json](../artifacts.json) file which specifies what contracts BindGen should generate Go bindings and metadata files for. More information on how to configure `artifacts.json` can be found [here](#anatomy-of-artifactsjson).

## Using the Makefile Commands

### `bindgen`

```bash
ETHERSCAN_APIKEY_ETH=your_api_key \
ETHERSCAN_APIKEY_OP=your_api_key \
RPC_URL_ETH=your_rpc_url \
RPC_URL_OP=your_rpc_url \
make bindgen
```

This command will run `forge clean` to remove any existing Forge artifacts found in the [contracts-bedrock](../../packages/contracts-bedrock/) directory, re-build the Forge artifacts, then will use BindGen to generate Go bindings and metadata files for the contracts specified in [artifacts.json](../artifacts.json).

#### Required ENVs

- `ETHERSCAN_APIKEY_ETH` An Etherscan API key for querying Ethereum Mainnet.

  - [Here's a guide](https://docs.etherscan.io/getting-started/viewing-api-usage-statistics) on how to obtain a key.

- `ETHERSCAN_APIKEY_OP` An Etherscan API key for querying Optimism Mainnet.

  - You can follow the above guide to obtain a key, but make sure you're on the [Optimistic Etherscan](https://optimistic.etherscan.io/)

- `RPC_URL_ETH` This is any HTTP URL that can be used to query an Ethereum Mainnet RPC node.

  - Expected to use API key authentication.

- `RPC_URL_OP` This is any HTTP URL that can be used to query an Optimism Mainnet RPC node.

  - Expected to use API key authentication.

### `bindgen-local`

```bash
make bindgen-local
```

This command will run `forge clean` to remove any existing Forge artifacts found in the [contracts-bedrock](../../packages/contracts-bedrock/) directory, re-build the Forge artifacts, then will use BindGen to generate Go bindings and metadata files for the `"local"` contracts specified in [artifacts.json](../artifacts.json).

### `bindgen-remote`

```bash
ETHERSCAN_APIKEY_ETH=your_api_key \
ETHERSCAN_APIKEY_OP=your_api_key \
RPC_URL_ETH=your_rpc_url \
RPC_URL_OP=your_rpc_url \
make bindgen-remote
```

This command will use BindGen to generate Go bindings and metadata files for the `"remote"` contracts specified in [artifacts.json](../artifacts.json).

#### Required ENVs

- `ETHERSCAN_APIKEY_ETH` An Etherscan API key for querying Ethereum Mainnet.

  - [Here's a guide](https://docs.etherscan.io/getting-started/viewing-api-usage-statistics) on how to obtain a key.

- `ETHERSCAN_APIKEY_OP` An Etherscan API key for querying Optimism Mainnet.

  - You can follow the above guide to obtain a key, but make sure you're on the [Optimistic Etherscan](https://optimistic.etherscan.io/)

- `RPC_URL_ETH` This is any HTTP URL that can be used to query an Ethereum Mainnet RPC node.

  - Expected to use API key authentication.

- `RPC_URL_OP` This is any HTTP URL that can be used to query an Optimism Mainnet RPC node.

  - Expected to use API key authentication.

## Using the CLI Directly

Currently the CLI only has one command, `generate`, which expects one of the following sub-commands:

Command  | Description                                                                | Flags                         | Usage
-------- | -------------------------------------------------------------------------- | ----------------------------- | ------------------------------------------------------------------
`all`    | Generates bindings for both local and remotely sourced contracts.          | [Global Flags](#global-flags) | `bindgen generate [global-flags] all [local-flags] [remote-flags]`
`local`  | Generates bindings for contracts with locally available Forge artifacts.   | [Local Flags](#local-flags)   | `bindgen generate [global-flags] local [local-flags]`
`remote` | Generates bindings for contracts whose metadata is sourced from Etherscan. | [Remote Flags](#remote-flags) | `bindgen generate [global-flags] remote [remote-flags]`

The following displays how the CLI can be invoked from the monorepo root:

```bash
go run ./op-bindings/cmd/ <bindgen-command> <flags> <sub-command> <sub-command-flags>
```

## CLI Flags

### Global Flags

These flags are used by all CLI commands

Flag               | Type   | Description                                                                    | Required
------------------ | ------ | ------------------------------------------------------------------------------ | --------
`metadata-out`     | String | Output directory for Go bindings contract metadata files                       | Yes
`bindings-package` | String | Go package name used for generated Go bindings                                 | Yes
`contracts-list`   | String | Path to the list of `local` and/or `remote` contracts                          | Yes
`log.level`        | String | Log level (`none`, `debug`, `info`, `warn`, `error`, `crit`) (Default: `info`) | No

## Local Flags

These flags are used with `all` and `local` commands

Flag               | Type   | Description                                                   | Required
------------------ | ------ | ------------------------------------------------------------- | --------
`source-maps-list` | String | Comma-separated list of contracts to generate source-maps for | No
`forge-artifacts`  | String | Path to the directory with compiled Forge artifacts           | Yes

## Remote Flags

These flags are used with `all` and `remote` commands

Flag                   | Type   | Description                                                                 | Required
---------------------- | ------ | --------------------------------------------------------------------------- | --------
`etherscan.apikey.eth` | String | An Etherscan API key for querying Ethereum Mainnet                          | Yes
`etherscan.apikey.op`  | String | An Etherscan API key for querying Optimism Mainnet                          | Yes
`rpc.url.eth`          | String | This is any HTTP URL that can be used to query an Ethereum Mainnet RPC node | Yes
`rpc.url.op`           | String | This is any HTTP URL that can be used to query an Optimism Mainnet RPC node | Yes

# Using BindGen to Add New Preinstalls to L2 Genesis

**Note** While we encourage hacking on the OP stack, we are not actively looking to integrate more contracts to the official OP stack genesis.

BindGen uses the provided `contracts-list` to generate Go bindings and metadata files which are used when building the L2 genesis. The first step in adding a new preinstall to L2 genesis is adding the contract to your `contracts-list` (by default this list is [artifacts.json](../artifacts.json)).

## Anatomy of `artifacts.json`

Below is a condensed version of the default [artifacts.json](../artifacts.json) file for reference:

```json
{
  "local": [
    "SystemConfig",
    "L1CrossDomainMessenger",

    ...

    "StorageSetter",
    "SuperchainConfig"
  ],
  "remote": [
    {
      "name": "MultiCall3",
      "verified": true,
      "deployments": {
        "eth": "0xcA11bde05977b3631167028862bE2a173976CA11",
        "op": "0xcA11bde05977b3631167028862bE2a173976CA11"
      }
    },

    ...

    {
      "name": "EntryPoint",
      "verified": true,
      "deployments": {
        "eth": "0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789",
        "op": "0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789"
      },
      "deploymentSalt": "0000000000000000000000000000000000000000000000000000000000000000"
    }
  ]
}
```

### `"local"` Contracts

The first property of this JSON object, `"local"`, specifies the name of the contracts that have locally available Forge artifacts which BindGen will use to generate Go bindings and metadata files. This property specifies an array of strings where each string corresponds to the name of the contract which is used in the name of its corresponding Forge artifact.

For example, the first contract listed in the default contract list is `SystemConfig`. After running `pnpm build` in the [contract-bedrock](../../packages/contracts-bedrock/), you'll have a [forge-artifacts](../../packages/contracts-bedrock/forge-artifacts/) directory where you can find [SystemConfig.sol](../../packages/contracts-bedrock/forge-artifacts/SystemConfig.sol/). Inside is the Forge artifacts BindGen will use to generate the Go bindings and metadata file.

In some cases, such as `Safe`, there will exist multiple versioned Forge artifacts (e.g. [contracts-bedrock/forge-artifacts/Safe.sol/](../../packages/contracts-bedrock/forge-artifacts/Safe.sol/) contains `Safe.0.8.15.json` and `Safe.0.8.19.json`). In this case BindGen will default to using the lesser version (`Safe.0.8.19.json` in this case), and when running BindGen you will see a warning logged to the console to notify you:

```bash
...
WARN [12-22|13:39:19.217] Multiple versions of forge artifacts exist, using lesser version contract=Safe
...
INFO [12-22|13:39:20.253] Generating bindings and metadata for local contract contract=Safe
```

### `"remote"` Contracts

The second property specifies a list of `RemoteContract` objects which contain metadata used to fetch the needed contract info to generate Go bindings from Etherscan; these contracts do **not** have locally available Forge artifacts.

There are a couple different variations of the `RemoteContract` object, but the following is the Go struct for reference:

```go
type Deployments struct {
    Eth common.Address `json:"eth"`
    Op  common.Address `json:"op"`
}

type RemoteContract struct {
    Name           string         `json:"name"`
    Verified       bool           `json:"verified"`
    Deployments    Deployments    `json:"deployments"`
    DeploymentSalt string         `json:"deploymentSalt"`
    Deployer       common.Address `json:"deployer"`
    ABI            string         `json:"abi"`
    InitBytecode   string         `json:"initBytecode"`
}
```

Name                   | Description
---------------------- | -----------
`name` | The name of the remote contract that will be used for the Go bindings and metadata files
`verified` | Denotes whether the contract is verified on Etherscan
`deployments` | An object that maps a network and the address the contract is deployed to on that network
`deployments.eth` | The address the contract is deployed to on Ethereum Mainnet
`deployments.op` | The address the contract is deployed to on Optimism Mainnet
`deploymentSalt` | If the contract was deployed using CREATE2 or a CREATE2 proxy deployer, here is where you specify the salt that was used for creation
`deployer` | The address used to deploy the contract, used to mimic CREATE2 deployments
`abi` | The ABI of the contract, required if the contract is **not** verified on Etherscan
`initBytecode` | The initialization bytecode for the contract, required if the contract is a part of the initialization of another contract (i.e. the `input` data of the deployment transaction contains initialization bytecode other than what belongs to the specific contract you're adding)

### Adding A New `"remote"` Contract

After adding a `RemoteContract` object to your `contracts-list`, you will need to add the `name` of your contract to the `switch` statement found in the `processContracts` function in [generator_remote.go](./generator_remote.go):

```go
...

switch contract.Name {
		case "MultiCall3", "Safe_v130", "SafeL2_v130", "MultiSendCallOnly_v130",
			"EntryPoint", "SafeSingletonFactory", "DeterministicDeploymentProxy":
			err = generator.standardHandler(&contractMetadata)
		case "Create2Deployer":
			err = generator.create2DeployerHandler(&contractMetadata)
		case "MultiSend_v130":
			err = generator.multiSendHandler(&contractMetadata)
		case "SenderCreator":
			// The SenderCreator contract is deployed by EntryPoint, so the transaction data
			// from the deployment transaction is for the entire EntryPoint deployment.
			// So, we're manually providing the initialization bytecode
			contractMetadata.InitBin = contract.InitBytecode
			err = generator.senderCreatorHandler(&contractMetadata)
		case "Permit2":
			// Permit2 has an immutable Solidity variable that resolves to block.chainid,
			// so we can't use the deployed bytecode, and instead must generate it
			// at some later point not handled by BindGen.
			// DeployerAddress is intended to be used to help deploy Permit2 at it's deterministic address
			// to a chain set with the required id to be able to obtain a diff minimized deployed bytecode
			contractMetadata.Deployer = contract.Deployer
			err = generator.permit2Handler(&contractMetadata)
		default:
			err = fmt.Errorf("unknown contract: %s, don't know how to handle it", contract.Name)
		}

...
```

If your contract is verified on Etherscan, doesn't contain any Solidity `immutable`s, and doesn't require any special handling, then you most likely can add your contract's `name` to the first switch case. Then will use the `standardHandler` which:

1. Fetches the required contract metadata from Etherscan (i.e. initialization and deployed bytecode, ABI, deployment transaction hash, etc.)
2. Compares the retrieved deployed bytecode from Etherscan against the response of `eth_codeAt` from an RPC node for each network specified in `RemoteContract.deployments` (this is a sanity check to verify Etherscan is returning correct data)
3. If applicable, removes the provided `RemoteContract.deploymentSalt` from the initialization bytecode
4. Compares the initialization bytecode retrieved from Etherscan on Ethereum Mainnet against the bytecode retrieved from Etherscan on Optimism Mainnet
  - This is an important sanity check! If the initialization bytecode from Ethereum differs from Optimism, then there's a big chance the deployment from Ethereum may not behave as expected if preinstalled to an OP stack L2
5. Compares the deployment bytecode retrieved from Etherscan on Ethereum Mainnet against the bytecode retrieved from Etherscan on Optimism Mainnet
  - This has the same concern as differing initialization bytecode
6. Lastly, the Go bindings are generated and the metadata file is written to the path provided as `metadata-out` CLI flag

All other default `"remote"` contract have some variation of the above execution flow depending on the nuances of each contract. For example:

- `Create2Deployer`'s initialization and deployed bytecode is expected to differ from its Optimism Mainnet deployment
- `MultiSend_v130` has an `immutable` Solidity variable the resolves to `address(this)`, so we can't use the deployment bytecode from Ethereum Mainnet, we must get its deployment bytecode from Optimism Mainnet
- `SenderCreator` is deployed by `EntryPoint`, so its initialization bytecode is provided in [artifacts.json](../artifacts.json) and not being fetched from Etherscan like other contracts

#### Contracts that Don't Make Good Preinstalls

Not every contract can be added as a preinstall, and some contracts have nuances that make them potentially dangerous or troublesome to preinstall. Below are some examples of contracts that wouldn't make good preinstalls. This is not a comprehensive list, so make sure to use judgment for each contract added as a preinstall.

- Contracts that haven't been audited or stood the test of time
  - Once a contract is preinstalled and a network is started, if a vulnerability is discovered for the contract and there is no way to easily disable the contract, the only options to "disable" the vulnerable contract are to either (A) remove it from the L2 genesis and restart the L2 network, (B) Hardfork the network to remove/replace the preinstall, or (C) Warn users not to use the vulnerable preinstall
- Related to above, contracts that may become deprecated/unsupported relatively soon
  - As mentioned above, you're limited to options A, B, or C
- Upgradeable Contracts
  - While it's certainly feasible to preinstall an upgradeable contract, great care should be taken to minimize security risks to users if the contract is upgraded to a malicious or buggy implementation. Understanding who has the ability to upgrade the contract is key to avoiding this. Additionally, users might be expecting a preinstall to do something and may be caught off guard if the implementation was upgraded without their knowledge
- Contracts with Privileged Roles and Configuration Parameters
  - Similar to the upgradeable contracts, simply having an owner or other privileged role with the ability to make configuration changes can present a security risk and result in unexpected different behaviors across chains.
- Contracts that have dependencies
  - Dependencies has many definitions, for example:
    - Being reliant on specific Oracle contracts that may not be available on your L2
    - Specific contract state that's set on L1 but won't be on L2
    - Relying on specific values of block and transaction properties (e.g. `block.chainid`, `block.timestamp`, `block.number`, etc.)
    - Contract libraries that may not be deployed on L2

### Adding the Contract to L2 Genesis

Once you've configured the `contracts-list` to include the contracts you'd like to add as preinstalls, the next step is utilizing the BindGen outputs to configure the L2 genesis.

1. First we must update the [addresses.go](../predeploys/addresses.go) file to include the address we're preinstalling our contracts to
1. Update the `switch` case found in [layer_two.go](../../op-chain-ops/genesis/layer_two.go) to include the `name` of your contracts
1. Update [immutables.go](../../op-chain-ops/immutables/immutables.go) to include your added contracts
1. Update [Predeploys.sol](../../packages/contracts-bedrock/src/libraries/Predeploys.sol) to include your added contracts at their expected addresses
1. Update [Predeploys.t.sol](../../packages/contracts-bedrock/test/Predeploys.t.sol) to include the `name` of your contracts to avoid being tested for `Predeploys.PROXY_ADMIN`
