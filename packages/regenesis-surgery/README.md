# @eth-optimism/regenesis-surgery

## What is this?

`regenesis-surgery` contains a series of scripts and tests necessary to perform a regenesis on Optimistic Ethereum.

## Getting started

After cloning and switching to the repository, install dependencies:

```bash
$ yarn
```

### Configuration

We're using `dotenv` for our configuration.
To configure the project, clone this repository and copy the `env.example` file to `.env`.

### Runnign scripts

To run an individual script directly:

```bash
$ npx ts-node scripts/event-indexer.ts
```
