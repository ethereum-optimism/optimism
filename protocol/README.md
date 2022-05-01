[![codecov](https://codecov.io/gh/ethereum-optimism/optimistic-specs/branch/main/graph/badge.svg?token=19JPIN9XPB)](https://codecov.io/gh/ethereum-optimism/optimistic-specs)

# The Optimism Spec

This repository holds the work-in-progress specification for the next version of
Optimism.

This spec is developed iteratively, specifying a rollup of increasing
complexity. The current stage specifies a "rollup" with deposits, withdrawals and sequenced (L2-native) transactions.
Some aspects, such as the fee logic and calldata compression, are still missing or in placeholder state.

## Work in Progress

Please note that this specification is currently under heavy construction.

## Local Devnet Setup

You can spin up a local devnet via `docker-compose`.
For convenience, we have defined `make` targets to start and stop the devnet with a single command.
To run the devnet, you will need `docker` and `docker-compose` installed.
Then, as a precondition, make sure that you have compiled the contracts by `cd`ing into `packages/contracts`
and running `yarn` followed by `yarn build`. You'll only need to do this if you change the contracts in the future.

Then, run the following:

```bash
make devnet-up # starts the devnet
make devnet-down # stops the devnet
make devnet-clean # removes the devnet by deleting images and persistent volumes
```

L1 is accessible at `http://localhost:8545`, and L2 is accessible at `http://localhost:8546`.
Any Ethereum tool - Metamask, `seth`, etc. - can use these endpoints.
Note that you will need to specify the L2 chain ID manually if you use Metamask. The devnet's L2 chain ID is 901.

The devnet comes with a pre-funded account you can use as a faucet:

- Address: `0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266`
- Private key: `ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80`

The faucet account exists on both L1 and L2. To deposit onto L2 from L1, you can use the `deposit` hardhat task.
Run the following from the `packags/contracts` directory:

```bash
npx hardhat deposit --amount-eth <amount in eth> --to <address>
````

You'll need a `.env` with the following contents:

```
L1_PROVIDER_URL=http://localhost:8545
L2_PROVIDER_URL=http://localhost:8546
PRIVATE_KEY=bf7604d9d3a1c7748642b1b7b05c2bd219c9faa91458b370f85e5a40f3b03af7
```

The batch submitter uses the account below to submit batches to L1:

- Address: `0xde3829a23df1479438622a08a116e8eb3f620bb5`
- Private key: `bf7604d9d3a1c7748642b1b7b05c2bd219c9faa91458b370f85e5a40f3b03af7`

## Contributing

### Basic Contributions

Contributing to the Optimism specification is easy.

You'll find a list of open questions and active research topics over on the
[Discussions] page for this repo. Specific tasks and TODOs can be found on the
[Issues] page. You can edit content or add new pages by creating a [pull
request].

[Discussions]: https://github.com/ethereum-optimism/optimistic-specs/discussions
[Issues]: https://github.com/ethereum-optimism/optimistic-specs/issues
[pull request]: https://github.com/ethereum-optimism/optimistic-specs/pulls

## License

Specification: CC0 1.0 Universal, see [`specs/LICENSE`](./specs/LICENSE) file.

Reference software: MIT, see [`LICENSE`](./LICENSE) file.
