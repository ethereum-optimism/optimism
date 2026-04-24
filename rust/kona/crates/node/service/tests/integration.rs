//! Integration tests for the node service crate.
// Integration tests use `pub(crate)` freely across nested modules; suppressing
// these noisy stylistic lints keeps the test support code readable.
#![allow(clippy::redundant_pub_crate, clippy::default_trait_access)]

/// Tests for the node actors.
mod actors;
