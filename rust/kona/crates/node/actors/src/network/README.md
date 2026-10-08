# Network actor

The network actor is responsible for handling interactions with the p2p layer of the kona-node, specifically the libp2p gossip driver and the discv5 handler.

### Integration

> **Warning**
>
> Notice, the socket address uses `0.0.0.0`.
> If you are experiencing issues connecting to peers for discovery,
> check to make sure you are not using the loopback address,
> `127.0.0.1` aka "localhost", which can prevent outward facing connections.

The node service starts the gossip and discovery drivers, constructs the network actor with its clients and channels, and supervises a lifetime future that repeatedly calls `NodeActor::step`.

When an actor exits or the node lifetime is dropped, the supervisor aborts the remaining actor tasks. See [the node service](../../../service/src/service/node.rs) for the wiring.
