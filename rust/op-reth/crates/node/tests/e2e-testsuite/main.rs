// E2E tests are heavy and wrap upstream test-utils with many on-chain literals, panicking
// helpers, and deep trait bounds. Keep the pedantic-noise allows scoped to the test runner.
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
    reason = "e2e tests; match on-chain values verbatim and rely on test-utils panics"
)]
#![allow(missing_docs)]

mod p2p;
mod testsuite;

const fn main() {}
