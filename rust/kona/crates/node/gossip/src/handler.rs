//! Block Handler

use crate::{
    HandlerEncodeError,
    config::snappy_decompressed_len_within_bound,
    payload::{GossipPayloadVersion, SignedGossipPayload},
};
use alloy_primitives::{Address, B256, Signature};
use kona_genesis::RollupConfig;
use libp2p::gossipsub::{IdentTopic, Message, MessageAcceptance, TopicHash};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::{
    collections::{BTreeMap, HashSet},
    sync::Arc,
};
use tokio::sync::watch::Receiver;

/// This trait defines the functionality required to process incoming messages
/// and determine their acceptance within the network.
///
/// Implementors of this trait can specify how messages are handled and which
/// topics they are interested in.
pub trait Handler: Send {
    /// Manages validation and further processing of messages
    /// This is a stateful method, because the handler needs to keep track of seen hashes.
    fn handle(&mut self, msg: Message) -> (MessageAcceptance, Option<OpExecutionPayloadEnvelope>);

    /// Specifies which topics the handler is interested in
    fn topics(&self) -> Vec<TopicHash>;
}

/// Responsible for managing blocks received via p2p gossip
#[derive(Debug, Clone)]
pub struct BlockHandler {
    /// The rollup config used to validate the block.
    pub rollup_config: RollupConfig,
    /// A [`Receiver`] to monitor changes to the unsafe block signer.
    pub signer_recv: Receiver<Address>,
    /// The libp2p topic for pre Canyon/Shangai blocks.
    pub blocks_v1_topic: IdentTopic,
    /// The libp2p topic for Canyon/Delta blocks.
    pub blocks_v2_topic: IdentTopic,
    /// The libp2p topic for Ecotone V3 blocks.
    pub blocks_v3_topic: IdentTopic,
    /// The libp2p topic for V4 blocks.
    pub blocks_v4_topic: IdentTopic,
    /// A map of seen block height to block hash set.
    /// This map is pruned when it contains more than [`Self::SEEN_HASH_CACHE_SIZE`] entries.
    pub seen_hashes: BTreeMap<u64, HashSet<B256>>,
    /// The L2 chain ID, pre-rendered as the `chain_id` metric label value.
    ///
    /// Cached as an [`Arc<str>`] so the gossip hot paths clone a refcount rather than
    /// re-allocating the label on every emit.
    pub chain_id_label: Arc<str>,
}

