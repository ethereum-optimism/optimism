# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.3](https://github.com/op-rs/kona/compare/kona-executor-v0.2.2...kona-executor-v0.2.3) - 2025-01-16

### Other

- Update Maili Deps (#908)

## [0.2.2](https://github.com/op-rs/kona/compare/kona-executor-v0.2.1...kona-executor-v0.2.2) - 2025-01-13

### Other

- update Cargo.toml dependencies

## [0.2.1](https://github.com/op-rs/kona/compare/kona-executor-v0.2.0...kona-executor-v0.2.1) - 2025-01-07

### Fixed

- op-rs rename (#883)

### Other

- Bump Dependencies (#880)
- remove redundant words in comment (#882)
- Isthmus Withdrawals Root (#881)

## [0.2.0](https://github.com/op-rs/kona/compare/kona-executor-v0.1.0...kona-executor-v0.2.0) - 2024-12-03

### Added

- *(driver)* refines the executor interface for the driver ([#850](https://github.com/op-rs/kona/pull/850))

### Fixed

- bump ([#855](https://github.com/op-rs/kona/pull/855))
- use non problematic hashmap fns ([#853](https://github.com/op-rs/kona/pull/853))

### Other

- update deps and clean up misc features ([#864](https://github.com/op-rs/kona/pull/864))

## [0.0.6](https://github.com/op-rs/kona/compare/kona-executor-v0.0.5...kona-executor-v0.0.6) - 2024-11-20

### Added

- *(mpt)* Extend `TrieProvider` in `kona-executor` ([#813](https://github.com/op-rs/kona/pull/813))

### Other

- *(driver)* use tracing macros ([#823](https://github.com/op-rs/kona/pull/823))
- *(workspace)* Migrate back to `thiserror` v2 ([#811](https://github.com/op-rs/kona/pull/811))

## [0.0.5](https://github.com/op-rs/kona/compare/kona-executor-v0.0.4...kona-executor-v0.0.5) - 2024-11-06

### Added

- *(TrieProvider)* Abstract TrieNode retrieval ([#787](https://github.com/op-rs/kona/pull/787))

### Other

- *(executor)* rm upstream util ([#755](https://github.com/op-rs/kona/pull/755))

## [0.0.4](https://github.com/op-rs/kona/compare/kona-executor-v0.0.3...kona-executor-v0.0.4) - 2024-10-29

### Other

- update Cargo.toml dependencies

## [0.0.3](https://github.com/op-rs/kona/compare/kona-executor-v0.0.2...kona-executor-v0.0.3) - 2024-10-25

### Added

- remove thiserror ([#735](https://github.com/op-rs/kona/pull/735))
- *(executor)* Clean ups ([#719](https://github.com/op-rs/kona/pull/719))
- *(executor)* EIP-1559 configurability spec updates ([#716](https://github.com/op-rs/kona/pull/716))
- *(executor)* Update EIP-1559 configurability ([#648](https://github.com/op-rs/kona/pull/648))
- *(executor)* Use EIP-1559 parameters from payload attributes ([#616](https://github.com/op-rs/kona/pull/616))
- *(derive)* bump op-alloy dep ([#605](https://github.com/op-rs/kona/pull/605))
- kona-providers ([#596](https://github.com/op-rs/kona/pull/596))
- *(executor)* Migrate to `thiserror` ([#544](https://github.com/op-rs/kona/pull/544))
- *(mpt)* Migrate to `thiserror` ([#541](https://github.com/op-rs/kona/pull/541))
- *(primitives)* Remove Attributes ([#529](https://github.com/op-rs/kona/pull/529))
- large dependency update ([#528](https://github.com/op-rs/kona/pull/528))

### Fixed

- *(executor)* Holocene EIP-1559 params in Header ([#622](https://github.com/op-rs/kona/pull/622))
- *(workspace)* hoist and fix lints ([#577](https://github.com/op-rs/kona/pull/577))

### Other

- re-org imports ([#711](https://github.com/op-rs/kona/pull/711))
- *(workspace)* Removes Primitives ([#638](https://github.com/op-rs/kona/pull/638))
- *(executor)* move todo to issue: ([#680](https://github.com/op-rs/kona/pull/680))
- *(executor)* Cover Builder ([#676](https://github.com/op-rs/kona/pull/676))
- *(executor)* Use Upstreamed op-alloy Methods  ([#651](https://github.com/op-rs/kona/pull/651))
- *(executor)* Test Coverage over Executor Utilities ([#650](https://github.com/op-rs/kona/pull/650))
- doc logos ([#609](https://github.com/op-rs/kona/pull/609))
- *(workspace)* Allow stdlib in `cfg(test)` ([#548](https://github.com/op-rs/kona/pull/548))
- Bumps Dependency Versions ([#520](https://github.com/op-rs/kona/pull/520))
- *(primitives)* rm RawTransaction ([#505](https://github.com/op-rs/kona/pull/505))

## [0.0.2](https://github.com/op-rs/kona/compare/kona-executor-v0.0.1...kona-executor-v0.0.2) - 2024-09-04

### Added
- *(executor)* Expose full revm Handler ([#475](https://github.com/op-rs/kona/pull/475))
- *(workspace)* Workspace Re-exports ([#468](https://github.com/op-rs/kona/pull/468))
- *(executor)* `StatelessL2BlockExecutor` benchmarks ([#350](https://github.com/op-rs/kona/pull/350))
- *(executor)* Generic precompile overrides ([#340](https://github.com/op-rs/kona/pull/340))
- *(executor)* Builder pattern for `StatelessL2BlockExecutor` ([#339](https://github.com/op-rs/kona/pull/339))

### Fixed
- *(workspace)* Use published `revm` version ([#459](https://github.com/op-rs/kona/pull/459))
- downgrade for release plz ([#458](https://github.com/op-rs/kona/pull/458))
- *(workspace)* Add Unused Dependency Lint ([#453](https://github.com/op-rs/kona/pull/453))
- Don't hold onto intermediate execution cache across block boundaries ([#396](https://github.com/op-rs/kona/pull/396))

### Other
- *(workspace)* Alloy Version Bumps ([#467](https://github.com/op-rs/kona/pull/467))
- *(workspace)* Update for `op-rs` org transfer ([#474](https://github.com/op-rs/kona/pull/474))
- *(workspace)* Hoist Dependencies ([#466](https://github.com/op-rs/kona/pull/466))
- refactor types out of kona-derive ([#454](https://github.com/op-rs/kona/pull/454))
- *(deps)* Bump revm version to v13 ([#422](https://github.com/op-rs/kona/pull/422))

## [0.0.1](https://github.com/op-rs/kona/releases/tag/kona-executor-v0.0.1) - 2024-06-22

### Other
- *(workspace)* Prep release ([#301](https://github.com/op-rs/kona/pull/301))
- version dependencies ([#296](https://github.com/op-rs/kona/pull/296))
- *(deps)* fast forward op alloy dep ([#267](https://github.com/op-rs/kona/pull/267))
- *(workspace)* `kona-executor` ([#259](https://github.com/op-rs/kona/pull/259))
