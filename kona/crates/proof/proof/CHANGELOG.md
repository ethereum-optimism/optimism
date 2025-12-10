# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.3](https://github.com/op-rs/kona/compare/kona-proof-v0.2.2...kona-proof-v0.2.3) - 2025-01-16

### Added

- *(client)* Interop binary (#903)

### Other

- Update Maili Deps (#908)

## [0.2.2](https://github.com/op-rs/kona/compare/kona-proof-v0.2.1...kona-proof-v0.2.2) - 2025-01-13

### Other

- *(deps)* Replace dep `alloy-rs/op-alloy-registry`->`op-rs/maili-registry` (#892)
- *(deps)* Replace dep `alloy-rs/op-alloy-protocol`->`op-rs/maili-protocol` (#890)

## [0.2.1](https://github.com/op-rs/kona/compare/kona-proof-v0.2.0...kona-proof-v0.2.1) - 2025-01-07

### Fixed

- op-rs rename (#883)

## [0.2.0](https://github.com/op-rs/kona/compare/kona-proof-v0.1.0...kona-proof-v0.2.0) - 2024-12-03

### Added

- *(driver)* wait for engine ([#851](https://github.com/op-rs/kona/pull/851))
- *(client)* Invalidate impossibly old claims ([#852](https://github.com/op-rs/kona/pull/852))
- *(driver)* refines the executor interface for the driver ([#850](https://github.com/op-rs/kona/pull/850))
- *(workspace)* Isolate FPVM-specific platform code ([#821](https://github.com/op-rs/kona/pull/821))

### Fixed

- bump ([#865](https://github.com/op-rs/kona/pull/865))

### Other

- update deps and clean up misc features ([#864](https://github.com/op-rs/kona/pull/864))
- *(host)* Hint Parsing Cleanup ([#844](https://github.com/op-rs/kona/pull/844))
- *(derive)* remove indexed blob hash ([#847](https://github.com/op-rs/kona/pull/847))
- L2ExecutePayloadProof Hint Type ([#832](https://github.com/op-rs/kona/pull/832))

## [0.0.1](https://github.com/op-rs/kona/releases/tag/kona-proof-v0.0.1) - 2024-11-20

### Added

- *(workspace)* `kona-proof` ([#818](https://github.com/op-rs/kona/pull/818))

### Fixed

- imports ([#829](https://github.com/op-rs/kona/pull/829))

### Other

- op-alloy 0.6.8 ([#830](https://github.com/op-rs/kona/pull/830))
- *(driver)* use tracing macros ([#823](https://github.com/op-rs/kona/pull/823))

## [0.0.4](https://github.com/op-rs/kona/compare/kona-common-v0.0.3...kona-common-v0.0.4) - 2024-10-25

### Added

- remove thiserror ([#735](https://github.com/op-rs/kona/pull/735))
- *(preimage/common)* Migrate to `thiserror` ([#543](https://github.com/op-rs/kona/pull/543))

### Fixed

- *(workspace)* hoist and fix lints ([#577](https://github.com/op-rs/kona/pull/577))

### Other

- re-org imports ([#711](https://github.com/op-rs/kona/pull/711))
- *(preimage)* Test Coverage ([#634](https://github.com/op-rs/kona/pull/634))
- test coverage for common ([#629](https://github.com/op-rs/kona/pull/629))
- doc logos ([#609](https://github.com/op-rs/kona/pull/609))
- *(workspace)* Allow stdlib in `cfg(test)` ([#548](https://github.com/op-rs/kona/pull/548))

## [0.0.3](https://github.com/op-rs/kona/compare/kona-common-v0.0.2...kona-common-v0.0.3) - 2024-09-04

### Added
- add zkvm target for io ([#394](https://github.com/op-rs/kona/pull/394))

### Other
- *(workspace)* Update for `op-rs` org transfer ([#474](https://github.com/op-rs/kona/pull/474))
- *(workspace)* Hoist Dependencies ([#466](https://github.com/op-rs/kona/pull/466))
- *(bin)* Remove `kt` ([#461](https://github.com/op-rs/kona/pull/461))
- *(common)* Remove need for cursors in `NativeIO` ([#416](https://github.com/op-rs/kona/pull/416))

## [0.0.2](https://github.com/op-rs/kona/compare/kona-common-v0.0.1...kona-common-v0.0.2) - 2024-06-22

### Added
- *(client)* Derivation integration ([#257](https://github.com/op-rs/kona/pull/257))
- *(client/host)* Oracle-backed Blob fetcher ([#255](https://github.com/op-rs/kona/pull/255))
- *(host)* Host program scaffold ([#184](https://github.com/op-rs/kona/pull/184))
- *(preimage)* `OracleServer` + `HintReader` ([#96](https://github.com/op-rs/kona/pull/96))
- *(common)* Move from `RegisterSize` to native ptr size type ([#95](https://github.com/op-rs/kona/pull/95))
- *(workspace)* Add `rustfmt.toml`

### Fixed
- *(common)* Pipe IO support ([#282](https://github.com/op-rs/kona/pull/282))

### Other
- *(common)* Use `Box::leak` rather than `mem::forget` ([#180](https://github.com/op-rs/kona/pull/180))
- Add simple blocking async executor ([#38](https://github.com/op-rs/kona/pull/38))
- Make versions of packages independent ([#36](https://github.com/op-rs/kona/pull/36))
