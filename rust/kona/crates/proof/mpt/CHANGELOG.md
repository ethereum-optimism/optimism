# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.2](https://github.com/op-rs/kona/compare/kona-mpt-v0.1.1...kona-mpt-v0.1.2) - 2025-01-07

### Fixed

- op-rs rename (#883)

### Other

- Bump Dependencies (#880)

## [0.1.1](https://github.com/op-rs/kona/compare/kona-mpt-v0.1.0...kona-mpt-v0.1.1) - 2024-12-03

### Other

- update deps and clean up misc features ([#864](https://github.com/op-rs/kona/pull/864))

## [0.0.7](https://github.com/op-rs/kona/compare/kona-mpt-v0.0.6...kona-mpt-v0.0.7) - 2024-11-20

### Added

- *(mpt)* Extend `TrieProvider` in `kona-executor` ([#813](https://github.com/op-rs/kona/pull/813))

### Fixed

- *(mpt)* Remove unused collapse ([#808](https://github.com/op-rs/kona/pull/808))

### Other

- v0.6.6 op-alloy ([#804](https://github.com/op-rs/kona/pull/804))
- *(workspace)* Migrate back to `thiserror` v2 ([#811](https://github.com/op-rs/kona/pull/811))
- Revert "chore: bump alloy deps ([#788](https://github.com/op-rs/kona/pull/788))" ([#791](https://github.com/op-rs/kona/pull/791))

## [0.0.6](https://github.com/op-rs/kona/compare/kona-mpt-v0.0.5...kona-mpt-v0.0.6) - 2024-11-06

### Added

- *(TrieProvider)* Abstract TrieNode retrieval ([#787](https://github.com/op-rs/kona/pull/787))

### Other

- bump alloy deps ([#788](https://github.com/op-rs/kona/pull/788))

## [0.0.5](https://github.com/op-rs/kona/compare/kona-mpt-v0.0.4...kona-mpt-v0.0.5) - 2024-10-29

### Fixed

- add feature for `alloy-provider`, fix `test_util` ([#738](https://github.com/op-rs/kona/pull/738))

## [0.0.4](https://github.com/op-rs/kona/compare/kona-mpt-v0.0.3...kona-mpt-v0.0.4) - 2024-10-25

### Added

- remove thiserror ([#735](https://github.com/op-rs/kona/pull/735))
- *(executor)* Clean ups ([#719](https://github.com/op-rs/kona/pull/719))
- use derive more display ([#675](https://github.com/op-rs/kona/pull/675))
- kona-providers ([#596](https://github.com/op-rs/kona/pull/596))
- *(ci)* Split online/offline tests ([#582](https://github.com/op-rs/kona/pull/582))
- *(mpt)* Migrate to `thiserror` ([#541](https://github.com/op-rs/kona/pull/541))

### Fixed

- *(mpt)* Empty root node case ([#705](https://github.com/op-rs/kona/pull/705))
- typos ([#690](https://github.com/op-rs/kona/pull/690))
- *(workspace)* hoist and fix lints ([#577](https://github.com/op-rs/kona/pull/577))
- *(mpt)* Empty list walker ([#493](https://github.com/op-rs/kona/pull/493))

### Other

- cleans up kona-mpt deps ([#725](https://github.com/op-rs/kona/pull/725))
- re-org imports ([#711](https://github.com/op-rs/kona/pull/711))
- *(mpt)* codecov ([#655](https://github.com/op-rs/kona/pull/655))
- *(mpt)* mpt noop trait impls ([#649](https://github.com/op-rs/kona/pull/649))
- *(mpt)* account conversion tests ([#647](https://github.com/op-rs/kona/pull/647))
- doc logos ([#609](https://github.com/op-rs/kona/pull/609))
- *(workspace)* Allow stdlib in `cfg(test)` ([#548](https://github.com/op-rs/kona/pull/548))

## [0.0.3](https://github.com/op-rs/kona/compare/kona-mpt-v0.0.2...kona-mpt-v0.0.3) - 2024-09-04

### Added
- *(mpt)* `TrieNode` benchmarks ([#351](https://github.com/op-rs/kona/pull/351))

### Fixed
- *(workspace)* Add Unused Dependency Lint ([#453](https://github.com/op-rs/kona/pull/453))
- *(deps)* Bump Alloy Dependencies ([#409](https://github.com/op-rs/kona/pull/409))

### Other
- *(workspace)* Alloy Version Bumps ([#467](https://github.com/op-rs/kona/pull/467))
- *(workspace)* Update for `op-rs` org transfer ([#474](https://github.com/op-rs/kona/pull/474))
- *(workspace)* Hoist Dependencies ([#466](https://github.com/op-rs/kona/pull/466))
- *(bin)* Remove `kt` ([#461](https://github.com/op-rs/kona/pull/461))
- *(deps)* Bump revm version to v13 ([#422](https://github.com/op-rs/kona/pull/422))

## [0.0.2](https://github.com/op-rs/kona/compare/kona-mpt-v0.0.1...kona-mpt-v0.0.2) - 2024-06-22

### Added
- *(client)* Derivation integration ([#257](https://github.com/op-rs/kona/pull/257))
- *(client)* Oracle-backed derive traits ([#252](https://github.com/op-rs/kona/pull/252))
- *(client)* Account + Account storage hinting in `TrieDB` ([#228](https://github.com/op-rs/kona/pull/228))
- *(client)* Add `current_output_root` to block executor ([#225](https://github.com/op-rs/kona/pull/225))
- *(ci)* Dependabot config ([#236](https://github.com/op-rs/kona/pull/236))
- *(client)* `StatelessL2BlockExecutor` ([#210](https://github.com/op-rs/kona/pull/210))
- *(mpt)* Block hash walkback ([#199](https://github.com/op-rs/kona/pull/199))
- *(mpt)* Simplify `TrieDB` ([#198](https://github.com/op-rs/kona/pull/198))
- *(mpt)* Trie DB commit ([#196](https://github.com/op-rs/kona/pull/196))
- *(mpt)* Trie node insertion ([#195](https://github.com/op-rs/kona/pull/195))
- *(host)* Host program scaffold ([#184](https://github.com/op-rs/kona/pull/184))
- *(workspace)* Client programs in workspace ([#178](https://github.com/op-rs/kona/pull/178))
- *(mpt)* `TrieCacheDB` scaffold ([#174](https://github.com/op-rs/kona/pull/174))
- *(mpt)* `TrieNode` retrieval ([#173](https://github.com/op-rs/kona/pull/173))
- *(mpt)* Refactor `TrieNode` ([#172](https://github.com/op-rs/kona/pull/172))

### Fixed
- *(mpt)* Fix extension node truncation ([#300](https://github.com/op-rs/kona/pull/300))
- *(ci)* Release plz ([#145](https://github.com/op-rs/kona/pull/145))

### Other
- version dependencies ([#296](https://github.com/op-rs/kona/pull/296))
- *(mpt)* Do not expose recursion vars ([#197](https://github.com/op-rs/kona/pull/197))
