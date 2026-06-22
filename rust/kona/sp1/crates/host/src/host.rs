//! Defines the [`OPSuccinctHost`] trait for the kona-sp1 host.

use alloy_primitives::B256;
use anyhow::Result;
use async_trait::async_trait;
use kona_host::single::{SingleChainHost, SingleChainHostError};
use kona_preimage::{BidirectionalChannel, Channel};
use tokio::task::JoinHandle;

use crate::{fetcher::OPSuccinctDataFetcher, witness_generation::WitnessGenerator};

/// Trait for starting a preimage server.
#[async_trait]
pub trait PreimageServerStarter {
    /// Start the preimage server.
    async fn start_server<C>(
        &self,
        hint: C,
        preimage: C,
    ) -> Result<JoinHandle<Result<(), SingleChainHostError>>, SingleChainHostError>
    where
        C: Channel + Send + Sync + 'static;
}

#[async_trait]
impl PreimageServerStarter for SingleChainHost {
    async fn start_server<C>(
        &self,
        hint: C,
        preimage: C,
    ) -> Result<JoinHandle<Result<(), SingleChainHostError>>, SingleChainHostError>
    where
        C: Channel + Send + Sync + 'static,
    {
        self.start_server(hint, preimage).await
    }
}

/// Trait representing an OP Succinct Host.
#[async_trait]
pub trait OPSuccinctHost: Send + Sync + 'static {
    /// The type of arguments required to run the host.
    type Args: Send + Sync + 'static + Clone + PreimageServerStarter;
    /// The type of witness generator used by the host.
    type WitnessGenerator: WitnessGenerator + Send + Sync;

    /// Returns the witness generator.
    fn witness_generator(&self) -> &Self::WitnessGenerator;

    /// Fetch the host arguments.
    ///
    /// Parameters:
    /// - `l2_start_block`: The starting L2 block number.
    /// - `l2_end_block`: The ending L2 block number.
    /// - `l1_head_hash`: Optionally supplied L1 head block hash used as the L1 origin.
    /// - `safe_db_fallback`: Flag to indicate whether to fallback to timestamp-based L1 head
    ///   estimation when `SafeDB` is not available.
    async fn fetch(
        &self,
        l2_start_block: u64,
        l2_end_block: u64,
        l1_head_hash: Option<B256>,
        safe_db_fallback: bool,
    ) -> Result<Self::Args>;

    /// Run the host and client program.
    ///
    /// Returns the witness which can be supplied to the zkVM.
    async fn run(
        &self,
        args: &Self::Args,
    ) -> Result<<Self::WitnessGenerator as WitnessGenerator>::WitnessData> {
        let preimage = BidirectionalChannel::new()?;
        let hint = BidirectionalChannel::new()?;

        let server_task = args.start_server(hint.host, preimage.host).await?;

        let witness = self.witness_generator().run(preimage.client, hint.client).await?;
        // Unlike the upstream, manually abort the server task, as it will hang if you wait for both
        // tasks to complete.
        server_task.abort();

        Ok(witness)
    }

    /// Get the L1 head hash from the host args.
    fn get_l1_head_hash(&self, args: &Self::Args) -> Option<B256>;

    /// Get the finalized L2 block number. This is used to determine the highest block that can be
    /// included in a range proof.
    ///
    /// For ETH DA, this is the finalized L2 block number.
    ///
    /// The latest proposed block number is assumed to be the highest block number that has been
    /// successfully processed by the host.
    async fn get_finalized_l2_block_number(
        &self,
        fetcher: &OPSuccinctDataFetcher,
        latest_proposed_block_number: u64,
    ) -> Result<Option<u64>>;

    /// Calculate a safe L1 head hash for the given L2 end block.
    ///
    /// This method is DA-specific:
    /// - For ETH DA: Uses simple offset logic.
    ///
    /// Parameters:
    /// - `fetcher`: The data fetcher for accessing blockchain data.
    /// - `l2_end_block`: The ending L2 block number for the range.
    /// - `safe_db_fallback`: Whether to fallback to timestamp-based estimation when `SafeDB` is
    ///   unavailable.
    async fn calculate_safe_l1_head(
        &self,
        fetcher: &OPSuccinctDataFetcher,
        l2_end_block: u64,
        safe_db_fallback: bool,
    ) -> Result<B256>;
}
