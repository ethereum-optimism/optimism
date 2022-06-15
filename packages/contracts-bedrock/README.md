# Optimism: Bedrock Edition - Contracts

## Install

The repo currently uses a mix of typescript tests (run with HardHat) and solidity tests (run with Forge). The project
uses the default hardhat directory structure, and all build/test steps should be run using the yarn scripts to ensure
the correct options are set.

Install node modules with yarn (v1), and Node.js (14+).

```shell
yarn
```

See installation instructions for forge [here](https://github.com/gakonst/foundry).

## Build

```shell
yarn build
```

## Running Tests

First get the dependencies:

`git submodule init` and `git submodule update`

Then the full test suite can be executed via `yarn`:

```shell
yarn test
```

To run only typescript tests:

```shell
yarn test:hh
```

To run only solidity tests:

```shell
yarn test:forge
```

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
