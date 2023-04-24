# cannon-extra

This is a collection of extra / legacy scripts. Cannon v0.3 and onwards reduces scope.
- The challenge/response bookkeeping contract part of the fraud proof is in development
  separate from Cannon, hence moved into `extra` and deprecated.
- No usage of Merkle Patricia Tries (MPTs) for the VM state anymore.
- No `minigeth` anymore: see [`op-program`](https://github.com/ethereum-optimism/optimism/tree/develop/op-program) instead.
- No `mipigo` anymore: the ELF-loading and startup preparation is now part of the Go `mipsevm` toolset.

