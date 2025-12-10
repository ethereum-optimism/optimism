//! This module contains the prologue phase of the client program, pulling in the boot information
//! through the `PreimageOracle` ABI as local keys.

use crate::{HintType, INVALID_TRANSITION, INVALID_TRANSITION_HASH, PreState};
use alloc::{string::ToString, vec::Vec};
use alloy_primitives::{B256, Bytes, U256};
use alloy_rlp::Decodable;
use kona_genesis::{L1ChainConfig, RollupConfig};
use kona_preimage::{
    CommsClient, HintWriterClient, PreimageKey, PreimageKeyType, PreimageOracleClient,
    errors::PreimageOracleError,
};
use kona_proof::errors::OracleProviderError;
use kona_registry::{HashMap, L1_CONFIGS, ROLLUP_CONFIGS};
use serde::{Deserialize, Serialize};
use thiserror::Error;
use tracing::warn;

/// The local key ident for the L1 head hash.
pub const L1_HEAD_KEY: U256 = U256::from_be_slice(&[1]);

/// The local key ident for the agreed upon L2 pre-state claim.
pub const L2_AGREED_PRE_STATE_KEY: U256 = U256::from_be_slice(&[2]);

/// The local key ident for the L2 post-state claim.
pub const L2_CLAIMED_POST_STATE_KEY: U256 = U256::from_be_slice(&[3]);

/// The local key ident for the L2 claim timestamp.
pub const L2_CLAIMED_TIMESTAMP_KEY: U256 = U256::from_be_slice(&[4]);

/// The local key ident for the L2 rollup config.
pub const L2_ROLLUP_CONFIG_KEY: U256 = U256::from_be_slice(&[6]);

/// The local key ident for the l1 config.
pub const L1_CONFIG_KEY: U256 = U256::from_be_slice(&[7]);

/// The boot information for the interop client program.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct BootInfo {
    /// The L1 head hash containing the safe L2 chain data that may reproduce the post-state claim.
    pub l1_head: B256,
    /// The agreed upon superchain pre-state commitment.
    pub agreed_pre_state_commitment: B256,
    /// The agreed upon superchain pre-state.
    pub agreed_pre_state: PreState,
    /// The claimed (disputed) superchain post-state commitment.
    pub claimed_post_state: B256,
    /// The L2 claim timestamp.
    pub claimed_l2_timestamp: u64,
    /// The rollup config for the L2 chain.
    pub rollup_configs: HashMap<u64, RollupConfig>,
    /// The L1 config for the L2 chain.
    pub l1_config: L1ChainConfig,
}

