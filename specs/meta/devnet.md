<!-- DOCTOC SKIP -->
# Bedrock Local Devnet Setup

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [Bedrock Local Devnet Setup](#bedrock-local-devnet-setup)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

You can spin up a local devnet via `docker compose`.
For convenience, we have defined `make` targets to start and stop the devnet with a single command.
To run the devnet, you will need `docker` installed.
Then, as a precondition, make sure that you have compiled the contracts by `cd`ing into `packages/contracts-bedrock`
and running `pnpm i` followed by `pnpm build`. You'll only need to do this if you change the contracts in the future.

Then, run the following:

```bash
make devnet-up # starts the devnet
make devnet-down # stops the devnet
make devnet-clean # removes the devnet by deleting images and persistent volumes
```

L1 is accessible at `http://localhost:8545`, and L2 is accessible at `http://localhost:9545`.
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

```bash
L1_PROVIDER_URL=http://localhost:8545
L2_PROVIDER_URL=http://localhost:9545
PRIVATE_KEY=bf7604d9d3a1c7748642b1b7b05c2bd219c9faa91458b370f85e5a40f3b03af7
```

The batch submitter uses the account below to submit batches to L1:

- Address: `0xde3829a23df1479438622a08a116e8eb3f620bb5`
- Private key: `bf7604d9d3a1c7748642b1b7b05c2bd219c9faa91458b370f85e5a40f3b03af7`
