//! Proof-related utilities for SP1.

use std::sync::Arc;

use kona_sp1_elfs::{AGGREGATION_ELF, RANGE_ELF};
use kona_sp1_ethereum_host_utils::host::SingleChainOPSuccinctHost;
use kona_sp1_host_utils::fetcher::OPSuccinctDataFetcher;

/// Get the range ELF.
pub const fn get_range_elf() -> &'static [u8] {
    RANGE_ELF
}

/// Get the aggregation ELF.
pub const fn get_agg_elf() -> &'static [u8] {
    AGGREGATION_ELF
}

/// Initialize the default (ETH-DA) host.
pub fn initialize_host(fetcher: Arc<OPSuccinctDataFetcher>) -> Arc<SingleChainOPSuccinctHost> {
    tracing::info!("Initializing host with Ethereum DA");
    Arc::new(SingleChainOPSuccinctHost::new(fetcher))
}
