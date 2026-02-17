//! Shows how to manually access the database

use reth_op::{chainspec::BASE_MAINNET, node::OpNode, provider::providers::ReadOnlyConfig};

// Providers are zero-cost abstractions on top of an opened MDBX Transaction
// exposing a familiar API to query the chain's information without requiring knowledge
// of the inner tables.
//
// These abstractions do not include any caching and the user is responsible for doing that.
// Other parts of the code which include caching are parts of the `EthApi` abstraction.
fn main() -> eyre::Result<()> {
    // The path to data directory, e.g. "~/.local/reth/share/base"
    let datadir = std::env::var("RETH_DATADIR")?;

    // Instantiate a provider factory for Ethereum mainnet using the provided datadir path.
    let factory = OpNode::provider_factory_builder()
        .open_read_only(BASE_MAINNET.clone(), ReadOnlyConfig::from_datadir(datadir))?;

    // obtain a provider access that has direct access to the database.
    let _provider = factory.provider();

    Ok(())
}
