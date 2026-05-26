use core::{
    convert::{Infallible, TryFrom},
    fmt::Debug,
    time::Duration,
};
use reth_node_builder::components::PayloadServiceBuilder;
use reth_optimism_evm::OpEvmConfig;
use reth_optimism_payload_builder::config::{OpDAConfig, OpGasLimitConfig};

use crate::{
    args::OpRbuilderArgs,
    flashtestations::args::FlashtestationsArgs,
    gas_limiter::args::GasLimiterArgs,
    traits::{NodeBounds, PoolBounds},
    tx_signer::Signer,
};

mod builder_tx;
mod context;
mod flashblocks;
mod generator;
mod standard;

pub use builder_tx::{
    BuilderTransactionCtx, BuilderTransactionError, BuilderTransactions, InvalidContractDataError,
    SimulationSuccessResult, get_balance, get_nonce,
};
pub use context::OpPayloadBuilderCtx;
pub use flashblocks::FlashblocksBuilder;
pub use standard::StandardBuilder;

/// Defines the payload building mode for the OP builder.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub enum BuilderMode {
    /// Uses the plain OP payload builder that produces blocks every chain blocktime.
    #[default]
    Standard,
    /// Uses the flashblocks payload builder that progressively builds chunks of a
    /// block every short interval and makes it available through a websocket update
    /// then merges them into a full block every chain block time.
    Flashblocks,
}

/// Defines the interface for any block builder implementation API entry point.
///
/// Instances of this trait are used during Reth node construction as an argument
/// to the `NodeBuilder::with_components` method to construct the payload builder
/// service that gets called whenver the current node is asked to build a block.
pub trait PayloadBuilder: Send + Sync + 'static {
    /// The type that has an implementation specific variant of the Config<T> struct.
    /// This is used to configure the payload builder service during startup.
    type Config: TryFrom<OpRbuilderArgs, Error: Debug> + Clone + Debug + Send + Sync + 'static;

    /// The type that is used to instantiate the payload builder service
    /// that will be used by reth to build blocks whenever the node is
    /// asked to do so.
    type ServiceBuilder<Node, Pool>: PayloadServiceBuilder<Node, Pool, OpEvmConfig>
    where
        Node: NodeBounds,
        Pool: PoolBounds;

    /// Called during node startup by reth. Returns a [`PayloadBuilderService`] instance
    /// that is preloaded with a [`PayloadJobGenerator`] instance specific to the builder
    /// type.
    fn new_service<Node, Pool>(
        config: BuilderConfig<Self::Config>,
    ) -> eyre::Result<Self::ServiceBuilder<Node, Pool>>
    where
        Node: NodeBounds,
        Pool: PoolBounds;
}

/// Configuration values that are applicable to any type of block builder.
#[derive(Clone)]
pub struct BuilderConfig<Specific: Clone> {
    /// Secret key of the builder that is used to sign the end of block transaction.
    pub builder_signer: Option<Signer>,

    /// When set to true, transactions are simulated by the builder and excluded from the block
    /// if they revert. They may still be included in the block if individual transactions
    /// opt-out of revert protection.
    pub revert_protection: bool,

    /// When enabled, this will invoke the flashtestions workflow. This involves a
    /// bootstrapping step that generates a new pubkey for the TEE service
    pub flashtestations_config: FlashtestationsArgs,

    /// The interval at which blocks are added to the chain.
    /// This is also the frequency at which the builder will be receiving FCU requests from the
    /// sequencer.
    pub block_time: Duration,

    /// Data Availability configuration for the OP builder
    /// Defines constraints for the maximum size of data availability transactions.
    pub da_config: OpDAConfig,

    /// Gas limit configuration for the payload builder
    pub gas_limit_config: OpGasLimitConfig,

