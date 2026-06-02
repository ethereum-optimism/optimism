use crate::{
    builders::{
        BuilderConfig, OpPayloadBuilderCtx, context::compute_post_exec_mode,
        flashblocks::FlashblocksConfig,
    },
    gas_limiter::{AddressGasLimiter, args::GasLimiterArgs},
    metrics::OpRBuilderMetrics,
    sdm_admin::SdmPostExecOptInFlag,
    traits::ClientBounds,
};
use op_revm::OpSpecId;
use reth_basic_payload_builder::PayloadConfig;
use reth_evm::EvmEnv;
use reth_optimism_chainspec::OpChainSpec;
use reth_optimism_evm::{OpEvmConfig, OpNextBlockEnvAttributes};
use reth_optimism_payload_builder::{
    OpPayloadBuilderAttributes,
    config::{OpDAConfig, OpGasLimitConfig},
};
use reth_optimism_primitives::OpTransactionSigned;
use std::sync::Arc;
use tokio_util::sync::CancellationToken;

#[derive(Debug, Clone)]
pub(super) struct OpPayloadSyncerCtx {
    /// The type that knows how to perform system calls and configure the evm.
    evm_config: OpEvmConfig,
    /// The DA config for the payload builder
    da_config: OpDAConfig,
    /// The chainspec
    chain_spec: Arc<OpChainSpec>,
    /// Max gas that can be used by a transaction.
    max_gas_per_txn: Option<u64>,
    /// The metrics for the builder
    metrics: Arc<OpRBuilderMetrics>,
    /// Operator opt-in flag for SDM PostExec production, shared with the admin RPC.
    sdm_post_exec_opt_in: SdmPostExecOptInFlag,
}

impl OpPayloadSyncerCtx {
    pub(super) fn new<Client>(
        client: &Client,
        builder_config: BuilderConfig<FlashblocksConfig>,
        evm_config: OpEvmConfig,
        metrics: Arc<OpRBuilderMetrics>,
    ) -> eyre::Result<Self>
    where
        Client: ClientBounds,
    {
        let chain_spec = client.chain_spec();
        Ok(Self {
            evm_config,
            da_config: builder_config.da_config.clone(),
            chain_spec,
            max_gas_per_txn: builder_config.max_gas_per_txn,
            metrics,
            sdm_post_exec_opt_in: builder_config.sdm_post_exec_opt_in.clone(),
        })
    }

    pub(super) fn evm_config(&self) -> &OpEvmConfig {
        &self.evm_config
    }

    pub(super) fn max_gas_per_txn(&self) -> Option<u64> {
        self.max_gas_per_txn
    }

    pub(super) fn into_op_payload_builder_ctx(
        self,
        payload_config: PayloadConfig<OpPayloadBuilderAttributes<OpTransactionSigned>>,
        evm_env: EvmEnv<OpSpecId>,
        block_env_attributes: OpNextBlockEnvAttributes,
        cancel: CancellationToken,
    ) -> OpPayloadBuilderCtx {
        let post_exec_mode = compute_post_exec_mode(
            &self.evm_config,
            payload_config.attributes.timestamp(),
            &self.sdm_post_exec_opt_in,
        );
        OpPayloadBuilderCtx {
            evm_config: self.evm_config,
            da_config: self.da_config,
            gas_limit_config: OpGasLimitConfig::default(),
            chain_spec: self.chain_spec,
            config: payload_config,
            evm_env,
            block_env_attributes,
            cancel,
            builder_signer: None,
            metrics: self.metrics,
            extra_ctx: (),
            max_gas_per_txn: self.max_gas_per_txn,
            address_gas_limiter: AddressGasLimiter::new(GasLimiterArgs::default()),
            post_exec_mode,
        }
    }
}
