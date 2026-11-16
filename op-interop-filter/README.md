# op-interop-filter

A lightweight service that validates interop executing messages for op-geth or op-reth transaction filtering.

Any reorg will trigger the failsafe which disables all interop transactions. If `--reorg-recovery-enabled` is set, reorg-triggered failsafe is automatically resolved by rewinding logs DB state to finalized.

An operator can enable the failsafe manually with `admin_setFailsafeEnabled`. The filter stores this state in `failsafe.json` in `--data-dir`, so the failsafe stays on after a restart. Without `--data-dir`, the manual failsafe does not survive a restart. Set `--failsafe-enabled` to enable the manual failsafe at every start. If `failsafe.json` is unreadable, the filter does not start unless `--failsafe-enabled` is set.

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
