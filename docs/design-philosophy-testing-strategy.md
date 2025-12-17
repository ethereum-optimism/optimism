# Rollup-Boost: Design Philosophy and Testing Strategy

## Design Philosophy

Rollup-Boost occupies a critical position in the Op-stack architecture, serving as the bridge between the sequencer and the block builder. This positioning informs our entire design philosophy: **reliability above all else**.

At its core, Rollup-Boost exists to ensure system liveness. The sequencer must be able to produce blocks even when external components like dedicated builders experience issues. This critical responsibility guides our primary design principle:

**Minimize Complexity**: Rollup-Boost has a defined scope: it proxies and mirrors RPC requests for block building. While the system's position could enable additional features (like broadcasting transactions too), we deliberately limit scope creep. If a new feature introduces complexity and potential failure points, we carefully evaluate its inclusion. We prioritize rock-solid core functionality over feature richness. In particular, we focus on minimizing complexity on the hot path, keeping critical components like the proxy layer as simple and reliable as possible.

Looking ahead, we are considering restructuring Rollup-Boost into a modular rollup-boost-sdk library. This approach would make it easier to add different flavors or strategies safely. By extracting core functionality into a well-tested library, we could enable various implementations while maintaining the reliability of critical components. This architectural evolution would help balance innovation with our commitment to stability on the critical path.

### Architecture Decisions

The `src/proxy.rs` component contains the core logic of mirroring engine requests and stands as the most critical part of the system. We maintain this component with exceptional care, recognizing that changes here carry the highest risk. Despite its importance, this module remains relatively simple by design.

## Testing Strategy

We employ a layered testing strategy that provides defense in depth:

**Unit Tests** verify individual components, but as this is a distributed system, they only get us so far.

**Integration Tests** serve as our most critical testing layer. Located in `tests`, these tests use a simulated environment to verify system behavior under various conditions:

- How does the system respond when the builder produces invalid blocks?
- What happens when the builder experiences high latency?
- How does the system behave during execution mode transitions?
- Can the system recover when the builder becomes unavailable and later returns?

Each integration test creates a complete test environment with mock L2 and builder nodes, simulating real-world scenarios that exercise the system's resilience features. One limitation of this approach is that issues in the mock CL node might not surface in these tests, offering quick feedback but potentially missing certain edge cases.

**End-to-End Tests** (planned) will use actual components in a production-like environment. These tests using Builder Playground will help ensure our test assumptions match real-world behavior.
