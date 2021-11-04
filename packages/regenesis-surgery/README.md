# @eth-optimism/regenesis-surgery

Scripts used to perform the transition process between OVMv1 and OVMv2.

## Installation

```sh
git clone git@github.com:ethereum-optimism/optimism.git
yarn clean
yarn install
yarn build
```

## Usage

1. Open `.env` and add values for all environment variables listed below.
2. Run `yarn start` to start the surgery process.
3. Grab a coffee or something.

## Environment Variables

| Variable                      | Description                                                   |
| ----------------------------- | --------------------------------------------------------------|
| `REGEN__STATE_DUMP_FILE`      | Path to the state dump file                                   |
| `REGEN__ETHERSCAN_FILE`       | Path to the etherscan dump file                               |
| `REGEN__GENESIS_FILE`         | Path to the initial genesis file                              |
| `REGEN__OUTPUT_FILE`          | Path where the output genesis will be saved                   |
| `REGEN__L2_PROVIDER_URL`      | RPC provider for the L2 network being upgraded                |
| `REGEN__ETH_PROVIDER_URL`     | RPC provider for Ethereum mainnet                             |
| `REGEN__ROPSTEN_PROVIDER_URL` | RPC provider for the Ropsten testnet                          |
| `REGEN__ROPSTEN_PRIVATE_KEY`  | Private key of an account that has Ropsten ETH                |
| `REGEN__STATE_DUMP_HEIGHT`    | Height at which the state dump was taken                      |
| `REGEN__START_INDEX`          | Start index to begin processing the regenesis at (do not set) |
| `REGEN__END_INDEX`            | End index to finish processing the regenesis at (do not set)  |
