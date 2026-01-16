# Network actor

The network actor is responsible for handling interactions with the p2p layer of the kona-node, specifically the libp2p gossip driver and the discv5 handler.

### Example

> **Warning**
>
> Notice, the socket address uses `0.0.0.0`.
> If you are experiencing issues connecting to peers for discovery,
> check to make sure you are not using the loopback address,
> `127.0.0.1` aka "localhost", which can prevent outward facing connections.

```rust,no_run
use std::net::{IpAddr, Ipv4Addr, SocketAddr};
use alloy_primitives::address;
use tokio_util::sync::CancellationToken;
use kona_genesis::RollupConfig;
use kona_p2p::{LocalNode, Config};
use kona_node_service::{NetworkActor};
use libp2p::Multiaddr;
use discv5::enr::CombinedKey;

#[tokio::main]
async fn main() {
    // Construct the Network
    let signer = address!("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb");
    let gossip = SocketAddr::new(IpAddr::V4(Ipv4Addr::UNSPECIFIED), 9099);
    let mut gossip_addr = Multiaddr::from(gossip.ip());
    gossip_addr.push(libp2p::multiaddr::Protocol::Tcp(gossip.port()));
    let advertise_ip = IpAddr::V4(Ipv4Addr::UNSPECIFIED);

    let CombinedKey::Secp256k1(k256_key) = CombinedKey::generate_secp256k1() else {
        unreachable!()
    };
    let disc = LocalNode::new(k256_key, advertise_ip, 9097, 9098);

    // The unsafe blocks are sent by the network actor to `blocks_rx`. This channel receiver can be
    // used by external actors/modules to handle incoming unsafe blocks.
    let (blocks, blocks_rx) = tokio::sync::mpsc::channel(1024);

    let (inbound_data, network) = NetworkActor::new(NetworkActor::builder(Config::new(
        RollupConfig::default(),
        disc,
        gossip_addr,
        signer
    )));

    // This will start the p2p stack of the kona-node (ie the libp2p gossip and discovery layers)
    network.start(NetworkContext { blocks, cancellation: CancellationToken::new() }).await?;
}
```

[!WARNING]: ###example