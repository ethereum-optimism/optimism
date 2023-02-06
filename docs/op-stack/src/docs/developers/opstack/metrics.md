---
title: Metrics
lang: en-US
---

The Bedrock `op-node` exposes a variety of metrics to help observe the health of the system and debug issues. Metrics are formatted for use with Prometheus, and exposed via a metrics endpoint. The default metrics endpoint is `http://localhost:7300/metrics`.

To enable metrics, pass the `--metrics.enabled` flag to the `op-node`. You can customize the metrics port and address via the `--metrics.port` and `--metrics.addr` flags, respectively.

## Important Metrics

To monitor the health of your node, you should monitor the following metrics:
 
- `op_node_default_refs_number`: This metric represents the `op-node`'s current L1/L2 reference block number for different sync types. If it stops increasing, it means that the node is not syncing. If it goes backwards, it means your node is reorging.
- `op_node_default_peer_count`: This metric represents how many peers the `op-node` is connected to. Without peers, the `op-node` cannot sync unsafe blocks and your node will lag behind the sequencer as it will fall back to syncing purely from L1.
- `op_node_default_rpc_client_request_duration_seconds`: This metrics measures the latency of RPC requests initiated by the `op-node`. This metric is important when debugging sync performance, as it will reveal which specific RPC calls are slowing down sync. This metric exposes one timeseries per RPC method. The most important RPC methods to monitor are:
  - `engine_forkChoiceUpdatedV1`, `engine_getPayloadV1`, and `engine_newPayloadV1`: These methods are used to execute blocks on `op-geth`. If these methods are slow, it means that sync time is bottlenecked by either `op-geth` itself or your connection to it.
  - `eth_getBlockByHash`, `eth_getTransactionReceipt`, and `eth_getBlockByNumber`: These methods are used by the `op-node` to fetch transaction data from L1. If these methods are slow, it means that sync time is bottlenecked by your L1 RPC.

## Available Metrics

A complete list of available metrics is below:

|                       METRIC                        |                                           DESCRIPTION                                            |    LABELS    |   TYPE    |
|-----------------------------------------------------|--------------------------------------------------------------------------------------------------|--------------|-----------|
| op_node_default_info                                | Pseudo-metric tracking version and config info                                                   | version      | gauge     |
| op_node_default_up                                  | 1 if the op node has finished starting up                                                        |              | gauge     |
| op_node_default_rpc_server_requests_total           | Total requests to the RPC server                                                                 | method       | counter   |
| op_node_default_rpc_server_request_duration_seconds | Histogram of RPC server request durations                                                        | method       | histogram |
| op_node_default_rpc_client_requests_total           | Total RPC requests initiated by the opnode's RPC client                                          | method       | counter   |
| op_node_default_rpc_client_request_duration_seconds | Histogram of RPC client request durations                                                        | method       | histogram |
| op_node_default_rpc_client_responses_total          | Total RPC request responses received by the opnode's RPC client                                  | method,error | counter   |
| op_node_default_l1_source_cache_size                | L1 Source cache cache size                                                                       | type         | gauge     |
| op_node_default_l1_source_cache_get                 | L1 Source cache lookups, hitting or not                                                          | type,hit     | counter   |
| op_node_default_l1_source_cache_add                 | L1 Source cache additions, evicting previous values or not                                       | type,evicted | counter   |
| op_node_default_l2_source_cache_size                | L2 Source cache cache size                                                                       | type         | gauge     |
| op_node_default_l2_source_cache_get                 | L2 Source cache lookups, hitting or not                                                          | type,hit     | counter   |
| op_node_default_l2_source_cache_add                 | L2 Source cache additions, evicting previous values or not                                       | type,evicted | counter   |
| op_node_default_derivation_idle                     | 1 if the derivation pipeline is idle                                                             |              | gauge     |
| op_node_default_pipeline_resets_total               | Count of derivation pipeline resets events                                                       |              | counter   |
| op_node_default_last_pipeline_resets_unix           | Timestamp of last derivation pipeline resets event                                               |              | gauge     |
| op_node_default_unsafe_payloads_total               | Count of unsafe payloads events                                                                  |              | counter   |
| op_node_default_last_unsafe_payloads_unix           | Timestamp of last unsafe payloads event                                                          |              | gauge     |
| op_node_default_derivation_errors_total             | Count of derivation errors events                                                                |              | counter   |
| op_node_default_last_derivation_errors_unix         | Timestamp of last derivation errors event                                                        |              | gauge     |
| op_node_default_sequencing_errors_total             | Count of sequencing errors events                                                                |              | counter   |
| op_node_default_last_sequencing_errors_unix         | Timestamp of last sequencing errors event                                                        |              | gauge     |
| op_node_default_publishing_errors_total             | Count of p2p publishing errors events                                                            |              | counter   |
| op_node_default_last_publishing_errors_unix         | Timestamp of last p2p publishing errors event                                                    |              | gauge     |
| op_node_default_unsafe_payloads_buffer_len          | Number of buffered L2 unsafe payloads                                                            |              | gauge     |
| op_node_default_unsafe_payloads_buffer_mem_size     | Total estimated memory size of buffered L2 unsafe payloads                                       |              | gauge     |
| op_node_default_refs_number                         | Gauge representing the different L1/L2 reference block numbers                                   | layer,type   | gauge     |
| op_node_default_refs_time                           | Gauge representing the different L1/L2 reference block timestamps                                | layer,type   | gauge     |
| op_node_default_refs_hash                           | Gauge representing the different L1/L2 reference block hashes truncated to float values          | layer,type   | gauge     |
| op_node_default_refs_seqnr                          | Gauge representing the different L2 reference sequence numbers                                   | type         | gauge     |
| op_node_default_refs_latency                        | Gauge representing the different L1/L2 reference block timestamps minus current time, in seconds | layer,type   | gauge     |
| op_node_default_l1_reorg_depth                      | Histogram of L1 Reorg Depths                                                                     |              | histogram |
| op_node_default_transactions_sequenced_total        | Count of total transactions sequenced                                                            |              | gauge     |
| op_node_default_p2p_peer_count                      | Count of currently connected p2p peers                                                           |              | gauge     |
| op_node_default_p2p_stream_count                    | Count of currently connected p2p streams                                                         |              | gauge     |
| op_node_default_p2p_gossip_events_total             | Count of gossip events by type                                                                   | type         | counter   |
| op_node_default_p2p_bandwidth_bytes_total           | P2P bandwidth by direction                                                                       | direction    | gauge     |
| op_node_default_sequencer_building_diff_seconds     | Histogram of Sequencer building time, minus block time                                           |              | histogram |
| op_node_default_sequencer_building_diff_total       | Number of sequencer block building jobs                                                          |              | counter   |
| op_node_default_sequencer_sealing_seconds           | Histogram of Sequencer block sealing time                                                        |              | histogram |
| op_node_default_sequencer_sealing_total             | Number of sequencer block sealing jobs                                                           |              | counter   |
