//! Integration tests.
//!
//! All the individual modules are rooted here to produce a single binary.

// Integration-test plumbing relies on fixture builders, timing casts, and async
// harnesses where the pedantic/nursery categories below fire pervasively.
#![allow(
    clippy::needless_pass_by_value,
    clippy::significant_drop_tightening,
    clippy::similar_names,
    clippy::items_after_statements,
    clippy::too_many_lines,
    clippy::missing_panics_doc,
    clippy::missing_errors_doc,
    clippy::unreadable_literal,
    clippy::uninlined_format_args,
    clippy::default_trait_access,
    clippy::cast_possible_truncation,
    clippy::cast_possible_wrap,
    clippy::cast_sign_loss,
    clippy::cast_precision_loss,
    clippy::unused_async,
    clippy::unnecessary_wraps,
    clippy::used_underscore_binding,
    clippy::future_not_send,
    clippy::large_futures
)]

mod harness;
mod service;
mod stream;
