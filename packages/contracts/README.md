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

### Deploying
TODO: You can deploy by running:

```sh
yarn run deploy:<contract-specific-task-here> <environment>
```

The `environment` parameter tells the deployment script which config file to use (expected filename `.<environment>.env`).

