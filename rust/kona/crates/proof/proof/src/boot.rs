//! This module contains the prologue phase of the client program, pulling in the boot information
//! through the `PreimageOracle` ABI as local keys.

use crate::errors::OracleProviderError;
use alloy_primitives::{B256, U256};
use kona_genesis::{L1ChainConfig, RollupConfig};
use kona_preimage::{PreimageKey, PreimageOracleClient};
use kona_registry::{L1_CONFIGS, ROLLUP_CONFIGS};
use serde::{Deserialize, Serialize};

/// The local key identifier for the L1 head hash.
///
/// This key is used to retrieve the L1 block hash that contains all the data
/// necessary to derive the disputed L2 blocks. The L1 head serves as the
/// starting point for L1 data extraction during the derivation process.
pub const L1_HEAD_KEY: U256 = U256::from_be_slice(&[1]);

/// The local key identifier for the agreed L2 output root.
///
/// This key retrieves the baseline L2 output root that both parties agree upon.
/// It represents the last known good state before the disputed blocks and serves
/// as the starting point for derivation verification.
pub const L2_OUTPUT_ROOT_KEY: U256 = U256::from_be_slice(&[2]);

/// The local key identifier for the disputed L2 output root claim.
///
/// This key retrieves the user's claimed L2 output root at the target block.
/// The fault proof will compare the derived output root against this claim
/// to determine if the claim is valid or invalid.
pub const L2_CLAIM_KEY: U256 = U256::from_be_slice(&[3]);

/// The local key identifier for the disputed L2 block number.
///
/// This key retrieves the L2 block number at which the output root disagreement
/// occurs. The derivation process will produce blocks up to this number to
/// verify the claim.
pub const L2_CLAIM_BLOCK_NUMBER_KEY: U256 = U256::from_be_slice(&[4]);

/// The local key identifier for the L2 chain ID.
///
/// This key retrieves the L2 network identifier, which is used to load the
/// appropriate rollup configuration and ensure network-specific validation
/// rules are applied correctly.
pub const L2_CHAIN_ID_KEY: U256 = U256::from_be_slice(&[5]);

/// The local key identifier for the L2 rollup configuration.
///
/// This key is used as a fallback to retrieve the rollup configuration from
/// the preimage oracle when no hardcoded configuration is available for the
/// given chain ID. Oracle-loaded configs require additional validation.
pub const L2_ROLLUP_CONFIG_KEY: U256 = U256::from_be_slice(&[6]);

/// The local key identifier for the L1 chain configuration.
///
/// This key is used as a fallback to retrieve the chain configuration from
/// the preimage oracle when no hardcoded configuration is available for the
/// given chain ID. Oracle-loaded configs require additional validation.
pub const L1_CONFIG_KEY: U256 = U256::from_be_slice(&[7]);

