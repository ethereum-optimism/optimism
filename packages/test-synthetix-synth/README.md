# Overview
This is just an example of using our `solc-transpiler` as the `solc-js` compiler within `waffle` and `truffle`.

# Truffle
## Transpiling
1. Make sure the `@eth-optimisim/solc-transpiler` dependency points to the [latest release](https://www.npmjs.com/package/@eth-optimism/solc-transpiler)
2. Run `yarn install`
3. Run `truffle compile --config truffle-config-ovm.js`
4. See the compiled + transpiled output in the contract JSON in the `build/contracts/` directory

## Testing
The beauty of the OVM and our compatibility with Ethereum dev tools is that you can test regularly or test against the OVM _without any code changes_. 

### Testing Regularly
1. `yarn install`
2. `rm -rf build`
3. `truffle compile`
4. `truffle test ./truffle-tests/test-erc20.js`

### Testing w/ OVM
1. `yarn install`
2. `rm -rf build`
3. `truffle compile --config truffle-config-ovm.js`
4. Make sure the `rollup-full-node` is [running](https://github.com/ethereum-optimism/optimism-monorepo/blob/master/packages/rollup-full-node/README.md#running-the-fullnode-server)
5. `truffle test ./truffle-tests/test-erc20.js --config truffle-config-ovm.js`


# Waffle 

## Transpiling
1. Make sure the `@eth-optimisim/solc-transpiler` dependency points to the [latest release](https://www.npmjs.com/package/@eth-optimism/solc-transpiler)
2. Run `yarn install`
3. Run `yarn build:waffle`
4. See the compiled + transpiled output in the contract JSON in the `build/waffle/` directory 
