# crossdomain/testdata

Real world test data is used to generate test vectors for the withdrawal
hashing. The `trace.sh` script will generate artifacts used as part of the
tests. It accepts a single argument, being the transaction hash to fetch
artifacts for. It will fetch a receipt, a call trace and a state diff.
The tests require that a file named after the transaction hash exists
in each of the directories `call-traces`, `receipts` and `state-diffs`.
The `trace.sh` script will ensure that the files are created correctly.
