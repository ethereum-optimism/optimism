# Vendored interop services

This directory is a source snapshot of
[`ethereum-optimism/interop-services`](https://github.com/ethereum-optimism/interop-services),
vendored for local interop development and smoke scenarios.

- Upstream commit: `279b721069e085ba40a84a49b5d6f10e98aeffd2`
- Snapshot date: 2026-08-24
- Runtime applications: `apps/ponder-interop` and
  `apps/autorelayer-interop`

The complete workspace is retained because those applications use local
workspace packages. Generated dependencies and build artifacts are not
vendored.

The upstream root package declares `"license": "UNLICENSED"` and the snapshot
contains no standalone license file. This vendored copy preserves that
metadata; confirm the intended redistribution license before publishing it.

The snapshot has three integration-only changes:

- `package.json` explicitly allows the pinned `better-sqlite3` install script.
  pnpm 10 otherwise suppresses that native build, and the upstream
  autorelayer cannot open its local state database.
- `mise.toml` retains only the Node tool needed by these services. Supersim and
  Foundry are provided by the surrounding Optimism workspace and demo setup.
- a minimal root `go.mod` keeps an unrelated Supersim README test in this Node
  workspace outside the surrounding Optimism module's repository-wide Go
  package scan. The vendored runtime applications do not use Go.

To refresh this snapshot, archive its reviewed source commit here, preserve
this file with the new commit and date, update the frozen pnpm lock, then run
the Ponder and autorelayer build/tests documented by the interop smoke
package.
