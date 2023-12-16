# Optimism Smart Contracts (Bedrock)

[![codecov](https://codecov.io/gh/ethereum-optimism/optimism/branch/develop/graph/badge.svg?token=0VTG7PG7YR&flag=contracts-bedrock-tests)](https://codecov.io/gh/ethereum-optimism/optimism)

This package contains the smart contracts that compose the on-chain component of Optimism's upcoming Bedrock upgrade.
We've tried to maintain 100% backwards compatibility with the existing system while also introducing new useful features.
You can find detailed specifications for the contracts contained within this package [here](../../specs).

A style guide we follow for writing contracts can be found [here](./STYLE_GUIDE.md).

## Contracts Overview

### Contracts deployed to L1

| Name                                                                                     | Proxy Type                                                              | Description                                                                                         |
| ---------------------------------------------------------------------------------------- | ----------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------- |
| [`L1CrossDomainMessenger`](../../specs/messengers.md)                                    | [`ResolvedDelegateProxy`](./contracts/legacy/ResolvedDelegateProxy.sol) | High-level interface for sending messages to and receiving messages from Optimism                   |
| [`L1StandardBridge`](../../specs/bridges.md)                                             | [`L1ChugSplashProxy`](./contracts/legacy/L1ChugSplashProxy.sol)         | Standardized system for transferring ERC20 tokens to/from Optimism                                   |
| [`L2OutputOracle`](../../specs/proposals.md#l2-output-oracle-smart-contract)             | [`Proxy`](./contracts/universal/Proxy.sol)                              | Stores commitments to the state of Optimism which can be used by contracts on L1 to access L2 state |
| [`OptimismPortal`](../../specs/deposits.md#deposit-contract)                             | [`Proxy`](./contracts/universal/Proxy.sol)                              | Low-level message passing interface                                                                 |
| [`OptimismMintableERC20Factory`](../../specs/predeploys.md#optimismmintableerc20factory) | [`Proxy`](./contracts/universal/Proxy.sol)                              | Deploys standard `OptimismMintableERC20` tokens that are compatible with either `StandardBridge`    |
| [`ProxyAdmin`](../../specs/TODO)                                                         | -                                                                       | Contract that can upgrade L1 contracts                                                              |

### Contracts deployed to L2

| Name                                                                                     | Proxy Type                                 | Description                                                                                      |
| ---------------------------------------------------------------------------------------- | ------------------------------------------ | ------------------------------------------------------------------------------------------------ |
| [`GasPriceOracle`](../../specs/predeploys.md#ovm_gaspriceoracle)                         | [`Proxy`](./contracts/universal/Proxy.sol) | Stores L2 gas price configuration values                                                         |
| [`L1Block`](../../specs/predeploys.md#l1block)                                           | [`Proxy`](./contracts/universal/Proxy.sol) | Stores L1 block context information (e.g., latest known L1 block hash)                           |
| [`L2CrossDomainMessenger`](../../specs/predeploys.md#l2crossdomainmessenger)             | [`Proxy`](./contracts/universal/Proxy.sol) | High-level interface for sending messages to and receiving messages from L1                      |
| [`L2StandardBridge`](../../specs/predeploys.md#l2standardbridge)                         | [`Proxy`](./contracts/universal/Proxy.sol) | Standardized system for transferring ERC20 tokens to/from L1                                     |
| [`L2ToL1MessagePasser`](../../specs/predeploys.md#ovm_l2tol1messagepasser)               | [`Proxy`](./contracts/universal/Proxy.sol) | Low-level message passing interface                                                              |
| [`SequencerFeeVault`](../../specs/predeploys.md#sequencerfeevault)                       | [`Proxy`](./contracts/universal/Proxy.sol) | Vault for L2 transaction fees                                                                    |
| [`OptimismMintableERC20Factory`](../../specs/predeploys.md#optimismmintableerc20factory) | [`Proxy`](./contracts/universal/Proxy.sol) | Deploys standard `OptimismMintableERC20` tokens that are compatible with either `StandardBridge` |
| [`L2ProxyAdmin`](../../specs/TODO)                                                       | -                                          | Contract that can upgrade L2 contracts when sent a transaction from L1                           |

### Legacy and deprecated contracts

| Name                                                            | Location | Proxy Type                                 | Description                                                                           |
| --------------------------------------------------------------- | -------- | ------------------------------------------ | ------------------------------------------------------------------------------------- |
| [`AddressManager`](./contracts/legacy/AddressManager.sol)       | L1       | -                                          | Legacy upgrade mechanism (unused in Bedrock)                                          |
| [`DeployerWhitelist`](./contracts/legacy/DeployerWhitelist.sol) | L2       | [`Proxy`](./contracts/universal/Proxy.sol) | Legacy contract for managing allowed deployers (unused since EVM Equivalence upgrade) |
| [`L1BlockNumber`](./contracts/legacy/L1BlockNumber.sol)         | L2       | [`Proxy`](./contracts/universal/Proxy.sol) | Legacy contract for accessing latest known L1 block number, replaced by `L1Block`     |

## Installation

We export contract ABIs, contract source code, and contract deployment information for this package via `npm`:

```shell
npm install @eth-optimism/contracts-bedrock
```

## Contributing

For all information about working on and contributing to Optimism's smart contracts, please see [CONTRIBUTING.md](./CONTRIBUTING.md)

## Deployment

The smart contracts are deployed using `foundry` with a `hardhat-deploy` compatibility layer. When the contracts are deployed,
they will write a temp file to disk that can then be formatted into a `hardhat-deploy` style artifact by calling another script.

The addresses in the `deployments` directory will be read into the script based on the backend's chain id.
To manually define the set of addresses used in the script, set the `CONTRACT_ADDRESSES_PATH` env var to a path on the local
filesystem that points to a JSON file full of key value pairs where the keys are names of contracts and the
values are addresses. This works well with the JSON files in `superchain-ops`.

### Configuration

Create or modify a file `<network-name>.json` inside of the [`deploy-config`](./deploy-config/) folder.
By default, the network name will be selected automatically based on the chainid. Alternatively, the `DEPLOYMENT_CONTEXT` env var can be used to override the network name.
The spec for the deploy config is defined by the `deployConfigSpec` located inside of the [`hardhat.config.ts`](./hardhat.config.ts).

### Execution

Before deploying the contracts, you can verify the state diff produced by the deploy script using the `runWithStateDiff()` function signature which produces the outputs inside [`snapshots/state-diff/`](./snapshots/state-diff).
Run the deployment with state diffs by executing: `forge script -vvv scripts/Deploy.s.sol:Deploy --sig 'runWithStateDiff()' --rpc-url $ETH_RPC_URL --broadcast --private-key $PRIVATE_KEY`.

1. Set the env vars `ETH_RPC_URL`, `PRIVATE_KEY` and `ETHERSCAN_API_KEY` if contract verification is desired
1. Deploy the contracts with `forge script -vvv scripts/Deploy.s.sol:Deploy --rpc-url $ETH_RPC_URL --broadcast --private-key $PRIVATE_KEY`
   Pass the `--verify` flag to verify the deployments automatically with Etherscan.
1. Generate the hardhat deploy artifacts with `forge script -vvv scripts/Deploy.s.sol:Deploy --sig 'sync()' --rpc-url $ETH_RPC_URL --broadcast --private-key $PRIVATE_KEY`

### Deploying a single contract

All of the functions for deploying a single contract are `public` meaning that the `--sig` argument to `forge script` can be used to
target the deployment of a single contract.

### Test Setup

The Solidity unit tests use the same codepaths to set up state that are used in production. The same L1 deploy script is used to deploy the L1 contracts for the in memory tests
and the L2 state is set up using the same L2 genesis generation code that is used for production and then loaded into foundry via the `vm.loadAllocs` cheatcode. This helps
to reduce the overhead of maintaining multiple ways to set up the state as well as give additional coverage to the "actual" way that the contracts are deployed.

The L1 contract addresses are held in `deployments/hardhat/.deploy` and the L2 test state is held in a `.testdata` directory. The L1 addresses are used to create the L2 state
and it is possible for stale addresses to be pulled into the L2 state, causing tests to fail. Stale addresses may happen if the order of the L1 deployments happen differently
since some contracts are deployed using `CREATE`. Run `pnpm clean` and rerun the tests if they are failing for an unknown reason.

### Static Analysis

`contracts-bedrock` uses [slither](https://github.com/crytic/slither) as its primary static analysis tool. When opening a pr that includes changes to `contracts-bedrock`, you should
verify that slither did not detect any new issues by running `pnpm slither:check`.

If there are new issues, you should triage them.
Run `pnpm slither:triage` to step through findings.
You should _carefully_ walk through these findings, specifying which to triage/ignore (default is to keep all, outputting them into `slither-report.json`).
Findings can be triaged into `slither.db.json` or kept in the `slither-report.json`.
You should triage issues with extreme _care_ and security sign-off.

After issues are triaged, or an updated slither report is generated, make sure to check in your changes to git.
Once checked in, the changes can be verified by running `pnpm slither:check`.
This will fail if there are issues missing from the `slither-report.json` that are _not_ triaged into `slither.db.json`.
