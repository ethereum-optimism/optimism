//! Gossipsub Config

use lazy_static::lazy_static;
use libp2p::gossipsub::{Config, ConfigBuilder, Message, MessageId};
use openssl::sha::sha256;
use snap::raw::Decoder;
use std::{sync::Arc, time::Duration};

////////////////////////////////////////////////////////////////////////////////////////////////
// GossipSub Constants
////////////////////////////////////////////////////////////////////////////////////////////////

/// The maximum gossip size.
/// Limits the total size of gossip RPC containers as well as decompressed individual messages.
pub const MAX_GOSSIP_SIZE: usize = 10 * (1 << 20);

/// The minimum gossip size.
/// Used to make sure that there is at least some data to validate the signature against.
pub const MIN_GOSSIP_SIZE: usize = 66;

/// The maximum outbound queue.
pub const MAX_OUTBOUND_QUEUE: usize = 256;

/// The maximum validate queue.
pub const MAX_VALIDATE_QUEUE: usize = 256;

/// The global validate throttle.
pub const GLOBAL_VALIDATE_THROTTLE: usize = 512;

/// The default mesh D.
pub const DEFAULT_MESH_D: usize = 8;

/// The default mesh D low.
pub const DEFAULT_MESH_DLO: usize = 6;

/// The default mesh D high.
pub const DEFAULT_MESH_DHI: usize = 12;

/// The default mesh D lazy.
pub const DEFAULT_MESH_DLAZY: usize = 6;

////////////////////////////////////////////////////////////////////////////////////////////////
// Duration Constants
////////////////////////////////////////////////////////////////////////////////////////////////

lazy_static! {
    /// The gossip heartbeat.
    pub static ref GOSSIP_HEARTBEAT: Duration = Duration::from_millis(500);

    /// The seen messages TTL.
    /// Limits the duration that message IDs are remembered for gossip deduplication purposes.
    pub static ref SEEN_MESSAGES_TTL: Duration = 130 * *GOSSIP_HEARTBEAT;

    /// The peer score inspect frequency.
    /// The frequency at which peer scores are inspected.
    pub static ref PEER_SCORE_INSPECT_FREQUENCY: Duration = 15 * Duration::from_secs(1);
}

////////////////////////////////////////////////////////////////////////////////////////////////
// Config Building
////////////////////////////////////////////////////////////////////////////////////////////////

/// Builds the default gossipsub configuration.
///
/// Notable defaults:
/// - `flood_publish`: false (call `.flood_publish(true)` on the [`ConfigBuilder`] to enable)
/// - `backoff_slack`: 1
/// - heart beat interval: 1 second
/// - peer exchange is disabled
/// - maximum byte size for gossip messages: 2048 bytes
///
/// `chain_id` is the L2 chain ID; the message-id function attaches it as the `chain_id` metric
/// label so a multi-chain process can attribute malformed-frame counts to the right chain.
///
/// # Returns
///
/// A [`ConfigBuilder`] with the default gossipsub configuration already set.
/// Call `.build()` on the returned builder to get the final [`libp2p::gossipsub::Config`].
pub fn default_config_builder(chain_id: u64) -> ConfigBuilder {
    // `message_id_fn` takes a `'static` callable, so the label is rendered once here and moved
    // into the closure rather than re-allocated on every inbound frame.
    let chain_id_label: Arc<str> = Arc::from(chain_id.to_string());
    let mut builder = ConfigBuilder::default();
    builder
        .mesh_n(DEFAULT_MESH_D)
        .mesh_n_low(DEFAULT_MESH_DLO)
        .mesh_n_high(DEFAULT_MESH_DHI)
        .gossip_lazy(DEFAULT_MESH_DLAZY)
        .heartbeat_interval(*GOSSIP_HEARTBEAT)
        .fanout_ttl(Duration::from_secs(60))
        .history_length(12)
        .history_gossip(3)
        .flood_publish(false)
        .support_floodsub()
        .max_transmit_size(MAX_GOSSIP_SIZE)
        .duplicate_cache_time(Duration::from_secs(120))
        .validation_mode(libp2p::gossipsub::ValidationMode::None)
        .validate_messages()
        .message_id_fn(move |msg| compute_message_id(msg, &chain_id_label));

    builder
}

