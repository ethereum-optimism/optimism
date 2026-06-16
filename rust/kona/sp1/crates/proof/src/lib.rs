//! Proof-related utilities for SP1.

use std::sync::Arc;

use kona_sp1_elfs::RANGE_ELF_EMBEDDED;
use kona_sp1_ethereum_host_utils::host::SingleChainOPSuccinctHost;
use kona_sp1_host_utils::fetcher::OPSuccinctDataFetcher;

/// Get the range ELF depending on the feature flag.
pub const fn get_range_elf_embedded() -> &'static [u8] {
    RANGE_ELF_EMBEDDED
}

/// Initialize the default (ETH-DA) host.
pub fn initialize_host(fetcher: Arc<OPSuccinctDataFetcher>) -> Arc<SingleChainOPSuccinctHost> {
    tracing::info!("Initializing host with Ethereum DA");
    Arc::new(SingleChainOPSuccinctHost::new(fetcher))
}
