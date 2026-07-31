# revm-ee-tests

Shared test utilities and integration tests for **revm** and **op-revm**.

## Running Tests

```bash
# Run all tests
cargo test -p revm-ee-tests

# Run a specific test subset (e.g. TIP-1016 state gas tests)
cargo test -p revm-ee-tests tip1016
```

Snapshot testdata is auto-generated on first run and compared on subsequent runs.