/// Returns the default [Config] for gossipsub.
pub fn default_config(chain_id: u64) -> Config {
    default_config_builder(chain_id).build().expect("default gossipsub config must be valid")
}

/// Returns the snappy-declared decompressed length of `data` if it is within
/// [`MAX_GOSSIP_SIZE`], or `None` when the frame header is malformed or declares a larger size.
///
/// Only the snappy frame header (a varint) is decoded; the decompressed buffer is never
/// allocated. Gossip receive paths call this to reject oversized frames before running a
/// decompressor, which would otherwise pre-allocate `vec![0; decompress_len(input)]` — up to
/// `u32::MAX` (~4 GiB) — from a tiny attacker-controlled frame. These paths are reachable by
/// unauthenticated remote peers, so the bound must be enforced before decompression.
///
/// Op-node applies the same `snappy.DecodedLen(...) <= maxGossipSize` bound in both its
/// gossipsub `message_id_fn` and its topic validator (in `op-node/p2p/gossip.go`).
pub(crate) fn snappy_decompressed_len_within_bound(data: &[u8]) -> Option<usize> {
    snap::raw::decompress_len(data).ok().filter(|&n| n <= MAX_GOSSIP_SIZE)
}

/// Message-id domain separator for a frame that decompressed within [`MAX_GOSSIP_SIZE`], matching
/// op-node's `MessageDomainValidSnappy`.
const MESSAGE_DOMAIN_VALID_SNAPPY: [u8; 4] = [1, 0, 0, 0];

/// Message-id domain separator for an oversized or undecompressable frame, matching op-node's
/// `MessageDomainInvalidSnappy`.
const MESSAGE_DOMAIN_INVALID_SNAPPY: [u8; 4] = [0, 0, 0, 0];

/// Computes the [`MessageId`] of a `gossipsub` message.
///
/// Mirrors op-node's `BuildMsgIdFn`: the id is
/// `sha256(domain || uint64_le(topic.len()) || topic || payload)[..20]`, where `payload` is the
/// decompressed frame under the valid-snappy domain when it decompresses within
/// [`MAX_GOSSIP_SIZE`], otherwise the raw frame under the invalid-snappy domain. Binding the topic
/// keeps a valid frame replayed on the wrong block-version topic from colliding with the real
/// message's id, which libp2p's cross-topic duplicate cache would otherwise suppress.
///
/// Invoked as gossipsub's `message_id_fn` on every inbound PUBLISH before signature validation, so
/// the input is unauthenticated; oversized frames are bounded via
/// [`snappy_decompressed_len_within_bound`] before decompression.
fn compute_message_id(msg: &Message, _chain_id_label: &Arc<str>) -> MessageId {
    // Only attempt decompression once the header's declared length is within bound, so an oversized
    // frame never triggers a large allocation (see `snappy_decompressed_len_within_bound`).
    let decompressed = snappy_decompressed_len_within_bound(&msg.data)
        .and_then(|_| Decoder::new().decompress_vec(&msg.data).ok());

    let (domain, payload) = decompressed.as_deref().map_or_else(
        || {
            // Oversized or undecompressable frame: invalid-snappy domain. Count and debug-log,
            // never warn — this runs on unauthenticated remote input.
            kona_macros::inc!(
                counter,
                crate::Metrics::MESSAGE_ID_INVALID_SNAPPY,
                crate::Metrics::CHAIN_ID_LABEL => _chain_id_label.clone()
            );
            debug!(target: "gossip", len = msg.data.len(), "Snappy frame failed to decompress within bound in message-id");
            (MESSAGE_DOMAIN_INVALID_SNAPPY, msg.data.as_slice())
        },
        |data| (MESSAGE_DOMAIN_VALID_SNAPPY, data),
    );

    let topic = msg.topic.as_str().as_bytes();
    let topic_len = (topic.len() as u64).to_le_bytes();
    let digest =
        sha256([domain.as_slice(), topic_len.as_slice(), topic, payload].concat().as_slice());

    MessageId(digest[..20].to_vec())
}

