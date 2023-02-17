# Erigon Build Hack

Because Erigon was originally forked from the go-ethereum codebase, and,
because it still imports go-ethereum, it's generally not possible for Geth and
Erigon to both be referenced in the same go Binary.

This is not only because the different packages have similar, but different
library requirements, but because they additionally have conflicting CGO
symbols.

Additionally, because of the structure of the Optimism project, relying on a
go.work file with git sub-modules, the build of Erigon is further complicated.

This directory is a simple hack to build Erigon in isolation, at a specific
pinned version (one supporting the Bedrock upgrades until such support can be
merged upstream).

You may verify that this build directory works by executing a simple:

```
go build github.com/ledgerwatch/erigon/cmd/erigon
```

Or, you may run the simple automated test which will do the same.
