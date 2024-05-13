# `SystemConfig` Invariants

## Gas limit boundaries
**Test:** [`SystemConfig.t.sol#L69`](../test/invariants/SystemConfig.t.sol#L69)

The gas limit of the `SystemConfig` contract can never be lower than the hard-coded lower bound or higher than the hard-coded upper bound. The lower bound must never be higher than the upper bound. 