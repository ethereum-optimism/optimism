//! Network Behaviour Module.

use derive_more::Debug;
use libp2p::{
    gossipsub::{
        Config, IdentTopic, IdentityTransform, MessageAuthenticity, WhitelistSubscriptionFilter,
    },
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

/// The gossipsub behaviour, restricted to the topics the handlers serve.
pub type Gossipsub = libp2p::gossipsub::Behaviour<IdentityTransform, WhitelistSubscriptionFilter>;

/// Specifies the [`NetworkBehaviour`] of the node
#[derive(NetworkBehaviour, Debug)]
#[behaviour(out_event = "Event")]
pub struct Behaviour {
    /// Responds to inbound pings and send outbound pings.
    #[debug(skip)]
    pub ping: libp2p::ping::Behaviour,
    /// Enables gossipsub as the routing layer.
    pub gossipsub: Gossipsub,
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

        let topics = handlers.iter().flat_map(|handler| handler.topics()).collect::<Vec<_>>();

        let mut gossipsub = libp2p::gossipsub::Behaviour::new_with_subscription_filter(
            MessageAuthenticity::Anonymous,
            cfg,
            WhitelistSubscriptionFilter(topics.iter().cloned().collect()),
        )
        .map_err(|_| BehaviourError::GossipsubCreationFailed)?;

        let identify = libp2p::identify::Behaviour::new(
            libp2p::identify::Config::new(String::new(), public_key)
                .with_agent_version("kona".to_string()),
        );

        let sync_req_resp = libp2p_stream::Behaviour::new();

        if !topics.is_empty() {
            tracing::info!(target: "gossip", "Subscribed to topics:");
        }
        for topic in &topics {
            let topic = IdentTopic::new(topic.to_string());
            gossipsub.subscribe(&topic).map_err(|_| BehaviourError::SubscriptionFailed)?;
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
    use libp2p::gossipsub::{IdentTopic, SubscriptionError, TopicHash};

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
        let cfg = config::default_config(10);
        let handlers = vec![];
        let _ = Behaviour::new(key.public(), cfg, &handlers).unwrap();
    }

    #[test]
    fn test_behaviour_rejects_topics_outside_the_handler_set() {
        let key = libp2p::identity::Keypair::generate_secp256k1();
        let cfg = config::default_config();
        let (_, recv) = tokio::sync::watch::channel(Address::default());
        let block_handler = BlockHandler::new(
            RollupConfig { l2_chain_id: Chain::optimism_mainnet(), ..Default::default() },
            recv,
        );
        let handlers: Vec<Box<dyn Handler>> = vec![Box::new(block_handler)];
        let mut behaviour = Behaviour::new(key.public(), cfg, &handlers).unwrap();

        // The filter gates our own subscriptions and the ones peers announce.
        let result = behaviour.gossipsub.subscribe(&IdentTopic::new("/optimism/10/9/blocks"));
        assert!(matches!(result, Err(SubscriptionError::NotAllowed)), "unexpected: {result:?}");

        // `Ok(false)` means we already subscribed.
        let result = behaviour.gossipsub.subscribe(&IdentTopic::new("/optimism/10/0/blocks"));
        assert!(matches!(result, Ok(false)), "unexpected: {result:?}");

        assert_eq!(behaviour.gossipsub.topics().count(), op_mainnet_topics().len());
    }

    #[test]
    fn test_behaviour_with_handlers() {
        let key = libp2p::identity::Keypair::generate_secp256k1();
        let cfg = config::default_config(10);
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
