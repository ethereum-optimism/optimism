---
title: Rollup Operations
lang: en-US
---

## Stopping your Rollup

An orderly shutdown is done in the reverse order to the order in which components were started:

1. To stop the batcher, use this command:

   ```sh
   curl -d '{"id":0,"jsonrpc":"2.0","method":"admin_stopBatcher","params":[]}' \
       -H "Content-Type: application/json" http://localhost:8548 | jq
   ```

   This way the batcher knows to save any data it has cached to L1.
   Wait until you see `Batch Submitter stopped` in batcher's output before you stop the process.

1. Stop `op-node`.
   This component is stateless, so you can just stop the process.

1. Stop `op-geth`.
   Make sure you use **CTRL-C** to avoid database corruption.


## Starting your Rollup

To restart the blockchain, use the same order of components you did when you initialized it.

1. `op-geth`
1. `op-node`
1. `op-batcher`

   If `op-batcher` is still running and you just stopped it using RPC, you can start it with this command:

   ```sh
   curl -d '{"id":0,"jsonrpc":"2.0","method":"admin_startBatcher","params":[]}' \
       -H "Content-Type: application/json" http://localhost:8548 | jq   
   ```

::: tip Synchronization takes time

`op-batcher` might have warning messages similar to:

```
WARN [03-21|14:13:55.248] Error calculating L2 block range         err="failed to get sync status: Post \"http://localhost:8547\": context deadline exceeded"
WARN [03-21|14:13:57.328] Error calculating L2 block range         err="failed to get sync status: Post \"http://localhost:8547\": context deadline exceeded"
```

This means that `op-node` is not yet synchronized up to the present time.
Just wait until it is.

:::


## Adding nodes

To add nodes to the rollup, you need to initialize `op-node` and `op-geth`, similar to what you did for the first node.
You should *not* add an `op-bathcer`, there should be only one.

1. Configure the OS and prerequisites as you did for the first node.
1. Build the Optimism monorepo and `op-geth` as you did for the first node.
1. Copy from the first node these files:
    
    ```bash
    ~/op-geth/genesis.json
    ~/optimism/op-node/rollup.json
    ```
    
1. Create a new `jwt.txt` file as a shared secret:
    
    ```bash
    cd ~/op-geth
    openssl rand -hex 32 > jwt.txt
    cp jwt.txt ~/optimism/op-node
    ```
    
1. Initialize the new op-geth:
    
    ```bash
    cd ~/op-geth
    ./build/bin/geth init --datadir=./datadir ./genesis.json
    ```

1. To enable L2 nodes to synchronize directly, rather than wait until the transactions are written to L1, turn on [peer to peer synchronization](http://localhost:8081/docs/build/getting-started/#run-op-node).
   If you already have peer to peer synchronization, add the new node to the `--p2p.static` list so it can synchronize.

1. Start `op-geth` (using the same command line you used on the initial node)
1. Start `op-node` (using the same command line you used on the initial node)
