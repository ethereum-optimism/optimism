# Rollup Node

The consensus module of Optimisc Ethereum.

## Summary

The Rollup Node is a consenus client that determines the latest state of the
rollup. It reads state from the canon chain (L1) to compute the rollup state.

## Components

#### Feed Oracle

The Feed Oracle is an adapter that monitors the canon chain for new events in
the rollup contract. The rollup contract provides the *canonical* order for the
rollup chain.

The Feed Oracle can index the batch submissions and deposits locally into the
database. The batch submissions should be converted into equivalent rollup
blocks.

#### Consensus

Additional checks should be imposed on the execution engine. Roughly:

```python
def on_block(db, block):
    ctx = None
    for tx in block.transactions:
        # verify context tx updates canon light client with correct data
        if tx.is_ctx():
            assert db.is_valid_ctx(tx)
            ctx = tx

        # verify deposits are valid in the canon chain and in the context of the ctx tx
        if tx.is_deposit():
            assert db.is_valid_deposit(ctx, tx)

    # run the block in the executin engine, if it's valid it's the new head of the
    # rollup chain
    assert verify_block_in_execution_engine(block)
```

#### Execution Engine

The execution engine implements the [execution specification][execution-spec].
There already many teams who plan to transition their legacy clients into
execution engines. The rollup client will communicate to the engine via a
bidirectional JSON-RPC interface ([WIP][execution-engine-rpc]).

One of the main goal of the rollup client is to use the exeuction engine
without modification.

### Miner

The miner assembles the block via the Execution Engine. It injects context
transactions that update the light client to the canon chain and it inject new
depsit transactions. It submits the transaction batch to and waits until the
transaction is mined (potentially bumping fee price if needed). 

#### Database

The database persists the rollup chain and indexed data from the canon chain.
Pruning of canon chain data can be aggressive since indexed data is only relavent
to consensus during the force-inclusion period of deposits.

#### Networking

The Rollup Node should provide proxied access to the Execution Engine's
networking stack. This allows for two main things:

* Context updates and deposits are injected into the execution engine via
  transactions. This means that it's possible for the Execution Engine to
  recieve signed system updates in the public mempool. This could cause block
  producing rollup nodes to submit bundles with invalid contexts/deposits.
* To provide fast-confirmations, a group of rollup block producers can define a
  "unconfirmed" blocks that haven't yet been submitted to mainnet (but will
  likely be in the near future). 

[execution-spec]: https://github.com/ethereum/execution-specs
[execution-engine-rpc]: https://hackmd.io/@n0ble/consensus_api_design_space
