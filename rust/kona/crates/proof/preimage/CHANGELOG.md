# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.1](https://github.com/op-rs/kona/compare/kona-preimage-v0.2.0...kona-preimage-v0.2.1) - 2025-01-07

### Fixed

- op-rs rename (#883)

## [0.2.0](https://github.com/op-rs/kona/compare/kona-preimage-v0.1.0...kona-preimage-v0.2.0) - 2024-12-03

### Added

- *(workspace)* Isolate FPVM-specific platform code ([#821](https://github.com/op-rs/kona/pull/821))

## [0.0.5](https://github.com/op-rs/kona/compare/kona-preimage-v0.0.4...kona-preimage-v0.0.5) - 2024-11-20

### Added

- *(preimage)* Decouple from `kona-common` ([#817](https://github.com/op-rs/kona/pull/817))

### Other

- *(driver)* use tracing macros ([#823](https://github.com/op-rs/kona/pull/823))
- *(workspace)* Reorganize SDK ([#816](https://github.com/op-rs/kona/pull/816))

## [0.0.4](https://github.com/op-rs/kona/compare/kona-preimage-v0.0.3...kona-preimage-v0.0.4) - 2024-10-25

### Added

- remove thiserror ([#735](https://github.com/op-rs/kona/pull/735))
- *(preimage/common)* Migrate to `thiserror` ([#543](https://github.com/op-rs/kona/pull/543))

### Fixed

- hashmap ([#732](https://github.com/op-rs/kona/pull/732))
- *(workspace)* hoist and fix lints ([#577](https://github.com/op-rs/kona/pull/577))
- *(preimage)* Improve error differentiation in preimage servers ([#535](https://github.com/op-rs/kona/pull/535))

### Other

- re-org imports ([#711](https://github.com/op-rs/kona/pull/711))
- *(preimage)* Test Coverage ([#634](https://github.com/op-rs/kona/pull/634))
- doc logos ([#609](https://github.com/op-rs/kona/pull/609))
- *(workspace)* Bump dependencies ([#550](https://github.com/op-rs/kona/pull/550))
- *(workspace)* Allow stdlib in `cfg(test)` ([#548](https://github.com/op-rs/kona/pull/548))

## [0.0.3](https://github.com/op-rs/kona/compare/kona-preimage-v0.0.2...kona-preimage-v0.0.3) - 2024-09-04

### Added
- *(workspace)* Workspace Re-exports ([#468](https://github.com/op-rs/kona/pull/468))
- *(client)* providers generic over oracles ([#336](https://github.com/op-rs/kona/pull/336))

### Fixed
- *(workspace)* Add Unused Dependency Lint ([#453](https://github.com/op-rs/kona/pull/453))

### Other
- *(workspace)* Update for `op-rs` org transfer ([#474](https://github.com/op-rs/kona/pull/474))
- *(workspace)* Hoist Dependencies ([#466](https://github.com/op-rs/kona/pull/466))
- *(common)* Remove need for cursors in `NativeIO` ([#416](https://github.com/op-rs/kona/pull/416))
- *(preimage)* Remove dynamic dispatch ([#354](https://github.com/op-rs/kona/pull/354))

## [0.0.2](https://github.com/op-rs/kona/compare/kona-preimage-v0.0.1...kona-preimage-v0.0.2) - 2024-06-22

### Added
- *(preimage)* add serde feature flag to preimage crate for keys ([#271](https://github.com/op-rs/kona/pull/271))
- *(client)* Derivation integration ([#257](https://github.com/op-rs/kona/pull/257))
- *(ci)* Dependabot config ([#236](https://github.com/op-rs/kona/pull/236))
- *(client)* `StatelessL2BlockExecutor` ([#210](https://github.com/op-rs/kona/pull/210))
- *(client)* `BootInfo` ([#205](https://github.com/op-rs/kona/pull/205))
- *(preimage)* Async client handles ([#200](https://github.com/op-rs/kona/pull/200))
- *(host)* Add local key value store ([#189](https://github.com/op-rs/kona/pull/189))
- *(host)* Host program scaffold ([#184](https://github.com/op-rs/kona/pull/184))
- *(preimage)* Async server components ([#183](https://github.com/op-rs/kona/pull/183))
- *(precompile)* Add `precompile` key type ([#179](https://github.com/op-rs/kona/pull/179))
- *(preimage)* `OracleServer` + `HintReader` ([#96](https://github.com/op-rs/kona/pull/96))
- *(common)* Move from `RegisterSize` to native ptr size type ([#95](https://github.com/op-rs/kona/pull/95))
- *(workspace)* Add `rustfmt.toml`

### Other
- *(workspace)* Move `alloy-primitives` to workspace dependencies ([#103](https://github.com/op-rs/kona/pull/103))
- Make versions of packages independent ([#36](https://github.com/op-rs/kona/pull/36))
