# Optimistic specs

Shared open-source specification for an optimistic-rollup that strives for 1:1 Ethereum on L2 while minimizing additional code.

The dream: *EVM compatibility, all Execution clients, all L1 tooling, L2 security (rollup), no L1-like limits*

This project is a combined effort of the Optimism, EF research and Quilt/Consensys research teams.

Enabling rollups on execution-layer data today, and shard-data in the future.

[**Specs overview**](./overview.md) - experimental!

Components (under construction):
- [Layer 1 Contracts](./components/layer1.md)
- [Rollup Synchronizer](./components/rollup_synchronizer.md)
- [Execution Engine](./components/exec_engine.md)
- [Batch Submitter](./components/batch_submitter.md)
- [Witness Generator](./components/witness_gen.md)
- [Challenge Agent](./components/challenge_agent.md)

## Contribute

Contribute by opening a PR. There are weekly dev calls you can join, 
chat with [@protolambda](https://github.com/protolambda/) or [@karlfloersch](https://twitter.com/karl_dot_tech/).


## License

CC0 1.0 Universal, see [`LICENSE`](./LICENSE) file.
