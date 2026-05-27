//! Re-runs the `op-superchain` build when the committed zip changes.
//!
//! `include_bytes!` does not register file dependencies with cargo, so any
//! refresh of `data/superchain-configs.zip` (via `cargo run -p op-superchain
//! --bin regenerate`) must be declared here for downstream consumers to see
//! the new bytes.

fn main() {
    println!("cargo:rerun-if-changed=data/superchain-configs.zip");
    println!("cargo:rerun-if-changed=build.rs");
}
