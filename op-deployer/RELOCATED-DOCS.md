# Decision: op-deployer Documentation Location

**Status:** Accepted  
**Date:** 2025-02-12

## Summary

op-deployer documentation has been moved from the monorepo (mdBook in `op-deployer/book/`) to the [ethereum-optimism/docs](https://github.com/ethereum-optimism/docs) repository and the docs website ([docs.optimism.io](https://docs.optimism.io)). The mdBook has been removed from the monorepo.

## Rationale

- **Single source of truth:** Documentation for chain operators is consolidated in the docs repo alongside other operator tools (op-conductor, op-validator, etc.).
- **Easier maintenance:** Contributors can update op-deployer docs without cloning the optimism monorepo.
- **Consistency:** All Optimism documentation follows the same structure and deployment pipeline.

## Canonical Documentation Location

- **Primary:** [docs.optimism.io - Chain Operators > Tools > Deployer](https://docs.optimism.io/operators/chain-operators/tools/op-deployer)
- **Source:** [ethereum-optimism/docs - pages/operators/chain-operators/tools/op-deployer.mdx](https://github.com/ethereum-optimism/docs/blob/main/pages/operators/chain-operators/tools/op-deployer.mdx)
- **Tutorial:** [Create L2 Rollup - op-deployer setup](https://docs.optimism.io/operators/chain-operators/tutorials/create-l2-rollup/op-deployer-setup)

## For op-deployer Maintainers and Contributors

- Submit documentation changes to the [ethereum-optimism/docs](https://github.com/ethereum-optimism/docs) repository.
- Open PRs against the `main` branch of the docs repo.
- Documentation updates do not require changes to the optimism monorepo.

## What Was Removed

- `optimism/op-deployer/book/` (entire directory)
  - `book.toml`, `custom.css`
  - `src/user-guide/` (init.md, apply.md, bootstrap.md, usage.md, etc.)
  - `src/reference-guide/` (architecture.md, pipeline.md, etc.)
  - `src/SUMMARY.md`, `src/README.md`
  - `src/assets/`

## References

- [ethereum-optimism/docs](https://github.com/ethereum-optimism/docs) — Mintlify docs at [docs.optimism.io](https://docs.optimism.io)
- [op-deployer README](README.md) — links to docs repo
