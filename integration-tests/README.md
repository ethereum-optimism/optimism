# @eth-optimism/integration-tests

## Setup

Follow installation + build instructions in the [primary README](../README.md).
Then, run:

```bash
yarn build:integration
```

## Running tests

### Testing a live network

Create an `.env` file and fill it out.
Look at `.env.example` to know which variables to include.

Once you have your environment set up, run:

```bash
yarn test:integration:live
```

You can also set environment variables on the command line instead of inside `.env` if you want:

```bash
L1_URL=whatever L2_URL=whatever yarn test:integration:live
```

Note that this can take an extremely long time (~1hr).
