
# v103 tag (revm v34.0.0)

* BAL (EIP-7928) support added to Database implementations.
* `GasParams` is new struct where you can set dynamic opcode gas params. Initialized and can be found in cfg.
  * Gas calculation functions moved from `revm-interpreter` to be part of gas params.
  * Gas constants moved from `revm_interpreter`::gas to `revm_context_interface::cfg::gas`
* `CreateInputs` struct fields made private with accessor pattern.
  * Use `CreateInputs::created_address()` getter (now cached).
* `Host::selfdestruct` signature changed to support OOG on cold load for target account.
* Inspector `log` function renamed:
  * `Inspector::log` renamed to `log` and `log_full`.
  * `log_full` default impl calls `log`.
  * `log_full` has `Interpreter` input while `log` does not.
  * `log` is called in places where Interpreter is not found.
* `PrecompileError::Other` now contains `Cow<'static, str>` instead of `&'static str`.
  * Allows setting both `&'static str` (no perf penalty) and `String` if needed.
* `JournaledAccount` struct added for tracking account changes.
  * `JournalTr` functions that fetch account now return a ref.
  * New function `load_account_mut` returns `JournaledAccount`.
* `JournalTr::load_account_code` deprecated, renamed to `JournalTr::load_account_with_code`.
* `JournalTr::warm_account_and_storage` and `JournalTr::warm_account` removed.
  * Access list is now separate from the main Journal EvmState.
  * Use `JournalTr::warm_access_list` to import access list.
* Declarative macros `tri!`, `gas_or_fail!`, `otry!` removed from `revm-interpreter`.
* `MemoryGas` API signature changes.
* Removed deprecated methods including `into_plain_state`, `regenerate_hash`.
* `State.bal_state` field added (breaks struct literal constructors).
* `DatabaseCommitExt::drain_balances` and `increment_balances` added.
* First precompile error now bubbles up with detailed error messages.
  * New `PrecompileError` variants added.

# 102 tag ( revm v33.1.0)

No breaking changes

# 101 tag ( revm v33.1.0)

No breaking changes

# v100 tag (revm v33.0.0)

* Additionally to v99 version:
  * `Host::selfdestruct` function got changed to support oog on cold load for target account. 

# v99 tag ( revm v32.0.0 )

(yanked version)

* Added support for transmitting Logs set from precompiles.
   * `Inspector::log` function got renamed to `log` and `log_full`
   * `log_full` default impl will call `log`
   * difference is that `log_full` has `Interpreter` input while `log` does not
     and `log` will be called in places where Interpreter is not found.
* `PrecompileError` now contains `Other` as `Cow<'static, str>`
   * It allows setting both `&'static str` that is without perf penalty and `String` if needed.

# v98 tag

No breaking changes.

# v97 tag

No breaking changes.

# v96 tag

No breaking changes.

# v95 tag ( revm v31.0.0)

* Cfg added `memory_limit()` function
* `JournaledAccount` have been added that wraps account changes, touching and creating journal entry.
  * Past function that fetches account now return a ref and new function `load_account_mut` now return `JournaledAccount`
  * `JournalEntry` type is added to `JournalTr` so JournaledAccount can create it.
* `JournalTr::load_account_code` is deprecated/renamed to `JournalTr::load_account_with_code`
* `JournalTr::warm_account_and_storage` and `JournalTr::warm_account` are removed as access list is now separate from the main
  Journal EvmState. Function that imports access list to the Journal is `JournalTr::warm_access_list`

# v94 tag ( op-revm )

No breaking changes

# v93 tag ( op-revm )

No breaking changes

# v93 tag ( revm v30.1.0)

No breaking changes

# v92 tag ( revm v30.1.2)

No breaking changes

# v91 tag ( revm v30.1.1)

No breaking changes

# v90 tag ( revm v30.1.0)

* Removal of some deprecated functions. `into_plain_state`, `regenerate_hash` were deprecated few releases ago.

# v89 tag ( op-revm)

No breaking changes

# v88 tag (revm v30.0.0)

* `ContextTr`, `EvmTr` gained `all_mut()` and `all()` functions.
  * `InspectorEvmTr` got `all_inspector` and `all_mut_inspector` functions.
  * For custom Evm, you only need to implement `all_*` functions.
* `InspectorFrame` got changed to allow CustomFrame to be added.
  * If there is no need for inspection `fn eth_frame` can return None.
* `kzg-rs` feature and library removed. Default KZG implementation is now c-kzg.
  * for no-std variant, arkworks lib is used.