#[cfg(test)]
mod tests {
    use super::*;

    /// Recomputes op-node's message-id construction for a topic + payload, used to pin
    /// `compute_message_id`'s exact byte layout.
    fn expected_message_id(domain: [u8; 4], topic: &str, payload: &[u8]) -> Vec<u8> {
        let topic_len = (topic.len() as u64).to_le_bytes();
        sha256(
            [domain.as_slice(), topic_len.as_slice(), topic.as_bytes(), payload]
                .concat()
                .as_slice(),
        )[..20]
            .to_vec()
    }

    #[test]
    fn test_constructs_default_config() {
        let cfg = default_config(10);
        assert_eq!(cfg.mesh_n(), DEFAULT_MESH_D);
        assert_eq!(cfg.mesh_n_low(), DEFAULT_MESH_DLO);
        assert_eq!(cfg.mesh_n_high(), DEFAULT_MESH_DHI);
    }

    #[test]
    fn test_compute_message_id_invalid_snappy() {
        let msg = Message {
            source: None,
            data: vec![1, 2, 3, 4, 5],
            sequence_number: None,
            topic: libp2p::gossipsub::TopicHash::from_raw("test"),
        };

        let id = compute_message_id(&msg, &Arc::from("10"));
        assert_eq!(
            id.0,
            expected_message_id(MESSAGE_DOMAIN_INVALID_SNAPPY, "test", &[1, 2, 3, 4, 5])
        );
    }

    #[test]
    fn test_compute_message_id_valid_snappy() {
        let compressed = snap::raw::Encoder::new().compress_vec(&[1, 2, 3, 4, 5]).unwrap();
        let msg = Message {
            source: None,
            data: compressed,
            sequence_number: None,
            topic: libp2p::gossipsub::TopicHash::from_raw("test"),
        };

        let id = compute_message_id(&msg, &Arc::from("10"));
        assert_eq!(
            id.0,
            expected_message_id(MESSAGE_DOMAIN_VALID_SNAPPY, "test", &[1, 2, 3, 4, 5])
        );
    }

    #[test]
    fn compute_message_id_binds_topic() {
        // The same payload on two different block-version topics must produce different ids, so a
        // valid frame replayed on the wrong topic cannot suppress the real message via libp2p's
        // cross-topic duplicate cache.
        let payload = vec![1u8, 2, 3, 4, 5];
        let make = |topic: &str| Message {
            source: None,
            data: payload.clone(),
            sequence_number: None,
            topic: libp2p::gossipsub::TopicHash::from_raw(topic),
        };

        let id_v1 = compute_message_id(&make("/optimism/10/0/blocks"), &Arc::from("10"));
        let id_v2 = compute_message_id(&make("/optimism/10/1/blocks"), &Arc::from("10"));
        assert_ne!(id_v1.0, id_v2.0, "message id must depend on the topic");
        assert_eq!(
            id_v1.0,
            expected_message_id(MESSAGE_DOMAIN_INVALID_SNAPPY, "/optimism/10/0/blocks", &payload)
        );
    }

    /// Golden vectors computed independently of this crate (Python `hashlib`), pinning byte-parity
    /// with op-node's `BuildMsgIdFn`: `sha256(domain || uint64_le(topic.len()) || topic ||
    /// payload)`.
    #[test]
    fn compute_message_id_matches_op_node_golden_vectors() {
        let make = |data: Vec<u8>, topic: &str| Message {
            source: None,
            data,
            sequence_number: None,
            topic: libp2p::gossipsub::TopicHash::from_raw(topic),
        };

        // Invalid snappy: raw payload [1,2,3,4,5] (fails decompression), topic "test".
        assert_eq!(
            compute_message_id(&make(vec![1, 2, 3, 4, 5], "test"), &Arc::from("10")).0,
            alloy_primitives::hex::decode("b6897dcba59347fedcd694cc0f5117093c9dc727").unwrap(),
        );

        // Valid snappy: compress([1,2,3,4,5]) decompresses to [1,2,3,4,5], topic "test".
        let valid = snap::raw::Encoder::new().compress_vec(&[1, 2, 3, 4, 5]).unwrap();
        assert_eq!(
            compute_message_id(&make(valid, "test"), &Arc::from("10")).0,
            alloy_primitives::hex::decode("adbe547b27f41294a08a09210a5e8531e83cdc16").unwrap(),
        );
    }

