[![codecov](https://codecov.io/gh/ethereum-optimism/optimism/branch/master/graph/badge.svg?token=0VTG7PG7YR&flag=contracts)](https://codecov.io/gh/ethereum-optimism/optimism)

# Optimism Smart Contracts

`@eth-optimism/contracts` contains the various Solidity smart contracts used within the Optimism system.
Some of these contracts are [meant to be deployed to Ethereum ("Layer 1")](https://github.com/ethereum-optimism/optimism/tree/develop/packages/contracts/contracts/L1), while others are [meant to be deployed to Optimism ("Layer 2")](https://github.com/ethereum-optimism/optimism/tree/develop/packages/contracts/contracts/L2).
Within each contract file you'll find the network upon which the contract is meant to be deloyed, listed as either `EVM` (for Ethereum) or `OVM` (for Optimism).
If neither `EVM` nor `OVM` are listed, the contract is likely intended to be used on either network.

## Usage (npm)

You can import `@eth-optimism/contracts` to use the Optimism contracts within your own codebase.
Install via `npm` or `yarn`:

```shell
npm install @eth-optimism/contracts
```

Within your contracts:

```solidity
import { SomeContract } from "@eth-optimism/contracts/path/to/SomeContract.sol";
```

Note that the `/path/to/SomeContract.sol` is the path to the target contract within the [contracts folder](https://github.com/ethereum-optimism/optimism/tree/develop/packages/contracts/contracts) inside of this package.
For example, the [L1CrossDomainMessenger](/contracts/L1/messaging/L1CrossDomainMessenger.sol) contract is located at `/contracts/L1/messaging/L1CrossDomainMessenger.sol`, relative to this README.
You would therefore import the contract as:


```solidity
import { L1CrossDomainMessenger } from "@eth-optimism/contracts/L1/messaging/L1CrossDomainMessenger.sol";
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

#### Required environment variables

You must set the following environment variables to execute a deployment:

```bash
# Name for the network to deploy to ("mainnet", "kovan", etc.)
export CONTRACTS_TARGET_NETWORK=...

# Private key that will send deployment transactions
export CONTRACTS_DEPLOYER_KEY=...

# RPC URL connected to the L1 chain we're deploying to
export CONTRACTS_RPC_URL=...

# Your Etherscan API key for the L1 network
export ETHERSCAN_API_KEY=...
```

#### Creating a deployment script

Before you can carry out a deployment, you must create a deployment script.
See [mainnet.sh](./scripts/deploy-scripts/mainnet.sh) for an example deployment script.
We recommend duplicating an existing deployment script and modifying it to satisfy your requirements.

Most variables within the deploy script are relatively self-explanatory.
If you intend to upgrade an existing system you **MUST** [include the following argument](https://github.com/ethereum-optimism/optimism/blob/6f633f915b34a46ac14430724bed9722af8bd05e/packages/contracts/scripts/deploy-scripts/mainnet.sh#L33) in the deploy script:

```
--tags upgrade
```

If you are deploying a system from scratch, you should **NOT** include `--tags upgrade` or you will fail to deploy several contracts.

#### Executing a deployment

Once you've created your deploy script, simply run the script to trigger a deployment.
During the deployment process, you will be asked to transfer ownership of several contracts to a special contract address.
You will also be asked to verify various configuration values.
This is a safety mechanism to make sure that actions within an upgrade are performed atomically.
Ownership of these addresses will be automatically returned to the original owner address once the upgrade is complete.
The original owner can always recover ownership from the upgrade contract in an emergency.
Please read these instructions carefully, verify each of the presented configuration values, and carefully confirm that the contract you are giving ownership to has not been compromised (e.g., check the code on Etherscan).

After your deployment is complete, your new contracts will be written to an artifacts directory in `./deployments/<name>`.
Your contracts will also be automatically verified as part of the deployment script.

#### Creating a genesis file

Optimism expects that certain contracts (called "predeploys") be deployed to the L2 network at pre-determined addresses.
Doing this requires that you generate a special genesis file to be used by your corresponding L2Geth nodes.
You must first create a genesis generation script.
Like in the deploy script, we recommend starting from an [existing script](./scripts/deploy-scripts/mainnet-genesis.sh).
Modify each of the values within this script to match the values of your own deployment, taking any L1 contract addresses from the `./deployments/<name>` folder that was just generated or modified.

Execute this script to generate the genesis file.
You will find this genesis file at `./dist/dumps/state-dump.latest.json`.
You can then ingest this file via `geth init`.

### Hardhat tasks

#### Whitelisting

Optimism has removed the whitelist from the Optimism mainnet.
However, if you are running your own network and still wish to use the whitelist, you can manage the whitelist with the `whitelist` task.
Run the following to get help text for the `whitelist` command:

```
npx hardhat whitelist --help
```

#### Withdrawing fees

Any wallet can trigger a withdrawal of fees within the `SequencerFeeWallet` contract on L2 back to L1 as long as a threshold balance has been reached.
Fees within the wallet will return to a fixed address on L1.
Run the following to get help text for the `withdraw-fees` command:

```
npx hardhat withdraw-fees --help
```

## Security
Please refer to our [Security Policy](https://github.com/ethereum-optimism/.github/security/policy) for information about how to disclose security issues with this code.
We also maintain a [bug bounty program via Immunefi](https://immunefi.com/bounty/optimism/) with a maximum payout of $2,000,042 for critical bug reports.
