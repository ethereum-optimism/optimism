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
| [`L1StandardBridge`](../../specs/bridges.md)                                             | [`L1ChugSplashProxy`](./contracts/legacy/L1ChugSplashProxy.sol)         | Standardized system for transfering ERC20 tokens to/from Optimism                                   |
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

## Development

### Dependencies

We work on this repository with a combination of [Hardhat](https://hardhat.org) and [Foundry](https://getfoundry.sh/).

1. Install Foundry by following [the instructions located here](https://getfoundry.sh/).
   A specific version must be used.
   ```shell
   foundryup -C 3b1129b5bc43ba22a9bcf4e4323c5a9df0023140
   ```
2. Install node modules with yarn (v1) and Node.js (16+):

   ```shell
   yarn install
   ```

### Build

```shell
yarn build
```

### Tests

```shell
yarn test
```

#### Running Echidna tests

You must have [Echidna](https://github.com/crytic/echidna) installed.

Contracts targetted for Echidna testing are located in `./contracts/echidna`
Each target contract is tested with a separate yarn command, for example:

```shell
yarn echidna:aliasing
```

### Deployment

The smart contracts are deployed using `foundry` with a `hardhat-deploy` compatibility layer. When the contracts are deployed,
they will write a temp file to disk that can then be formatted into a `hardhat-deploy` style artifact by calling another script.

#### Configuration

Create or modify a file `<network-name>.json` inside of the [`deploy-config`](./deploy-config/) folder.
By default, the network name will be selected automatically based on the chainid. Alternatively, the `DEPLOYMENT_CONTEXT` env var can be used to override the network name.
The spec for the deploy config is defined by the `deployConfigSpec` located inside of the [`hardhat.config.ts`](./hardhat.config.ts).

#### Execution

1. Set the env vars `ETH_RPC_URL`, `PRIVATE_KEY` and `ETHERSCAN_API_KEY` if contract verification is desired
1. Deploy the contracts with `forge script -vvv scripts/Deploy.s.sol:Deploy --rpc-url $ETH_RPC_URL --broadcast --private-key $PRIVATE_KEY`
   Pass the `--verify` flag to verify the deployments automatically with Etherscan.
1. Generate the hardhat deploy artifacts with `forge script -vvv scripts/Deploy.s.sol:Deploy --sig 'sync()' --rpc-url $ETH_RPC_URL --broadcast --private-key $PRIVATE_KEY`

#### Deploying a single contract

All of the functions for deploying a single contract are `public` meaning that the `--sig` argument to `forge script` can be used to
target the deployment of a single contract.

## Tools

### Layout Locking

We use a system called "layout locking" as a safety mechanism to prevent certain contract variables from being moved to different storage slots accidentally.
To lock a contract variable, add it to the `layout-lock.json` file which has the following format:

```json
{
  "MyContractName": {
    "myVariableName": {
      "slot": 1,
      "offset": 0,
      "length": 32
    }
  }
}
```

With the above config, the `validate-spacers` hardhat task will check that we have a contract called `MyContractName`, that the contract has a variable named `myVariableName`, and that the variable is in the correct position as defined in the lock file.
You should add things to the `layout-lock.json` file when you want those variables to **never** change.
Layout locking should be used in combination with diffing the `.storage-layout` file in CI.
