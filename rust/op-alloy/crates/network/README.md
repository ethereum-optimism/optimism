## `op-alloy-network`

<a href="https://github.com/alloy-rs/op-alloy/actions/workflows/ci.yml"><img src="https://github.com/alloy-rs/op-alloy/actions/workflows/ci.yml/badge.svg?label=ci" alt="CI"></a>
<a href="https://crates.io/crates/op-alloy-network"><img src="https://img.shields.io/crates/v/op-alloy-network.svg" alt="op-alloy-network crate"></a>
<a href="https://github.com/alloy-rs/op-alloy/blob/main/LICENSE-MIT"><img src="https://img.shields.io/badge/License-MIT-d1d1f6.svg?label=license&labelColor=2a2f35" alt="MIT License"></a>
<a href="https://github.com/alloy-rs/op-alloy/blob/main/LICENSE-APACHE"><img src="https://img.shields.io/badge/License-APACHE-d1d1f6.svg?label=license&labelColor=2a2f35" alt="Apache License"></a>
<a href="https://alloy-rs.github.io/op-alloy"><img src="https://img.shields.io/badge/Book-854a15?logo=mdBook&labelColor=2a2f35" alt="Book"></a>


Optimism blockchain RPC behavior abstraction.

This crate contains a simple abstraction of the RPC behavior of an
Op-stack blockchain. It is intended to be used by the Alloy client to
provide a consistent interface to the rest of the library, regardless of
changes the underlying blockchain makes to the RPC interface.
