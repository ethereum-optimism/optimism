# Network actor

The network actor is responsible for handling interactions with the p2p layer of the kona-node, specifically the libp2p gossip driver and the discv5 handler.

### Example

[`examples/gossip`](../../../../../../examples/gossip/src/main.rs) builds a `NetworkConfig`, starts the network actor from it, and logs the blocks it receives over gossip.

What the node advertises to other nodes depends on the IP it listens on: on all interfaces (`0.0.0.0`), its record has no IP until enough peers agree on its address. See [advertised address](https://docs.optimism.io/node-operators/kona-node/configuration#advertised-address).
