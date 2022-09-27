# Optimism Smart Contracts (Bedrock)

This package contains the smart contracts that compose the on-chain component of Optimism's upcoming Bedrock upgrade.
We've tried to maintain 100% backwards compatibility with the existing system while also introducing new useful features.
You can find detailed specifications for the contracts contained within this package [here](../../specs).

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

### Deployment

#### Configuration

1. Create or modify a file `<network-name>.json` inside of the [`deploy-config`](./deploy-config/) folder.
2. Fill out this file according to the `deployConfigSpec` located inside of the [`hardhat.config.ts](./hardhat.config.ts)

#### Execution

1. Copy `.env.example` into `.env`
2. Fill out the `L1_RPC` and `PRIVATE_KEY_DEPLOYER` environment variables in `.env`
3. Run `npx hardhat deploy --network <network-name>` to deploy the L1 contracts
4. Run `npx hardhat etherscan-verify --network <network-name> --sleep` to verify contracts on Etherscan

## Standards and Conventions

### Style

#### Comments

We use [Seaport](https://github.com/ProjectOpenSea/seaport/blob/main/contracts/Seaport.sol)-style comments with some minor modifications.
Some basic rules:

- Always use `@notice` since it has the same general effect as `@dev` but avoids confusion about when to use one over the other.
- Include a newline between `@notice` and the first `@param`.
- Include a newline between `@param` and the first `@return`.
- Use a line-length of 100 characters.

We also have the following custom tags:

- `@custom:proxied`: Add to a contract whenever it's meant to live behind a proxy.
- `@custom:upgradeable`: Add to a contract whenever it's meant to be used in an upgradeable contract.
- `@custom:semver`: Add to a constructor to indicate the version of a contract.
- `@custom:legacy`: Add to an event or function when it only exists for legacy support.

#### Errors

- Use `require` statements when making simple assertions.
- Use `revert` if throwing an error where an assertion is not being made (no custom errors). See [here](https://github.com/ethereum-optimism/optimism/blob/861ae315a6db698a8c0adb1f8eab8311fd96be4c/packages/contracts-bedrock/contracts/L2/OVM_ETH.sol#L31) for an example of this in practice.
- Error strings MUST have the format `"{ContractName}: {message}"` where `message` is a lower case string.

#### Function Parameters

- Function parameters should be prefixed with an underscore.

#### Event Parameters

- Event parameters should NOT be prefixed with an underscore.

#### Spacers

We use spacer variables to account for old storage slots that are no longer being used.
The name of a spacer variable MUST be in the format `spacer_<slot>_<offset>_<length>` where `<slot>` is the original storage slot number, `<offset>` is the original offset position within the storage slot, and `<length>` is the original size of the variable.
Spacers MUST be `private`.

### Proxy by Default

All contracts should be assumed to live behind proxies (except in certain special circumstances).
This means that new contracts MUST be built under the assumption of upgradeability.
We use a minimal [`Proxy`](./contracts/universal/Proxy.sol) contract designed to be owned by a corresponding [`ProxyAdmin`](./contracts/universal/ProxyAdmin.sol) which follow the interfaces of OpenZeppelin's `Proxy` and `ProxyAdmin` contracts, respectively.

Unless explicitly discussed otherwise, you MUST include the following basic upgradeability pattern for each new implementation contract:

1. Extend OpenZeppelin's `Initializable` base contract.
2. Include a `uint8 public constant VERSION = X` at the TOP of your contract.
3. Include a function `initialize` with the modifier `reinitializer(VERSION)`.
4. In the `constructor`, set any `immutable` variables and call the `initialize` function for setting mutables.

### Versioning

All (non-library and non-abstract) contracts MUST extend the `Semver` base contract which exposes a `version()` function that returns a semver-compliant version string.
During the Bedrock development process the `Semver` value for all contracts SHOULD return `0.0.1` (this is not particularly important, but it's an easy standard to follow).
When the initial Bedrock upgrade is released, the `Semver` value MUST be updated to `1.0.0`.

After the initial Bedrock upgrade, contracts MUST use the following versioning scheme:

- `patch` releases are to be used only for changes that do NOT modify contract bytecode (such as updating comments).
- `minor` releases are to be used for changes that modify bytecode OR changes that expand the contract ABI provided that these changes do NOT break the existing interface.
- `major` releases are to be used for changes that break the existing contract interface OR changes that modify the security model of a contract.

#### Exceptions

We have made an exception to the `Semver` rule for the `WETH` contract to avoid making changes to a well-known, simple, and recognizable contract.
