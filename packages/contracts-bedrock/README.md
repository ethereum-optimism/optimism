# Optimism: Bedrock Edition - Contracts

## Install

The repo currently uses solidity tests (run with Forge). The project uses the default hardhat directory structure, and all build/test steps should be run using the yarn scripts to ensure
the correct options are set.

Install node modules with yarn (v1), and Node.js (16+).

```shell
yarn
```

See installation instructions for forge [here](https://github.com/gakonst/foundry).

## Build

```shell
yarn build
```

## Running Tests

Then the full test suite can be executed via `yarn`:

```shell
yarn test
```

The differential tests require typescript to be compiled to javascript.

## Deployment

Create a file that corresponds to the network name in the `deploy-config`
directory and then run the command:

```shell
L1_RPC=<ETHEREUM L1 RPC endpoint> \
PRIVATE_KEY_DEPLOYER=<PRIVATE KEY TO PAY FOR THE DEPLOYMENT> \
    npx hardhat deploy --network <network-name>
```

In the `hardhat.config.ts`, there is a `deployConfigSpec` field that validates that the types
are correct, be sure to export an object in the `deploy-config/<network-name>.ts` file that
has a key for each property in the `deployConfigSpec`.

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
- `@custom:legacy`: Add to an event or function when it only exists for legacy support.

#### Errors

- Use `require` statements when making simple assertions.
- Use `revert` if throwing an error where an assertion is not being made (no custom errors). See [here](https://github.com/ethereum-optimism/optimism/blob/861ae315a6db698a8c0adb1f8eab8311fd96be4c/packages/contracts-bedrock/contracts/L2/OVM_ETH.sol#L31) for an example of this in practice.
- Error strings MUST have the format `"{ContractName}: {message}"` where `message` is a lower case string.

#### Function Parameters

- Function parameters should be prefixed with an underscore.

#### Event Parameters

- Event parameters should NOT be prefixed with an underscore.

### Proxy by Default

All contracts should be assumed to live behind proxies (except in certain special circumstances).
This means that new contracts MUST be built under the assumption of upgradeability.
We use a minimal [`Proxy`](./contracts/universal/Proxy.sol) contract designed to be owned by a corresponding [`ProxyAdmin`](./contracts/universal/ProxyAdmin.sol) which follow the interfaces of OpenZeppelin's `Proxy` and `ProxyAdmin` contracts, respectively.

Unless explicitly discussed otherwise, you MUST include the following basic upgradeability pattern for each new implementation contract:

1. Extend OpenZeppelin's `Initializable` base contract.
2. Include a `uint8 public constant VERSION = X` at the TOP of your contract.
3. Include a function `initialize` with the modifier `reinitializer(VERSION)`.
4. In the `constructor`, set any `immutable` variables and call the `initialize` function for setting mutables.
