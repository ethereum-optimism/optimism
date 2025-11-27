//! Command that initializes the node from a genesis file.

use alloy_consensus::Header;
use clap::Parser;
use reth_cli::chainspec::ChainSpecParser;
use reth_cli_commands::common::{AccessRights, CliNodeTypes, Environment};
use reth_db_common::init::init_from_state_dump;
use reth_optimism_chainspec::OpChainSpec;
use reth_optimism_primitives::{
    bedrock::{BEDROCK_HEADER, BEDROCK_HEADER_HASH},
    OpPrimitives,
};
use reth_primitives_traits::{header::HeaderMut, SealedHeader};
use reth_provider::{
    BlockNumReader, DBProvider, DatabaseProviderFactory, StaticFileProviderFactory,
    StaticFileWriter,
};
use std::{io::BufReader, sync::Arc};
use tracing::info;

/// Initializes the database with the genesis block.
#[derive(Debug, Parser)]
pub struct InitStateCommandOp<C: ChainSpecParser> {
    #[command(flatten)]
    init_state: reth_cli_commands::init_state::InitStateCommand<C>,

    /// Specifies whether to initialize the state without relying on OVM or EVM historical data.
    ///
    /// When enabled, and before inserting the state, it creates a dummy chain up to the last OVM
    /// block (#105235062) (14GB / 90 seconds). It then, appends the Bedrock block. This is
    /// hardcoded for OP mainnet, for other OP chains you will need to pass in a header.
    ///
    /// - **Note**: **Do not** import receipts and blocks beforehand, or this will fail or be
    ///   ignored.
    #[arg(long, default_value = "false")]
    without_ovm: bool,
}

impl<C: ChainSpecParser<ChainSpec = OpChainSpec>> InitStateCommandOp<C> {
    /// Execute the `init` command
    pub async fn execute<N: CliNodeTypes<ChainSpec = C::ChainSpec, Primitives = OpPrimitives>>(
        mut self,
    ) -> eyre::Result<()> {
        // If using --without-ovm for OP mainnet, handle the special case with hardcoded Bedrock
        // header. Otherwise delegate to the base InitStateCommand implementation.
        if self.without_ovm {
            if self.init_state.env.chain.is_optimism_mainnet() {
                return self.execute_with_bedrock_header::<N>();
            }

            // For non-mainnet OP chains with --without-ovm, use the base implementation
            // by setting the without_evm flag
            self.init_state.without_evm = true;
        }

        self.init_state.execute::<N>().await
    }

    /// Execute init-state with hardcoded Bedrock header for OP mainnet.
    fn execute_with_bedrock_header<
        N: CliNodeTypes<ChainSpec = C::ChainSpec, Primitives = OpPrimitives>,
    >(
        self,
    ) -> eyre::Result<()> {
        info!(target: "reth::cli", "Reth init-state starting for OP mainnet");
        let env = self.init_state.env.init::<N>(AccessRights::RW)?;

        let Environment { config, provider_factory, .. } = env;
        let static_file_provider = provider_factory.static_file_provider();
        let provider_rw = provider_factory.database_provider_rw()?;

        let last_block_number = provider_rw.last_block_number()?;

        if last_block_number == 0 {
            reth_cli_commands::init_state::without_evm::setup_without_evm(
                &provider_rw,
                SealedHeader::new(BEDROCK_HEADER, BEDROCK_HEADER_HASH),
                |number| {
                    let mut header = Header::default();
                    header.set_number(number);
                    header
                },
            )?;

            // SAFETY: it's safe to commit static files, since in the event of a crash, they
            // will be unwound according to database checkpoints.
            //
            // Necessary to commit, so the BEDROCK_HEADER is accessible to provider_rw and
            // init_state_dump
            static_file_provider.commit()?;
        } else if last_block_number > 0 && last_block_number < BEDROCK_HEADER.number {
            return Err(eyre::eyre!(
                "Data directory should be empty when calling init-state with --without-ovm."
            ))
        }

        info!(target: "reth::cli", "Initiating state dump");

        let reader = BufReader::new(reth_fs_util::open(self.init_state.state)?);
        let hash = init_from_state_dump(reader, &provider_rw, config.stages.etl)?;

        provider_rw.commit()?;

        info!(target: "reth::cli", hash = ?hash, "Genesis block written");
        Ok(())
    }
}

impl<C: ChainSpecParser> InitStateCommandOp<C> {
    /// Returns the underlying chain being used to run this command
    pub fn chain_spec(&self) -> Option<&Arc<C::ChainSpec>> {
        self.init_state.chain_spec()
    }
}