    // The deadline is critical for payload availability. If we reach the deadline,
    // the payload job stops and cannot be queried again. With tight deadlines close
    // to the block number, we risk reaching the deadline before the node queries the payload.
    //
    // Adding 0.5 seconds as wiggle room since block times are shorter here.
    // TODO: A better long-term solution would be to implement cancellation logic
    // that cancels existing jobs when receiving new block building requests.
    //
    // When batcher's max channel duration is big enough (e.g. 10m), the
    // sequencer would send an avalanche of FCUs/getBlockByNumber on
    // each batcher update (with 10m channel it's ~800 FCUs at once).
    // At such moment it can happen that the time b/w FCU and ensuing
    // getPayload would be on the scale of ~2.5s. Therefore we should
    // "remember" the payloads long enough to accommodate this corner-case
    // (without it we are losing blocks). Postponing the deadline for 5s
    // (not just 0.5s) because of that.
    pub block_time_leeway: Duration,

    /// Inverted sampling frequency in blocks. 1 - each block, 100 - every 100th block.
    pub sampling_ratio: u64,

    /// Configuration values that are specific to the block builder implementation used.
    pub specific: Specific,

    /// Maximum gas a transaction can use before being excluded.
    pub max_gas_per_txn: Option<u64>,

    /// Address gas limiter stuff
    pub gas_limiter_config: GasLimiterArgs,
}

impl<S: Debug + Clone> core::fmt::Debug for BuilderConfig<S> {
    fn fmt(&self, f: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
        f.debug_struct("Config")
            .field(
                "builder_signer",
                &match self.builder_signer.as_ref() {
                    Some(signer) => signer.address.to_string(),
                    None => "None".into(),
                },
            )
            .field("revert_protection", &self.revert_protection)
            .field("flashtestations", &self.flashtestations_config)
            .field("block_time", &self.block_time)
            .field("block_time_leeway", &self.block_time_leeway)
            .field("da_config", &self.da_config)
            .field("gas_limit_config", &self.gas_limit_config)
            .field("sampling_ratio", &self.sampling_ratio)
            .field("specific", &self.specific)
            .field("max_gas_per_txn", &self.max_gas_per_txn)
            .field("gas_limiter_config", &self.gas_limiter_config)
            .finish()
    }
}

impl<S: Default + Clone> Default for BuilderConfig<S> {
    fn default() -> Self {
        Self {
            builder_signer: None,
            revert_protection: false,
            flashtestations_config: FlashtestationsArgs::default(),
            block_time: Duration::from_secs(2),
            block_time_leeway: Duration::from_millis(500),
            da_config: OpDAConfig::default(),
            gas_limit_config: OpGasLimitConfig::default(),
            specific: S::default(),
            sampling_ratio: 100,
            max_gas_per_txn: None,
            gas_limiter_config: GasLimiterArgs::default(),
        }
    }
}

impl<S> TryFrom<OpRbuilderArgs> for BuilderConfig<S>
where
    S: TryFrom<OpRbuilderArgs, Error: Debug> + Clone,
{
    type Error = S::Error;

    fn try_from(args: OpRbuilderArgs) -> Result<Self, Self::Error> {
        Ok(Self {
            builder_signer: args.builder_signer,
            revert_protection: args.enable_revert_protection,
            flashtestations_config: args.flashtestations.clone(),
            block_time: Duration::from_millis(args.chain_block_time),
            block_time_leeway: Duration::from_secs(args.extra_block_deadline_secs),
            da_config: Default::default(),
            gas_limit_config: Default::default(),
            sampling_ratio: args.telemetry.sampling_ratio,
            max_gas_per_txn: args.max_gas_per_txn,
            gas_limiter_config: args.gas_limiter.clone(),
            specific: S::try_from(args)?,
        })
    }
}

#[expect(clippy::infallible_try_from)]
impl TryFrom<OpRbuilderArgs> for () {
    type Error = Infallible;

    fn try_from(_: OpRbuilderArgs) -> Result<Self, Self::Error> {
        Ok(())
    }
}
