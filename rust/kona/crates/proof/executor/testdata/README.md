# Kona executor fixtures

## `block-7-sdm-premium.tar.gz`

This fixture pins a non-empty SDM/PostExec transition produced by
`op-reth-premium`, not by the stock op-reth payload builder in this repository.

- Producer repository: `ethereum-optimism/optimism-premium`
- Producer commit: `10c2d76` (`main`)
- Producer's Optimism dependency: `dbaf2aed6a1ed52ca3bd3caba60bb896a7bf4ce1`
- Producer path: premium subblocks payload builder
- Capture harness: Optimism devstack at `ca40d43e51`
- Ephemeral chain IDs: L1 `900`, L2 `901`
- Fork schedule: Isthmus, Jovian, Karst, and Lagoon active at genesis
- Block number: `7`
- Block hash: `0xa128f899f6bff19247a2656364c3e440de4bc1292347bf3b8f6f4f717765c0b2`
- State root: `0x483d0a88146a80c0d5a592879f2d04f060b7515a1bad20f295fd4b601db20b93`
- Raw EVM gas: `1,515,842`; refund payload total: `630,000`; canonical gas: `885,842`

The premium sequencer built the block from the repeated-slot SDM workload. A
stock op-reth verifier imported the exact block and supplied execution-witness
and proof data to `execution-fixture`; the verifier did not build the block.
The fixture contains a trailing `0x7D` with non-empty, non-zero refund entries.