    /// The classic 7-byte snappy bomb declaring `u32::MAX` decompressed length takes the
    /// invalid-snappy branch rather than being decompressed. The bound is enforced by
    /// [`snappy_decompressed_len_within_bound`], which reads only the frame header — see
    /// `snappy_len_bound_rejects_oversize_without_allocating` for the allocation-free guarantee.
    #[test]
    fn compute_message_id_rejects_declared_oversize_bomb() {
        // Header: varint(u32::MAX) + literal-chunk-tag + one literal byte.
        let bomb = vec![0xFFu8, 0xFF, 0xFF, 0xFF, 0x0F, 0x00, 0x41];
        assert_eq!(
            snap::raw::decompress_len(&bomb).unwrap(),
            u32::MAX as usize,
            "sanity: bomb header must declare u32::MAX"
        );

        let msg = Message {
            source: None,
            data: bomb.clone(),
            sequence_number: None,
            topic: libp2p::gossipsub::TopicHash::from_raw("test"),
        };

        let id = compute_message_id(&msg, &Arc::from("10"));
        // Bomb takes the oversized-header rejection branch: hash uses invalid-snappy domain.
        assert_eq!(id.0, expected_message_id(MESSAGE_DOMAIN_INVALID_SNAPPY, "test", &bomb));
    }

    /// Proves the bound actually gates decompression (distinguishing this implementation from
    /// an unbounded one): a frame that *validly* decompresses to more than [`MAX_GOSSIP_SIZE`]
    /// takes the invalid-snappy domain, not the valid-snappy domain an unbounded implementation
    /// would produce after decompressing it.
    #[test]
    fn compute_message_id_rejects_oversize_frame_via_invalid_snappy() {
        let over = snap::raw::Encoder::new().compress_vec(&vec![0u8; MAX_GOSSIP_SIZE + 1]).unwrap();
        // The compressed frame is tiny, so the gossipsub transmit-size limit does not stop it.
        assert!(over.len() <= MAX_GOSSIP_SIZE);

        let msg = Message {
            source: None,
            data: over.clone(),
            sequence_number: None,
            topic: libp2p::gossipsub::TopicHash::from_raw("test"),
        };
        let id = compute_message_id(&msg, &Arc::from("10"));

        // Bounded path: invalid-snappy domain over the raw (still-compressed) bytes.
        assert_eq!(id.0, expected_message_id(MESSAGE_DOMAIN_INVALID_SNAPPY, "test", &over));

        // The id an unbounded implementation would produce (decompress, then valid-snappy
        // domain) must NOT be the one returned.
        let decompressed = snap::raw::Decoder::new().decompress_vec(&over).unwrap();
        assert_ne!(id.0, expected_message_id(MESSAGE_DOMAIN_VALID_SNAPPY, "test", &decompressed));
    }

    #[test]
    fn snappy_len_bound_rejects_oversize_without_allocating() {
        // A 7-byte frame declaring `u32::MAX` decompressed length. `decompress_len` reads only
        // the varint header, so the declared buffer is never allocated to evaluate the bound.
        let bomb = vec![0xFFu8, 0xFF, 0xFF, 0xFF, 0x0F, 0x00, 0x41];
        assert_eq!(snap::raw::decompress_len(&bomb).unwrap(), u32::MAX as usize);
        assert_eq!(snappy_decompressed_len_within_bound(&bomb), None);

        // A genuine frame that decompresses to one byte over the limit is also rejected.
        let over = snap::raw::Encoder::new().compress_vec(&vec![0u8; MAX_GOSSIP_SIZE + 1]).unwrap();
        assert_eq!(snappy_decompressed_len_within_bound(&over), None);

        // A truncated varint header is malformed and rejected.
        assert_eq!(snappy_decompressed_len_within_bound(&[0xFF]), None);

        // Empty input declares a zero-length payload, which is within bound (downstream
        // decoding still rejects it); it must not error out here.
        assert_eq!(snappy_decompressed_len_within_bound(&[]), Some(0));
    }

