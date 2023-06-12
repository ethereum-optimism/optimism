# Versioning

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [Go modules](#go-modules)
  - [versioning process](#versioning-process)
- [Typescript](#typescript)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Go modules

Go modules that are currently versioned:

```text
./op-service
./op-bindings
./op-batcher
./op-node
./op-proposer
./op-e2e
```

Go modules which are not yet versioned:

```text
./batch-submitter  (changesets)
./indexer          (changesets)
./l2geth           (changesets)
./op-exporter      (changesets)
./proxyd           (changesets)
./state-surgery
```

### versioning process

Since changesets versioning is not compatible with Go we are moving away from it.
Starting with new bedrock modules, Go-compatible tags will be used,
formatted as `modulename/vX.Y.Z` where `vX.Y.Z` is semver.

## Typescript

See Changesets.
