use eyre::WrapErr as _;
use libp2p::{
    Swarm, autonat,
    connection_limits::{self, ConnectionLimits},
    identify, identity, mdns, ping,
    swarm::NetworkBehaviour,
};
use std::{convert::Infallible, time::Duration};

const PROTOCOL_VERSION: &str = "1.0.0";

#[derive(NetworkBehaviour)]
#[behaviour(to_swarm = "BehaviourEvent")]
pub(crate) struct Behaviour {
    // connection gating
    connection_limits: connection_limits::Behaviour,

    // discovery
    mdns: mdns::tokio::Behaviour,

    // protocols
    identify: identify::Behaviour,
    ping: ping::Behaviour,
    stream: libp2p_stream::Behaviour,

    // nat traversal
    autonat: autonat::Behaviour,
}

#[allow(clippy::large_enum_variant)]
#[derive(Debug, derive_more::From)]
pub(crate) enum BehaviourEvent {
    Autonat(autonat::Event),
    Identify(identify::Event),
    Mdns(mdns::Event),
    Ping(ping::Event),
}

impl From<()> for BehaviourEvent {
    fn from(_: ()) -> Self {
        unreachable!("() cannot be converted to BehaviourEvent")
    }
}

impl From<Infallible> for BehaviourEvent {
    fn from(_: Infallible) -> Self {
        unreachable!("Infallible cannot be converted to BehaviourEvent")
    }
}

impl Behaviour {
    pub(crate) fn new(
        keypair: &identity::Keypair,
        agent_version: String,
        max_peer_count: u32,
    ) -> eyre::Result<Self> {
        let peer_id = keypair.public().to_peer_id();

        let autonat = autonat::Behaviour::new(peer_id, autonat::Config::default());
        let mdns = mdns::tokio::Behaviour::new(mdns::Config::default(), peer_id)
            .wrap_err("failed to create mDNS behaviour")?;
        let connection_limits = connection_limits::Behaviour::new(
            ConnectionLimits::default().with_max_established(Some(max_peer_count)),
        );

        let identify = identify::Behaviour::new(
            identify::Config::new(PROTOCOL_VERSION.to_string(), keypair.public())
                .with_agent_version(agent_version),
        );
        let ping = ping::Behaviour::new(ping::Config::new().with_interval(Duration::from_secs(10)));
        let stream = libp2p_stream::Behaviour::new();

        Ok(Self {
            autonat,
            connection_limits,
            identify,
            ping,
            mdns,
            stream,
        })
    }

    pub(crate) fn new_control(&mut self) -> libp2p_stream::Control {
        self.stream.new_control()
    }
}

impl BehaviourEvent {
    pub(crate) fn handle(self, swarm: &mut Swarm<Behaviour>) {
        match self {
            BehaviourEvent::Autonat(_event) => {}
            BehaviourEvent::Identify(_event) => {}
            BehaviourEvent::Mdns(event) => match event {
                mdns::Event::Discovered(list) => {
                    for (peer_id, multiaddr) in list {
                        if swarm.is_connected(&peer_id) {
                            continue;
                        }

                        tracing::debug!("mDNS discovered peer {peer_id} at {multiaddr}");
                        swarm.add_peer_address(peer_id, multiaddr);
                        swarm.dial(peer_id).unwrap_or_else(|e| {
                            tracing::error!("failed to dial mDNS discovered peer {peer_id}: {e}")
                        });
                    }
                }
                mdns::Event::Expired(list) => {
                    for (peer_id, multiaddr) in list {
                        tracing::debug!("mDNS expired peer {peer_id} at {multiaddr}");
                    }
                }
            },
            BehaviourEvent::Ping(_event) => {}
        }
    }
}
