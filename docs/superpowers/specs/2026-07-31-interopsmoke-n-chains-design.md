# Interop smoke: N chains

## Scope

Replace the two-chain RPC inputs with a repeatable `--l2-rpc` flag. Require at least two RPC endpoints. Build one remote-chain and remote-user entry per endpoint.

## Behavior

- Identity verifies every configured chain ID is unique.
- Transfers run once per configured chain.
- Bridge, valid-message, and invalid-message run for every ordered pair of distinct chains.
- Three endpoints therefore exercise A->B, A->C, B->A, B->C, C->A, and C->B.
- Invalid-message no longer selects fixed A/B directions; it covers every ordered pair.

## Compatibility

This deliberately replaces the A/B-specific CLI flags and environment variables with the repeatable endpoint input. Commands retain their names and existing options unrelated to chain selection.

## Tests

Unit tests cover endpoint validation, direction generation for two and three chains, and rejection of duplicate chain IDs. Existing package tests remain green.
