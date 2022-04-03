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
For example, the [L1CrossDomainMessenger](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts/contracts/L1/messaging/L1CrossDomainMessenger.sol) contract is located at `packages/contracts/contracts/L1/messaging/L1CrossDomainMessenger.sol`, relative to this README.
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

Compile and build the various required with the `build` command:

```shell
yarn build
```

### Deploying the Contracts

#### Required environment variables

You must set several required environment variables before you can execute a deployment.
Duplicate the file [`.env.example`](./.env.example) and rename your duplicate to `.env`.
Fill out each of the environment variables before continuing.

#### Creating a deployment configuration

Before you can carry out a deployment, you must create a deployment configuration file inside of the [deploy-config](./deploy-config/) folder.
Deployment configuration files are TypeScript files that export an object that conforms to the `DeployConfig` type.
See [mainnet.ts](./deploy-config/mainnet.ts) for an example deployment configuration.
We recommend duplicating an existing deployment config and modifying it to satisfy your requirements.

#### Executing a deployment

Once you've created your deploy config, you can execute a deployment with the following command:

```
npx hardhat deploy --network <my network name>
```

Note that this only applies to fresh deployments.
If you want to upgrade an existing system (instead of deploying a new system from scratch), you must use the following command instead:

```
npx hardhat deploy --network <my network name> --tags upgrade
```

During the deployment process, you will be asked to transfer ownership of several contracts to a special contract address.
You will also be asked to verify various configuration values.
This is a safety mechanism to make sure that actions within an upgrade are performed atomically.
Ownership of these addresses will be automatically returned to the original owner address once the upgrade is complete.
The original owner can always recover ownership from the upgrade contract in an emergency.
Please read these instructions carefully, verify each of the presented configuration values, and carefully confirm that the contract you are giving ownership to has not been compromised (e.g., check the code on Etherscan).

After your deployment is complete, your new contracts will be written to an artifacts directory in `./deployments/<my network name>`.

#### Verifying contract source code

Contracts will be automatically verified via both [Etherscan](https://etherscan.io) and [Sourcify](https://sourcify.dev/) during the deployment process.
If there was an issue with verification during the deployment, you can manually verify your contracts with the command:

```
npx hardhat etherscan-verify --network <my network name>
```

#### Creating a genesis file

Optimism expects that certain contracts (called "predeploys") be deployed to the L2 network at pre-determined addresses.
We guarantee this by creating a genesis file in which certain contracts are already within the L2 state at the genesis block.
To create the genesis file for your network, you must first deploy the L1 contracts using the appropriate commands from above.
Once you've deployed your contracts, run the following command:

```
npx hardhat take-dump --network <my network name>
```

A genesis file will be created for you at `/genesis/<my network name>.json`.
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
