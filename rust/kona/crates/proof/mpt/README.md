# `kona-mpt`

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
