mod apis;
mod contracts;
mod driver;
mod external;
mod instance;
mod txs;
mod utils;

use alloy_primitives::{B256, b256};
pub use apis::*;
pub use contracts::*;
pub use driver::*;
pub use external::*;
pub use instance::*;
pub use txs::*;
pub use utils::*;

// anvil default key[1]
pub const BUILDER_PRIVATE_KEY: &str =
    "0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d";
// anvil default key[0]
pub const FUNDED_PRIVATE_KEY: &str =
    "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80";
// anvil default key[8]
pub const FLASHBLOCKS_DEPLOY_KEY: &str =
    "0xdbda1821b80551c9d65939329250298aa3472ba22feea921c0cf5d620ea67b97";
// anvil default key[9]
pub const FLASHTESTATION_DEPLOY_KEY: &str =
    "0x2a871d0798f97d79848a013d4936a73bf4cc922c825d33c1cf7073dff6d409c6";

pub const DEFAULT_GAS_LIMIT: u64 = 10_000_000;

pub const DEFAULT_DENOMINATOR: u32 = 50;

pub const DEFAULT_ELASTICITY: u32 = 2;
pub const DEFAULT_JWT_TOKEN: &str =
    "688f5d737bad920bdfb2fc2f488d6b6209eebda1dae949a8de91398d932c517a";

pub const ONE_ETH: u128 = 1_000_000_000_000_000_000;

// flashtestations constants
pub const TEE_DEBUG_ADDRESS: alloy_primitives::Address =
    alloy_primitives::address!("6Af149F267e1e62dFc431F2de6deeEC7224746f4");

pub const WORKLOAD_ID: B256 =
    b256!("952569f637f3f7e36cd8f5a7578ae4d03a1cb05ddaf33b35d3054464bb1c862e");

pub const SOURCE_LOCATORS: &[&str] = &[
    "https://github.com/flashbots/flashbots-images/commit/53d431f58a0d1a76f6711518ef8d876ce8181fc2",
];

pub const COMMIT_HASH: &str = "53d431f58a0d1a76f6711518ef8d876ce8181fc2";

/// This gets invoked before any tests, when the cargo test framework loads the test library.
/// It injects itself into
#[ctor::ctor]
fn init_tests() {
    use tracing_subscriber::{filter::filter_fn, prelude::*};
    if let Ok(v) = std::env::var("TEST_TRACE") {
        let level = match v.as_str() {
            "false" | "off" => return,
            "true" | "debug" | "on" => tracing::Level::DEBUG,
            "trace" => tracing::Level::TRACE,
            "info" => tracing::Level::INFO,
            "warn" => tracing::Level::WARN,
            "error" => tracing::Level::ERROR,
            _ => return,
        };

        // let prefix_blacklist = &["alloy_transport_ipc", "storage::db::mdbx"];
        let prefix_blacklist = &["storage::db::mdbx"];

        tracing_subscriber::registry()
            .with(tracing_subscriber::fmt::layer())
            .with(filter_fn(move |metadata| {
                metadata.level() <= &level
                    && !prefix_blacklist
                        .iter()
                        .any(|prefix| metadata.target().starts_with(prefix))
            }))
            .init();
    }

    #[cfg(not(windows))]
    let _ = rlimit::setrlimit(rlimit::Resource::NOFILE, 500_000, 500_000);
}
