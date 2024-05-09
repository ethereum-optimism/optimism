# Goldsky subgraphs
Subgraphs have been migrated to Goldsky, more contracts may follow (we may need to evaluate which subgraphs are still needed).
For old subgraphs please refer to the [legacy repository](https://github.com/bobanetwork/boba/tree/develop/packages/boba/subgraph).

## Install GoldSky
`curl https://goldsky.com | sh`

or refer to the official documentation: https://docs.goldsky.com/introduction

Hint: You can also directly migrate your already deployed subgraphs from The Graph to GoldSky.

## Configuration
With GoldSky you only need to add your ABI into the `./abi` folder and create a configuration file (refer to `lightbridge.json`).

Here is also some official [GoldSky documentation](https://docs.goldsky.com/subgraphs/introduction).

To deploy your subgraphs execute:
`goldsky subgraph deploy {NAME_OF_SUBGRAPH}/v{VERSION} --from-abi ./{YOUR_CONFIG_JSON}.json`

Here is an example:
`goldsky subgraph deploy light-bridge/v1 --from-abi ./lightbridge.json`

## API / Account
We already have a GoldSky account. As of 18. Dec 23 following people have already access that you can reach out to for access:
- Boyuan
- Souradeep
- Kevin

## Online dashboard
You can view your subgraphs [here](https://app.goldsky.com/dashboard/subgraphs).
This dashboard also gives you access to your subgraphs directly, so that you can test out queries.

## Queries
GoldSky already generates GraphQL types for, that you can use in your queries (which make your querying much more powerful and easier).

An example:

```graphql
query Teleportation($wallet: String!, $sourceChainId: BigInt!) {
  assetReceiveds(
    where: {and: [{emitter_contains_nocase: $wallet}, { sourceChainId: $sourceChainId }]}
  ) {
    token
    sourceChainId
    toChainId
    depositId
    emitter
    amount
    block_number
    timestamp_
    transactionHash_
  }
}
```

- `emitter_contains_nocase` is a non-standard operation, most GraphQL providers don't support case insensitive querying.
- `block_number`, `timestamp_`, `transactionHash_` and many others are indexed/queryable as well, despite not being part of the on-chain event.

