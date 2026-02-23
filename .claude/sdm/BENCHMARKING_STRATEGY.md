# SDM Benchmarking Strategy

19 February 2026

### Motivation for and against SDM

*(or at least PoC1; or why wall-clock is insufficient?)*

SDM / adaptive / multidimensional gas does not make blocks bigger, it makes blocks fully utilized instead of worst-case limited.

Ethereum today is limited by a single bottleneck assumption:

**block limit = safe under worst possible transaction mix**

That is extremely conservative.

One scalar gas limit for all.

So the block must be safe for the worst-case transaction repeated 30M times.
**Result: massive idle capacity**

---

However nodes (validators and sequencers) are not limited by one dimension (CPU).

A blockchain node is not a CPU — it’s a distributed verifier with asymmetric costs across participants and across time.

---

**Current Ethereum (single gas)**

Worst-case resource dominates:

Throughput = min(CPU, Storage, Bandwidth) ~= Storage

So CPU sits idle.

**Multidimensional Ethereum (SDM)**

We fill the unused dimensions:

Throughput ~= CPU + Storage + Bandwidth

We try to approach the ***sum*** instead of the **minimum**.

---

### 1. Determine `actors` sets - `good` and `bad` actors

This is subjective.

We should establish a list of `contract addresses` and/or `from addresses` that constitute as `bad` actors and `good` actors, and any interaction with them should be considered `good` or `bad`.

For example:

1. Interactions with Uniswap routers, pools, etc. constitutes as `good`
2. Interactions with XEN (on Base) constitutes as `bad`

### 2a. Implement and run `replay` command

1. Pick a target chain - OPM or Base or Unichain.
2. Run an archive OP Stack node for the target chain.
3. Implement a **replay** command in the forked **op-geth**:
    - takes `--from-block` and `--to-block`
    - runs sequentially, measures wall-clock time in the core execution path in the EVM - `core/state_transition.go: func innerExecute`
        - Execute txs with *canonical gas semantics* for correctness (so roots match).
        - Additionally measure wall time and compute  “opgas gasUsed” purely as a metric.
    - writes line-delimited JSON results
4. Verify correctness by comparing **block state roots**.
5. Scale up ranges + repetitions; export to Parquet/CSV; plot distributions.

### 2b. Implement and run sync for a given chain with two EVMs emitting metrics

1. Pick a target chain - OPM or Base or Unichain.
2. Sync the chain from genesis or given snapshot
3. Execute blocks and transactions with both EVMs and emit metrics:
    1. existing EVM
    2. opgas EVM

### 3. Visualize results from `replay` command and `actors` sets

1. For each transaction in the `good` and `bad` actors set, visualize how the opgas model is affecting them - whether it is refunding gas for `good` transactions, and what the effects are on `bad` transactions.
2. Build benchmark reports that answer “does wall-time gas behave sanely?”
    1. distributions:
        1. wall-time per tx (p50/p95/p99)
        2. gas refund distribution (p50/p95/p99) - shows if refunds are edge cases or systemic
        3. wall-time per block
    2. correlations: correlate wall time with:
        1. calldata size
        2. number of storage writes (SSTORE)
        3. cold/warm access patterns
        4. external calls / depth
        5. precompiles used

### 4. Define worst offenders

Generate contracts and transactions that are `bad` and have gas refunded by a bloating state (many SSTORE calls, but transaction executes fast?) and benchmark them against the existing gas model.

### 4.1. State growth vs runtime

| **operation** | **CPU** | **network** |
| --- | --- | --- |
| Storage write | 5 ms | forever state growth |
| Hash computation | 5 ms | none |

### 4.2. **Network propagation**

| **operation** | **CPU** | **network** |
| --- | --- | --- |
| calldata blob | tiny | massive |
| zk proof verify | high | tiny |

### 4.3. **Create Adversarial Test Suites**

- **State bloat attacks**: Many SSTORE ops, minimal CPU usage. Does SDM refund enough to incentivize this? How much state growth happens?
- **Computation attacks**: Expensive precompiles or loops. Does SDM properly charge these?
- **Data attacks**: Large calldata blobs. Does SDM account for network bandwidth costs?
- **Hybrid attacks**: Combinations of the above

# Criteria for advancing

How to decide if we continue to production implementation or canceling the project?

Answer the following questions:

- [ ]  1. Measurements — are results from Benchmarks conclusively showing that `opgas` model is a net-good for a given OP Stack chain? Under what conditions does `opgas` outperform canonical gas, and at what cost?
- [ ]  2. State growth — can we release `opgas` model without incurring additional state growth penalties (i.e. refund only very fast transactions that use very little / no state writes?) How to trace all transactions and count storage writes/reads, so that we don’t refund transactions that make state growth worse?
- [ ]  3. EIPs for gas refunds — can gas refund EIPs (EIP-3529 and EIP-7623) be disabled without introducing DoS vectors and re-introducing GasToken?
- [ ]  4. Kill-switch — Can chain operators on-demand start/stop the `opgas` model functionality, i.e. if a DoS vector is discovered and triggered on a production chain (OPM or Unichain), how do we stop it?
- [ ]  5. Monitoring — How do we determine and monitor if a chain is being DoSed via `opgas`? What to look for? Is an alert going to be triggered?
- [ ]  6. ZK prover performance characteristics — How do we make sure that a ZK prover can still prove blocks that are discounted by `opgas`, given the difference in performance characteristics?

Based on results 

# Related documents

1. [SDM PoC 1 Review](https://www.notion.so/SDM-PoC-1-Review-2fef153ee16280fa992ac4a5cc19666e?pvs=21)
2.