//! Defines the Host implementation for Ethereum DA using single chain host.

use std::sync::Arc;

use crate::witness_generator::ETHDAWitnessGenerator;
use alloy_eips::BlockId;
use alloy_primitives::B256;
use anyhow::Result;
use async_trait::async_trait;
use kona_host::single::SingleChainHost;
use kona_sp1_host_utils::{fetcher::OPSuccinctDataFetcher, host::OPSuccinctHost};

/// Host implementation for Ethereum DA using single chain host.
#[expect(missing_debug_implementations)]
#[derive(Clone)]
pub struct SingleChainOPSuccinctHost {
    fetcher: Arc<OPSuccinctDataFetcher>,
    witness_generator: Arc<ETHDAWitnessGenerator>,
}

#[async_trait]
impl OPSuccinctHost for SingleChainOPSuccinctHost {
    type Args = SingleChainHost;
    type WitnessGenerator = ETHDAWitnessGenerator;

    fn witness_generator(&self) -> &Self::WitnessGenerator {
        &self.witness_generator
    }

    async fn fetch(
        &self,
        l2_start_block: u64,
        l2_end_block: u64,
        l1_head_hash: Option<B256>,
        safe_db_fallback: bool,
    ) -> Result<SingleChainHost> {
        // Calculate L1 head hash using simple logic if not provided.
        let l1_head_hash = match l1_head_hash {
            Some(hash) => hash,
            None => {
                self.calculate_safe_l1_head(&self.fetcher, l2_end_block, safe_db_fallback).await?
            }
        };

        let host = self.fetcher.get_host_args(l2_start_block, l2_end_block, l1_head_hash).await?;
        Ok(host)
    }

    fn get_l1_head_hash(&self, args: &Self::Args) -> Option<B256> {
        Some(args.l1_head)
    }

    async fn get_finalized_l2_block_number(
        &self,
        fetcher: &OPSuccinctDataFetcher,
        _: u64,
    ) -> Result<Option<u64>> {
        let finalized_l2_block_number = fetcher.get_l2_header(BlockId::finalized()).await?;
        Ok(Some(finalized_l2_block_number.number))
    }

    async fn calculate_safe_l1_head(
        &self,
        fetcher: &OPSuccinctDataFetcher,
        l2_end_block: u64,
        safe_db_fallback: bool,
    ) -> Result<B256> {
        // For Ethereum DA, use a simple approach with minimal offset.
        let (_, l1_head_number) = fetcher.get_l1_head(l2_end_block, safe_db_fallback).await?;

        // TODO(#18503): Investigate requirement for L1 head offset beyond batch posting block
        // with safe head > L2 end block.
        // Add a small buffer for Ethereum DA.
        let l1_head_number = l1_head_number + 20;

        // Ensure we don't exceed the finalized L1 header.
        let finalized_l1_header = fetcher.get_l1_header(BlockId::finalized()).await?;
        let safe_l1_head_number = std::cmp::min(l1_head_number, finalized_l1_header.number);

        Ok(fetcher.get_l1_header(safe_l1_head_number.into()).await?.hash_slow())
    }
}

impl SingleChainOPSuccinctHost {
    /// Creates a new [`SingleChainOPSuccinctHost`] instance.
    pub fn new(fetcher: Arc<OPSuccinctDataFetcher>) -> Self {
        Self { fetcher, witness_generator: Arc::new(ETHDAWitnessGenerator::new()) }
    }
}