impl Handler for BlockHandler {
    /// Authenticates and validates a signed execution payload received over P2P gossip.
    fn handle(&mut self, msg: Message) -> (MessageAcceptance, Option<OpExecutionPayloadEnvelope>) {
        // Reject frames whose Snappy header declares a decompressed size over MAX_GOSSIP_SIZE
        // before decoding. The decoder would otherwise pre-allocate the declared length (up to
        // roughly 4 GiB) from a tiny frame. This mirrors op-node's gossip topic validator.
        if snappy_decompressed_len_within_bound(&msg.data).is_none() {
            kona_macros::inc!(counter, crate::Metrics::INVALID_MESSAGE, "reason" => "invalid_snappy_length", crate::Metrics::CHAIN_ID_LABEL => self.chain_id_label.clone());
            debug!(target: "gossip", len = msg.data.len(), "Rejecting Snappy frame with invalid declared length before decode");
            return (MessageAcceptance::Reject, None);
        }

        let Some(version) = self.payload_version(&msg.topic) else {
            // Unreachable in practice because the driver dispatches only known block topics.
            kona_macros::inc!(counter, crate::Metrics::INVALID_MESSAGE, "reason" => "unknown_topic", crate::Metrics::CHAIN_ID_LABEL => self.chain_id_label.clone());
            debug!(target: "gossip", topic = ?msg.topic, "Received block with unknown topic");
            return (MessageAcceptance::Reject, None);
        };

        let signed = match SignedGossipPayload::decode_snappy(&msg.data) {
            Ok(signed) => signed,
            Err(err) => {
                kona_macros::inc!(counter, crate::Metrics::INVALID_MESSAGE, "reason" => "decode_error", crate::Metrics::CHAIN_ID_LABEL => self.chain_id_label.clone());
                debug!(target: "gossip", ?err, "Failed to decode signed gossip payload");
                return (MessageAcceptance::Reject, None);
            }
        };
        let payload_hash = signed.payload_hash();
        #[cfg(feature = "metrics")]
        kona_macros::inc!(counter, crate::Metrics::BLOCK_VALIDATION_TOTAL, crate::Metrics::CHAIN_ID_LABEL => self.chain_id_label.clone());

        // Authenticate the exact envelope bytes before decoding the SSZ payload and transactions.
        if let Err(err) = self.validate_signature(&signed) {
            debug!(target: "gossip", ?err, hash = ?payload_hash, "Received unauthenticated block");
            return (err.into(), None);
        }

        let envelope = match signed.into_envelope(version) {
            Ok(envelope) => envelope,
            Err(err) => {
                kona_macros::inc!(counter, crate::Metrics::INVALID_MESSAGE, "reason" => "decode_error", crate::Metrics::CHAIN_ID_LABEL => self.chain_id_label.clone());
                #[cfg(feature = "metrics")]
                kona_macros::inc!(counter, crate::Metrics::BLOCK_VALIDATION_FAILED, "reason" => "invalid_block", crate::Metrics::CHAIN_ID_LABEL => self.chain_id_label.clone());
                debug!(target: "gossip", ?err, hash = ?payload_hash, "Failed to decode authenticated execution payload");
                return (MessageAcceptance::Reject, None);
            }
        };

        match self.block_valid(&envelope) {
            Ok(()) => (MessageAcceptance::Accept, Some(envelope)),
            Err(err) => {
                // Already metered by BLOCK_VALIDATION_FAILED; debug-log for investigation without
                // letting invalid remote blocks spam warnings.
                debug!(target: "gossip", ?err, hash = ?payload_hash, "Received invalid block");
                (err.into(), None)
            }
        }
    }

    /// The gossip topics accepted for new blocks
    fn topics(&self) -> Vec<TopicHash> {
        vec![
            self.blocks_v1_topic.hash(),
            self.blocks_v2_topic.hash(),
            self.blocks_v3_topic.hash(),
            self.blocks_v4_topic.hash(),
        ]
    }
}

impl BlockHandler {
    /// Creates a new [`BlockHandler`].
    ///
    /// Requires the chain ID and a receiver channel for the unsafe block signer.
    pub fn new(rollup_config: RollupConfig, signer_recv: Receiver<Address>) -> Self {
        let chain_id = rollup_config.l2_chain_id.id();
        Self {
            rollup_config,
            signer_recv,
            chain_id_label: kona_macros::chain_id_label(chain_id),
            blocks_v1_topic: IdentTopic::new(format!("/optimism/{chain_id}/0/blocks")),
            blocks_v2_topic: IdentTopic::new(format!("/optimism/{chain_id}/1/blocks")),
            blocks_v3_topic: IdentTopic::new(format!("/optimism/{chain_id}/2/blocks")),
            blocks_v4_topic: IdentTopic::new(format!("/optimism/{chain_id}/3/blocks")),
            seen_hashes: BTreeMap::new(),
        }
    }

    /// Returns the topic using the specified timestamp and optional [`RollupConfig`].
    ///
    /// Reference: <https://github.com/ethereum-optimism/optimism/blob/0bc5fe8d16155dc68bcdf1fa5733abc58689a618/op-node/p2p/gossip.go#L604C1-L612C3>
    pub fn topic(&self, timestamp: u64) -> IdentTopic {
        if self.rollup_config.is_isthmus_active(timestamp) {
            self.blocks_v4_topic.clone()
        } else if self.rollup_config.is_ecotone_active(timestamp) {
            self.blocks_v3_topic.clone()
        } else if self.rollup_config.is_canyon_active(timestamp) {
            self.blocks_v2_topic.clone()
        } else {
            self.blocks_v1_topic.clone()
        }
    }

