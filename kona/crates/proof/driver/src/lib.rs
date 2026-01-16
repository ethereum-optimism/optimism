#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/op-rs/kona/main/assets/square.png",
    html_favicon_url = "https://raw.githubusercontent.com/op-rs/kona/main/assets/favicon.ico",
    issue_tracker_base_url = "https://github.com/op-rs/kona/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg, doc_auto_cfg))]
#![cfg_attr(not(test), no_std)]

extern crate alloc;

#[macro_use]
extern crate tracing;

mod errors;
pub use errors::{DriverError, DriverResult};

mod pipeline;
pub use pipeline::DriverPipeline;

mod executor;
pub use executor::Executor;

mod core;
pub use core::Driver;

mod cursor;
pub use cursor::PipelineCursor;

mod tip;
pub use tip::TipCursor;
