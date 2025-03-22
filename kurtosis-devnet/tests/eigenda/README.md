# EigenDA tests

All tests in here are run against an eigenda kurtosis devnet, which is started using the justfile command `just eigenda-devnet-start`.
See the justfile for the options there.

## Backends

The devnet can either be started with proxy in `memstore` mode, or in `holesky` mode where it connects to the eigenda holesky network.

> Every test in this package MUST have a `_Memstore` or `_Holesky` suffix to indicate which backend it is testing.

The testing commands in the justfile are `eigenda-devnet-test-memstore` and `eigenda-devnet-test-holesky`, and pattern match the test names to run the correct tests.
