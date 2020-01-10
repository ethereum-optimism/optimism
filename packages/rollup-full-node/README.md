# Aggregator
TODO: this.

# Dependencies
The `/exec/` scripts depend on [parity](https://github.com/paritytech/parity-ethereum/releases/tag/v2.5.13) being installed.

For other dependencies, please refer to the root README of this repo.

# Setup
Run `yarn install` to install necessary dependencies.

# Building
Run `yarn build` to build the code. Note: `yarn all` may be used to build and run tests.

# Testing
Run `yarn test` to run the unit tests.

# Configuration
`/config/default.json` specifies the default configuration. 
Overrides will be read from environment variables with the same key.

`/config/parity/local-chain-config.json` configures the local parity chain. This should not normally need modification.

# Running the Server
Run `yarn server` to run the aggregator server.

# Running a Persistent Chain
Run `./exec/startChain.sh` to start a local persistent blockchain.
Note: This chain will be initiated with a LOT of ETH in the following account:
* address: `0x77e3E8EF810e2eD36c396A80EC21379e345b862e`
* mnemonic: `response fresh afford leader twice silent table exist aisle pelican focus bird`

# Deleting Persistent Chain DB
Run `./exec/purgeChainDb.sh`

