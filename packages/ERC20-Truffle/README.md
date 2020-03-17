# Overview
This is just an example of using our `solc-transpiler` as the `solc-js` compiler within `truffle` to run a full ERC20 test suite.

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

Alternatively, just run: `yarn all`

### Testing w/ OVM
1. `yarn install`
2. `rm -rf build`
3. `truffle compile --config truffle-config-ovm.js`
4. `truffle test ./truffle-tests/test-erc20.js --config truffle-config-ovm.js`

Alternatively, just run: `yarn all:ovm`


# Waffle 

## Transpiling
1. Make sure the `@eth-optimisim/solc-transpiler` dependency points to the [latest release](https://www.npmjs.com/package/@eth-optimism/solc-transpiler)
2. Run `yarn install`
3. Run `yarn build:waffle`
4. See the compiled + transpiled output in the contract JSON in the `build/waffle/` directory 