    #[test]
    fn snappy_len_bound_accepts_up_to_and_including_max() {
        // Exactly `MAX_GOSSIP_SIZE` is within bound (inclusive), matching op-node.
        let at_max = snap::raw::Encoder::new().compress_vec(&vec![0u8; MAX_GOSSIP_SIZE]).unwrap();
        assert_eq!(snappy_decompressed_len_within_bound(&at_max), Some(MAX_GOSSIP_SIZE));

        let small = snap::raw::Encoder::new().compress_vec(&[1, 2, 3, 4, 5]).unwrap();
        assert_eq!(snappy_decompressed_len_within_bound(&small), Some(5));
    }

    #[cfg(feature = "metrics")]
    fn message_id_invalid_snappy_count(snapshot: metrics_util::debugging::Snapshot) -> u64 {
        use metrics_util::debugging::DebugValue;
        for (ckey, _unit, _desc, value) in snapshot.into_vec() {
            if ckey.key().name() != crate::Metrics::MESSAGE_ID_INVALID_SNAPPY {
                continue;
            }
            if let DebugValue::Counter(c) = value {
                return c;
            }
        }
        0
    }

    #[cfg(feature = "metrics")]
    #[test]
    fn compute_message_id_records_invalid_snappy_on_decompress_failure() {
        use metrics_util::debugging::DebuggingRecorder;

        // Declares a 1-byte payload but the body is corrupt, so `decompress_vec` fails.
        let msg = Message {
            source: None,
            data: vec![1, 2, 3, 4, 5],
            sequence_number: None,
            topic: libp2p::gossipsub::TopicHash::from_raw("test"),
        };

        let recorder = DebuggingRecorder::new();
        let snapshotter = recorder.snapshotter();
        metrics::with_local_recorder(&recorder, || {
            let _ = compute_message_id(&msg, &Arc::from("10"));
        });

        assert_eq!(message_id_invalid_snappy_count(snapshotter.snapshot()), 1);
    }

    #[cfg(feature = "metrics")]
    #[test]
    fn compute_message_id_records_invalid_snappy_on_oversize_frame() {
        use metrics_util::debugging::DebuggingRecorder;

        // A tiny frame that validly declares a decompressed size over the bound.
        let over = snap::raw::Encoder::new().compress_vec(&vec![0u8; MAX_GOSSIP_SIZE + 1]).unwrap();
        let msg = Message {
            source: None,
            data: over,
            sequence_number: None,
            topic: libp2p::gossipsub::TopicHash::from_raw("test"),
        };

        let recorder = DebuggingRecorder::new();
        let snapshotter = recorder.snapshotter();
        metrics::with_local_recorder(&recorder, || {
            let _ = compute_message_id(&msg, &Arc::from("10"));
        });

        assert_eq!(message_id_invalid_snappy_count(snapshotter.snapshot()), 1);
    }

    #[cfg(feature = "metrics")]
    #[test]
    fn compute_message_id_does_not_record_invalid_snappy_on_valid_frame() {
        use metrics_util::debugging::DebuggingRecorder;

        let valid = snap::raw::Encoder::new().compress_vec(&[1, 2, 3, 4, 5]).unwrap();
        let msg = Message {
            source: None,
            data: valid,
            sequence_number: None,
            topic: libp2p::gossipsub::TopicHash::from_raw("test"),
        };

        let recorder = DebuggingRecorder::new();
        let snapshotter = recorder.snapshotter();
        metrics::with_local_recorder(&recorder, || {
            let _ = compute_message_id(&msg, &Arc::from("10"));
        });

        assert_eq!(message_id_invalid_snappy_count(snapshotter.snapshot()), 0);
    }
}
