## Challenging Invalid Output Proposals

The dispute game factory deployed to Goerli reads from the permissioned L2 Output Oracle contract. This restricts games
to challenging valid output proposals and an honest challenger should win every game. To test creating games that
challenge an invalid output proposal, a custom chain is required. The simplest way to do this is using the end-to-end
test utilities in [`op-e2e`](https://github.com/ethereum-optimism/optimism/tree/develop/op-e2e).

A simple starting point has been provided in the `TestCannonProposedOutputRootInvalid` test case
in [`faultproof_test.go`](https://github.com/ethereum-optimism/optimism/blob/6e174ae2b2587d9ac5e2930d7574f85d254ca8b4/op-e2e/faultproof_test.go#L334).
This is a table test that takes the output root to propose, plus functions for move and step to counter the honest
claims. The test asserts that the defender always wins and thus the output root is found to be invalid.