    /// Encodes an execution payload and signature for the specified gossip topic.
    pub fn encode(
        &self,
        topic: IdentTopic,
        envelope: OpExecutionPayloadEnvelope,
        signature: Signature,
    ) -> Result<Vec<u8>, HandlerEncodeError> {
        let topic_hash = topic.hash();
        let version = self
            .payload_version(&topic_hash)
            .ok_or(HandlerEncodeError::UnknownTopic(topic_hash))?;
        SignedGossipPayload::from_envelope(&envelope, signature, version)?.encode_snappy()
    }

    /// Returns the payload version selected by a gossip topic.
    fn payload_version(&self, topic: &TopicHash) -> Option<GossipPayloadVersion> {
        if topic == &self.blocks_v1_topic.hash() {
            Some(GossipPayloadVersion::V1)
        } else if topic == &self.blocks_v2_topic.hash() {
            Some(GossipPayloadVersion::V2)
        } else if topic == &self.blocks_v3_topic.hash() {
            Some(GossipPayloadVersion::V3)
        } else if topic == &self.blocks_v4_topic.hash() {
            Some(GossipPayloadVersion::V4)
        } else {
            None
        }
    }
}

#[cfg(test)]
mod tests {
    use alloy_chains::Chain;
    use alloy_rpc_types_engine::{ExecutionPayloadV2, ExecutionPayloadV3};
    use op_alloy_rpc_types_engine::{
        OpExecutionPayloadEnvelope, OpExecutionPayloadV4, PayloadHash,
    };

    use crate::{v2_valid_block, v3_valid_block, v4_valid_block};

    use super::*;
    use alloy_primitives::{Address, B256, Signature};

    fn encode_with_test_signature(
        handler: &BlockHandler,
        topic: IdentTopic,
        envelope: OpExecutionPayloadEnvelope,
    ) -> (Vec<u8>, Address) {
        let signature = Signature::test_signature();
        let message = envelope.payload_hash().signature_message(10);
        let signer = signature.recover_address_from_prehash(&message).unwrap();
        let encoded = handler.encode(topic, envelope, signature).unwrap();
        (encoded, signer)
    }

    fn message(topic: IdentTopic, data: Vec<u8>) -> Message {
        Message { source: None, sequence_number: None, topic: topic.into(), data }
    }

    fn v2_envelope() -> OpExecutionPayloadEnvelope {
        OpExecutionPayloadEnvelope::V2(ExecutionPayloadV2::from_block_slow(&v2_valid_block()))
    }

    fn v3_envelope() -> OpExecutionPayloadEnvelope {
        let block = v3_valid_block();
        OpExecutionPayloadEnvelope::V3 {
            payload: ExecutionPayloadV3::from_block_slow(&block),
            parent_beacon_block_root: block.header.parent_beacon_block_root.unwrap(),
        }
    }

    fn v4_envelope() -> OpExecutionPayloadEnvelope {
        let block = v4_valid_block();
        OpExecutionPayloadEnvelope::V4 {
            payload: OpExecutionPayloadV4::from_v3_with_withdrawals_root(
                ExecutionPayloadV3::from_block_slow(&block),
                block.withdrawals_root.unwrap(),
            ),
            parent_beacon_block_root: block.header.parent_beacon_block_root.unwrap(),
        }
    }

    fn handler_with_signer(signer: Address) -> BlockHandler {
        let (_, unsafe_signer) = tokio::sync::watch::channel(signer);
        BlockHandler::new(
            RollupConfig { l2_chain_id: Chain::optimism_mainnet(), ..Default::default() },
            unsafe_signer,
        )
    }

