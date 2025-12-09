# `kona-executor` test fixtures

The `StatelessL2Builder` type uses static test data fixtures to run stateless execution of certain blocks offline. The
test data fixtures include:
* The `RollupConfig` of the chain that the block belongs to.
* The parent block header, which we apply state on top of.
* The payload attributes for building the new block.
* A `rocksdb` database containing the witness data for stateless execution of the block building job.

Sometimes, updates in the block building code can add new state accesses, requiring these fixtures to be re-generated.

To generate a new fixture and add it to the test suite, run:

```sh
cargo r -p execution-fixture \
    --l2-rpc <archival_l2_el_rpc> \
    --block-number <l2_block_number_to_execute>
```

this command will add a new compressed test fixture for the given L2 block into `kona-executor`'s `testdata` directory.
The test suite will automatically pick this new test fixture up, and no further action is needed to register it.
