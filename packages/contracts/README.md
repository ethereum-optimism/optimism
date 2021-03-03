# Optimistic Ethereum Smart Contracts

`@eth-optimism/contracts` contains the various Solidity smart contracts used within the Optimistic Ethereum system.
Some of these contracts are deployed on Ethereum ("Layer 1"), while others are meant to be deployed to Optimistic Ethereum ("Layer 2").

Within each contract file you'll find a comment that lists:
1. The compiler with which a contract is intended to be compiled, `solc` or `optimistic-solc`.
2. The network upon to which the contract will be deployed, `OVM` or `EVM`.

A more detailed overview of these contracts can be found on the [community hub](http://community.optimism.io/docs/protocol/protocol.html#system-overview).

<!-- TODO: Add link to final contract docs here when finished. -->

## Usage (npm)
If your development stack is based on Node/npm:

```shell
npm install @eth-optimism/contracts
```

Within your contracts:

```solidity
import { SomeContract } from "@eth-optimism/contracts/SomeContract.sol";
```

## Guide for Developers
### Setup
Install the following:
- [`Node.js` (14+)](https://nodejs.org/en/)
- [`npm`](https://www.npmjs.com/get-npm)
- [`yarn`](https://classic.yarnpkg.com/en/docs/install/)

Clone the repo:

```shell
git clone https://github.com/ethereum-optimism/contracts.git
cd contracts
```

Install `npm` packages:
```shell
yarn install
```

### Running Tests
Tests are executed via `yarn`:
```shell
yarn test
```

Run specific tests by giving a path to the file you want to run:
```shell
yarn test ./test/path/to/my/test.spec.ts
```

### Compiling and Building
Easiest way is to run the primary build script:
```shell
yarn build
```

Running the full build command will perform the following actions:
1. `build:contracts` - Compile all Solidity contracts with both the EVM and OVM compilers.
2. `build:typescript` - Builds the typescript files that are used to export utilities into js.
3. `build:copy` - Copies various other files into the build folder.
4. `build:dump` - Generates a genesis state from the contracts that L2 geth will use.
5. `build:typechain` - Generates [TypeChain](https://github.com/ethereum-ts/TypeChain) artifacts.

You can also build specific components as follows:
```shell
yarn build:contracts
```

## Security
Please refer to our [Security Policy](https://github.com/ethereum-optimism/.github/security/policy) for information about how to disclose security issues with this code.
