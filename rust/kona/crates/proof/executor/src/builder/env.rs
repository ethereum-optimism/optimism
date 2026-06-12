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
use alloy_evm::{EvmEnv, EvmFactory};
use alloy_primitives::U256;
use kona_genesis::RollupConfig;
use kona_mpt::TrieHinter;
use op_alloy_rpc_types_engine::OpPayloadAttributes;
use op_revm::OpSpecId;
use revm::{
    context::{BlockEnv, CfgEnv},
    context_interface::block::BlobExcessGasAndPrice,
};

impl<P, H, Evm, R> StatelessL2Builder<'_, P, H, Evm, R>
where
    P: TrieDBProvider,
    H: TrieHinter,
    Evm: EvmFactory,
{
    /// Returns the active [`EvmEnv`] for the executor.
    pub(crate) fn evm_env(
        &self,
        spec_id: OpSpecId,
        parent_header: &Header,
        payload_attrs: &OpPayloadAttributes,
        base_fee_params: &BaseFeeParams,
        min_base_fee: u64,
    ) -> ExecutorResult<EvmEnv<OpSpecId>> {
        let block_env = self.prepare_block_env(
            spec_id,
            parent_header,
            payload_attrs,
            base_fee_params,
            min_base_fee,
        )?;
        let cfg_env = self.evm_cfg_env(payload_attrs.payload_attributes.timestamp);
        Ok(EvmEnv::new(cfg_env, block_env))
    }

    /// Returns the active [`CfgEnv`] for the executor.
    pub(crate) fn evm_cfg_env(&self, timestamp: u64) -> CfgEnv<OpSpecId> {
        CfgEnv::new()
            .with_chain_id(self.config.l2_chain_id.id())
            .with_spec_and_mainnet_gas_params(self.config.spec_id(timestamp))
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

    /// Prepares a [`BlockEnv`] with the given [`OpPayloadAttributes`].
    pub(crate) fn prepare_block_env(
        &self,
        spec_id: OpSpecId,
        parent_header: &Header,
        payload_attrs: &OpPayloadAttributes,
        base_fee_params: &BaseFeeParams,
        min_base_fee: u64,
    ) -> ExecutorResult<BlockEnv> {
        // On the OP Stack the BLOBBASEFEE opcode is always 1 from Ecotone onward. Post-Jovian the
        // header's `blobGasUsed` field carries the block's DA footprint, so it must not feed the
        // EIP-4844 excess-blob-gas rule; pin the blob env instead.
        let blob_excess_gas_and_price = spec_id
            .is_enabled_in(OpSpecId::ECOTONE)
            .then_some(BlobExcessGasAndPrice { excess_blob_gas: 0, blob_gasprice: 1 });

        let next_block_base_fee = self
            .next_block_base_fee(*base_fee_params, parent_header, min_base_fee)
            .unwrap_or_default();

        Ok(BlockEnv {
            number: U256::from(parent_header.number + 1),
            beneficiary: payload_attrs.payload_attributes.suggested_fee_recipient,
            timestamp: U256::from(payload_attrs.payload_attributes.timestamp),
            gas_limit: payload_attrs.gas_limit.ok_or(ExecutorError::MissingGasLimit)?,
            basefee: next_block_base_fee,
            prevrandao: Some(payload_attrs.payload_attributes.prev_randao),
            blob_excess_gas_and_price,
            ..Default::default()
        })
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

#[cfg(test)]
mod tests {
    use super::*;
    use crate::NoopTrieDBProvider;
    use alloy_consensus::{Header, Sealable};
    use alloy_op_evm::OpEvmFactory;
    use alloy_rpc_types_engine::PayloadAttributes;
    use kona_mpt::NoopTrieHinter;

    /// The `BLOBBASEFEE` opcode must always be 1 on the OP Stack. Post-Jovian the header's
    /// `blobGasUsed` carries the block's DA footprint, which must not influence the blob env. The
    /// footprint here is from op-mainnet block 152635937.
    #[test]
    fn prepare_block_env_pins_blob_gasprice_to_one() {
        let config = RollupConfig::default();
        let parent_header = Header {
            number: 100,
            timestamp: 1_000_000,
            gas_limit: 60_000_000,
            base_fee_per_gas: Some(1_000_000_000),
            blob_gas_used: Some(30_406_400),
            excess_blob_gas: Some(0),
            ..Default::default()
        };

        let builder = StatelessL2Builder::new(
            &config,
            OpEvmFactory::<alloy_op_evm::OpTx>::default(),
            alloy_op_evm::block::OpAlloyReceiptBuilder::default(),
            NoopTrieDBProvider,
            NoopTrieHinter,
            parent_header.clone().seal_slow(),
        );

        let payload_attrs = OpPayloadAttributes {
            payload_attributes: PayloadAttributes {
                timestamp: parent_header.timestamp + 2,
                ..Default::default()
            },
            gas_limit: Some(parent_header.gas_limit),
            ..Default::default()
        };

        let block_env = builder
            .prepare_block_env(
                OpSpecId::ISTHMUS,
                &parent_header,
                &payload_attrs,
                &BaseFeeParams::new(250, 6),
                0,
            )
            .expect("prepare_block_env should succeed");

        let blob =
            block_env.blob_excess_gas_and_price.expect("blob env should be present for Isthmus");
        assert_eq!(
            blob.blob_gasprice, 1,
            "BLOBBASEFEE must be pinned to 1 on the OP Stack regardless of the parent DA footprint"
        );
        assert_eq!(
            blob.excess_blob_gas, 0,
            "excess blob gas must be pinned to 0 regardless of the parent DA footprint"
        );
    }
}
