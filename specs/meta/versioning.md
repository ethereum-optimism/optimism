# Versioning

## Go modules

Go modules that are currently versioned:

```text
./op-batcher
./op-bindings
./op-node
./op-proposer
./op-e2e
```

Go modules which are not yet versioned:

```text
./batch-submitter  (changesets)
./bss-core
./gas-oracle       (changesets)
./indexer          (changesets)
./l2geth           (changesets)
./l2geth-exporter  (changesets)
./op-exporter      (changesets)
./proxyd           (changesets)
./teleportr        (changesets)
./state-surgery
```

### versioning process

Since changesets versioning is not compatible with Go we are moving away from it.
Starting with new bedrock modules, Go-compatible tags will be used,
formatted as `modulename/vX.Y.Z` where `vX.Y.Z` is semver.

## Typescript

See Changesets.