    #[test]
    fn test_valid_decode() {
        let mut handler = handler_with_signer(Address::ZERO);
        let topic = handler.blocks_v2_topic.clone();
        let (encoded, signer) = encode_with_test_signature(&handler, topic.clone(), v2_envelope());
        handler.signer_recv = tokio::sync::watch::channel(signer).1;

        let (acceptance, payload) = handler.handle(message(topic, encoded));
        assert!(matches!(acceptance, MessageAcceptance::Accept));
        assert!(matches!(payload, Some(OpExecutionPayloadEnvelope::V2(_))));
    }

    #[test]
    fn test_invalid_decode_payload_hash() {
        let signature = Signature::test_signature();
        let wrong_message = PayloadHash(B256::ZERO).signature_message(10);
        let wrong_signer = signature.recover_address_from_prehash(&wrong_message).unwrap();
        let mut handler = handler_with_signer(wrong_signer);
        let topic = handler.blocks_v2_topic.clone();
        let encoded = handler.encode(topic.clone(), v2_envelope(), signature).unwrap();

        assert!(matches!(handler.handle(message(topic, encoded)).0, MessageAcceptance::Reject));
    }

    fn assert_version_mismatch(
        envelope: OpExecutionPayloadEnvelope,
        encoded_topic: fn(&BlockHandler) -> IdentTopic,
        received_topic: fn(&BlockHandler) -> IdentTopic,
    ) {
        let mut handler = handler_with_signer(Address::ZERO);
        let (encoded, signer) =
            encode_with_test_signature(&handler, encoded_topic(&handler), envelope);
        handler.signer_recv = tokio::sync::watch::channel(signer).1;

        assert!(matches!(
            handler.handle(message(received_topic(&handler), encoded)).0,
            MessageAcceptance::Reject
        ));
    }

    #[test]
    fn test_invalid_decode_version_mismatch() {
        assert_version_mismatch(
            v2_envelope(),
            |h| h.blocks_v2_topic.clone(),
            |h| h.blocks_v1_topic.clone(),
        );
    }

    #[test]
    fn test_invalid_decode_version_mismatch_v3_with_v2() {
        assert_version_mismatch(
            v3_envelope(),
            |h| h.blocks_v3_topic.clone(),
            |h| h.blocks_v2_topic.clone(),
        );
    }

    #[test]
    fn test_invalid_decode_version_mismatch_v2_with_v3() {
        assert_version_mismatch(
            v2_envelope(),
            |h| h.blocks_v2_topic.clone(),
            |h| h.blocks_v3_topic.clone(),
        );
    }

    #[test]
    fn test_invalid_decode_version_mismatch_v4_with_v3() {
        assert_version_mismatch(
            v4_envelope(),
            |h| h.blocks_v4_topic.clone(),
            |h| h.blocks_v3_topic.clone(),
        );
    }

    fn assert_valid_decode(
        envelope: OpExecutionPayloadEnvelope,
        topic: fn(&BlockHandler) -> IdentTopic,
    ) {
        let mut handler = handler_with_signer(Address::ZERO);
        let selected_topic = topic(&handler);
        let (encoded, signer) =
            encode_with_test_signature(&handler, selected_topic.clone(), envelope);
        handler.signer_recv = tokio::sync::watch::channel(signer).1;

        let (acceptance, payload) = handler.handle(message(selected_topic, encoded));
        assert!(matches!(acceptance, MessageAcceptance::Accept));
        assert!(payload.is_some());
    }

    #[test]
    fn test_valid_decode_v4() {
        assert_valid_decode(v4_envelope(), |h| h.blocks_v4_topic.clone());
    }

    #[test]
    fn test_valid_decode_v3() {
        assert_valid_decode(v3_envelope(), |h| h.blocks_v3_topic.clone());
    }