impl BootInfo {
    /// Load the boot information from the preimage oracle.
    ///
    /// ## Takes
    /// - `oracle`: The preimage oracle reader.
    ///
    /// ## Returns
    /// - `Ok(BootInfo)`: The boot information.
    /// - `Err(_)`: Failed to load the boot information.
    pub async fn load<O>(oracle: &O) -> Result<Self, BootstrapError>
    where
        O: PreimageOracleClient + HintWriterClient + Clone + Send,
    {
        let mut l1_head: B256 = B256::ZERO;
        oracle
            .get_exact(PreimageKey::new_local(L1_HEAD_KEY.to()), l1_head.as_mut())
            .await
            .map_err(OracleProviderError::Preimage)?;

        let mut l2_pre: B256 = B256::ZERO;
        oracle
            .get_exact(PreimageKey::new_local(L2_AGREED_PRE_STATE_KEY.to()), l2_pre.as_mut())
            .await
            .map_err(OracleProviderError::Preimage)?;

        let mut l2_post: B256 = B256::ZERO;
        oracle
            .get_exact(PreimageKey::new_local(L2_CLAIMED_POST_STATE_KEY.to()), l2_post.as_mut())
            .await
            .map_err(OracleProviderError::Preimage)?;

        let l2_claim_block = u64::from_be_bytes(
            oracle
                .get(PreimageKey::new_local(L2_CLAIMED_TIMESTAMP_KEY.to()))
                .await
                .map_err(OracleProviderError::Preimage)?
                .as_slice()
                .try_into()
                .map_err(OracleProviderError::SliceConversion)?,
        );

        let raw_pre_state = read_raw_pre_state(oracle, l2_pre).await?;
        if raw_pre_state == INVALID_TRANSITION {
            warn!(
                target: "boot_loader",
                "Invalid pre-state, short-circuiting to check post-state claim."
            );

            if l2_post == INVALID_TRANSITION_HASH {
                return Err(BootstrapError::InvalidToInvalid);
            } else {
                return Err(BootstrapError::InvalidPostState(l2_post));
            }
        }

        let agreed_pre_state =
            PreState::decode(&mut raw_pre_state.as_ref()).map_err(OracleProviderError::Rlp)?;

        let chain_ids: Vec<_> = match agreed_pre_state {
            PreState::SuperRoot(ref super_root) => {
                super_root.output_roots.iter().map(|r| r.chain_id).collect()
            }
            PreState::TransitionState(ref transition_state) => {
                transition_state.pre_state.output_roots.iter().map(|r| r.chain_id).collect()
            }
        };

        // Attempt to load the rollup config from the chain ID. If there is no config for the chain,
        // fall back to loading the config from the preimage oracle.
        let rollup_configs: HashMap<u64, RollupConfig> = if chain_ids
            .iter()
            .all(|id| ROLLUP_CONFIGS.contains_key(id))
        {
            chain_ids.iter().map(|id| (*id, ROLLUP_CONFIGS[id].clone())).collect()
        } else {
            warn!(
                target: "boot_loader",
                "No rollup config found for chain IDs {:?}, falling back to preimage oracle. This is insecure in production without additional validation!",
                chain_ids
            );
            let ser_cfg = oracle
                .get(PreimageKey::new_local(L2_ROLLUP_CONFIG_KEY.to()))
                .await
                .map_err(OracleProviderError::Preimage)?;
            serde_json::from_slice(&ser_cfg).map_err(OracleProviderError::Serde)?
        };

        // Attempt to load the l1 config from the chain ID. If there is no config for the chain,
        // fall back to loading the config from the preimage oracle.

        // Note that there should be only one l1 config per interop cluster. Let's ensure that all
        // the chain ids are the same.
        let l1_chain_ids = rollup_configs.values().map(|cfg| cfg.l1_chain_id).collect::<Vec<_>>();
        if l1_chain_ids.iter().any(|id| *id != l1_chain_ids[0]) {
            return Err(BootstrapError::InvalidL1Config);
        }

        let l1_chain_id = l1_chain_ids[0];

        let l1_config = if let Some(config) = L1_CONFIGS.get(&l1_chain_id) {
            config.clone()
        } else {
            warn!(
                target: "boot_loader",
                "No l1 config found for chain ID {}, falling back to preimage oracle. This is insecure in production without additional validation!",
                l1_chain_id
            );
            let ser_cfg = oracle
                .get(PreimageKey::new_local(L1_CONFIG_KEY.to()))
                .await
                .map_err(OracleProviderError::Preimage)?;
            serde_json::from_slice(&ser_cfg).map_err(OracleProviderError::Serde)?
        };

        Ok(Self {
            l1_head,
            l1_config,
            rollup_configs,
            agreed_pre_state_commitment: l2_pre,
            agreed_pre_state,
            claimed_post_state: l2_post,
            claimed_l2_timestamp: l2_claim_block,
        })
    }

    /// Returns the [RollupConfig] corresponding to the [PreState::active_l2_chain_id].
    pub fn active_rollup_config(&self) -> Option<RollupConfig> {
        let active_l2_chain_id = self.agreed_pre_state.active_l2_chain_id()?;
        self.rollup_configs.get(&active_l2_chain_id).cloned()
    }

    /// Returns the [L1ChainConfig] corresponding to the [PreState::active_l2_chain_id] through the
    /// l2 [RollupConfig].
    pub fn active_l1_config(&self) -> L1ChainConfig {
        self.l1_config.clone()
    }

    /// Returns the [RollupConfig] corresponding to the given `chain_id`.
    pub fn rollup_config(&self, chain_id: u64) -> Option<RollupConfig> {
        self.rollup_configs.get(&chain_id).cloned()
    }
}

/// An error that occurred during the bootstrapping phase.
#[derive(Debug, Error)]
pub enum BootstrapError {
    /// An error occurred while reading from the preimage oracle.
    #[error(transparent)]
    Oracle(#[from] OracleProviderError),
    /// The pre-state is invalid and the post-state claim is not invalid.
    #[error("`INVALID` pre-state claim; Post-state {0} unexpected.")]
    InvalidPostState(B256),
    /// The pre-state is invalid and the post-state claim is also invalid.
    #[error("No-op state transition detected; both pre and post states are `INVALID`.")]
    InvalidToInvalid,
    /// The l1 config is invalid because the chain ids are not the same.
    #[error("The l1 config is invalid because the chain ids are not the same.")]
    InvalidL1Config,
}

/// Reads the raw pre-state from the preimage oracle.
pub(crate) async fn read_raw_pre_state<O>(
    caching_oracle: &O,
    agreed_pre_state_commitment: B256,
) -> Result<Bytes, OracleProviderError>
where
    O: CommsClient,
{
    HintType::AgreedPreState
        .with_data(&[agreed_pre_state_commitment.as_ref()])
        .send(caching_oracle)
        .await?;
    let pre = caching_oracle
        .get(PreimageKey::new(*agreed_pre_state_commitment, PreimageKeyType::Keccak256))
        .await
        .map_err(OracleProviderError::Preimage)?;

    if pre.is_empty() {
        return Err(OracleProviderError::Preimage(PreimageOracleError::Other(
            "Invalid pre-state preimage".to_string(),
        )));
    }

    Ok(Bytes::from(pre))
}