* Opcodes that load account/storage will not load item if there is not enough gas for cold load.
  * This is in preparation for BAL.
  * Journal functions for loading now have skip_cold_load bool.
* `libsecp256k1` parity library is deprecated and removed.
* `JumpTable` internal representation changed from `BitVec` to `Bytes`. No changes in API.
* `SpecId` enum gained new `Amsterdam` variant and `OpSpecId` gained `Jovian` variant.
* `SELFDESTRUCT` constant renamed to `SELFDESTRUCT_REFUND`.
* `FrameStack::push` and `FrameStack::end_init` marked as `unsafe` as it can cause UB.
* First precompile error is now bubble up with detailed error messages. New `PrecompileError` variants added.
* Batch execution errors now include transaction index.
* `CallInput` now contains bytecode that is going to be executed (Previously it had address).
  * This allows skipping double bytecode fetching.
* `InvalidTransaction` enum gained `Str(Cow<'static, str>)` variant for custom error messages.
* `calc_excess_blob_gas` and `calc_excess_blob_gas` removes as they are unused and not accurate for Prague.

# v86 tag (revm v29.0.0)

* `PrecompileWithAddress` is renamed to `Precompile` and it became a struct.
  * `Precompile` contains`PrecompileId`, `Address` and function.
  * The reason is adding `PrecompileId` as it is needed for fusaka hardfork

# v85 tag (revm v28.0.1) from v84 tag (revm v28.0.0)

Forward compatible version.

# v84 tag (revm v28.0.0) from v83 tag (revm v27.1.0)

* `SystemCallEvm` functions got renamed and old ones are deprecated. Renaming is done to align it with other API calls.
   * `transact_system_call_finalize` is now `system_call`.
   * `transact_system_call` is now `system_call_one`.
* `ExtBytecode::regenerate_hash` got deprecated in support for `get_or_calculate_hash` or `calculate_hash`.
* Precompiles:
  * Bn128 renamed to Bn254. https://github.com/ethereum/EIPs/pull/10029#issue-3240867404
* `InstructionResult` now starts from 1 (previous 0) for perf purposes.
* In `JournalInner` previous `precompiles`, `warm_coinbase_address` and `warm_preloaded_addresses` pub fields are now moved to `warm_addresses` to encapsulate addresses that are warm by default. All access list account are all loaded from database.


# v83 tag (revm v27.1.0) from v82 tag (revm v27.0.3)

* `ContextTr` gained `Host` supertrait.
  * Previously Host was implemented for any T that has ContextTr, this restricts specializations.
  https://github.com/bluealloy/revm/issues/2732
  * `Host` is moved to `revm-context-interface`
  * If you custom struct that implement `ContextTr` you would need to manually implement `Host` trait, in most cases no action needed.
* In `revm-interpreter`, fn `cast_slice_to_u256` was removed and `push_slice` fn is added to `StackTrait`.
* `PrecompileOutput` now contains revert flag.
  * It is safe to put to false.
* In `kzg` and `blake2` modules few internal functions were made private or removed.

# v80 tag (revm v27.0.0) -> v81 tag ( revm v27.0.1)

* Inspector fn `step_end` is now called even if Inspector `step` sets the action. Previously this was not the  case.
    * https://github.com/bluealloy/revm/pull/2687
    * this additionally fixes panic bug where `bytecode.opcode()` would panic in `step_end`

# v70 tag (revm v22.0.2) -> v71 tag ( revm v23.0.0)

* Removal of `EvmData`.
    * It got flattened and ctx/inspector fields moved directly to Evm, additional layering didn't have purpose.
* Merging of `Handler`'s `validate_tx_against_state` and `deduct_caller` into one function `validate_against_state_and_deduct_caller`
    * If you dont override those functions there is no action. If you do please look at `pre_execution::validate_against_state_and_deduct_caller`
    function or `OpHandler` for examples of migration.
* Breaking changed for EOF to support eof-devnet1. 
* `SharedMemory` is not longer Rc<RefCell<>> and internally uses Rc<RefCell<Vec<u8>>> buffer.
    * No action if you dont use it inside Interpreter.
* In `JournalExt` fn `last_journal_mut()` is renamed to `journal_mut()`
* EOF is disabled from Osaka and not accessible.

# v68 tag (revm v21.0.0) -> v70 tag ( revm v22.0.2)

No breaking changes

# v67 tag (revm v21.0.0) -> v68 tag ( revm v22.0.0)

* No code breaking changes
* alloy-primitives bumped to v1.0.0 and we had a major bump because of it.
