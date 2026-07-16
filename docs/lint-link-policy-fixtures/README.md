# lint-link-policy fixtures

Synthetic fixtures for `docs/public-docs/scripts/lint-link-policy.mjs --self-test`
(run in CI by `.github/workflows/docs-link-policy.yml`).

*   `failing/` — one `.mdx` file per violation class; the file name is the rule
    it must trigger. The self-test fails if any fixture stops firing its rule.
*   `passing/` — policy-conformant examples of the same link shapes; the
    self-test fails if the linter reports anything here.
*   `specs-src/` — a two-file stand-in for an `ethereum-optimism/specs`
    checkout, so spec path/anchor resolution is exercised hermetically
    (no network).

These files are test data, not documentation pages. They live outside the
Mintlify content root (`docs/public-docs/`) on purpose, so the deliberate
violations in `failing/` never appear in real lint runs or in
`mint broken-links` output.
