# diffmips

This is a collection of MIPS testing tools.
The Unicorn emulator first backed Cannon directly, but has been separated as a testing-only tool,
and is replaced with a Go implementation of the minimal MIPS functionality.

Directory layout
```
unicorn     -- Sub-module, used by mipsevm for offchain MIPS emulation.
unicorntest -- Go module with Go tests, diffed against the Cannon state. [Work in Progress]
```

### `unicorn`

To build unicorn from source (git sub-module), run:
```
make libunicorn
```

### `unicorntest`

This requires `unicorn` to be built, as well as the `contracts` for testing.

To test:
```
make test
```

