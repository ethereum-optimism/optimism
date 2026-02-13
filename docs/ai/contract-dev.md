# Smart Contract Development

This document provides guidance for AI agents working with smart contracts in the OP Stack.

## Non-Idempotent Initializers

When reviewing `initialize()` or `reinitializer` functions, check whether the function is **idempotent** — calling it multiple times with the same arguments should produce the same end state as calling it once.

### The Risk

Proxied contracts in the OP Stack can be re-initialized during upgrades (via `reinitializer(version)`). Orchestrators like `OPContractsManagerV2._apply()` call `initialize()` on contracts that may already hold state from a previous initialization. If the initializer is not idempotent, re-initialization can corrupt state.

**Example**: `ETHLockbox.initialize()` calls `_authorizePortal()` for each portal passed in. Currently safe because `_authorizePortal()` is idempotent — setting `authorizedPortals[portal] = true` twice has the same effect as once. But if someone later added a portal count that increments on each authorization, re-initialization would double-count portals.

### What Makes an Initializer Non-Idempotent

- Incrementing counters or nonces
- Appending to arrays (creates duplicates on re-init)
- External calls with lasting side-effects (e.g., minting tokens, sending ETH)
- Operations that depend on prior state (e.g., "add 10 to balance" vs "set balance to 10")


### Other Reasons an Initializer may be Unsafe to Re-Run

- Emitting events that trigger off-chain actions (e.g., indexers that process each event exactly once)
- Overwriting a variable that other contracts or off-chain systems already depend on (e.g., resetting a registry address that live contracts are pointing to, or changing a config value that should be immutable after first init)

### Rule

Non-idempotent or unsafe-to-rerun behavior in `initialize()` / `reinitializer` functions is **disallowed** unless the consequences are explicitly acknowledged in a `@notice` comment on the function. The comment must explain why the non-idempotent behavior is safe given how callers use the function.

Without this comment, the code must not be approved.

### Review Checklist

When reviewing changes to `initialize()` or its callers:

1. **Is every operation in this initializer idempotent?** Assigning a variable to a fixed value is idempotent. Incrementing, appending, or calling external contracts may not be.
2. **Could overwriting any variable be unsafe?** Some values should only be set once — overwriting them during re-initialization could break other contracts or systems that depend on the original value.
3. **Can this contract be re-initialized?** Check for `reinitializer` modifier. If it only uses `initializer` (one-shot), the risk does not apply.
4. **If non-idempotent or unsafe behavior exists, is there a `@notice` comment acknowledging it?** The comment must explain why it's safe. If the comment is missing, flag it as a blocking issue.
