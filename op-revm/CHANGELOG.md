# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [15.0.0](https://github.com/bluealloy/revm/compare/op-revm-v14.1.0...op-revm-v15.0.0) - 2026-01-15

### Added

- new gas params, tx initial gas and codedeposit ([#3260](https://github.com/bluealloy/revm/pull/3260))
- move GasParams to Cfg ([#3229](https://github.com/bluealloy/revm/pull/3229))
- BAL EIP-7928 ([#3070](https://github.com/bluealloy/revm/pull/3070))
- early return if the l1 fee scalar is zero ([#3213](https://github.com/bluealloy/revm/pull/3213))
- Restrict Database::Error. JournaledAccountTr ([#3199](https://github.com/bluealloy/revm/pull/3199))

### Other

- fix typos, grammar errors, and improve documentation consistency ([#3294](https://github.com/bluealloy/revm/pull/3294))
- happy new year, 2026 licence ([#3272](https://github.com/bluealloy/revm/pull/3272))
- Remove redundant tx fetch in Optimism handler gas accounting ([#3220](https://github.com/bluealloy/revm/pull/3220))
- *(fmt)* merge all imports ([#3184](https://github.com/bluealloy/revm/pull/3184))

## [14.1.0](https://github.com/bluealloy/revm/compare/op-revm-v14.0.0...op-revm-v14.1.0) - 2025-11-14

### Fixed

- *(op-revm)* return error when enveloped_tx is missing ([#3143](https://github.com/bluealloy/revm/pull/3143))

## [14.0.0](https://github.com/bluealloy/revm/compare/op-revm-v12.0.2...op-revm-v14.0.0) - 2025-11-10

### Added

- *(precompiles)* add performant PrecompileError::OtherCowStr variant ([#3144](https://github.com/bluealloy/revm/pull/3144))
- process precompile logs to inspector ([#3148](https://github.com/bluealloy/revm/pull/3148))

## [12.0.2](https://github.com/bluealloy/revm/compare/op-revm-v12.0.1...op-revm-v12.0.2) - 2025-11-10

### Fixed

- *(op)* Ensure L1Block account is always loaded ([#3150](https://github.com/bluealloy/revm/pull/3150))

## [12.0.1](https://github.com/bluealloy/revm/compare/op-revm-v12.0.0...op-revm-v12.0.1) - 2025-11-07

### Other

- updated the following local packages: revm

## [12.0.0](https://github.com/bluealloy/revm/compare/op-revm-v11.2.0...op-revm-v12.0.0) - 2025-10-30

### Added

- JournaledAccount, a nice way to update and track changes ([#3086](https://github.com/bluealloy/revm/pull/3086))

### Fixed

- *(jovian)* fixes the DA footprint update storage slot. fix l1 fork associated with Jovian. ([#3120](https://github.com/bluealloy/revm/pull/3120))
- *(op-revm)* add missing enveloped_tx validation in validate_env ([#3094](https://github.com/bluealloy/revm/pull/3094))

### Other

- *(op)* use helper function in validate against state ([#3069](https://github.com/bluealloy/revm/pull/3069))


## [11.3.0](https://github.com/bluealloy/revm/compare/op-revm-v11.2.0...op-revm-v11.3.0) - 2025-10-28

### Added

- *(precompiles/jovian)* add jovian precompiles to revm ([#3128](https://github.com/bluealloy/revm/pull/3128))


## [11.2.0](https://github.com/bluealloy/revm/compare/op-revm-v11.1.2...op-revm-v11.2.0) - 2025-10-17

### Other

- updated the following local packages: revm

## [11.1.2](https://github.com/bluealloy/revm/compare/op-revm-v11.1.1...op-revm-v11.1.2) - 2025-10-15

### Other

- updated the following local packages: revm

## [11.1.1](https://github.com/bluealloy/revm/compare/op-revm-v11.1.0...op-revm-v11.1.1) - 2025-10-15

### Other

- updated the following local packages: revm

## [11.1.0](https://github.com/bluealloy/revm/compare/op-revm-v11.0.0...op-revm-v11.1.0) - 2025-10-09

### Fixed

- *(op-revm)* return error instead of panic when enveloped_tx is missing ([#3055](https://github.com/bluealloy/revm/pull/3055))

### Other

- *(op)* backport of #3073 fix for l1block info ([#3076](https://github.com/bluealloy/revm/pull/3076))
- backport v89 changelog ([#3075](https://github.com/bluealloy/revm/pull/3075))
- *(op)* split paths for deposit tx in caller deduction ([#3041](https://github.com/bluealloy/revm/pull/3041))

## [10.1.1](https://github.com/bluealloy/revm/compare/op-revm-v10.0.0...op-revm-v10.1.1) - 2025-09-23

## [11.0.0](https://github.com/bluealloy/revm/compare/op-revm-v10.1.0...op-revm-v11.0.0) - 2025-10-07

### Added

- *(jovian)* add da footprint block limit. ([#3003](https://github.com/bluealloy/revm/pull/3003))
- *(op-revm)* implement jovian operator fee fix ([#2996](https://github.com/bluealloy/revm/pull/2996))
- *(op-revm)* Add an option to disable "fee-charge" on `op-revm` ([#2980](https://github.com/bluealloy/revm/pull/2980))
- [**breaking**] Remove kzg-rs ([#2909](https://github.com/bluealloy/revm/pull/2909))

### Fixed

- add missing is_fee_charge_disabled check ([#3007](https://github.com/bluealloy/revm/pull/3007))
- Apply spelling corrections from PRs #2926, #2915, #2908 ([#2978](https://github.com/bluealloy/revm/pull/2978))
- *(op-revm)* clear enveloped_tx for deposit txs in build_fill and align docs ([#2957](https://github.com/bluealloy/revm/pull/2957))

### Other

- changelog update for v87 ([#3056](https://github.com/bluealloy/revm/pull/3056))
- add boundless ([#3043](https://github.com/bluealloy/revm/pull/3043))
- helper function gas_balance_spending ([#3030](https://github.com/bluealloy/revm/pull/3030))
- helper caller_initial_modification added ([#3032](https://github.com/bluealloy/revm/pull/3032))
- EvmTr and InspectorEvmTr receive all/all_mut fn ([#3037](https://github.com/bluealloy/revm/pull/3037))
- add ensure_enough_balance helper ([#3033](https://github.com/bluealloy/revm/pull/3033))
- *(op-revm)* propagate optional_fee_charge feature ([#3020](https://github.com/bluealloy/revm/pull/3020))
- Set l2_block in try_fetch for pre-Isthmus forks; add reload tests ([#2994](https://github.com/bluealloy/revm/pull/2994))
- prealloc few frames ([#2965](https://github.com/bluealloy/revm/pull/2965))
- treat empty input as zero operator fee in operator_fee_charge ([#2973](https://github.com/bluealloy/revm/pull/2973))
- add SECURITY.md ([#2956](https://github.com/bluealloy/revm/pull/2956))
- *(op-revm)* rm redundant phantom ([#2943](https://github.com/bluealloy/revm/pull/2943))
- *(op-revm)* add serialize DepositTransactionParts test ([#2942](https://github.com/bluealloy/revm/pull/2942))
- *(handler)* provide `&CallInputs`to`PrecompileProvider::run` ([#2921](https://github.com/bluealloy/revm/pull/2921))

## [10.1.0](https://github.com/bluealloy/revm/compare/op-revm-v10.0.0...op-revm-v10.1.0) - 2025-09-23

### Added

- *(op-revm)* Add an option to disable "fee-charge" on `op-revm` ([#2980](https://github.com/bluealloy/revm/pull/2980))

## [10.0.0](https://github.com/bluealloy/revm/compare/op-revm-v9.0.1...op-revm-v10.0.0) - 2025-08-23

### Added

- *(fusaka)* Add PrecompileId ([#2904](https://github.com/bluealloy/revm/pull/2904))

### Fixed

- *(handler)* correct transaction ID decrement logic ([#2892](https://github.com/bluealloy/revm/pull/2892))

## [9.0.1](https://github.com/bluealloy/revm/compare/op-revm-v9.0.0...op-revm-v9.0.1) - 2025-08-12

### Other

- updated the following local packages: revm

## [9.0.0](https://github.com/bluealloy/revm/compare/op-revm-v8.1.0...op-revm-v9.0.0) - 2025-08-06

### Added

- fix renamed functions for system_call ([#2824](https://github.com/bluealloy/revm/pull/2824))
- refactor test utils ([#2813](https://github.com/bluealloy/revm/pull/2813))
- add system transaction inspection support ([#2808](https://github.com/bluealloy/revm/pull/2808))
- Align naming of SystemCallEvm function to ExecuteEvm ([#2814](https://github.com/bluealloy/revm/pull/2814))
- rename bn128 to bn254 for Ethereum standard consistency ([#2810](https://github.com/bluealloy/revm/pull/2810))

### Fixed

- *(op-revm)* system tx not enveloped ([#2807](https://github.com/bluealloy/revm/pull/2807))
- nonce changed is not reverted in journal if fail due to insufficient balance ([#2805](https://github.com/bluealloy/revm/pull/2805))

### Other

- update README.md ([#2842](https://github.com/bluealloy/revm/pull/2842))
- *(op-revm)* Adds caller nonce assertion to op-revm intergation tests ([#2815](https://github.com/bluealloy/revm/pull/2815))
- *(op-revm)* Full test coverage `OpTransactionError` ([#2818](https://github.com/bluealloy/revm/pull/2818))
- Update test data for renamed tests ([#2817](https://github.com/bluealloy/revm/pull/2817))
- reuse global crypto provide idea ([#2786](https://github.com/bluealloy/revm/pull/2786))
- add rust-version and note about MSRV ([#2789](https://github.com/bluealloy/revm/pull/2789))
- add OnceLock re-export with no_std support ([#2787](https://github.com/bluealloy/revm/pull/2787))
- Add dyn Crypto trait to PrecompileFn ([#2772](https://github.com/bluealloy/revm/pull/2772))

## [8.1.0](https://github.com/bluealloy/revm/compare/op-revm-v8.0.3...op-revm-v8.1.0) - 2025-07-23

### Added

- *(osaka)* update EIP-7825 constant ([#2753](https://github.com/bluealloy/revm/pull/2753))

### Fixed

- gas deduction with `disable_balance_check` ([#2699](https://github.com/bluealloy/revm/pull/2699))

### Other

- *(op-revm)* test for optional balance check ([#2746](https://github.com/bluealloy/revm/pull/2746))
- change gas parameter to immutable reference ([#2702](https://github.com/bluealloy/revm/pull/2702))

## [8.0.3](https://github.com/bluealloy/revm/compare/op-revm-v8.0.2...op-revm-v8.0.3) - 2025-07-14

### Other

- simplify gas calculations by introducing a used() method ([#2703](https://github.com/bluealloy/revm/pull/2703))

## [8.0.2](https://github.com/bluealloy/revm/compare/op-revm-v8.0.1...op-revm-v8.0.2) - 2025-07-03

### Other

- updated the following local packages: revm

## [8.0.1](https://github.com/bluealloy/revm/compare/op-revm-v7.0.1...op-revm-v8.0.1) - 2025-06-30

### Added

- optional_eip3541 ([#2661](https://github.com/bluealloy/revm/pull/2661))

### Other

- cargo clippy --fix --all ([#2671](https://github.com/bluealloy/revm/pull/2671))
- *(op/handler)* verify caller account is touched by zero value transfer ([#2669](https://github.com/bluealloy/revm/pull/2669))
- use TxEnv::builder ([#2652](https://github.com/bluealloy/revm/pull/2652))

## [7.0.1](https://github.com/bluealloy/revm/compare/op-revm-v7.0.0...op-revm-v7.0.1) - 2025-06-20

### Fixed

- call stack_frame.clear() at end ([#2656](https://github.com/bluealloy/revm/pull/2656))

## [7.0.0](https://github.com/bluealloy/revm/compare/op-revm-v6.0.0...op-revm-v7.0.0) - 2025-06-19

### Added

- add fallible conversion from OpHaltReason to HaltReason ([#2649](https://github.com/bluealloy/revm/pull/2649))
- remove EOF ([#2644](https://github.com/bluealloy/revm/pull/2644))
- *(precompile)* rug/gmp-based modexp ([#2596](https://github.com/bluealloy/revm/pull/2596))
- enable P256 in Osaka ([#2601](https://github.com/bluealloy/revm/pull/2601))

### Other

- re-use frame allocation ([#2636](https://github.com/bluealloy/revm/pull/2636))
- rename `transact` methods ([#2616](https://github.com/bluealloy/revm/pull/2616))

## [6.0.0](https://github.com/bluealloy/revm/compare/op-revm-v5.0.1...op-revm-v6.0.0) - 2025-06-06

### Added

- add with_caller for system_transact ([#2587](https://github.com/bluealloy/revm/pull/2587))
- *(Osaka)* EIP-7825 tx limit cap ([#2575](https://github.com/bluealloy/revm/pull/2575))
- expand timestamp/block_number to u256 ([#2546](https://github.com/bluealloy/revm/pull/2546))
- transact multi tx ([#2517](https://github.com/bluealloy/revm/pull/2517))

### Fixed

- *(multitx)* Add local flags for create and selfdestruct ([#2581](https://github.com/bluealloy/revm/pull/2581))

### Other

- tag v75 revm v24.0.1 ([#2563](https://github.com/bluealloy/revm/pull/2563)) ([#2589](https://github.com/bluealloy/revm/pull/2589))
- *(op-revm)* impl type alias for Default OpEvm ([#2576](https://github.com/bluealloy/revm/pull/2576))
- *(docs)* add lints to database-interface and op-revm crates ([#2568](https://github.com/bluealloy/revm/pull/2568))
- ContextTr rm *_ref, and add *_mut fn ([#2560](https://github.com/bluealloy/revm/pull/2560))
- *(test)* preserve order of fields in json fixtures ([#2541](https://github.com/bluealloy/revm/pull/2541))

## [5.0.1](https://github.com/bluealloy/revm/compare/op-revm-v5.0.0...op-revm-v5.0.1) - 2025-05-31

### Other

- updated the following local packages: revm

## [5.0.0](https://github.com/bluealloy/revm/compare/op-revm-v4.0.2...op-revm-v5.0.0) - 2025-05-22

### Added

- *(op-revm)* add testdata comparison utility for EVM execution output ([#2525](https://github.com/bluealloy/revm/pull/2525))

### Other

- make crates.io version badge clickable ([#2526](https://github.com/bluealloy/revm/pull/2526))

## [4.0.2](https://github.com/bluealloy/revm/compare/op-revm-v4.0.1...op-revm-v4.0.2) - 2025-05-09

### Fixed

- *(op)* bump nonce on deposit ([#2503](https://github.com/bluealloy/revm/pull/2503))
- *(op)* call cleanup on local context ([#2499](https://github.com/bluealloy/revm/pull/2499))

### Other

- *(op)* revert previous and localize fix ([#2504](https://github.com/bluealloy/revm/pull/2504))

## [4.0.1](https://github.com/bluealloy/revm/compare/op-revm-v4.0.0...op-revm-v4.0.1) - 2025-05-09

### Fixed

- *(op)* mark caller account as touched ([#2495](https://github.com/bluealloy/revm/pull/2495))

### Other

- *(op)* Add test coverage to OP result module ([#2491](https://github.com/bluealloy/revm/pull/2491))
- *(op)* Add test coverage to `OpTransactionError` ([#2490](https://github.com/bluealloy/revm/pull/2490))

## [4.0.0](https://github.com/bluealloy/revm/compare/op-revm-v3.1.0...op-revm-v4.0.0) - 2025-05-07

Dependency bump

## [3.1.0](https://github.com/bluealloy/revm/compare/op-revm-v3.0.2...op-revm-v3.1.0) - 2025-05-07

### Added

- system_call switch order of inputs, address than bytes ([#2485](https://github.com/bluealloy/revm/pull/2485))
- *(Osaka)* disable EOF ([#2480](https://github.com/bluealloy/revm/pull/2480))
- skip cloning of call input from shared memory ([#2462](https://github.com/bluealloy/revm/pull/2462))
- *(Handler)* merge state validation with deduct_caller ([#2460](https://github.com/bluealloy/revm/pull/2460))
- *(tx)* Add Either RecoveredAuthorization ([#2448](https://github.com/bluealloy/revm/pull/2448))
- add precompiles getter to OpPrecompiles ([#2444](https://github.com/bluealloy/revm/pull/2444))
- *(EOF)* Changes needed for devnet-1 ([#2377](https://github.com/bluealloy/revm/pull/2377))

### Other

- *(op)* Set l2 block num in reloaded isthmus l1 block info ([#2465](https://github.com/bluealloy/revm/pull/2465))
- Add clones to FrameData ([#2482](https://github.com/bluealloy/revm/pull/2482))
- *(op)* Add test for verifying default OpSpecId update ([#2478](https://github.com/bluealloy/revm/pull/2478))
- copy edit The Book ([#2463](https://github.com/bluealloy/revm/pull/2463))
- bump dependency version ([#2431](https://github.com/bluealloy/revm/pull/2431))
- fixed broken link ([#2421](https://github.com/bluealloy/revm/pull/2421))
- backport from release branch ([#2415](https://github.com/bluealloy/revm/pull/2415)) ([#2416](https://github.com/bluealloy/revm/pull/2416))

## [3.0.2](https://github.com/bluealloy/revm/compare/op-revm-v3.0.1...op-revm-v3.0.2) - 2025-04-15

### Other

## [3.0.1](https://github.com/bluealloy/revm/compare/op-revm-v3.0.0...op-revm-v3.0.1) - 2025-04-13

### Fixed

- *(isthmus)* Add input size limitations to bls12-381 {G1/G2} MSM + pairing ([#2406](https://github.com/bluealloy/revm/pull/2406))

## [3.0.0](https://github.com/bluealloy/revm/compare/op-revm-v2.0.0...op-revm-v3.0.0) - 2025-04-09

### Added

- support for system calls ([#2350](https://github.com/bluealloy/revm/pull/2350))

### Other

- bump alloy 13.0.0 and alloy-primitives v1.0.0 ([#2394](https://github.com/bluealloy/revm/pull/2394))
- fixed `EIP` to `RIP` ([#2388](https://github.com/bluealloy/revm/pull/2388))
- clean unsed indicatif ([#2379](https://github.com/bluealloy/revm/pull/2379))
- *(op-inspector)* Add test for inspecting logs ([#2352](https://github.com/bluealloy/revm/pull/2352))
- *(op-tx)* Cover DepositTransactionParts constructor in test ([#2358](https://github.com/bluealloy/revm/pull/2358))
- add 0x prefix to b256! and address! calls ([#2345](https://github.com/bluealloy/revm/pull/2345))

## [2.0.0](https://github.com/bluealloy/revm/compare/op-revm-v1.0.0...op-revm-v2.0.0) - 2025-03-28

### Added

- cache precompile warming ([#2317](https://github.com/bluealloy/revm/pull/2317))
- Add arkworks wrapper for bls12-381 ([#2316](https://github.com/bluealloy/revm/pull/2316))
- provide more context to precompiles ([#2318](https://github.com/bluealloy/revm/pull/2318))
- Add a wrapper for arkworks for EIP196 ([#2305](https://github.com/bluealloy/revm/pull/2305))

### Fixed

- *(isthmus)* Correctly filter refunds for deposit transactions ([#2330](https://github.com/bluealloy/revm/pull/2330))

### Other

- Remove LATEST variant from SpecId enum ([#2299](https://github.com/bluealloy/revm/pull/2299))

## [1.0.0](https://github.com/bluealloy/revm/compare/op-revm-v1.0.0-alpha.6...op-revm-v1.0.0) - 2025-03-24

### Other

- *(op-precompiles)* Add test for checking that op default precompiles is updated ([#2291](https://github.com/bluealloy/revm/pull/2291))
- *(op-precompiles)* Add missing g2 add tests ([#2253](https://github.com/bluealloy/revm/pull/2253))

## [1.0.0-alpha.6](https://github.com/bluealloy/revm/compare/op-revm-v1.0.0-alpha.5...op-revm-v1.0.0-alpha.6) - 2025-03-21

### Added

- InspectEvm fn renames, inspector docs, book cleanup ([#2275](https://github.com/bluealloy/revm/pull/2275))
- Return Fatal error on bls precompiles if in no_std ([#2249](https://github.com/bluealloy/revm/pull/2249))
- Remove PrecompileError from PrecompileProvider ([#2233](https://github.com/bluealloy/revm/pull/2233))

### Fixed

- *(op)* deposit txs are identifier 126 or 0x7e not 0x7f ([#2237](https://github.com/bluealloy/revm/pull/2237))

### Other

- bring operator fee fixes to trunk ([#2273](https://github.com/bluealloy/revm/pull/2273))
- *(op-test-cov)* Add test for serializing deposit transaction parts ([#2267](https://github.com/bluealloy/revm/pull/2267))
- *(op-precompiles)* Check subset of l1 precompiles in op ([#2204](https://github.com/bluealloy/revm/pull/2204))
- *(op-handler)* Add test for halted deposit tx post regolith ([#2269](https://github.com/bluealloy/revm/pull/2269))
- *(op)* Remove redundant trait DepositTransaction ([#2265](https://github.com/bluealloy/revm/pull/2265))
- Fix sys deposit tx gas test ([#2263](https://github.com/bluealloy/revm/pull/2263))
- remove wrong `&mut` and duplicated spec ([#2276](https://github.com/bluealloy/revm/pull/2276))
- *(op-precompiles)* clean up op tx tests ([#2242](https://github.com/bluealloy/revm/pull/2242))
- make str to SpecId conversion fallible ([#2236](https://github.com/bluealloy/revm/pull/2236))
- *(op-precompiles)* Add tests for bls12-381 map fp to g ([#2241](https://github.com/bluealloy/revm/pull/2241))
- add a safe blst wrapper ([#2223](https://github.com/bluealloy/revm/pull/2223))
- *(op-precompiles)* Reuse tests for bls12-381 msm tests for pairing ([#2239](https://github.com/bluealloy/revm/pull/2239))
- *(op-precompiles)* add bls12-381 g2 add and msm tests ([#2231](https://github.com/bluealloy/revm/pull/2231))
- *(op-precompiles)* Add test for g1 msm ([#2227](https://github.com/bluealloy/revm/pull/2227))
- simplify single UT for OpSpecId compatibility. ([#2216](https://github.com/bluealloy/revm/pull/2216))
- use AccessListItem associated type instead of AccessList ([#2214](https://github.com/bluealloy/revm/pull/2214))

## [1.0.0-alpha.5](https://github.com/bluealloy/revm/compare/op-revm-v1.0.0-alpha.4...op-revm-v1.0.0-alpha.5) - 2025-03-16

### Added

- *(docs)* MyEvm example and book cleanup ([#2218](https://github.com/bluealloy/revm/pull/2218))
- add test for calling `bn128_pair` before and after granite ([#2200](https://github.com/bluealloy/revm/pull/2200))

### Other

- *(op-precompiles)* Add test for calling g1 add ([#2205](https://github.com/bluealloy/revm/pull/2205))
- *(op-test)* Clean up precompile tests ([#2206](https://github.com/bluealloy/revm/pull/2206))
- fix typo in method name ([#2202](https://github.com/bluealloy/revm/pull/2202))
- Add tests for checking fjord precompile activation ([#2199](https://github.com/bluealloy/revm/pull/2199))

## [1.0.0-alpha.4](https://github.com/bluealloy/revm/compare/op-revm-v1.0.0-alpha.3...op-revm-v1.0.0-alpha.4) - 2025-03-12

### Added

- Add tx/block to EvmExecution trait ([#2195](https://github.com/bluealloy/revm/pull/2195))
- rename inspect_previous to inspect_replay ([#2194](https://github.com/bluealloy/revm/pull/2194))

### Other

- add debug to precompiles type ([#2193](https://github.com/bluealloy/revm/pull/2193))

## [1.0.0-alpha.3](https://github.com/bluealloy/revm/compare/op-revm-v1.0.0-alpha.2...op-revm-v1.0.0-alpha.3) - 2025-03-11

### Fixed

- fix(op) enable proper precompiles p256 ([#2186](https://github.com/bluealloy/revm/pull/2186))
- *(op)* fix inspection call ([#2184](https://github.com/bluealloy/revm/pull/2184))
- correct propagate features ([#2177](https://github.com/bluealloy/revm/pull/2177))

### Other

- remove CTX phantomdata from precompile providers ([#2178](https://github.com/bluealloy/revm/pull/2178))

## [1.0.0-alpha.2](https://github.com/bluealloy/revm/compare/op-revm-v1.0.0-alpha.1...op-revm-v1.0.0-alpha.2) - 2025-03-10

### Other

- updated the following local packages: revm

## [1.0.0-alpha.1](https://github.com/bluealloy/revm/releases/tag/revm-optimism-v1.0.0-alpha.1) - 2025-02-16

### Added

- Split Inspector trait from EthHandler into standalone crate (#2075)
- Introduce Auth and AccessList traits (#2079)
- derive Eq for OpSpec (#2073)
- *(op)* Isthmus precompiles (#2054)
- Evm structure (Cached Instructions and Precompiles) (#2049)
- simplify InspectorContext (#2036)
- Context execution (#2013)
- EthHandler trait (#2001)
- extract and export `estimate_tx_compressed_size` (#1985)
- *(EIP-7623)* Increase calldata cost. backport from rel/v51 (#1965)
- simplify Transaction trait (#1959)
- Split inspector.rs (#1958)
- align Block trait (#1957)
- expose precompile address in Journal, DB::Error: StdError (#1956)
- add isthmus spec (#1938)
- integrate codspeed (#1935)
- Make Ctx journal generic (#1933)
- Restucturing Part7 Handler and Context rework (#1865)
- restructuring Part6 transaction crate (#1814)
- Restructuring Part3 inspector crate (#1788)
- restructure Part2 database crate (#1784)
- project restructuring Part1 (#1776)
- introducing EvmWiring, a chain-specific configuration (#1672)
- *(examples)* generate block traces (#895)
- implement EIP-4844 (#668)
- *(Shanghai)* All EIPs: push0, warm coinbase, limit/measure initcode (#376)
- Migrate `primitive_types::U256` to `ruint::Uint<256, 4>` (#239)
- Introduce ByteCode format, Update Readme (#156)

### Fixed

- make macro crate-agnostic (#1802)
- fix typos ([#620](https://github.com/bluealloy/revm/pull/620))

### Other

- set alpha.1 version
- backport op l1 fetch perf (#2076)
- remove OpSpec (#2074)
- Add helpers with_inspector with_precompile (#2063)
- *(op)* backport isthmus operator fee (#2059)
- Bump licence year to 2025 (#2058)
- rename OpHaltReason (#2042)
- simplify some generics (#2032)
- align crates versions (#1983)
- Make inspector use generics, rm associated types (#1934)
- add OpTransaction conversion tests (#1939)
- fix comments and docs into more sensible (#1920)
- Rename PRAGUE_EOF to OSAKA (#1903)
- refactor L1BlockInfo::tx_estimated_size_fjord (#1856)
- *(primitives)* replace HashMap re-exports with alloy_primitives::map (#1805)
- Test for l1 gas used and l1 fee for ecotone tx (#1748)
- *(deps)* bump anyhow from 1.0.88 to 1.0.89 (#1772)
- Bump new logo (#1735)
- *(README)* add rbuilder to used-by (#1585)
- added simular to used-by (#1521)
- add Trin to used by list (#1393)
- Fix typo in readme ([#1185](https://github.com/bluealloy/revm/pull/1185))
- Add Hardhat to the "Used by" list ([#1164](https://github.com/bluealloy/revm/pull/1164))
- Add VERBS to used by list ([#1141](https://github.com/bluealloy/revm/pull/1141))
- license date and revm docs (#1080)
- *(docs)* Update the benchmark docs to point to revm package (#906)
- *(docs)* Update top-level benchmark docs (#894)
- clang requirement (#784)
- Readme Updates (#756)
- Logo (#743)
- book workflow ([#537](https://github.com/bluealloy/revm/pull/537))
- add example to revm crate ([#468](https://github.com/bluealloy/revm/pull/468))
- Update README.md ([#424](https://github.com/bluealloy/revm/pull/424))
- add no_std to primitives ([#366](https://github.com/bluealloy/revm/pull/366))
- revm-precompiles to revm-precompile
- Bump v20, changelog ([#350](https://github.com/bluealloy/revm/pull/350))
- typos (#232)
- Add support for old forks. ([#191](https://github.com/bluealloy/revm/pull/191))
- revm bump 1.8. update libs. snailtracer rename ([#159](https://github.com/bluealloy/revm/pull/159))
- typo fixes
- fix readme typo
- Big Refactor. Machine to Interpreter. refactor instructions. call/create struct ([#52](https://github.com/bluealloy/revm/pull/52))
- readme. debuger update
- Bump revm v0.3.0. README updated
- readme
- Add time elapsed for tests
- readme updated
- Include Basefee into cost calc. readme change
- Initialize precompile accounts
- Status update. Taking a break
- Merkle calc. Tweaks and debugging for eip158
- Replace aurora bn lib with parity's. All Bn128Add/Mul/Pair tests passes
- TEMP
- one tab removed
- readme
- README Example simplified
- Gas calculation for Call/Create. Example Added
- readme usage
- README changes
- Static gas cost added
- Subroutine changelogs and reverts
- Readme postulates
- Spelling
- Restructure project
- First iteration. Machine is looking okay
