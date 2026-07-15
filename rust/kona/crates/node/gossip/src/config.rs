//! Gossipsub Config

use lazy_static::lazy_static;
use libp2p::gossipsub::{Config, ConfigBuilder, Message, MessageId};
use openssl::sha::sha256;
use snap::raw::Decoder;
use std::time::Duration;

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
/// # Returns
///
/// A [`ConfigBuilder`] with the default gossipsub configuration already set.
/// Call `.build()` on the returned builder to get the final [`libp2p::gossipsub::Config`].
pub fn default_config_builder() -> ConfigBuilder {
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
        .message_id_fn(compute_message_id);

    builder
}

/// Returns the default [Config] for gossipsub.
pub fn default_config() -> Config {
    default_config_builder().build().expect("default gossipsub config must be valid")
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

/// Computes the [`MessageId`] of a `gossipsub` message.
///
/// Oversized or malformed snappy frames are rejected via [`snappy_decompressed_len_within_bound`]
/// before decompression and take the invalid-snappy domain, matching op-node's `BuildMsgIdFn`.
/// This is invoked as gossipsub's `message_id_fn` on every inbound PUBLISH before signature
/// validation, so the input is unauthenticated.
fn compute_message_id(msg: &Message) -> MessageId {
    // Only attempt decompression once the header's declared length is within bound, so an oversized
    // frame never triggers a large allocation (see `snappy_decompressed_len_within_bound`).
    let decompressed = snappy_decompressed_len_within_bound(&msg.data)
        .and_then(|_| Decoder::new().decompress_vec(&msg.data).ok());

    let id = decompressed.map_or_else(
        || {
            // Oversized or undecompressable frame: take the invalid-snappy domain. Count and
            // debug-log, never warn — this runs on unauthenticated remote input.
            kona_macros::inc!(counter, crate::Metrics::MESSAGE_ID_INVALID_SNAPPY);
            debug!(target: "gossip", len = msg.data.len(), "Snappy frame failed to decompress within bound in message-id");
            let domain_invalid_snappy = [0u8; 4];
            sha256([domain_invalid_snappy.as_slice(), msg.data.as_slice()].concat().as_slice())
                [..20]
                .to_vec()
        },
        |data| {
            let domain_valid_snappy = [0x1u8, 0x0, 0x0, 0x0];
            sha256([domain_valid_snappy.as_slice(), data.as_slice()].concat().as_slice())[..20]
                .to_vec()
        },
    );

    MessageId(id)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_constructs_default_config() {
        let cfg = default_config();
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

        let id = compute_message_id(&msg);
        let hashed = sha256(&[&[0u8; 4], [1, 2, 3, 4, 5].as_slice()].concat());
        assert_eq!(id.0, hashed[..20].to_vec());
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

        let id = compute_message_id(&msg);
        let hashed = sha256(&[&[0x1, 0x0, 0x0, 0x0], [1, 2, 3, 4, 5].as_slice()].concat());
        assert_eq!(id.0, hashed[..20].to_vec());
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

        let id = compute_message_id(&msg);
        // Bomb takes the oversized-header rejection branch: hash uses invalid-snappy domain.
        let expected = sha256(&[&[0u8; 4], bomb.as_slice()].concat());
        assert_eq!(id.0, expected[..20].to_vec());
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
        let id = compute_message_id(&msg);

        // Bounded path: invalid-snappy domain over the raw (still-compressed) bytes.
        let bounded = sha256(&[[0u8; 4].as_slice(), over.as_slice()].concat());
        assert_eq!(id.0, bounded[..20].to_vec());

        // The id an unbounded implementation would produce (decompress, then valid-snappy
        // domain) must NOT be the one returned.
        let decompressed = snap::raw::Decoder::new().decompress_vec(&over).unwrap();
        let unbounded =
            sha256(&[[0x1u8, 0x0, 0x0, 0x0].as_slice(), decompressed.as_slice()].concat());
        assert_ne!(id.0, unbounded[..20].to_vec());
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
            let _ = compute_message_id(&msg);
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
            let _ = compute_message_id(&msg);
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
            let _ = compute_message_id(&msg);
        });

        assert_eq!(message_id_invalid_snappy_count(snapshotter.snapshot()), 0);
    }
}
