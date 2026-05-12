# Decision: op-deployer Documentation Location

**Status:** Accepted  
**Date:** 2025-02-12

## Summary

op-deployer documentation has been moved from the mdBook in `op-deployer/book/` to the public docs tree in this repository and the docs website ([docs.optimism.io](https://docs.optimism.io)). The mdBook has been removed from the monorepo.

## Rationale

- **Single source of truth:** Documentation for chain operators is consolidated in `docs/public-docs` alongside other operator tools (op-conductor, op-validator, etc.).
- **Easier maintenance:** Contributors can update op-deployer docs through the same repository and pull request flow used for the rest of the public docs.
- **Consistency:** All Optimism documentation follows the same structure and deployment pipeline.

## Canonical Documentation Location

- **Primary:** [docs.optimism.io - Chain Operators > Tools > OP Deployer](https://docs.optimism.io/chain-operators/tools/op-deployer/overview)
- **Source:** [`docs/public-docs/chain-operators/tools/op-deployer`](../docs/public-docs/chain-operators/tools/op-deployer)
- **Tutorial:** [Create L2 Rollup - op-deployer setup](https://docs.optimism.io/chain-operators/tutorials/create-l2-rollup/op-deployer-setup)

## For op-deployer Maintainers and Contributors

- Submit documentation changes under [`docs/public-docs`](../docs/public-docs).
- Open PRs against the `develop` branch of the optimism monorepo.
- Documentation updates usually do not require changes under `op-deployer/`.

## What Was Removed

- `optimism/op-deployer/book/` (entire directory)
  - `book.toml`, `custom.css`
  - `src/user-guide/` (init.md, apply.md, bootstrap.md, usage.md, etc.)
  - `src/reference-guide/` (architecture.md, pipeline.md, etc.)
  - `src/SUMMARY.md`, `src/README.md`
  - `src/assets/`

## References

- [`docs/public-docs`](../docs/public-docs) — source for [docs.optimism.io](https://docs.optimism.io)
- [op-deployer README](README.md) — links to the current public docs
