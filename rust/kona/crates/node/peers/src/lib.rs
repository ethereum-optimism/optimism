#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/op-rs/kona/main/assets/square.png",
    html_favicon_url = "https://raw.githubusercontent.com/op-rs/kona/main/assets/favicon.ico",
    issue_tracker_base_url = "https://github.com/op-rs/kona/issues/"
)]
#![cfg_attr(not(test), warn(unused_crate_dependencies))]
#![cfg_attr(docsrs, feature(doc_cfg, doc_auto_cfg))]

#[macro_use]
extern crate tracing;

/// Alias for a peer identifier.
///
/// This is the most primitive secp256k1 public key identifier for a given peer.
pub type PeerId = alloy_primitives::B512;

mod nodes;
pub use nodes::{BootNodes, OP_RAW_BOOTNODES, OP_RAW_TESTNET_BOOTNODES};

mod store;
pub use store::{BootStore, BootStoreFile};

mod score;
pub use score::PeerScoreLevel;

mod enr;
pub use enr::{EnrValidation, OpStackEnr, OpStackEnrError};

mod any;
pub use any::{AnyNode, DialOptsError};

mod boot;
pub use boot::BootNode;

mod record;
pub use record::{NodeRecord, NodeRecordParseError};

mod utils;
pub use utils::{
    PeerIdConversionError, enr_to_multiaddr, local_id_to_p2p_id, peer_id_to_secp256k1_pubkey,
};

mod monitoring;
pub use monitoring::PeerMonitoring;
