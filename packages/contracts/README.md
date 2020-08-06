# contracts
`rollup-contracts` is the set of smart contracts written in Solidity for Optimism's Optimistic Rollup.

### Requirements and Setup
Clone the parent repo `optimism-monorepo` and follow its instructions.

#### Node.js
`rollup-contracts` is tested with [`Node.js`](https://nodejs.org/en/) and has been tested on the following versions of Node:

- 11.6.0

If you're having trouble getting `rollup-contracts` tests running, please make sure you have one of the above `Node.js` versions installed.

### Running Tests
`rollup-contracts` makes use of a combination of [`Mocha`](https://mochajs.org/) (a testing framework) and [`Chai`](https://www.chaijs.com/) (an assertion library) for testing.

Run all tests with:

```sh
yarn test
```
So that Python and Vyper aren't requirements for our other components, we do include a `compiled-contracts` folder which contains JS exports of the bytecode and ABI. Compilation is done automatically before testing.

### Deployment
#### Configuration
The following environment variables must be configured to deploy contracts:

*L1 Node:*

Either:
* `L1_NODE_INFURA_NETWORK` - The network to use for Infura deployments.
* `L1_NODE_INFURA_PROJECT_ID` - The Project ID to use for Infura deployments.

Or:
* `L1_NODE_WEB3_URL` - The URL of the node through which the deployment will be done.

*Deployment Wallet*

Either:
* `L1_CONTRACT_DEPLOYMENT_PRIVATE_KEY` - The private key to use for contract deployment.

Or:
* `L1_CONTRACT_DEPLOYMENT_MNEMONIC` - The BIP-39/BIP-44 wallet mnemonic to use for contract deployment.

*Contract / Deployment Variables*
* `L1_CONTRACT_OWNER_ADDRESS` - The owner of the deployed contracts (where applicable). Defaults to deployer address if not provided.
* `FORCE_INCLUSION_PERIOD_SECONDS` - The maximum time in seconds between when a tx may be executed in L2 and when it must be mined on-chain
* `L1_SEQUENCER_ADDRESS` - The address of the sequencer that will be authorized to submit rollup blocks & roots.
* `L1_ADDRESS_RESOLVER_CONTRACT_ADDRESS` - (optional) The Address Resolver contract to use to determine which contracts actually need to be deployed. Contracts registered with this AddressResolver will not be re-deployed.

#### Deployment
After proper configuration, you can deploy all contracts by running:

```sh
yarn run deploy:all
```
