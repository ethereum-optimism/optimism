//! Block Handler

use crate::{HandlerEncodeError, config::snappy_decompressed_len_within_bound};
use alloy_primitives::{Address, B256};
use kona_genesis::RollupConfig;
use libp2p::gossipsub::{IdentTopic, Message, MessageAcceptance, TopicHash};
use op_alloy_rpc_types_engine::OpNetworkPayloadEnvelope;
use std::collections::{BTreeMap, HashSet};
use tokio::sync::watch::Receiver;

/// This trait defines the functionality required to process incoming messages
/// and determine their acceptance within the network.
///
/// Implementors of this trait can specify how messages are handled and which
/// topics they are interested in.
pub trait Handler: Send {
    /// Manages validation and further processing of messages
    /// This is a stateful method, because the handler needs to keep track of seen hashes.
    fn handle(&mut self, msg: Message) -> (MessageAcceptance, Option<OpNetworkPayloadEnvelope>);

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
}

impl Handler for BlockHandler {
    /// Checks validity of a [`OpNetworkPayloadEnvelope`] received over P2P gossip.
    /// If valid, sends the [`OpNetworkPayloadEnvelope`] to the block update channel.
    fn handle(&mut self, msg: Message) -> (MessageAcceptance, Option<OpNetworkPayloadEnvelope>) {
        // Reject frames whose snappy header declares a decompressed size over MAX_GOSSIP_SIZE
        // before decoding. `OpNetworkPayloadEnvelope::decode_v*` otherwise pre-allocates a
        // buffer of the declared length (up to ~4 GiB) from a tiny frame. Mirrors op-node's
        // gossip topic validator, which rejects `outLen > maxGossipSize` before decompressing.
        if snappy_decompressed_len_within_bound(&msg.data).is_none() {
            // Count for alerting and debug-log for investigation, but never warn: unauthenticated
            // remote input must not be able to spam warnings just by being invalid.
            kona_macros::inc!(counter, crate::Metrics::INVALID_MESSAGE, "reason" => "oversized_snappy");
            debug!(target: "gossip", len = msg.data.len(), "Rejecting oversized snappy frame before decode");
            return (MessageAcceptance::Reject, None);
        }

        let decoded = if msg.topic == self.blocks_v1_topic.hash() {
            OpNetworkPayloadEnvelope::decode_v1(&msg.data)
        } else if msg.topic == self.blocks_v2_topic.hash() {
            OpNetworkPayloadEnvelope::decode_v2(&msg.data)
        } else if msg.topic == self.blocks_v3_topic.hash() {
            OpNetworkPayloadEnvelope::decode_v3(&msg.data)
        } else if msg.topic == self.blocks_v4_topic.hash() {
            OpNetworkPayloadEnvelope::decode_v4(&msg.data)
        } else {
            // Unreachable in practice (the driver only dispatches known block topics), but count
            // and debug-log it rather than warn if that ever changes.
            kona_macros::inc!(counter, crate::Metrics::INVALID_MESSAGE, "reason" => "unknown_topic");
            debug!(target: "gossip", topic = ?msg.topic, "Received block with unknown topic");
            return (MessageAcceptance::Reject, None);
        };

        match decoded {
            Ok(envelope) => match self.block_valid(&envelope) {
                Ok(()) => (MessageAcceptance::Accept, Some(envelope)),
                Err(err) => {
                    // Already metered by BLOCK_VALIDATION_FAILED; debug-log for investigation
                    // without letting invalid remote blocks spam warnings.
                    debug!(target: "gossip", ?err, hash = ?envelope.payload_hash, "Received invalid block");
                    (err.into(), None)
                }
            },
            Err(err) => {
                // Undecodable payload from unauthenticated input: count and debug-log, don't warn.
                kona_macros::inc!(counter, crate::Metrics::INVALID_MESSAGE, "reason" => "decode_error");
                debug!(target: "gossip", ?err, "Failed to decode block");
                (MessageAcceptance::Reject, None)
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

    /// Encodes a [`OpNetworkPayloadEnvelope`] into a byte array
    /// based on the specified topic.
    pub fn encode(
        &self,
        topic: IdentTopic,
        envelope: OpNetworkPayloadEnvelope,
    ) -> Result<Vec<u8>, HandlerEncodeError> {
        let encoded = match topic.hash() {
            hash if hash == self.blocks_v1_topic.hash() => envelope.encode_v1()?,
            hash if hash == self.blocks_v2_topic.hash() => envelope.encode_v2()?,
            hash if hash == self.blocks_v3_topic.hash() => envelope.encode_v3()?,
            hash if hash == self.blocks_v4_topic.hash() => envelope.encode_v4()?,
            hash => return Err(HandlerEncodeError::UnknownTopic(hash)),
        };
        Ok(encoded)
    }
}

#[cfg(test)]
mod tests {
    use alloy_chains::Chain;
    use alloy_rpc_types_engine::{ExecutionPayloadV2, ExecutionPayloadV3};
    use op_alloy_rpc_types_engine::{OpExecutionPayload, OpExecutionPayloadV4, PayloadHash};

    use crate::{v2_valid_block, v3_valid_block, v4_valid_block};

    use super::*;
    use alloy_primitives::{B256, Signature};

    #[test]
    fn test_valid_decode() {
        let block = v2_valid_block();

        let v2 = ExecutionPayloadV2::from_block_slow(&block);

        let payload = OpExecutionPayload::V2(v2);
        let envelope = OpNetworkPayloadEnvelope {
            payload,
            signature: Signature::test_signature(),
            payload_hash: PayloadHash(B256::ZERO),
            parent_beacon_block_root: None,
        };

        let msg = envelope.payload_hash.signature_message(10);
        let signer = envelope.signature.recover_address_from_prehash(&msg).unwrap();
        let (_, unsafe_signer) = tokio::sync::watch::channel(signer);
        let mut handler = BlockHandler::new(
            RollupConfig { l2_chain_id: Chain::optimism_mainnet(), ..Default::default() },
            unsafe_signer,
        );

        // TRICK: Since the decode method recomputes the payload hash, we need to change the unsafe
        // signer in the handler to ensure that the payload won't be rejected for invalid
        // signature.
        let encoded = handler.encode(handler.blocks_v2_topic.clone(), envelope).unwrap();
        let decoded = OpNetworkPayloadEnvelope::decode_v2(&encoded).unwrap();

        let msg = decoded.payload_hash.signature_message(10);
        let signer = decoded.signature.recover_address_from_prehash(&msg).unwrap();
        let (_, unsafe_signer) = tokio::sync::watch::channel(signer);
        handler.signer_recv = unsafe_signer;

        // Let's try to encode a message.
        let message = Message {
            source: None,
            sequence_number: None,
            topic: handler.blocks_v2_topic.clone().into(),
            data: encoded,
        };

        assert!(matches!(handler.handle(message).0, MessageAcceptance::Accept));
    }

    /// This payload has a wrong hash so the signature won't be valid.
    #[test]
    fn test_invalid_decode_payload_hash() {
        let block = v2_valid_block();

        let v2 = ExecutionPayloadV2::from_block_slow(&block);

        let payload = OpExecutionPayload::V2(v2);
        let envelope = OpNetworkPayloadEnvelope {
            payload,
            signature: Signature::test_signature(),
            payload_hash: PayloadHash(B256::ZERO),
            parent_beacon_block_root: None,
        };

        let msg = envelope.payload_hash.signature_message(10);
        let signer = envelope.signature.recover_address_from_prehash(&msg).unwrap();
        let (_, unsafe_signer) = tokio::sync::watch::channel(signer);
        let mut handler = BlockHandler::new(
            RollupConfig { l2_chain_id: Chain::optimism_mainnet(), ..Default::default() },
            unsafe_signer,
        );

        // Let's try to encode a message.
        let message = Message {
            source: None,
            sequence_number: None,
            topic: handler.blocks_v2_topic.clone().into(),
            data: handler.encode(handler.blocks_v2_topic.clone(), envelope).unwrap(),
        };

        assert!(matches!(handler.handle(message).0, MessageAcceptance::Reject));
    }

    /// The message contains a wrong version so the payload won't be properly decoded.
    #[test]
    fn test_invalid_decode_version_mismatch() {
        let block = v2_valid_block();

        let v2 = ExecutionPayloadV2::from_block_slow(&block);

        let payload = OpExecutionPayload::V2(v2);
        let envelope = OpNetworkPayloadEnvelope {
            payload,
            signature: Signature::test_signature(),
            payload_hash: PayloadHash(B256::ZERO),
            parent_beacon_block_root: None,
        };

        let msg = envelope.payload_hash.signature_message(10);
        let signer = envelope.signature.recover_address_from_prehash(&msg).unwrap();
        let (_, unsafe_signer) = tokio::sync::watch::channel(signer);
        let mut handler = BlockHandler::new(
            RollupConfig { l2_chain_id: Chain::optimism_mainnet(), ..Default::default() },
            unsafe_signer,
        );

        let encoded = handler.encode(handler.blocks_v2_topic.clone(), envelope).unwrap();

        // Let's try to encode a message.
        let message = Message {
            source: None,
            sequence_number: None,
            // Version mismatch!
            topic: handler.blocks_v1_topic.clone().into(),
            data: encoded,
        };

        assert!(matches!(handler.handle(message).0, MessageAcceptance::Reject));
    }

    /// The message contains a wrong version so the payload won't be properly decoded.
    #[test]
    fn test_invalid_decode_version_mismatch_v3_with_v2() {
        let block = v3_valid_block();

        let v3 = ExecutionPayloadV3::from_block_slow(&block);

        let payload = OpExecutionPayload::V3(v3);
        let envelope = OpNetworkPayloadEnvelope {
            payload,
            signature: Signature::test_signature(),
            payload_hash: PayloadHash(B256::ZERO),
            parent_beacon_block_root: Some(
                block.header.parent_beacon_block_root.unwrap_or_default(),
            ),
        };

        let msg = envelope.payload_hash.signature_message(10);
        let signer = envelope.signature.recover_address_from_prehash(&msg).unwrap();
        let (_, unsafe_signer) = tokio::sync::watch::channel(signer);
        let mut handler = BlockHandler::new(
            RollupConfig { l2_chain_id: Chain::optimism_mainnet(), ..Default::default() },
            unsafe_signer,
        );

        let encoded = handler.encode(handler.blocks_v3_topic.clone(), envelope).unwrap();

        // Let's try to encode a message.
        let message = Message {
            source: None,
            sequence_number: None,
            // Version mismatch!
            topic: handler.blocks_v2_topic.clone().into(),
            data: encoded,
        };

        assert!(matches!(handler.handle(message).0, MessageAcceptance::Reject));
    }

    /// The message contains a wrong version so the payload won't be properly decoded.
    #[test]
    fn test_invalid_decode_version_mismatch_v2_with_v3() {
        let block = v2_valid_block();

        let v2 = ExecutionPayloadV2::from_block_slow(&block);

        let payload = OpExecutionPayload::V2(v2);
        let envelope = OpNetworkPayloadEnvelope {
            payload,
            signature: Signature::test_signature(),
            payload_hash: PayloadHash(B256::ZERO),
            parent_beacon_block_root: Some(
                block.header.parent_beacon_block_root.unwrap_or_default(),
            ),
        };

        let msg = envelope.payload_hash.signature_message(10);
        let signer = envelope.signature.recover_address_from_prehash(&msg).unwrap();
        let (_, unsafe_signer) = tokio::sync::watch::channel(signer);
        let mut handler = BlockHandler::new(
            RollupConfig { l2_chain_id: Chain::optimism_mainnet(), ..Default::default() },
            unsafe_signer,
        );

        let encoded = handler.encode(handler.blocks_v2_topic.clone(), envelope).unwrap();

        // Let's try to encode a message.
        let message = Message {
            source: None,
            sequence_number: None,
            // Version mismatch!
            topic: handler.blocks_v3_topic.clone().into(),
            data: encoded,
        };

        assert!(matches!(handler.handle(message).0, MessageAcceptance::Reject));
    }

    /// The message contains a wrong version so the payload won't be properly decoded.
    #[test]
    fn test_invalid_decode_version_mismatch_v4_with_v3() {
        let block = v4_valid_block();

        let v3 = ExecutionPayloadV3::from_block_slow(&block);
        let v4 = OpExecutionPayloadV4::from_v3_with_withdrawals_root(
            v3,
            block.withdrawals_root.unwrap(),
        );

        let payload = OpExecutionPayload::V4(v4);
        let envelope = OpNetworkPayloadEnvelope {
            payload,
            signature: Signature::test_signature(),
            payload_hash: PayloadHash(B256::ZERO),
            parent_beacon_block_root: Some(
                block.header.parent_beacon_block_root.unwrap_or_default(),
            ),
        };

        let msg = envelope.payload_hash.signature_message(10);
        let signer = envelope.signature.recover_address_from_prehash(&msg).unwrap();
        let (_, unsafe_signer) = tokio::sync::watch::channel(signer);
        let mut handler = BlockHandler::new(
            RollupConfig { l2_chain_id: Chain::optimism_mainnet(), ..Default::default() },
            unsafe_signer,
        );

        let encoded = handler.encode(handler.blocks_v4_topic.clone(), envelope).unwrap();

        // Let's try to encode a message.
        let message = Message {
            source: None,
            sequence_number: None,
            // Version mismatch!
            topic: handler.blocks_v3_topic.clone().into(),
            data: encoded,
        };

        assert!(matches!(handler.handle(message).0, MessageAcceptance::Reject));
    }

    #[test]
    fn test_valid_decode_v4() {
        let block = v4_valid_block();

        let v3 = ExecutionPayloadV3::from_block_slow(&block);
        let v4 = OpExecutionPayloadV4::from_v3_with_withdrawals_root(
            v3,
            block.withdrawals_root.unwrap(),
        );

        let payload = OpExecutionPayload::V4(v4);
        let envelope = OpNetworkPayloadEnvelope {
            payload,
            signature: Signature::test_signature(),
            payload_hash: PayloadHash(B256::ZERO),
            parent_beacon_block_root: Some(
                block.header.parent_beacon_block_root.unwrap_or_default(),
            ),
        };

        let msg = envelope.payload_hash.signature_message(10);
        let signer = envelope.signature.recover_address_from_prehash(&msg).unwrap();
        let (_, unsafe_signer) = tokio::sync::watch::channel(signer);
        let mut handler = BlockHandler::new(
            RollupConfig { l2_chain_id: Chain::optimism_mainnet(), ..Default::default() },
            unsafe_signer,
        );

        // TRICK: Since the decode method recomputes the payload hash, we need to change the unsafe
        // signer in the handler to ensure that the payload won't be rejected for invalid
        // signature.
        let encoded = handler.encode(handler.blocks_v4_topic.clone(), envelope).unwrap();
        let decoded = OpNetworkPayloadEnvelope::decode_v4(&encoded).unwrap();

        let msg = decoded.payload_hash.signature_message(10);
        let signer = decoded.signature.recover_address_from_prehash(&msg).unwrap();
        let (_, unsafe_signer) = tokio::sync::watch::channel(signer);
        handler.signer_recv = unsafe_signer;

        // Let's try to encode a message.
        let message = Message {
            source: None,
            sequence_number: None,
            topic: handler.blocks_v4_topic.clone().into(),
            data: encoded,
        };

        assert!(matches!(handler.handle(message).0, MessageAcceptance::Accept));
    }

    #[test]
    fn test_valid_decode_v3() {
        let block = v3_valid_block();

        let v3 = ExecutionPayloadV3::from_block_slow(&block);

        let payload = OpExecutionPayload::V3(v3);
        let envelope = OpNetworkPayloadEnvelope {
            payload,
            signature: Signature::test_signature(),
            payload_hash: PayloadHash(B256::ZERO),
            parent_beacon_block_root: Some(
                block.header.parent_beacon_block_root.unwrap_or_default(),
            ),
        };

        let msg = envelope.payload_hash.signature_message(10);
        let signer = envelope.signature.recover_address_from_prehash(&msg).unwrap();
        let (_, unsafe_signer) = tokio::sync::watch::channel(signer);
        let mut handler = BlockHandler::new(
            RollupConfig { l2_chain_id: Chain::optimism_mainnet(), ..Default::default() },
            unsafe_signer,
        );

        // TRICK: Since the decode method recomputes the payload hash, we need to change the unsafe
        // signer in the handler to ensure that the payload won't be rejected for invalid
        // signature.
        let encoded = handler.encode(handler.blocks_v3_topic.clone(), envelope).unwrap();
        let decoded = OpNetworkPayloadEnvelope::decode_v3(&encoded).unwrap();

        let msg = decoded.payload_hash.signature_message(10);
        let signer = decoded.signature.recover_address_from_prehash(&msg).unwrap();
        let (_, unsafe_signer) = tokio::sync::watch::channel(signer);
        handler.signer_recv = unsafe_signer;

        // Let's try to encode a message.
        let message = Message {
            source: None,
            sequence_number: None,
            topic: handler.blocks_v3_topic.clone().into(),
            data: encoded,
        };

        assert!(matches!(handler.handle(message).0, MessageAcceptance::Accept));
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
    fn handle_records_invalid_message_metric_for_oversized_snappy() {
        use metrics_util::debugging::DebuggingRecorder;

        let mut handler = zero_signer_handler();
        // Tiny frame whose snappy header declares a decompressed size over MAX_GOSSIP_SIZE.
        let over = snap::raw::Encoder::new()
            .compress_vec(&vec![0u8; crate::config::MAX_GOSSIP_SIZE + 1])
            .unwrap();
        let message = Message {
            source: None,
            sequence_number: None,
            topic: handler.blocks_v1_topic.clone().into(),
            data: over,
        };

        let recorder = DebuggingRecorder::new();
        let snapshotter = recorder.snapshotter();
        let acc = metrics::with_local_recorder(&recorder, || handler.handle(message).0);

        assert!(matches!(acc, MessageAcceptance::Reject));
        assert_eq!(invalid_message_count(snapshotter.snapshot(), "oversized_snappy"), 1);
    }

    #[cfg(feature = "metrics")]
    #[test]
    fn handle_records_invalid_message_metric_for_decode_error() {
        use metrics_util::debugging::DebuggingRecorder;

        // A v2-encoded payload published on the v1 topic fails `decode_v1` before validation.
        let v2 = ExecutionPayloadV2::from_block_slow(&v2_valid_block());
        let envelope = OpNetworkPayloadEnvelope {
            payload: OpExecutionPayload::V2(v2),
            signature: Signature::test_signature(),
            payload_hash: PayloadHash(B256::ZERO),
            parent_beacon_block_root: None,
        };
        let mut handler = zero_signer_handler();
        let encoded = handler.encode(handler.blocks_v2_topic.clone(), envelope).unwrap();
        let message = Message {
            source: None,
            sequence_number: None,
            topic: handler.blocks_v1_topic.clone().into(),
            data: encoded,
        };

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
