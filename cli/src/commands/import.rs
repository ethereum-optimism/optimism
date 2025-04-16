//! Command that initializes the node by importing OP Mainnet chain segment below Bedrock, from a
//! file.
use clap::Parser;
use reth_cli::chainspec::ChainSpecParser;
use reth_cli_commands::{
    common::{AccessRights, CliNodeTypes, Environment, EnvironmentArgs},
    import::build_import_pipeline,
};
use reth_consensus::noop::NoopConsensus;
use reth_db_api::{tables, transaction::DbTx};
use reth_downloaders::file_client::{ChunkedFileReader, DEFAULT_BYTE_LEN_CHUNK_CHAIN_FILE};
use reth_node_builder::BlockTy;
use reth_node_core::version::SHORT_VERSION;
use reth_optimism_chainspec::OpChainSpec;
use reth_optimism_evm::OpExecutorProvider;
use reth_optimism_primitives::{bedrock::is_dup_tx, OpPrimitives};
use reth_provider::{BlockNumReader, ChainSpecProvider, HeaderProvider, StageCheckpointReader};
use reth_prune::PruneModes;
use reth_stages::StageId;
use reth_static_file::StaticFileProducer;
use std::{path::PathBuf, sync::Arc};
use tracing::{debug, error, info};

/// Syncs RLP encoded blocks from a file.
#[derive(Debug, Parser)]
pub struct ImportOpCommand<C: ChainSpecParser> {
    #[command(flatten)]
    env: EnvironmentArgs<C>,

    /// Chunk byte length to read from file.
    #[arg(long, value_name = "CHUNK_LEN", verbatim_doc_comment)]
    chunk_len: Option<u64>,

    /// The path to a block file for import.
    ///
    /// The online stages (headers and bodies) are replaced by a file import, after which the
    /// remaining stages are executed.
    #[arg(value_name = "IMPORT_PATH", verbatim_doc_comment)]
    path: PathBuf,
}

impl<C: ChainSpecParser<ChainSpec = OpChainSpec>> ImportOpCommand<C> {
    /// Execute `import` command
    pub async fn execute<N: CliNodeTypes<ChainSpec = C::ChainSpec, Primitives = OpPrimitives>>(
        self,
    ) -> eyre::Result<()> {
        info!(target: "reth::cli", "reth {} starting", SHORT_VERSION);

        info!(target: "reth::cli",
            "Disabled stages requiring state, since cannot execute OVM state changes"
        );

        debug!(target: "reth::cli",
            chunk_byte_len=self.chunk_len.unwrap_or(DEFAULT_BYTE_LEN_CHUNK_CHAIN_FILE),
            "Chunking chain import"
        );

        let Environment { provider_factory, config, .. } = self.env.init::<N>(AccessRights::RW)?;

        // we use noop here because we expect the inputs to be valid
        let consensus = Arc::new(NoopConsensus::default());

        // open file
        let mut reader = ChunkedFileReader::new(&self.path, self.chunk_len).await?;

        let mut total_decoded_blocks = 0;
        let mut total_decoded_txns = 0;
        let mut total_filtered_out_dup_txns = 0;

        let mut sealed_header = provider_factory
            .sealed_header(provider_factory.last_block_number()?)?
            .expect("should have genesis");

        while let Some(mut file_client) =
            reader.next_chunk::<BlockTy<N>>(consensus.clone(), Some(sealed_header)).await?
        {
            // create a new FileClient from chunk read from file
            info!(target: "reth::cli",
                "Importing chain file chunk"
            );

            let tip = file_client.tip().ok_or_else(|| eyre::eyre!("file client has no tip"))?;
            info!(target: "reth::cli", "Chain file chunk read");

            total_decoded_blocks += file_client.headers_len();
            total_decoded_txns += file_client.total_transactions();

            for (block_number, body) in file_client.bodies_iter_mut() {
                body.transactions.retain(|_| {
                    if is_dup_tx(block_number) {
                        total_filtered_out_dup_txns += 1;
                        return false
                    }
                    true
                })
            }

            let (mut pipeline, events) = build_import_pipeline(
                &config,
                provider_factory.clone(),
                &consensus,
                Arc::new(file_client),
                StaticFileProducer::new(provider_factory.clone(), PruneModes::default()),
                true,
                OpExecutorProvider::optimism(provider_factory.chain_spec()),
            )?;

            // override the tip
            pipeline.set_tip(tip);
            debug!(target: "reth::cli", ?tip, "Tip manually set");

            let provider = provider_factory.provider()?;

            let latest_block_number =
                provider.get_stage_checkpoint(StageId::Finish)?.map(|ch| ch.block_number);
            tokio::spawn(reth_node_events::node::handle_events(None, latest_block_number, events));

            // Run pipeline
            info!(target: "reth::cli", "Starting sync pipeline");
            tokio::select! {
                res = pipeline.run() => res?,
                _ = tokio::signal::ctrl_c() => {},
            }

            sealed_header = provider_factory
                .sealed_header(provider_factory.last_block_number()?)?
                .expect("should have genesis");
        }

        let provider = provider_factory.provider()?;

        let total_imported_blocks = provider.tx_ref().entries::<tables::HeaderNumbers>()?;
        let total_imported_txns = provider.tx_ref().entries::<tables::TransactionHashNumbers>()?;

        if total_decoded_blocks != total_imported_blocks ||
            total_decoded_txns != total_imported_txns + total_filtered_out_dup_txns
        {
            error!(target: "reth::cli",
                total_decoded_blocks,
                total_imported_blocks,
                total_decoded_txns,
                total_filtered_out_dup_txns,
                total_imported_txns,
                "Chain was partially imported"
            );
        }

        info!(target: "reth::cli",
            total_imported_blocks,
            total_imported_txns,
            total_decoded_blocks,
            total_decoded_txns,
            total_filtered_out_dup_txns,
            "Chain file imported"
        );

        Ok(())
    }
}

impl<C: ChainSpecParser> ImportOpCommand<C> {
    /// Returns the underlying chain being used to run this command
    pub const fn chain_spec(&self) -> Option<&Arc<C::ChainSpec>> {
        Some(&self.env.chain)
    }
}
