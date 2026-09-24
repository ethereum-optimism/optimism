#![doc = include_str!("../README.md")]
#![warn(missing_debug_implementations, missing_docs, unreachable_pub, rustdoc::all)]
#![deny(unused_must_use, rust_2018_idioms)]
#![cfg_attr(docsrs, feature(doc_cfg))]
#![allow(clippy::type_complexity)]
#![cfg_attr(not(test), no_std)]

extern crate alloc;

// The fault-proof program admits `sp1-private-projection-v1` spans exactly as the nodes do:
// without the verifier every sp1 span would be dropped here and admitted by op-node and
// kona-node. Enabled by this crate's own dependency declaration, not by feature unification.
const _: () = assert!(
    kona_protocol::projection::SP1_PROJECTION_VERIFIER_COMPILED,
    "kona-protocol must be built with `sp1-projection-verifier`"
);

pub mod fpvm_evm;
pub mod interop;
pub mod single;
