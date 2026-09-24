//! Public-projection attribute pre-execution.
//!
//! On a public projection every sequencer transaction of a block must execute successfully, or
//! the block is invalid (`OpBlockExecutionError::ProjectionSequencerTxFailed`, enabled by
//! `OpEvmConfig::optimism` for projection genesis). Derived attributes that break the rule must be
//! reported to the consensus client as invalid attributes, so that derivation (Holocene) replaces
//! them with a deposit-only block. The asynchronous payload job cannot do that: by the time it
//! fails, `engine_forkchoiceUpdated` has already answered VALID with a payload ID, and the failed
//! job only surfaces as an unknown payload, which the consensus client retries forever.
//!
//! So, like op-geth, which builds the attributes-only payload synchronously inside
//! `engine_forkchoiceUpdated` and answers `INVALID_PAYLOAD_ATTRIBUTES` when a forced transaction
//! fails, the engine validator pre-executes the attributes' transactions with this function
//! before a projection payload job is started.

use crate::{OpPayloadBuilderAttributes, OpPayloadPrimitives};
use reth_chainspec::EthChainSpec;
use reth_evm::{
    ConfigureEvm,
    execute::{BlockBuilder, BlockExecutionError, BlockValidationError},
};
use reth_optimism_forks::OpHardforks;
use reth_payload_builder_primitives::PayloadBuilderError;
use reth_payload_primitives::BuildNextEnv;
use reth_primitives_traits::{SealedHeader, SignedTransaction};
use reth_revm::{database::StateProviderDatabase, db::State};
use reth_storage_api::StateProvider;

/// Executes the sequencer transactions of `attributes` on top of `parent`, exactly as the payload
/// job does, and returns the first execution error the job would treat as fatal.
///
/// Returns `Ok(None)` when every sequencer transaction executes. Transactions the job skips
/// (`InvalidTx`: bad nonce, insufficient funds, ...) are skipped here too. `Err` means the check
/// itself could not run, which callers should treat as "no verdict".
pub fn first_failing_sequencer_tx<Evm, N, ChainSpec>(
    evm_config: &Evm,
    chain_spec: &ChainSpec,
    state: impl StateProvider,
    parent: &SealedHeader<N::BlockHeader>,
    attributes: &OpPayloadBuilderAttributes<N::SignedTx>,
) -> Result<Option<BlockExecutionError>, PayloadBuilderError>
where
    Evm: ConfigureEvm<
            Primitives = N,
            NextBlockEnvCtx: BuildNextEnv<
                OpPayloadBuilderAttributes<N::SignedTx>,
                N::BlockHeader,
                ChainSpec,
            >,
        >,
    N: OpPayloadPrimitives,
    ChainSpec: EthChainSpec + OpHardforks,
{
    let mut db = State::builder()
        .with_database(StateProviderDatabase::new(&state))
        .with_bundle_update()
        .build();
    let env = Evm::NextBlockEnvCtx::build_next_env(attributes, parent, chain_spec)
        .map_err(PayloadBuilderError::other)?;
    let mut builder = evm_config
        .builder_for_next_block(&mut db, parent, env)
        .map_err(PayloadBuilderError::other)?;
    builder
        .apply_pre_execution_changes()
        .map_err(|err| PayloadBuilderError::EvmExecutionError(Box::new(err)))?;
    for tx in &attributes.transactions {
        let Ok(tx) = tx.value().try_clone_into_recovered() else {
            // The job rejects the attributes on its own for an unrecoverable transaction.
            return Ok(None);
        };
        match builder.execute_transaction(tx) {
            Ok(_) |
            Err(BlockExecutionError::Validation(BlockValidationError::InvalidTx { .. })) => {}
            Err(err) => return Ok(Some(err)),
        }
    }
    Ok(None)
}
