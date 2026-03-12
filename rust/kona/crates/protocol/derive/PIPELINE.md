# Derivation Pipeline Architecture

A `no_std` compatible implementation of the OP Stack's derivation pipeline that transforms L1 blockchain data into L2 payload attributes.

## Pipeline Stages

```
┌─────────────────────────────────────────────────────────────────────┐
│                        L1 Chain (Ethereum)                         │
│                  blocks, calldata, blobs, receipts                 │
└──────────────────────────────┬──────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│                     TRAVERSAL STAGE                                 │
│  PollingTraversal (active) │ IndexedTraversal (passive/signals)     │
│  Iterates L1 blocks sequentially, provides SystemConfig            │
└──────────────────────────────┬──────────────────────────────────────┘
                               │ BlockInfo + SystemConfig
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│                     L1 RETRIEVAL                                    │
│  Fetches raw calldata + blob data from L1 via DataAvailability-    │
│  Provider for the current L1 block                                 │
└──────────────────────────────┬──────────────────────────────────────┘
                               │ raw bytes
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│                     FRAME QUEUE                                     │
│  Parses raw bytes → Frame objects                                  │
│  Buffers in VecDeque<Frame>                                        │
│  Holocene+: prunes invalid frame sequences                         │
└──────────────────────────────┬──────────────────────────────────────┘
                               │ Frame
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│                     CHANNEL STAGES                                  │
│  ┌──────────────┐    ┌───────────────────┐    ┌──────────────────┐ │
│  │ ChannelBank  │───▶│ ChannelAssembler  │───▶│ ChannelProvider  │ │
│  │ (frame→ch)   │    │ (frames→channel)  │    │ (top-level)      │ │
│  └──────────────┘    └───────────────────┘    └──────────────────┘ │
└──────────────────────────────┬──────────────────────────────────────┘
                               │ Channel (compressed)
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│                     CHANNEL READER                                  │
│  Decompresses channel data (zlib/brotli/zstd)                      │
│  RLP-decodes into Batch objects (SingleBatch or SpanBatch)         │
└──────────────────────────────┬──────────────────────────────────────┘
                               │ Batch
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│                     BATCH STAGES                                    │
│  ┌──────────────────┐    ┌──────────────────┐                      │
│  │ BatchStream      │───▶│ BatchProvider    │                      │
│  │ (Holocene+:      │    │ (validates &     │                      │
│  │  SpanBatch →     │    │  orders batches) │                      │
│  │  SingleBatch[])  │    │                  │                      │
│  └──────────────────┘    └──────────────────┘                      │
└──────────────────────────────┬──────────────────────────────────────┘
                               │ SingleBatch
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│                     ATTRIBUTES QUEUE                                │
│  SingleBatch → OpPayloadAttributes                                 │
│  Uses AttributesBuilder to construct deposit-only base             │
│  Appends sequencer transactions from the batch                     │
│  Outputs OpAttributesWithParent                                    │
└──────────────────────────────┬──────────────────────────────────────┘
                               │ OpAttributesWithParent
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│                  DERIVATION PIPELINE (core.rs)                      │
│  Buffers attributes in VecDeque<OpAttributesWithParent>            │
│  Exposes step() / peek() / Iterator interface                      │
│  Handles Signal propagation (Reset, Activation, FlushChannel)      │
└──────────────────────────────┬──────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│                     L2 Execution Engine                             │
│              Executes payload → canonical L2 blocks                 │
└─────────────────────────────────────────────────────────────────────┘
```

## Signal Flow (reverse direction)

Signals propagate top-down using head-recursion: each stage signals its inner stage first, then clears its own state bottom-up.

```
DerivationPipeline
  │  Signal::Reset / Activation / FlushChannel / ProvideBlock
  ▼
AttributesQueue → BatchProvider → BatchStream → ChannelReader
  → ChannelProvider → FrameQueue → L1Retrieval → Traversal
```

## Ownership Model

Each stage **owns** the previous stage, forming a nested composition. The pipeline only holds a reference to the top-level `AttributesQueue`; all lower stages are accessed through trait-based delegation.

```
AttributesQueue<
  BatchProvider<
    BatchStream<
      ChannelReader<
        ChannelProvider<
          FrameQueue<
            L1Retrieval<
              PollingTraversal | IndexedTraversal
            >>>>>>>
```

## Pipeline Construction

```rust
PipelineBuilder::new()
    .rollup_config(cfg)
    .chain_provider(l1_provider)
    .l2_chain_provider(l2_provider)
    .dap_source(data_availability)
    .builder(attributes_builder)
    .origin(l1_origin)
    .build_polled()   // PollingTraversal (active)
 // .build_indexed()  // IndexedTraversal (passive)
```

## Key Traits

| Trait | Purpose |
|-------|---------|
| `SignalReceiver` | Receives pipeline signals |
| `OriginProvider` | Provides current L1 origin |
| `OriginAdvancer` | Advances to next L1 origin |
| `NextAttributes` | Top-level interface producing payload attributes |
| `ChainProvider` | L1 blockchain data (blocks, transactions, receipts) |
| `L2ChainProvider` | L2 blockchain data and system config |
| `DataAvailabilityProvider` | Raw L1 data (calldata, blobs) |
| `AttributesBuilder` | Constructs initial payload attributes |

## Error Handling

Error categories drive signal types:

- **Temporary Errors** (EOF, NotEnoughData) — Continue stepping, wait for more data
- **Reset Errors** (BadParentHash, HoloceneActivation, ReorgDetected) — Trigger Reset/Activation signal
- **Critical Errors** — Fatal, stop derivation
