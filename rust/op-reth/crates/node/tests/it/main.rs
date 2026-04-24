// Integration tests bring in a wide set of on-chain test literals, panicking test helpers,
// and deep trait bounds from reth-test-utils. Keep the pedantic-noise allows here scoped
// to the test runner so the crate lib stays clean while tests remain readable.
#![allow(
    clippy::unreadable_literal,
    clippy::default_trait_access,
    clippy::missing_panics_doc,
    clippy::missing_errors_doc,
    clippy::too_many_lines,
    clippy::items_after_statements,
    clippy::large_futures,
    clippy::wildcard_imports,
    clippy::uninlined_format_args,
    clippy::cast_possible_truncation,
    clippy::cast_sign_loss,
    clippy::significant_drop_tightening,
    reason = "integration tests; match on-chain values verbatim and rely on test-utils panics"
)]
#![allow(missing_docs)]

mod builder;

mod priority;

mod rpc;

mod custom_genesis;

const fn main() {}
