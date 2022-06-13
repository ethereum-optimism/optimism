# Running Cannon on L2

The original Cannon prototype allows challenging L1 blocks on L1. Normally, the
challenger should always fail, as L1 blocks are valid by virtue of being
included on-chain.

The next milestone is to allow challenging L2 blocks on L2. What this proves is
that the extra logic we added in
[l2geth](https://github.com/ethereum-optimism/reference-optimistic-geth) (aka
reference-optimism-geth) does not break anything. It's also a good way to
exercise our devnet infrastructure.

Running the Cannon demo on a mainnet (L1) fork is a simple as doing:

```bash
# from repo root
demo/challenge_simple.sh
# or
demo/challenge_fault.sh
```

For L2, you'll need first to run the devnet locally. For this, clone the
`develop` branch of the [optimism
monorepo](https://github.com/ethereum-optimism/optimism), then run:

```bash
yarn && make build && make devnet-clean && make devnet-up
```

If you're having trouble building, here's a [full
transcript](https://github.com/ethereum-optimism/cannon/wiki/Bedrock-Full-Devnet-Setup)
of all the commands required to run on a fresh cloud linux machine.

Note it's important to run `make devnet-clean` before each invocation of `make
devnet-up` to work around some issues at the time of writing.

Then you can run the L2 demos:

```bash
# from repo root
demo/l2_challenge_simple.sh
# or
demo/l2_challenge_fault.sh
```
