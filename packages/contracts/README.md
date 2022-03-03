[![codecov](https://codecov.io/gh/ethereum-optimism/optimism/branch/master/graph/badge.svg?token=0VTG7PG7YR&flag=contracts)](https://codecov.io/gh/ethereum-optimism/optimism)

# Optimism Smart Contracts

`@eth-optimism/contracts` contains the various Solidity smart contracts used within the Optimism system.
Some of these contracts are deployed on Ethereum ("Layer 1"), while others are meant to be deployed to Optimism ("Layer 2").

Within each contract file you'll find a comment that lists:
1. The compiler with which a contract is intended to be compiled, `solc` or `optimistic-solc`.
2. The network upon to which the contract will be deployed, `OVM` or `EVM`.

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

### Measuring test coverage:
```shell
yarn test:coverage
```

The output is most easily viewable by opening the html file in your browser:
```shell
open ./coverage/index.html
```

### Compiling and Building
Easiest way is to run the primary build script:
```shell
yarn build
```

Running the full build command will perform the following actions:
1. `build:contracts` - Compile all Solidity contracts with both the EVM and OVM compilers.
2. `build:typescript` - Builds the typescript files that are used to export utilities into js.
3. `build:copy` - Copies various other files into the dist folder.
4. `build:dump` - Generates a genesis state from the contracts that L2 geth will use.
5. `build:typechain` - Generates [TypeChain](https://github.com/ethereum-ts/TypeChain) artifacts.

You can also build specific components as follows:
```shell
yarn build:contracts
```

### Deploying the Contracts
To deploy the contracts first clone, install, and build the contracts package.

Next set the following env vars:

```bash
CONTRACTS_TARGET_NETWORK=...
CONTRACTS_DEPLOYER_KEY=...
CONTRACTS_RPC_URL=...
```

Then to perform the actual deployment run:

```bash
npx hardhat deploy \
  --network ... \  # `network` MUST equal your env var `CONTRACTS_TARGET_NETWORK`
  --ovm-address-manager-owner ... \
  --ovm-proposer-address ... \
  --ovm-relayer-address ... \
  --ovm-sequencer-address ... \
  --scc-fraud-proof-window ... \
  --scc-sequencer-publish-window ...
```

This will deploy the contracts to the network specified in your env and create
an artifacts directory in `./deployments`.

To view all deployment options run:

```bash
npx hardhat deploy --help

Hardhat version 2.2.1

Usage: hardhat [GLOBAL OPTIONS] deploy [--ctc-force-inclusion-period-seconds <INT>] [--ctc-max-transaction-gas-limit <INT>] --deploy-scripts <STRING> [--em-max-gas-per-queue-per-epoch <INT>] [--em-max-transaction-gas-limit <INT>] [--em-min-transaction-gas-limit <INT>] [--em-ovm-chain-id <INT>] [--em-seconds-per-epoch <INT>] --export <STRING> --export-all <STRING> --gasprice <STRING> [--l1-block-time-seconds <INT>] [--no-compile] [--no-impersonation] --ovm-address-manager-owner <STRING> --ovm-proposer-address <STRING> --ovm-relayer-address <STRING> --ovm-sequencer-address <STRING> [--reset] [--scc-fraud-proof-window <INT>] [--scc-sequencer-publish-window <INT>] [--silent] --tags <STRING> [--watch] --write <BOOLEAN>

OPTIONS:

  --ctc-force-inclusion-period-seconds  Number of seconds that the sequencer has to include transactions before the L1 queue. (default: 2592000)
  --ctc-max-transaction-gas-limit       Max gas limit for L1 queue transactions. (default: 9000000)
  --deploy-scripts                      override deploy script folder path
  --em-max-gas-per-queue-per-epoch      Maximum gas allowed in a given queue for each epoch. (default: 250000000)
  --em-max-transaction-gas-limit        Maximum allowed transaction gas limit. (default: 9000000)
  --em-min-transaction-gas-limit        Minimum allowed transaction gas limit. (default: 50000)
  --em-ovm-chain-id                     Chain ID for the L2 network. (default: 420)
  --em-seconds-per-epoch                Number of seconds in each epoch. (default: 0)
  --export                              export current network deployments
  --export-all                          export all deployments into one file
  --gasprice                            gas price to use for transactions
  --l1-block-time-seconds               Number of seconds on average between every L1 block. (default: 15)
  --no-compile                          disable pre compilation
  --no-impersonation                    do not impersonate unknown accounts
  --ovm-address-manager-owner           Address that will own the Lib_AddressManager. Must be provided or this deployment will fail.
  --ovm-proposer-address                Address of the account that will propose state roots. Must be provided or this deployment will fail.
  --ovm-relayer-address                 Address of the message relayer. Must be provided or this deployment will fail.
  --ovm-sequencer-address               Address of the sequencer. Must be provided or this deployment will fail.
  --reset                               whether to delete deployments files first
  --scc-fraud-proof-window              Number of seconds until a transaction is considered finalized. (default: 604800)
  --scc-sequencer-publish-window        Number of seconds that the sequencer is exclusively allowed to post state roots. (default: 1800)
  --silent                              whether to remove log
  --tags                                specify which deploy script to execute via tags, separated by commas
  --watch                               redeploy on every change of contract or deploy script
  --write                               whether to write deployments to file

deploy: Deploy contracts

For global options help run: hardhat help
```

### Verifying Deployments on Etherscan
If you are using a network which Etherscan supports you can verify your contracts with:

```bash
npx hardhat etherscan-verify --api-key ... --network ...
```

### Other hardhat tasks

To whitelist deployers on Mainnet you must have the whitelist Owner wallet connected, then run:
```bash
npx hardhat whitelist \
  --use-ledger true \
  --contracts-rpc-url https://mainnet.optimism.io \
  --address ... \ # address to whitelist
```

To withdraw ETH fees to L1 on Mainnet, run:
```bash
npx hardhat withdraw-fees \
  --use-ledger \  # The ledger to withdraw fees with. Ensure this wallet has ETH on L2 to pay the tx fee.
  --contracts-rpc-url https://mainnet.optimism.io \
```


## Security
Please refer to our [Security Policy](https://github.com/ethereum-optimism/.github/security/policy) for information about how to disclose security issues with this code.
