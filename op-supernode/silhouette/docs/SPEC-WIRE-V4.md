# Silhouette proof-batch wire format v4

**Status: NORMATIVE. This is the version this branch produces.** The Go implementation is
`op-supernode/proofbatch`. The frozen cross-language fixture corpus currently pins v3; v4's appended
fields are pinned by Go layout/round-trip tests until the second implementation regenerates it.

Wire v4 is wire v3 plus the three block-reference fields required to reproduce an op-node
`L2BlockRef` exactly. All v3 rules for exported logs, imported messages, canonical ordering,
proof-slot handling and acceptance remain unchanged.

## 1. Why v4 exists

An L2 block hash does not encode the op-node reference metadata used by follow-mode LightCLs. In
particular, a block can have the expected hash while a verifier has heuristically assigned a
different L1 origin or sequence number. Stock follow-source logic correctly treats those references
as different chains and resets the unsafe head.

V4 removes that inference. Each block explicitly carries the public L1-info values from which its
canonical reference is formed:

- L1 origin hash;
- L1 origin number;
- L1 sequence number.

These values disclose no private transaction data. They are already present in the mandatory L1-info
deposit at the front of the real L2 block.

## 2. Envelope

```text
magic         = "KCPB"          (4 bytes, ASCII)
version       = 0x04            (u8)
pv_len        = u32 big-endian
public_values = pv_len bytes     (ABI encoding in section 3)
proof_len     = u32 big-endian
proof         = proof_len bytes  (empty in attested mode)
```

A verifier accepts exactly the version configured for that chain. `DecodeAny` support for older
versions is diagnostic and migration support; it is not permission for a verifier to accept more
than one layout.

## 3. Public values

The public values are `abi.encode(ProofBatch)`:

```solidity
struct ProofBatch {
    bytes32 prevOutputRoot;
    bytes32 newOutputRoot;
    bytes32 l1Head;
    bytes32 rollupConfigHash;
    bytes32 depSetHash;
    bytes32 exportPolicyHash;
    BlockExport[] blocks;
}

struct BlockExport {
    uint64 blockNumber;
    uint64 timestamp;
    bytes32 blockHash;
    bytes32 stateRoot;
    bytes32 messagePasserStorageRoot;
    LogExport[] logs;
    ExecMsg[] execMsgs;
    bytes32 l1OriginHash;      // new in v4
    uint64 l1OriginNumber;     // new in v4
    uint64 sequenceNumber;     // new in v4
}

struct LogExport {
    uint32 logIndex;
    bytes32 logHash;
    bytes preimage;
}

struct ExecMsg {
    Identifier id;
    bytes32 msgHash;
}

struct Identifier {
    address origin;
    uint64 blockNumber;
    uint32 logIndex;
    uint64 timestamp;
    uint256 chainId;
}
```

The three v4 fields are appended after `execMsgs`; no v3 field moves. Blocks are consecutive and in
ascending number/timestamp order. Exported logs and imported messages retain all structural and
canonical-order rules from `SPEC-WIRE-V3.md`.

## 4. Producer rule

For every exported block, the producer reads `l1OriginHash`, `l1OriginNumber`, and `sequenceNumber`
from the rollup node's `OutputResponse.BlockRef`. It must also verify that the response describes the
same block number, hash and timestamp as the execution payload. It must not infer these fields from
the proof-batch carrier block or from neighboring exports.

## 5. Verifier rule

In addition to the v3 acceptance rules, the verifier checks each explicit L1 origin against its
canonical L1 source and enforces ordinary rollup progression:

- the origin exists canonically at `(number, hash)`;
- an origin never moves backwards;
- it stays the same or advances by exactly one L1 block;
- `sequenceNumber` is zero when the origin advances;
- otherwise `sequenceNumber` increments by one;
- the L2 timestamp satisfies the rollup config's L1-origin and drift constraints.

The accepted values are recorded directly in the proven fact and returned by the magic EL. No
heuristic reconstruction is allowed for v4.

## 6. One-block overlap

The normal batcher may begin after a verifier-built Holocene replacement because that replacement is
already local-safe. To make the replacement itself proof-committed, a v4 channel may prepend exactly
one block equal to the verifier's current proven head.

The verifier trims that block only when every committed field matches its current fact: number,
timestamp, block hash, state root, message-passer root, output root, L1 origin, sequence number and
import list. Any other overlap, any overlap longer than one block, or any mismatch is rejected.

## 7. Data deliberately absent

The proof batch contains no ordinary transaction bytes or hashes, transaction senders, calldata,
receipts, contract state, or full log contents under the all-hashes export policy. The L1 blob
transaction envelope still publicly reveals the submitter, inbox, nonce, fee fields, blob
commitments and L1 inclusion block.
