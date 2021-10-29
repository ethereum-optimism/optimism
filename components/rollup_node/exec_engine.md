# L2 Execution Engine

## Overview

The L2 execution engine is where transaction execution actually takes place, and is where all L2 state is held. Its goal is to be as similar to L1 nodes as possible by using the Eth2 Merge Engine API. In normal ETH2, the consensus node sends ETH blocks from the beacon chain into the engine. In Optimistic Ethereum, the Rollup Driver (and perhaps other components, depending on the type of Rollup Node) sends L2 blocks that are canonically generated from L1 into the engine.

## Differences between L1 and L2 EE

1. **Deposit TX Type**: There is a new typed transaction which corresponds to deposits. This is a cross-chain message type; not propagated by the mempool, only insertable by the rollup node in correspondence to an on-chain event. This transaction type is also able to mint deposited ETH as new native L2 `account.value`.
2. **Deposit Insertability**: The TX type described above must be L1-authenticated and therefore cannot come from the unauthenticated mempool. This requires adding a `Transaction[]` argument to [`engine_perparePayload`](https://github.com/ethereum/execution-apis/blob/v1.0.0-alpha.2/src/engine/interop/specification.md#engine_preparepayload).
3. **Fee payment modification**: ETH is charged in addition to the `gasPrice * gasUsed`, to compensate the sequencer for rolling up the user transaction. This should be consistent with the [existing OVM 2.0 fee scheme](https://community.optimism.io/docs/users/fees-2.0.html), but using the L1 context information directly instead of via an oracle.
4. **EIP1559 modification**: Updates to the `BASEFEE` on L2 must be handled differently, in accordance to L1 time, as L2 blocktime may not be constant. (*Note: this modification may not be necessary; L1 devs are actively discussing modification of eip-1559 to handle "gap-slots" which would solve for the same timing.*)
5. **L1 Context Information**: The L1 blockhash, L1 blocknumber, and L1 basefee are frequently requested to be accessible within L2. *(TBD: it is possible to expose this information purely in a normal L2 contract, which is updated via the deposit TX type. This may minimize the diff more.)*
6. **Confirmation Status**: The Execution Engine will repurpose (but not otherwise modify) the three beacon chain confirmation strengths (`finalized, safe, unsafe`) to corresponding to (`l1_finalized, l1_confirmed, sequencer_confirmed`).