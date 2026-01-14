//! Network Behaviour Module.

use derive_more::Debug;
use libp2p::{
    gossipsub::{Config, IdentTopic, MessageAuthenticity},
    swarm::NetworkBehaviour,
};

use crate::{Event, Handler};

/// An error that can occur when creating a [`Behaviour`].
#[derive(Debug, Clone, PartialEq, Eq, thiserror::Error)]
pub enum BehaviourError {
    /// The gossipsub behaviour creation failed.
    #[error("gossipsub behaviour creation failed")]
    GossipsubCreationFailed,
    /// Subscription failed.
    #[error("subscription failed")]
    SubscriptionFailed,
    /// Failed to set the peer score on the gossipsub.
    #[error("{0}")]
    PeerScoreFailed(String),
}

/// Specifies the [`NetworkBehaviour`] of the node
#[derive(NetworkBehaviour, Debug)]
#[behaviour(out_event = "Event")]
pub struct Behaviour {
    /// Responds to inbound pings and send outbound pings.
    #[debug(skip)]
    pub ping: libp2p::ping::Behaviour,
    /// Enables gossipsub as the routing layer.
    pub gossipsub: libp2p::gossipsub::Behaviour,
    /// Enables the identify protocol.
    #[debug(skip)]
    pub identify: libp2p::identify::Behaviour,
    /// Enables the sync request/response protocol.
    /// See `<https://specs.optimism.io/protocol/rollup-node-p2p.html#payload_by_number>`
    #[debug(skip)]
    pub sync_req_resp: libp2p_stream::Behaviour,
}

impl Behaviour {
    /// Configures the swarm behaviors, subscribes to the gossip topics, and returns a new
    /// [`Behaviour`].
    pub fn new(
        public_key: libp2p::identity::PublicKey,
        cfg: Config,
        handlers: &[Box<dyn Handler>],
    ) -> Result<Self, BehaviourError> {
        let ping = libp2p::ping::Behaviour::default();

        let mut gossipsub = libp2p::gossipsub::Behaviour::new(MessageAuthenticity::Anonymous, cfg)
            .map_err(|_| BehaviourError::GossipsubCreationFailed)?;

        let identify = libp2p::identify::Behaviour::new(
            libp2p::identify::Config::new("".to_string(), public_key)
                .with_agent_version("kona".to_string()),
        );

        let sync_req_resp = libp2p_stream::Behaviour::new();

        let subscriptions = handlers
            .iter()
            .flat_map(|handler| {
                handler
                    .topics()
                    .iter()
                    .map(|topic| {
                        let topic = IdentTopic::new(topic.to_string());
                        gossipsub
                            .subscribe(&topic)
                            .map_err(|_| BehaviourError::SubscriptionFailed)?;
                        Ok(topic.to_string())
                    })
                    .collect::<Vec<_>>()
            })
            .collect::<Result<Vec<String>, BehaviourError>>()?;

        if !subscriptions.is_empty() {
            tracing::info!(target: "gossip", "Subscribed to topics:");
        }
        for topic in subscriptions {
            tracing::info!(target: "gossip", "-> {}", topic);
        }

        Ok(Self { identify, ping, gossipsub, sync_req_resp })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{config, handler::BlockHandler};
    use alloy_chains::Chain;
    use alloy_primitives::Address;
    use kona_genesis::RollupConfig;
    use libp2p::gossipsub::{IdentTopic, TopicHash};

    fn op_mainnet_topics() -> Vec<TopicHash> {
        vec![
            IdentTopic::new("/optimism/10/0/blocks").hash(),
            IdentTopic::new("/optimism/10/1/blocks").hash(),
            IdentTopic::new("/optimism/10/2/blocks").hash(),
            IdentTopic::new("/optimism/10/3/blocks").hash(),
        ]
    }

    #[test]
    fn test_behaviour_no_handlers() {
        let key = libp2p::identity::Keypair::generate_secp256k1();
        let cfg = config::default_config();
        let handlers = vec![];
        let _ = Behaviour::new(key.public(), cfg, &handlers).unwrap();
    }

    #[test]
    fn test_behaviour_with_handlers() {
        let key = libp2p::identity::Keypair::generate_secp256k1();
        let cfg = config::default_config();
        let (_, recv) = tokio::sync::watch::channel(Address::default());
        let block_handler = BlockHandler::new(
            RollupConfig { l2_chain_id: Chain::optimism_mainnet(), ..Default::default() },
            recv,
        );
        let handlers: Vec<Box<dyn Handler>> = vec![Box::new(block_handler)];
        let behaviour = Behaviour::new(key.public(), cfg, &handlers).unwrap();
        let mut topics = behaviour.gossipsub.topics().cloned().collect::<Vec<TopicHash>>();
        topics.sort();
        assert_eq!(topics, op_mainnet_topics());
    }
}
