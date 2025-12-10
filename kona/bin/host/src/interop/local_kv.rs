//! Contains a concrete implementation of the [KeyValueStore] trait that stores data on disk,
//! using the [InteropHost] config.

use super::InteropHost;
use crate::KeyValueStore;
use alloy_primitives::{B256, keccak256};
use anyhow::Result;
use kona_preimage::PreimageKey;
use kona_proof_interop::boot::{
    L1_CONFIG_KEY, L1_HEAD_KEY, L2_AGREED_PRE_STATE_KEY, L2_CLAIMED_POST_STATE_KEY,
    L2_CLAIMED_TIMESTAMP_KEY, L2_ROLLUP_CONFIG_KEY,
};

/// A simple, synchronous key-value store that returns data from a [InteropHost] config.
#[derive(Debug)]
pub struct InteropLocalInputs {
    cfg: InteropHost,
}

impl InteropLocalInputs {
    /// Create a new [InteropLocalInputs] with the given [InteropHost] config.
    pub const fn new(cfg: InteropHost) -> Self {
        Self { cfg }
    }
}

impl KeyValueStore for InteropLocalInputs {
    fn get(&self, key: B256) -> Option<Vec<u8>> {
        let preimage_key = PreimageKey::try_from(*key).ok()?;
        match preimage_key.key_value() {
            L1_HEAD_KEY => Some(self.cfg.l1_head.to_vec()),
            L2_AGREED_PRE_STATE_KEY => {
                Some(keccak256(self.cfg.agreed_l2_pre_state.as_ref()).to_vec())
            }
            L2_CLAIMED_POST_STATE_KEY => Some(self.cfg.claimed_l2_post_state.to_vec()),
            L2_CLAIMED_TIMESTAMP_KEY => Some(self.cfg.claimed_l2_timestamp.to_be_bytes().to_vec()),
            L2_ROLLUP_CONFIG_KEY => {
                let rollup_configs = self.cfg.read_rollup_configs()?.ok()?;
                serde_json::to_vec(&rollup_configs).ok()
            }
            L1_CONFIG_KEY => {
                let l1_configs = self.cfg.read_l1_configs()?.ok()?;
                serde_json::to_vec(&l1_configs).ok()
            }
            _ => None,
        }
    }

    fn set(&mut self, _: B256, _: Vec<u8>) -> Result<()> {
        unreachable!("LocalKeyValueStore is read-only")
    }
}
