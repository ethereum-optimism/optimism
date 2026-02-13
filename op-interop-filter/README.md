# op-interop-filter

A lightweight service that validates interop executing messages for op-geth or op-reth transaction filtering.

Any reorg will trigger the failsafe which disables all interop transactions.

## Usage

### Build from source

```bash
just op-interop-filter
./bin/op-interop-filter --help
```

### Run from source

```bash
go run ./cmd --help
```

### Build docker image

```bash
docker buildx bake op-interop-filter
```
