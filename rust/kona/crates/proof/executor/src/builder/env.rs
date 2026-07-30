//! Environment utility functions for [`StatelessL2Builder`].

use super::StatelessL2Builder;
use crate::{
    ExecutorError, ExecutorResult, TrieDBProvider,
    util::{
        decode_holocene_eip_1559_params_block_header, decode_jovian_eip_1559_params_block_header,
    },
};
use alloy_consensus::{BlockHeader, Header};
use alloy_eips::{calc_next_block_base_fee, eip1559::BaseFeeParams};
use alloy_evm::{EvmEnv, EvmFactory, eth::NextEvmEnvAttributes};
use alloy_op_evm::evm_env_for_op_next_block;
use kona_genesis::RollupConfig;
use kona_mpt::TrieHinter;
use op_alloy_rpc_types_engine::OpPayloadAttributes;
use op_revm::OpSpecId;

impl<P, H, Evm, R> StatelessL2Builder<'_, P, H, Evm, R>
where
    P: TrieDBProvider,
    H: TrieHinter,
    Evm: EvmFactory,
{
    /// Returns the active [`EvmEnv`] for the executor.
    pub(crate) fn evm_env(
        &self,
        parent_header: &Header,
        payload_attrs: &OpPayloadAttributes,
        base_fee_params: &BaseFeeParams,
        min_base_fee: u64,
    ) -> ExecutorResult<EvmEnv<OpSpecId>> {
        let gas_limit = payload_attrs.gas_limit.ok_or(ExecutorError::MissingGasLimit)?;
        let next_block_base_fee = self
            .next_block_base_fee(*base_fee_params, parent_header, min_base_fee)
            .unwrap_or_default();

        Ok(evm_env_for_op_next_block(
            parent_header,
            NextEvmEnvAttributes {
                timestamp: payload_attrs.payload_attributes.timestamp,
                suggested_fee_recipient: payload_attrs.payload_attributes.suggested_fee_recipient,
                prev_randao: payload_attrs.payload_attributes.prev_randao,
                gas_limit,
                slot_number: None,
            },
            next_block_base_fee,
            self.config,
            self.config.l2_chain_id.id(),
        ))
    }

    fn next_block_base_fee(
        &self,
        params: BaseFeeParams,
        parent: &Header,
        min_base_fee: u64,
    ) -> Option<u64> {
        if !self.config.is_jovian_active(parent.timestamp()) {
            return parent.next_block_base_fee(params);
        }

        // Starting from Jovian, we use the maximum of the gas used and the blob gas used to
        // calculate the next base fee.
        let gas_used = if parent.blob_gas_used().unwrap_or_default() > parent.gas_used() {
            parent.blob_gas_used().unwrap_or_default()
        } else {
            parent.gas_used()
        };

        let mut next_block_base_fee = calc_next_block_base_fee(
            gas_used,
            parent.gas_limit(),
            parent.base_fee_per_gas().unwrap_or_default(),
            params,
        );

        // If the next block base fee is less than the min base fee, set it to the min base fee.
        // # Note
        // Before Jovian activation, the min-base-fee is 0 so this is a no-op.
        if next_block_base_fee < min_base_fee {
            next_block_base_fee = min_base_fee;
        }

        Some(next_block_base_fee)
    }

    /// Returns the active base fee parameters for the parent header.
    /// Returns the min-base-fee as the second element of the tuple.
    ///
    /// ## Note
    /// Before Jovian activation, the min-base-fee is 0.
    pub(crate) fn active_base_fee_params(
        config: &RollupConfig,
        parent_header: &Header,
        payload_timestamp: u64,
    ) -> ExecutorResult<(BaseFeeParams, u64)> {
        match config {
            // After Holocene activation, the base fee parameters are stored in the
            // `extraData` field of the parent header. If Holocene wasn't active in the
            // parent block, the default base fee parameters are used.
            _ if config.is_jovian_active(parent_header.timestamp) => {
                decode_jovian_eip_1559_params_block_header(parent_header)
            }
            _ if config.is_holocene_active(parent_header.timestamp) => {
                decode_holocene_eip_1559_params_block_header(parent_header)
                    .map(|base_fee_params| (base_fee_params, 0))
            }
            // If the next payload attribute timestamp is past canyon activation,
            // use the canyon base fee params from the rollup config.
            _ if config.is_canyon_active(payload_timestamp) => {
                // If the payload attribute timestamp is past canyon activation,
                // use the canyon base fee params from the rollup config.
                Ok((config.chain_op_config.post_canyon_params(), 0))
            }
            _ => {
                // If the next payload attribute timestamp is prior to canyon activation,
                // use the default base fee params from the rollup config.
                Ok((config.chain_op_config.pre_canyon_params(), 0))
            }
        }
    }
}