/// The boot information for the client program.
///
/// [`BootInfo`] contains all the essential parameters needed to initialize the fault proof
/// client program. It separates verified inputs (cryptographically committed) from user
/// inputs (requiring validation through derivation).
///
/// This structure is loaded during the prologue phase from the preimage oracle and
/// establishes the initial state for the fault proof computation.
///
/// # Security Model
/// The boot information follows a two-tier security model:
/// - **Verified inputs**: Committed by the fault proof system, trusted
/// - **User inputs**: Provided by the claimant, must be verified through execution
///
/// # Usage in Fault Proof
/// 1. Load boot info from preimage oracle during prologue
/// 2. Initialize derivation pipeline with verified L1 head and safe L2 output
/// 3. Derive L2 blocks up to the claimed block number
/// 4. Compare derived output root with user's claim
/// 5. Proof succeeds if outputs match, fails otherwise
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct BootInfo {
    /// The L1 head hash containing safe L2 chain data for reproduction.
    ///
    /// This hash identifies the L1 block that contains all the data necessary
    /// to derive the L2 chain up to the disputed block. It serves as the
    /// starting point for L1 data extraction during derivation.
    ///
    /// **Security**: Verified input committed by the fault proof system.
    pub l1_head: B256,
    /// The agreed upon safe L2 output root.
    ///
    /// This represents the last known good L2 state that both parties agree upon.
    /// It serves as the starting point for derivation and the baseline against
    /// which the disputed claim is evaluated.
    ///
    /// **Security**: Verified input committed by the fault proof system.
    pub agreed_l2_output_root: B256,
    /// The disputed L2 output root claim.
    ///
    /// This is the user's claim about what the L2 output root should be at the
    /// target block number. The fault proof will derive the actual output root
    /// and compare it against this claim to determine validity.
    ///
    /// **Security**: User-submitted input requiring verification.
    pub claimed_l2_output_root: B256,
    /// The L2 block number being disputed.
    ///
    /// This specifies the target L2 block number at which the output root
    /// disagreement occurs. The derivation process will produce blocks up to
    /// this number and compute the resulting output root.
    ///
    /// **Security**: User-submitted input requiring verification.
    pub claimed_l2_block_number: u64,
    /// The L2 chain identifier.
    ///
    /// Used to identify which L2 network this proof applies to and to load
    /// the appropriate rollup configuration. This prevents cross-chain
    /// replay attacks and ensures proper network-specific validation.
    ///
    /// **Security**: Verified input committed by the fault proof system.
    pub chain_id: u64,
    /// The rollup configuration for the L2 chain.
    ///
    /// Contains all the network-specific parameters needed for proper L2 block
    /// derivation, including genesis configuration, system addresses, gas limits,
    /// and hard fork activation heights.
    ///
    /// **Security**: Loaded from registry (secure) or oracle (requires validation).
    pub rollup_config: RollupConfig,
    /// An optional configuration for the l1 chain associated with the l2 chain.
    ///
    /// **Security**: Loaded from registry (secure) or oracle (requires validation).
    pub l1_config: L1ChainConfig,
}

