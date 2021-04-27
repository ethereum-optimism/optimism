# Integration Tests

Typescript based integration test repo for OMGX.

## CONFIGURATION

Create a `.env` file in the root directory of this project. Add environment-specific variables on new lines in the form of `NAME=VALUE`.

To test on Rinkeby, ChainID4, you will need an Infura key and two accounts with Rinkeby ETH in them. The text wallets must contain enough ETH to cover the tests. 

**The full test suite includes some very slow transactions such as withdrawls, whic hcan take 300 seconds each. Please be patient.**

## Test local

```bash
NODE_ENV=local
L1_NODE_WEB3_URL=http://localhost:9545
L2_NODE_WEB3_URL=http://localhost:8545
ETH1_ADDRESS_RESOLVER_ADDRESS=0x3e4CFaa8730092552d9425575E49bB542e329981
TEST_PRIVATE_KEY_1=0x754fde3f5e60ef2c7649061e06957c29017fe21032a8017132c0078e37f6193a
TEST_PRIVATE_KEY_2=0x23d9aeeaa08ab710a57972eb56fc711d9ab13afdecc92c89586e0150bfa380a6
TARGET_GAS_LIMIT=9000000000
CHAIN_ID=420
```

## Test Rinkeby

```bash
NODE_ENV=local
L1_NODE_WEB3_URL=https://rinkeby.infura.io/v3/KEY
L2_NODE_WEB3_URL=http://54.161.5.63:8545
ETH1_ADDRESS_RESOLVER_ADDRESS=0xa32cf2433ba24595d3aCE5cc9A7079d3f1CC5E0c
TEST_PRIVATE_KEY_1=0xPRIVATE KEY OF THE FIRST TEST WALLET
TEST_PRIVATE_KEY_2=0xPRIVATE KEY OF THE SECOND TEST WALLET
TARGET_GAS_LIMIT=9000000000
CHAIN_ID=420
```

## PERFORM THE TESTS

```bash
$ yarn install
$ yarn build:integration
$ yarn test:integration
```
