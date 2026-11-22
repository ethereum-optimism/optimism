//! Contains a concrete implementation of the [`KeyValueStore`] trait that stores data on disk,
//! using the [`InteropHost`] config.

use super::InteropHost;
use crate::{KeyValueStore, Result};
use alloy_primitives::{B256, keccak256};
use kona_interop::DependencySet;
use kona_preimage::{
    DEPENDENCY_SET_KEY, L1_CONFIG_KEY, L1_HEAD_KEY, L2_CLAIM_BLOCK_NUMBER_KEY, L2_CLAIM_KEY,
    L2_OUTPUT_ROOT_KEY, L2_ROLLUP_CONFIG_KEY, PreimageKey,
};
use tracing::error;

/// A simple, synchronous key-value store that returns data from a [`InteropHost`] config.
#[derive(Debug)]
pub struct InteropLocalInputs {
    cfg: InteropHost,
}

impl InteropLocalInputs {
    /// Create a new [`InteropLocalInputs`] with the given [`InteropHost`] config.
    pub const fn new(cfg: InteropHost) -> Self {
        Self { cfg }
    }
}

impl KeyValueStore for InteropLocalInputs {
    fn get(&self, key: B256) -> Option<Vec<u8>> {
        let preimage_key = PreimageKey::try_from(*key).ok()?;
        match preimage_key.key_value() {
            L1_HEAD_KEY => Some(self.cfg.l1_head.to_vec()),
            L2_OUTPUT_ROOT_KEY => Some(keccak256(self.cfg.agreed_l2_pre_state.as_ref()).to_vec()),
            L2_CLAIM_KEY => Some(self.cfg.claimed_l2_post_state.to_vec()),
            L2_CLAIM_BLOCK_NUMBER_KEY => Some(self.cfg.claimed_l2_timestamp.to_be_bytes().to_vec()),
            L2_ROLLUP_CONFIG_KEY => {
                let rollup_configs = self.cfg.read_rollup_configs()?.ok()?;
                serde_json::to_vec(&rollup_configs).ok()
            }
            L1_CONFIG_KEY => {
                let l1_config = self.cfg.read_l1_config().ok()?;
                serde_json::to_vec(&l1_config).ok()
            }
            DEPENDENCY_SET_KEY => {
                // A dependency set that fails to parse is not replaced with an empty one, which
                // would make every executing message invalid; `start_server` refuses to start.
                let dependency_set = match self.cfg.read_dependency_set() {
                    Some(dependency_set) => dependency_set
                        .inspect_err(|e| error!(target: "host", "dependency set unavailable: {e}"))
                        .ok()?,
                    None => DependencySet {
                        dependencies: Default::default(),
                        override_message_expiry_window: None,
                    },
                };
                serde_json::to_vec(&dependency_set).ok()
            }
            _ => None,
        }
    }

    fn set(&mut self, _: B256, _: Vec<u8>) -> Result<()> {
        unreachable!("LocalKeyValueStore is read-only")
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use kona_interop::MESSAGE_EXPIRY_WINDOW;
    use std::path::PathBuf;

    fn dependency_set_preimage(dependency_set_path: Option<PathBuf>) -> Option<Vec<u8>> {
        let inputs =
            InteropLocalInputs::new(InteropHost { dependency_set_path, ..Default::default() });
        let key = PreimageKey::new_local(DEPENDENCY_SET_KEY.to());
        inputs.get(B256::from(key))
    }

    #[test]
    fn test_dependency_set_preimage_no_path_is_empty_set() {
        let preimage = dependency_set_preimage(None).expect("serves an empty dependency set");
        let dependency_set: DependencySet = serde_json::from_slice(&preimage).unwrap();
        assert!(dependency_set.dependencies.is_empty());
    }

    #[test]
    fn test_dependency_set_preimage_unparsable_file_is_not_served() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("depset.json");
        std::fs::write(
            &path,
            format!(
                r#"{{"dependencies":{{}},"overrideMessageExpiryWindow":{}}}"#,
                MESSAGE_EXPIRY_WINDOW + 1
            ),
        )
        .unwrap();
        assert_eq!(dependency_set_preimage(Some(path)), None);
    }
}