impl BootInfo {
    /// Load the boot information from the preimage oracle.
    ///
    /// This method retrieves all the necessary boot parameters from the preimage oracle
    /// using predefined local keys. It handles both verified inputs (from the fault proof
    /// system) and user-submitted inputs that need validation.
    ///
    /// # Arguments
    /// * `oracle` - The preimage oracle client for reading boot data
    ///
    /// # Returns
    /// * `Ok(BootInfo)` - Successfully loaded and validated boot information
    /// * `Err(OracleProviderError)` - Failed to load or parse boot information
    ///
    /// # Errors
    /// This method can fail due to:
    /// - **Preimage errors**: Oracle communication failures or missing keys
    /// - **Slice conversion errors**: Invalid data format for numeric values
    /// - **Serde errors**: Failed to deserialize rollup configuration
    /// - **Missing data**: Required boot parameters not available in oracle
    ///
    /// # Loading Process
    /// The method loads boot information in this order:
    /// 1. **L1 Head Hash** (`L1_HEAD_KEY`): The L1 block containing safe L2 data
    /// 2. **Agreed L2 Output Root** (`L2_OUTPUT_ROOT_KEY`): Known safe L2 state
    /// 3. **Claimed L2 Output Root** (`L2_CLAIM_KEY`): User's disputed claim
    /// 4. **Claimed Block Number** (`L2_CLAIM_BLOCK_NUMBER_KEY`): Target block height
    /// 5. **Chain ID** (`L2_CHAIN_ID_KEY`): L2 network identifier
    /// 6. **Rollup Config**: Either from registry (secure) or oracle (fallback)
    ///
    /// # Rollup Configuration Loading
    /// The rollup configuration is loaded with a security preference:
    /// - **Primary**: Lookup in hardcoded [`static@ROLLUP_CONFIGS`] registry by chain ID
    /// - **Fallback**: Load from oracle using `L2_ROLLUP_CONFIG_KEY` (with warning)
    ///
    /// The fallback method requires additional validation in production environments
    /// as oracle-provided configs are not verified by the fault proof system.
    ///
    /// # Security Considerations
    /// - Verified inputs are cryptographically committed in the fault proof
    /// - User inputs (claim data) require verification through the derivation process
    /// - Oracle-loaded configs should be validated against known good configurations
    ///
    /// # Example Usage
    /// ```rust,ignore
    /// let boot_info = BootInfo::load(&oracle).await?;
    /// println!("Loaded boot info for chain {}", boot_info.chain_id);
    /// println!("Target block: {}", boot_info.claimed_l2_block_number);
    /// ```
    pub async fn load<O>(oracle: &O) -> Result<Self, OracleProviderError>
    where
        O: PreimageOracleClient + Send,
    {
        let mut l1_head: B256 = B256::ZERO;
        oracle
            .get_exact(PreimageKey::new_local(L1_HEAD_KEY.to()), l1_head.as_mut())
            .await
            .map_err(OracleProviderError::Preimage)?;

        let mut l2_output_root: B256 = B256::ZERO;
        oracle
            .get_exact(PreimageKey::new_local(L2_OUTPUT_ROOT_KEY.to()), l2_output_root.as_mut())
            .await
            .map_err(OracleProviderError::Preimage)?;

        let mut l2_claim: B256 = B256::ZERO;
        oracle
            .get_exact(PreimageKey::new_local(L2_CLAIM_KEY.to()), l2_claim.as_mut())
            .await
            .map_err(OracleProviderError::Preimage)?;

        let l2_claim_block = u64::from_be_bytes(
            oracle
                .get(PreimageKey::new_local(L2_CLAIM_BLOCK_NUMBER_KEY.to()))
                .await
                .map_err(OracleProviderError::Preimage)?
                .as_slice()
                .try_into()
                .map_err(OracleProviderError::SliceConversion)?,
        );
        let chain_id = u64::from_be_bytes(
            oracle
                .get(PreimageKey::new_local(L2_CHAIN_ID_KEY.to()))
                .await
                .map_err(OracleProviderError::Preimage)?
                .as_slice()
                .try_into()
                .map_err(OracleProviderError::SliceConversion)?,
        );

        // Attempt to load the rollup config from the chain ID. If there is no config for the chain,
        // fall back to loading the config from the preimage oracle.
        let rollup_config = if let Some(config) = ROLLUP_CONFIGS.get(&chain_id) {
            config.clone()
        } else {
            warn!(
                target: "boot_loader",
                "No rollup config found for chain ID {}, falling back to preimage oracle. This is insecure in production without additional validation!",
                chain_id
            );
            let ser_cfg = oracle
                .get(PreimageKey::new_local(L2_ROLLUP_CONFIG_KEY.to()))
                .await
                .map_err(OracleProviderError::Preimage)?;
            serde_json::from_slice(&ser_cfg).map_err(OracleProviderError::Serde)?
        };

        // Attempt to load the rollup config from the chain ID. If there is no config for the chain,
        // fall back to loading the config from the preimage oracle.
        let l1_config = if let Some(config) = L1_CONFIGS.get(&rollup_config.l1_chain_id) {
            config.clone()
        } else {
            warn!(
                target: "boot_loader",
                "No l1 config found for chain ID {}, falling back to preimage oracle. This is insecure in production without additional validation!",
                rollup_config.l1_chain_id
            );
            let ser_cfg = oracle
                .get(PreimageKey::new_local(L1_CONFIG_KEY.to()))
                .await
                .map_err(OracleProviderError::Preimage)?;

            serde_json::from_slice(&ser_cfg).map_err(OracleProviderError::Serde)?
        };

        debug!(
            target: "boot_loader",
            l1_head = %l1_head,
            chain_id = chain_id,
            claimed_l2_block_number = l2_claim_block,
            "Successfully loaded boot information"
        );

        Ok(Self {
            l1_head,
            agreed_l2_output_root: l2_output_root,
            claimed_l2_output_root: l2_claim,
            claimed_l2_block_number: l2_claim_block,
            chain_id,
            rollup_config,
            l1_config,
        })
    }
}
