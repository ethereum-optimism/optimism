# Network actor

The network actor runs the libp2p gossip swarm of the kona-node. Peer discovery runs in the separate discovery actor, which passes the peers it finds to the network actor to dial.

### Integration

> **Warning**
>
> Notice, the socket address uses `0.0.0.0`.
> If you are experiencing issues connecting to peers for discovery,
> check to make sure you are not using the loopback address,
> `127.0.0.1` aka "localhost", which can prevent outward facing connections.

The node service calls `NetworkConfig::start`, which starts the gossip swarm listening and builds the discovery driver. It passes the gossip driver to `network::Builder::new`, gives the builder's handle to the discovery actor and the P2P RPC, and builds the actor's lifetime future with the engine, discovery and the signer's payloads.

When an actor exits or the node lifetime is dropped, the supervisor aborts the remaining actor tasks. See [the node service](../../../service/src/service/node.rs) for the wiring.