    #[cfg(feature = "metrics")]
    fn invalid_message_count(snapshot: metrics_util::debugging::Snapshot, reason: &str) -> u64 {
        use metrics_util::debugging::DebugValue;
        for (ckey, _unit, _desc, value) in snapshot.into_vec() {
            let key = ckey.key();
            let is_match = key.name() == crate::Metrics::INVALID_MESSAGE &&
                key.labels().any(|l| l.key() == "reason" && l.value() == reason);
            if !is_match {
                continue;
            }
            if let DebugValue::Counter(c) = value {
                return c;
            }
        }
        0
    }

    #[cfg(feature = "metrics")]
    fn zero_signer_handler() -> BlockHandler {
        let (_, unsafe_signer) = tokio::sync::watch::channel(Address::ZERO);
        BlockHandler::new(
            RollupConfig { l2_chain_id: Chain::optimism_mainnet(), ..Default::default() },
            unsafe_signer,
        )
    }

    #[cfg(feature = "metrics")]
    #[test]
    fn handle_records_invalid_message_metric_for_invalid_snappy_length() {
        use metrics_util::debugging::DebuggingRecorder;

        // Both an over-MAX_GOSSIP_SIZE header and a malformed/truncated header take the pre-decode
        // reject path and share the `invalid_snappy_length` reason.
        let oversized = snap::raw::Encoder::new()
            .compress_vec(&vec![0u8; crate::config::MAX_GOSSIP_SIZE + 1])
            .unwrap();
        let malformed = vec![0xffu8]; // truncated snappy varint header

        for data in [oversized, malformed] {
            let mut handler = zero_signer_handler();
            let message = Message {
                source: None,
                sequence_number: None,
                topic: handler.blocks_v1_topic.clone().into(),
                data,
            };

            let recorder = DebuggingRecorder::new();
            let snapshotter = recorder.snapshotter();
            let acc = metrics::with_local_recorder(&recorder, || handler.handle(message).0);

            assert!(matches!(acc, MessageAcceptance::Reject));
            assert_eq!(invalid_message_count(snapshotter.snapshot(), "invalid_snappy_length"), 1);
        }
    }

    #[cfg(feature = "metrics")]
    #[test]
    fn handle_records_invalid_message_metric_for_decode_error() {
        use metrics_util::debugging::DebuggingRecorder;

        // An authenticated V2 payload published on the V1 topic fails typed envelope decoding.
        let mut handler = zero_signer_handler();
        let (encoded, signer) =
            encode_with_test_signature(&handler, handler.blocks_v2_topic.clone(), v2_envelope());
        handler.signer_recv = tokio::sync::watch::channel(signer).1;
        let message = message(handler.blocks_v1_topic.clone(), encoded);

        let recorder = DebuggingRecorder::new();
        let snapshotter = recorder.snapshotter();
        let acc = metrics::with_local_recorder(&recorder, || handler.handle(message).0);

        assert!(matches!(acc, MessageAcceptance::Reject));
        assert_eq!(invalid_message_count(snapshotter.snapshot(), "decode_error"), 1);
    }

    #[cfg(feature = "metrics")]
    #[test]
    fn handle_records_invalid_message_metric_for_unknown_topic() {
        use metrics_util::debugging::DebuggingRecorder;

        let mut handler = zero_signer_handler();
        // Valid small snappy frame so the pre-decode bound passes; the topic is not a block topic.
        let data = snap::raw::Encoder::new().compress_vec(&[1, 2, 3, 4, 5]).unwrap();
        let message = Message {
            source: None,
            sequence_number: None,
            topic: TopicHash::from_raw("unknown"),
            data,
        };

        let recorder = DebuggingRecorder::new();
        let snapshotter = recorder.snapshotter();
        let acc = metrics::with_local_recorder(&recorder, || handler.handle(message).0);

        assert!(matches!(acc, MessageAcceptance::Reject));
        assert_eq!(invalid_message_count(snapshotter.snapshot(), "unknown_topic"), 1);
    }
}
