# OP Supernode

Run multiple OP Stack chains in a single process. OP Supernode virtualizes OP Node so each chain runs as an isolated in-memory worker with per-chain config, data, and logs.

## Major Features
### Chain Containers
Chain Containers represent the concerns of one specific Chain being managed by the `op-supernode`

#### Chain Isolation / Interfacing
Chain Containers abstract away the local Consensus Layer / Derivation Pipeline, as well as the Execution Engine.
Chain containers manage the runtime of the CL as a local process called a "Virtual Node", which presently is implemented
only by `op-node` itself.
Chain Containers provide a stable interface to get data from the chain or affect the chain without needing to operate on the internals of
that chain.
Chain Containers also allow for multiple chains to be derived inside of the same `op-supernode`. Because theya re isolated,
running many chains worth of derivation is trivial.


#### Shared Chain Resources
Chain Containers also benefit from running in a shared environment through the use of shared resources.
Shared resources are Dependencies which have been injected into the Virtual Node such that the original behavior is in-tact,
but redundant access is eliminated.

- Shared L1 and Beacon Client mean shared caching and only a single pipe to the L1 across all chains.
- Shared RPCs through a namespaced RPC registration system. Call `11155420/` for OP Sepolia's RPC capabilities, or `84532/` for Base Sepolia.
- Metrics are shared through similar namespacing, but will likely be joined via prometheus dimensions in the future.
- Data Directories are namespaced to protect SafeDB and P2P resources
- Flag configuration can be shared amongst chains for common setup.


#### Flag Configuration Tips

Because `op-node` is the only implementation of Virtual Nodes presently, it gets special treatment when the application starts up.
In specific, all those flags which are found in `op-node` are upstreamed with namespacing into the `op-supernode` flags.
This allows for *roughly* 1:1 setup and behavior between Node and Supernode, with cavets listed below.
- To set a value for one chain, use `--vn.<chainID>.<flag>`
- To set a value for *all* chains, use `--vn.all.<flag>`
- Some behaviors are expected to be configured at the `op-supernode` level and are not respected per usual when the application starts:
  - `l1` and `l1.beacon` are used to create the shared L1 client. Any L1 configuration will not be respected by a Virtual Node.
  - Log and Metric settings are passed down to the Virtual node from the top level Log/Metric flags, and individual chain settings may not be respected.
  - `p2p` is enabled/disabled at the top level and sets all listen ports to `0` to prevent collisions. Per-chain P2P functionality will added
  via a shared resource in the future.

Example launch of Supernode:
```
./bin/op-supernode \
  --l1='...' \
  --l1.beacon='...' \
  --disable-p2p=true \
  --log.level=DEBUG \
  --metrics.enabled=true \
  --metrics.port=7300 \
  --chains=11155420,84532 \
  --vn.11155420.network=op-sepolia \
  --vn.11155420.l2='...' \
  --vn.84532.network=base-sepolia \
  --vn.84532.l2='...'
  --vn.84532.l1.beacon-archiver='...' \
  --vn.all.l2.jwt-secret=../../jwt-secret.txt \
  ```

Note: consult the `help` printout of the application as currently there is a mistake in the naming of Environment Variables for flags.
Env Vars may be used, but at present their names include inappropriate `op-node` markers.

### Activities
Activities represent the concerns of `op-supernode` which fall outside of any one chain, and are modular plugins to the capabilities of the software.

#### RPC Activities
Components which expose RPC functionality and register as an Activity will have their RPC namespaces registered against the `op-supernode` root.

#### Runnable Activities
Components which expose Start/Stop are given a goroutine to work during `op-supernode` runtime

#### Current Activities:
- `Heartbeat`
  - RPC: `heartbeat_check` produces a random-hex sign of life when called.
  - Runtime: emits a simple heartbeat message to the logs to show liveness.
- `SuperRoot`
  - RPC: `superroot_atTimestamp` produces a SuperRoot from Verified L2 blocks, and includes sync/derivation information for Proofs.

### Quickstart
Build:
```bash
just op-supernode
```
