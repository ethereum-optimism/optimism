# @eth-optimism/integration-tests

Note that these tests are ran against the legacy system, see `op-e2e` for
the bedrock test suite.

## Setup

Follow installation + build instructions in the [primary README](../README.md).
Then, run:

```bash
yarn build
```

## Running integration tests

### Testing a live network

Testing on a live network is a bit more complicated than testing locally. You'll need the following in order to do so:

1. A pre-funded wallet with at least 40 ETH.
2. URLs to an L1 and L2 node.
3. The address of the address manager contract.
4. The chain ID of the L2.

Once you have all the necessary info, create a `.env` file like the one in `.env.example` and fill it in with the values above. Then, run:

```bash
yarn test:integration:live
```

This will take quite a long time. Kovan, for example, takes about 30 minutes to complete.

You can also set environment variables on the command line instead of inside `.env` if you want:

```bash
L1_URL=whatever L2_URL=whatever yarn test:integration:live
```

To run the Uniswap integration tests against a deployed set of Uniswap contracts, add the following env vars:

```
UNISWAP_POSITION_MANAGER_ADDRESS=<non fungible position manager address>
UNISWAP_ROUTER_ADDRESS=<router address>
```


## Running actor tests

Actor tests use the same environment variables as the integration tests, so set up your `.env` file if you haven't
already. Then, run `yarn test:actor <args>` to run the tests. Note that it will be **very expensive** to run the actor
tests against mainnet, and that the tests can take a while to complete.

See [actor-tests/README.md](actor-tests/README.md) for information on actor tests. 
