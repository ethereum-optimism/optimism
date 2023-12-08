## Running op-challenger

`op-challenger` is a program that implements the honest actor algorithm to automatically “play” the dispute games.

### Prerequisites

- The cannon pre-state downloaded from [Goerli deployment](./deployments.md#goerli).
- An account on the Goerli testnet with funds available. The amount of GöETH required depends on the number of claims
  the challenger needs to post, but 0.01 ETH should be plenty to start.
- A Goerli L1 node.
    - An archive node is not required.
    - Public RPC providers can be used, however a significant number of requests will need to be made which may exceed
      rate limits for free plans.
- An OP-Goerli L2 archive node with `debug` APIs enabled.
    - An archive node is required to ensure world-state pre-images remain available.
    - Public RPC providers are generally not usable as they don’t support the `debug_dbGet` RPC method.
- Approximately 3.5Gb of disk space for each game being played.

### Starting op-challenger

When executing `op-challenger`, there are a few placeholders that need to be set to concrete values:

- `<L1_URL>` the Goerli L1 JSON RPC endpoint
- `<DISPUTE_GAME_FACTORY_ADDRESS>` the address of the dispute game factory contract (see
  the [Goerli deployment details](./deployments.md#goerli))
- `<PRESTATE>` the prestate.json downloaded above. Note that this needs to precisely match the prestate used on-chain so
  must be the downloaded version and not a version built locally (see the [Goerli deployment details](./deployments.md#goerli))
- `<L2_URL>` the OP-Goerli L2 archive node JSON RPC endpoint
- `<PRIVATE_KEY>` the private key for a funded Goerli account. For other ways to specify the account to use
  see `./op-challenger/bin/op-challenger --help`

From inside the monorepo directory, run the challenger after setting these placeholders.

```bash
# Build the required components
make op-challenger op-program cannon

# Run op-challenger
./op-challenger/bin/op-challenger \
  --trace-type cannon \
  --l1-eth-rpc <L1_URL> \
  --game-factory-address <DISPUTE_GAME_FACTORY_ADDRESS> \
  --agree-with-proposed-output=true \
  --datadir temp/challenger-goerli \
  --cannon-network goerli \
  --cannon-bin ./cannon/bin/cannon \
  --cannon-server ./op-program/bin/op-program \
  --cannon-prestate <PRESTATE> \
  --cannon-l2 <L2_URL> \
  --private-key <PRIVATE_KEY>
```


### Restricting Games to Play

By default `op-challenger` will generate traces and respond to any game created by the dispute game factory contract. On
a public testnet like Goerli, that could be a large number of games, requiring significant CPU and disk resources. To
avoid this, `op-challenger` supports specifying an allowlist of games for it to respond to with the `--game-allowlist`
option.

```bash
./op-challenger/bin/op-challenger \
  ... \
  --game-allowlist <GAME_ADDR> <GAME_ADDR> <GAME_ADDR>...
```
