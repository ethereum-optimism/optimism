//! Network types

use kona_gossip::P2pRpcRequest;
use std::sync::Arc;

/// A type alias for the sender of a [`P2pRpcRequest`].
type P2pReqSender = tokio::sync::mpsc::Sender<P2pRpcRequest>;

/// `P2pRpc`
///
/// This is a server implementation of [`crate::OpP2PApiServer`].
#[derive(Debug)]
pub struct P2pRpc {
    /// The channel to send [`P2pRpcRequest`]s.
    pub sender: P2pReqSender,
    /// The L2 chain ID, pre-rendered as the `chain_id` metric label value.
    pub chain_id_label: Arc<str>,
}

impl P2pRpc {
    /// Constructs a new [`P2pRpc`] given a sender channel and the L2 chain ID.
    pub fn new(sender: P2pReqSender, chain_id: u64) -> Self {
        Self { sender, chain_id_label: Arc::from(chain_id.to_string()) }
    }
}
