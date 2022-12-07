# @eth-optimism/replica-healthcheck

[![codecov](https://codecov.io/gh/ethereum-optimism/optimism/branch/develop/graph/badge.svg?token=0VTG7PG7YR&flag=replica-healthcheck-tests)](https://codecov.io/gh/ethereum-optimism/optimism)

## What is this?

`replica-healthcheck` is an express server to be run alongside a replica instance, to ensure that the replica is healthy. Currently, it exposes metrics on syncing stats and exits when the replica has a mismatched state root against the sequencer.


## Installation

Clone, install, and build the Optimism monorepo:

```
git clone https://github.com/ethereum-optimism/optimism.git
yarn install
yarn build
```

## Running the service (manual)

Copy `.env.example` into a new file named `.env`, then set the environment variables listed there.
You can view a list of all environment variables and descriptions for each via:

```
yarn start --help
```

Once your environment variables have been set, run the relayer via:

```
yarn start
```
