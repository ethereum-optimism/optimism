# `kona-mpt`

<a href="https://github.com/op-rs/kona/actions/workflows/rust_ci.yaml"><img src="https://github.com/op-rs/kona/actions/workflows/rust_ci.yaml/badge.svg?label=ci" alt="CI"></a>
<a href="https://crates.io/crates/kona-mpt"><img src="https://img.shields.io/crates/v/kona-mpt.svg?label=kona-mpt&labelColor=2a2f35" alt="Kona MPT"></a>
<a href="https://github.com/op-rs/kona/blob/main/LICENSE.md"><img src="https://img.shields.io/badge/License-MIT-d1d1f6.svg?label=license&labelColor=2a2f35" alt="License"></a>
<a href="https://img.shields.io/codecov/c/github/op-rs/kona"><img src="https://img.shields.io/codecov/c/github/op-rs/kona" alt="Codecov"></a>

A recursive, in-memory implementation of Ethereum's hexary Merkle Patricia Trie (MPT), supporting:
- Retrieval
- Insertion
- Deletion
- Root Computation
    - Trie Node RLP Encoding

This implementation is intended to serve as a backend for a stateless executor of Ethereum blocks, like
the one in the [`kona-executor`](../executor) crate. Starting with a trie root, the `TrieNode` can be
unravelled to access, insert, or delete values. These operations are all backed by the `TrieProvider`,
which enables fetching the preimages of hashed trie nodes.
