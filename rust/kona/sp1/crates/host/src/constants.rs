//! Constants for the sp1 host.

use cargo_metadata::MetadataCommand;
use lazy_static::lazy_static;
use std::path::PathBuf;

fn get_workspace_root() -> PathBuf {
    let metadata = MetadataCommand::new().exec().unwrap();
    metadata.workspace_root.into()
}

lazy_static! {
    /// Path to the kona SP1 Fault Dispute Game configuration file.
    pub static ref KONA_SP1_FAULT_DISPUTE_GAME_CONFIG_PATH: PathBuf = {
        std::env::var("KONA_SP1_FAULT_DISPUTE_GAME_CONFIG_PATH")
            .ok()
            .map(PathBuf::from)
            .unwrap_or_else(|| get_workspace_root().join("contracts").join("konasp1fdgconfig.json"))
    };
}
