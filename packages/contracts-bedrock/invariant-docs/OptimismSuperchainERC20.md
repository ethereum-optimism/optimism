# `OptimismSuperchainERC20` Invariants

## sum of supertoken total supply across all chains is always <= to convert(legacy, super)- convert(super, legacy)
**Test:** [`OptimismSuperchainERC20#L36`](../test/invariants/OptimismSuperchainERC20#L36)



## sum of supertoken total supply across all chains is equal to convert(legacy, super)- convert(super, legacy) when all when all cross-chain messages are processed
**Test:** [`OptimismSuperchainERC20#L57`](../test/invariants/OptimismSuperchainERC20#L57)



## many other assertion mode invariants are also defined  under `test/invariants/OptimismSuperchainERC20/fuzz/` .
**Test:** [`OptimismSuperchainERC20#L80`](../test/invariants/OptimismSuperchainERC20#L80)

since setting`fail_on_revert=false` also ignores StdAssertion failures, this invariant explicitly asks the handler for assertion test failures 