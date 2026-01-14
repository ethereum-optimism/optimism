#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/op-rs/kona/main/assets/square.png",
    html_favicon_url = "https://raw.githubusercontent.com/op-rs/kona/main/assets/favicon.ico",
    issue_tracker_base_url = "https://github.com/op-rs/kona/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg, doc_auto_cfg))]
#![cfg_attr(target_arch = "mips64", feature(asm_experimental_arch))]
#![cfg_attr(any(target_arch = "mips64", target_arch = "riscv64"), no_std)]

extern crate alloc;

pub mod errors;

pub mod io;

#[cfg(feature = "tracing")]
pub mod tracing;

pub mod malloc;

mod traits;
pub use traits::BasicKernelInterface;

mod types;
pub use types::FileDescriptor;

mod channel;
pub use channel::FileChannel;

pub(crate) mod linux;

#[cfg(target_arch = "mips64")]
pub(crate) mod mips64;

#[cfg(target_arch = "riscv64")]
pub(crate) mod riscv64;
