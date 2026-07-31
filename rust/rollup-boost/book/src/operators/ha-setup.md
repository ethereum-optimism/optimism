# High Availability Setup

The current OP Stack sequencer HA design relies on `op-conductor` to manage a cluster of sequencers. In the rollup-boost HA setup, each sequencer connects to its own builder instance. In the event of sequencer failover, `op-conductor` elects a new leader, promoting a different `op-node` along with its associated `rollup-boost` and builder instance.

![HA Setup](https://raw.githubusercontent.com/flashbots/rollup-boost/refs/heads/main/assets/1-1-builder-rb.png)

If the builder produces undesirable but valid blocks, operators must either manually disable external block production via the `rollup-boost` debug API, disable the block builder directly (causing health checks to fail), or manually select a new sequencer leader.

See the [design doc](https://github.com/flashbots/rollup-boost/blob/main/docs/rollup-boost-ha.md) for more detail on the design principles for the HA setup with rollup-boost.

## Health Checks

In high availability deployments, `op-conductor` must assess the full health of the block production path. Rollup Boost will expose a composite `/healthz` endpoint to report on both builder synchronization and payload production status. These checks allow `op-conductor` to detect degraded block building conditions and make informed leadership decisions.

Rollup Boost will continuously monitor two independent conditions to inform the health of the builder and the default execution client:

- **Builder Synchronization**:  
  A background task periodically queries the builder’s latest unsafe block via `engine_getBlockByNumber`. The task compares the timestamp of the returned block to the local system time. If the difference exceeds a configured maximum unsafe interval (`max_unsafe_interval`), the builder is considered out of sync. Failure to fetch a block from the builder or detection of an outdated block timestamp results in the health status being downgraded to Partial. If the builder is responsive and the block timestamp is within the acceptable interval, the builder is considered synchronized and healthy. Alternatively, instead of periodic polling, builder synchronization can be inferred if the builder returns a `VALID` response to a `newPayload` call forwarded from Rollup Boost.

- **Payload Production**:  
  During each `get_payload` request, Rollup Boost will verify payload availability from both the builder and the execution client. If the builder fails to deliver a payload, Rollup Boost will report partial health. If the execution client fails to deliver a payload, Rollup Boost will report unhealthy.

`op-conductor` should also be configurable in how it interprets health status for failover decisions. This allows chain operators to define thresholds based on their risk tolerance and operational goals. For example, operators may choose to maintain leadership with a sequencer reporting `206 Partial Content` to avoid unnecessary failovers or they may configure `op-conductor` to immediately fail over when any degradation is detected. This flexibility allows the chain operator to configure a failover policy that aligns with network performance expectations and builder reliability.

<br>

| Condition | Health Status |
|:----------|:--------------|
| Builder is synced and both execution client and builder return payloads | `200 OK` (Healthy) |
| Builder is out of sync| `206 Partial Content` (Partially Healthy) |
| Builder fails to return payload on `get_payload` request | `206 Partial Content` (Partially Healthy) |
| Execution client fails to return payload on `get_payload` request | `503 Service Unavailable` (Unhealthy) |

`op-conductor` should query the `/healthz` endpoint exposed by Rollup Boost in addition to the existing execution client health checks. Health should be interpreted as follows:

- `200 OK` (Healthy): The node is fully healthy and eligible for leadership.
- `206 Partial Content` (Partially Healthy): The node is degraded but may be considered for leadership if configured by operator
- `503 Service Unavailable` (Unhealthy): The node is unhealthy and must be excluded from leadership.

During normal operation and leadership transfers, `op-conductor` should prioritize sequencer candidates in the following order:

1. Prefer nodes reporting `200 OK`.
2. Nodes that return `503 Service Unavailable` are treated as unhealthy and must not be eligible for sequencer leadership. `op-conductor` should offer a configuration option to treat nodes returning `206 Partial Content` as either healthy or unhealthy.

Rollup Boost instances that are not actively sequencing rely on the builder sync check to report health, as they are not producing blocks. This behavior mirrors the existing `op-conductor` health checks for inactive sequencers and ensures readiness during failover without compromising network liveness guarantees. Note that `op-conductor` will still evaluate existing sequencer health checks to determine overall sequencer health.

Note that in the case where the builder is unhealthy, `rollup-boost` should bypass forwarding block production requests to the builder entirely and immediately use the default execution client for payload construction. This avoids introducing unnecessary latency while waiting for the builder response to timeout.

When builder health is restored, normal request forwarding and payload selection behavior will resume.

<br>

## Failure Scenarios

Below is a high level summary of how each failure scenario is handled. All existing failure modes assumed by upstream `op-conductor` are maintained:

| Failure Scenario | Category | Scenario and Solution |
| --- | --- | --- |
| Leader Sequencer Execution Client Fails | Sequencer Failure | `op-conductor` will detect an unhealthy status from both `rollup-boost` and pre-existing sequencer health checks, causing Conductor to elect a new leader. Once the default execution client has recovered, `rollup-boost` will update its health status to `200` and the sequencer will continue operating normally as a follower. |
| Follower Sequencer Execution Client Fails | Sequencer Failure | Both `rollup-boost` and pre-existing sequencer health checks will report "unhealthy". Once the default execution client has recovered, `rollup-boost` will update its health status to `200` and the sequencer will continue operating normally as a follower. In the event of leadership transfer, this sequencer instance will not be considered for leadership.|
| Leader `rollup-boost` Fails | Rollup Boost Failure | Leader sequencer `rollup-boost` becomes unhealthy, causing `op-conductor`'s sequencer health checks to fail and attempt to elect a new leader. This failure mode is the same as a typical leader sequencer failure. Once the sequencer recovers, it will continue to participate in the cluster as a follower|
| Follower `rollup-boost` Fails | Rollup Boost Failure | Follower sequencer `rollup-boost` becomes unhealthy. The leader sequencer is unaffected. Once the sequencer recovers, it will continue to participate in the cluster as a follower.|
| Leader Builder Stops Producing Blocks | Builder Failure | The builder associated with the sequencer leader stops producing new payloads. `rollup-boost` will detect the builder failure via background health checks and downgrade its health status to partial. This will result in `rollup-boost` ignoring the builder and selecting the default execution client's payload for block production. If `op-conductor` is configured to failover upon partial `rollup-boost` health, a new leader will attempt to be elected. Once the builder recovers and resumes payload production, `rollup-boost` will update its health to `200` and resume with normal operation. |
| Leader Builder Falls Out of Sync | Builder Failure | The builder associated with the sequencer leader falls out of sync with the chain head. `rollup-boost` will detect the unsynced state via the background health checks and downgrade its health status to partial. This will result in `rollup-boost` ignoring builder payloads and selecting the default execution client payload for block until the builder is resynced. If `op-conductor` is configured to failover upon partial `rollup-boost` health, a new leader will attempt to be elected. Once the builder recovers, `rollup-boost` will update its health to `200` and resume with normal operation. |
| Follower Builder Falls Out of Sync | Builder Failure | The builder associated with a follower sequencer falls out of sync with the chain head. Block production is unaffected while the node remains a follower. In the event a leader election occurs and `op-conductor` is configured to treat partial health as "unhealthy", this instance will not be eligible for leadership. Once the builder recovers, `rollup-boost` will report `200 OK` and resume normal operation.|
| Leader Builder Producing Bad Blocks| Builder Failure| In this scenario, the builder is "healthy" but producing bad blocks (eg. empty blocks). If the builder block passes validation via a `new_payload` call to the default execution client, it will be proposed to the network. Manual intervention is needed to either switch to a different sequencer or shutoff the builder. Further mitigation can be introduced via block selection policy allowing `rollup-boost` to select the "healthiest" block. Currently, it is unclear what block selection policy would provide the strongest guarantees.|
