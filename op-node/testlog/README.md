# testlog

`github.com/ethereum/go-ethereum/internal/testlog`: a Go-ethereum util for logging in tests.

Since we use the same logging, but as an external package, we have to move the test utility to our own internal package.

This fork also made minor modifications:

- Enable color by default.
- Add `estimateInfoLen` and use this for message padding in `flush()` to align the contents of the log entries,
  compensating for the different lengths of the log decoration that the Go library adds.
