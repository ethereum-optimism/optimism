#![doc = include_str!("../README.md")]
#![warn(missing_debug_implementations, missing_docs, unreachable_pub, rustdoc::all)]
#![deny(unused_must_use, rust_2018_idioms)]
#![cfg_attr(docsrs, feature(doc_cfg, doc_auto_cfg))]
#![allow(clippy::type_complexity)]
#![cfg_attr(not(test), no_std)]

extern crate alloc;

pub mod fpvm_evm;
pub mod interop;
pub mod single;
