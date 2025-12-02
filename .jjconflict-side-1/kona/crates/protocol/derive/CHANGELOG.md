# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.3](https://github.com/op-rs/kona/compare/kona-derive-v0.2.2...kona-derive-v0.2.3) - 2025-01-16

### Other

- Update Maili Deps (#908)
- fix some typos in comment (#906)

## [0.2.2](https://github.com/op-rs/kona/compare/kona-derive-v0.2.1...kona-derive-v0.2.2) - 2025-01-13

### Other

- Move temporary error logs to lower level than WARN ([#897](https://github.com/op-rs/kona/pull/897))
- *(deps)* Replace dep `alloy-rs/op-alloy-registry`->`op-rs/maili-registry` (#892)
- *(deps)* Replace dep `alloy-rs/op-alloy-protocol`->`op-rs/maili-protocol` (#890)
- *(derive)* Test Ignoring EIP-7702 (#887)

## [0.2.1](https://github.com/op-rs/kona/compare/kona-derive-v0.2.0...kona-derive-v0.2.1) - 2025-01-07

### Fixed

- op-rs rename (#883)
- *(derive)* `BatchStream` Past batch handling (#876)
- *(derive)* make tests compile (#878)

### Other

- Bump Dependencies (#880)
- patch for rust 1.81 ([#867](https://github.com/op-rs/kona/pull/867))

## [0.2.0](https://github.com/op-rs/kona/compare/kona-derive-v0.1.0...kona-derive-v0.2.0) - 2024-12-03

### Added

- *(workspace)* Isolate FPVM-specific platform code ([#821](https://github.com/op-rs/kona/pull/821))

### Fixed

- bump ([#855](https://github.com/op-rs/kona/pull/855))
- nightly lint ([#858](https://github.com/op-rs/kona/pull/858))

### Other

- *(derive)* remove indexed blob hash ([#847](https://github.com/op-rs/kona/pull/847))

## [0.0.7](https://github.com/op-rs/kona/compare/kona-derive-v0.0.6...kona-derive-v0.0.7) - 2024-11-20

### Added

- *(driver)* Abstract, Default Pipeline ([#796](https://github.com/op-rs/kona/pull/796))

### Other

- *(derive)* remove batch reader ([#826](https://github.com/op-rs/kona/pull/826))
- op-alloy 0.6.8 ([#830](https://github.com/op-rs/kona/pull/830))
- *(driver)* use tracing macros ([#823](https://github.com/op-rs/kona/pull/823))
- *(deps)* op-alloy 0.6.7 ([#824](https://github.com/op-rs/kona/pull/824))
- v0.6.6 op-alloy ([#804](https://github.com/op-rs/kona/pull/804))
- *(workspace)* Migrate back to `thiserror` v2 ([#811](https://github.com/op-rs/kona/pull/811))
- *(derive)* Re-export types ([#790](https://github.com/op-rs/kona/pull/790))
- Revert "chore: bump alloy deps ([#788](https://github.com/op-rs/kona/pull/788))" ([#791](https://github.com/op-rs/kona/pull/791))

## [0.0.6](https://github.com/op-rs/kona/compare/kona-derive-v0.0.5...kona-derive-v0.0.6) - 2024-11-06

### Added

- *(derive)* `From<BlobProviderError> for PipelineErrorKind` ([#780](https://github.com/op-rs/kona/pull/780))
- *(client)* Remove `anyhow` ([#779](https://github.com/op-rs/kona/pull/779))
- *(derive)* sources docs ([#754](https://github.com/op-rs/kona/pull/754))

### Fixed

- *(derive)* Data Availability Provider Abstraction ([#782](https://github.com/op-rs/kona/pull/782))
- *(derive)* hoist types out of traits ([#781](https://github.com/op-rs/kona/pull/781))
- *(client)* Trace extension support ([#778](https://github.com/op-rs/kona/pull/778))
- *(derive)* use signal value updated with system config. ([#776](https://github.com/op-rs/kona/pull/776))

### Other

- bump alloy deps ([#788](https://github.com/op-rs/kona/pull/788))
- *(derive)* pipeline error test coverage ([#784](https://github.com/op-rs/kona/pull/784))
- Only fill blob data when there is no calldata ([#764](https://github.com/op-rs/kona/pull/764))
- *(derive)* touchup kona-derive readme ([#762](https://github.com/op-rs/kona/pull/762))
- *(derive)* Error Exports ([#758](https://github.com/op-rs/kona/pull/758))
- *(derive)* Cleanup Exports ([#757](https://github.com/op-rs/kona/pull/757))

## [0.0.5](https://github.com/op-rs/kona/compare/kona-derive-v0.0.4...kona-derive-v0.0.5) - 2024-10-29

### Added

- *(derive)* use upstream op-alloy batch types ([#746](https://github.com/op-rs/kona/pull/746))
- *(derive)* Remove metrics ([#743](https://github.com/op-rs/kona/pull/743))
- *(derive)* sys config accessor ([#722](https://github.com/op-rs/kona/pull/722))

### Fixed

- tracing_subscriber problem in `kona-derive` tests ([#741](https://github.com/op-rs/kona/pull/741))
- *(derive)* Holocene action tests / fixes ([#733](https://github.com/op-rs/kona/pull/733))

### Other

- *(derive)* import hygiene ([#744](https://github.com/op-rs/kona/pull/744))

## [0.0.4](https://github.com/op-rs/kona/compare/kona-derive-v0.0.3...kona-derive-v0.0.4) - 2024-10-25

### Added

- remove thiserror ([#735](https://github.com/op-rs/kona/pull/735))
- *(derive)* `BatchProvider` multiplexed stage ([#726](https://github.com/op-rs/kona/pull/726))
- *(derive)* hoist stage traits ([#723](https://github.com/op-rs/kona/pull/723))
- frame queue test asserter ([#619](https://github.com/op-rs/kona/pull/619))
- *(workspace)* Distribute pipeline, not providers ([#717](https://github.com/op-rs/kona/pull/717))
- *(derive)* `BatchValidator` stage ([#703](https://github.com/op-rs/kona/pull/703))
- *(derive)* Add `ChannelAssembler` size limitation ([#700](https://github.com/op-rs/kona/pull/700))
- *(derive)* signal receiver logic ([#696](https://github.com/op-rs/kona/pull/696))
- *(derive)* Stage multiplexer ([#693](https://github.com/op-rs/kona/pull/693))
- *(derive)* `Past` batch validity variant ([#684](https://github.com/op-rs/kona/pull/684))
- codecov sources ([#657](https://github.com/op-rs/kona/pull/657))
- frame queue tests ([#613](https://github.com/op-rs/kona/pull/613))
- *(derive)* Holocene flush signal ([#612](https://github.com/op-rs/kona/pull/612))
- *(derive)* Add `Signal` API ([#611](https://github.com/op-rs/kona/pull/611))
- *(derive)* BatchQueue Update [Holocene] ([#601](https://github.com/op-rs/kona/pull/601))
- *(derive)* bump op-alloy dep ([#605](https://github.com/op-rs/kona/pull/605))
- kona-providers ([#596](https://github.com/op-rs/kona/pull/596))
- *(derive)* Span batch prefix checks ([#592](https://github.com/op-rs/kona/pull/592))
- *(derive)* `BatchStream` buffering ([#590](https://github.com/op-rs/kona/pull/590))
- *(derive)* BatchStreamProvider ([#591](https://github.com/op-rs/kona/pull/591))
- *(derive)* Refactor out Online Providers ([#569](https://github.com/op-rs/kona/pull/569))
- *(derive)* interleaved channel tests ([#585](https://github.com/op-rs/kona/pull/585))
- *(derive)* Holocene Buffer Flushing ([#575](https://github.com/op-rs/kona/pull/575))
- *(derive)* Holocene Channel Bank Checks ([#572](https://github.com/op-rs/kona/pull/572))
- *(derive)* Holocene Frame Queue ([#579](https://github.com/op-rs/kona/pull/579))
- *(derive)* Holocene Activation ([#574](https://github.com/op-rs/kona/pull/574))
- *(derive)* wire up the batch span stage ([#567](https://github.com/op-rs/kona/pull/567))
- *(derive)* New BatchStream Stage for Holocene ([#566](https://github.com/op-rs/kona/pull/566))
- *(derive)* Hoist AttributesBuilder ([#571](https://github.com/op-rs/kona/pull/571))
- *(derive)* Touchup Docs ([#555](https://github.com/op-rs/kona/pull/555))
- *(derive)* Latest BN ([#521](https://github.com/op-rs/kona/pull/521))
- Remove L2 Execution Payload ([#542](https://github.com/op-rs/kona/pull/542))
- *(derive)* Typed error handling ([#540](https://github.com/op-rs/kona/pull/540))
- *(primitives)* Remove Attributes ([#529](https://github.com/op-rs/kona/pull/529))
- large dependency update ([#528](https://github.com/op-rs/kona/pull/528))
- *(primitives)* reuse op-alloy-protocol channel and block types ([#499](https://github.com/op-rs/kona/pull/499))

### Fixed

- hashmap ([#732](https://github.com/op-rs/kona/pull/732))
- *(derive)* SpanBatch element limit + channel RLP size limit ([#692](https://github.com/op-rs/kona/pull/692))
- *(derive)* Holocene `SpanBatch` prefix checks ([#688](https://github.com/op-rs/kona/pull/688))
- *(derive)* Retain L1 blocks ([#683](https://github.com/op-rs/kona/pull/683))
- *(executor)* Holocene EIP-1559 params in Header ([#622](https://github.com/op-rs/kona/pull/622))
- derive pipeline params ([#587](https://github.com/op-rs/kona/pull/587))
- *(workspace)* hoist and fix lints ([#577](https://github.com/op-rs/kona/pull/577))
- *(derive)* move attributes builder trait ([#570](https://github.com/op-rs/kona/pull/570))
- *(client)* Channel reader error handling ([#539](https://github.com/op-rs/kona/pull/539))
- *(derive)* Sequence window expiry ([#532](https://github.com/op-rs/kona/pull/532))
- *(primitives)* use consensus hardforks ([#497](https://github.com/op-rs/kona/pull/497))
- *(primitives)* re-use op-alloy frame type ([#492](https://github.com/op-rs/kona/pull/492))

### Other

- re-org imports ([#711](https://github.com/op-rs/kona/pull/711))
- hoist trait test utilities ([#708](https://github.com/op-rs/kona/pull/708))
- *(workspace)* Removes Primitives ([#638](https://github.com/op-rs/kona/pull/638))
- *(derive)* Add tracing to `ChannelAssembler` ([#701](https://github.com/op-rs/kona/pull/701))
- *(derive)* remove span batch todo comments ([#682](https://github.com/op-rs/kona/pull/682))
- refactor test utils ([#677](https://github.com/op-rs/kona/pull/677))
- *(derive)* stage coverage ([#673](https://github.com/op-rs/kona/pull/673))
- *(executor)* Use Upstreamed op-alloy Methods  ([#651](https://github.com/op-rs/kona/pull/651))
- *(derive)* Test and Clean Batch Types ([#670](https://github.com/op-rs/kona/pull/670))
- *(derive)* Test Stage Resets and Flushes ([#669](https://github.com/op-rs/kona/pull/669))
- *(derive)* Batch Timestamp Tests ([#664](https://github.com/op-rs/kona/pull/664))
- *(derive)* test channel reader flushing ([#661](https://github.com/op-rs/kona/pull/661))
- *(derive)* adds more channel bank coverage ([#659](https://github.com/op-rs/kona/pull/659))
- *(derive)* test channel reader resets ([#660](https://github.com/op-rs/kona/pull/660))
- *(derive)* test channel bank reset ([#658](https://github.com/op-rs/kona/pull/658))
- *(derive)* hoist attributes queue test utils ([#654](https://github.com/op-rs/kona/pull/654))
- *(derive)* Pipeline Core Test Coverage ([#642](https://github.com/op-rs/kona/pull/642))
- *(derive)* Single Batch Test Coverage ([#643](https://github.com/op-rs/kona/pull/643))
- *(derive)* Blob Source Test Coverage ([#631](https://github.com/op-rs/kona/pull/631))
- refactor test providers ([#623](https://github.com/op-rs/kona/pull/623))
- doc logos ([#609](https://github.com/op-rs/kona/pull/609))
- use alloy primitives map ([#586](https://github.com/op-rs/kona/pull/586))
- *(derive)* [Holocene] Drain previous channel in one iteration ([#583](https://github.com/op-rs/kona/pull/583))
- channel reader docs ([#568](https://github.com/op-rs/kona/pull/568))
- Bumps Dependency Versions ([#520](https://github.com/op-rs/kona/pull/520))
- *(primitives)* rm RawTransaction ([#505](https://github.com/op-rs/kona/pull/505))

## [0.0.3](https://github.com/op-rs/kona/compare/kona-derive-v0.0.2...kona-derive-v0.0.3) - 2024-09-04

### Added
- Run cargo hack against workspace ([#485](https://github.com/op-rs/kona/pull/485))
- *(workspace)* Workspace Re-exports ([#468](https://github.com/op-rs/kona/pull/468))
- *(ci)* Add scheduled FPP differential tests ([#408](https://github.com/op-rs/kona/pull/408))
- *(derive+trusted-sync)* online blob provider with fallback ([#410](https://github.com/op-rs/kona/pull/410))
- increase granularity ([#365](https://github.com/op-rs/kona/pull/365))
- *(derive)* histogram for number of channels for given frame counts ([#337](https://github.com/op-rs/kona/pull/337))
- *(derive)* track the current channel size ([#331](https://github.com/op-rs/kona/pull/331))
- *(derive)* more stage metrics ([#326](https://github.com/op-rs/kona/pull/326))
- *(derive)* Granular Provider Metrics ([#325](https://github.com/op-rs/kona/pull/325))
- *(derive)* Stage Level Metrics ([#309](https://github.com/op-rs/kona/pull/309))
- *(examples)* Trusted Sync Metrics ([#308](https://github.com/op-rs/kona/pull/308))

### Fixed
- downgrade for release plz ([#458](https://github.com/op-rs/kona/pull/458))
- *(workspace)* Add Unused Dependency Lint ([#453](https://github.com/op-rs/kona/pull/453))
- *(derive)* remove fpvm tests ([#447](https://github.com/op-rs/kona/pull/447))
- *(derive)* Granite Hardfork Support ([#420](https://github.com/op-rs/kona/pull/420))
- remove data iter option ([#405](https://github.com/op-rs/kona/pull/405))
- *(deps)* Bump Alloy Dependencies ([#409](https://github.com/op-rs/kona/pull/409))
- *(kona-derive)* Remove SignedRecoverable Shim ([#400](https://github.com/op-rs/kona/pull/400))
- *(derive)* Pipeline Reset ([#383](https://github.com/op-rs/kona/pull/383))
- *(examples)* Start N Blocks Back from Tip ([#349](https://github.com/op-rs/kona/pull/349))
- *(derive)* Unused var w/o `metrics` feature ([#345](https://github.com/op-rs/kona/pull/345))
- *(derive)* bind the Pipeline trait to Iterator ([#334](https://github.com/op-rs/kona/pull/334))
- *(derive)* prefix all metric names ([#330](https://github.com/op-rs/kona/pull/330))
- *(examples)* don't panic on validation fetch failure ([#327](https://github.com/op-rs/kona/pull/327))
- *(derive)* Warnings with metrics macro ([#322](https://github.com/op-rs/kona/pull/322))

### Other
- *(workspace)* Alloy Version Bumps ([#467](https://github.com/op-rs/kona/pull/467))
- *(workspace)* Update for `op-rs` org transfer ([#474](https://github.com/op-rs/kona/pull/474))
- *(workspace)* Hoist Dependencies ([#466](https://github.com/op-rs/kona/pull/466))
- *(derive)* reset docs ([#464](https://github.com/op-rs/kona/pull/464))
- *(derive)* Remove udeps ([#462](https://github.com/op-rs/kona/pull/462))
- *(bin)* Remove `kt` ([#461](https://github.com/op-rs/kona/pull/461))
- refactor types out of kona-derive ([#454](https://github.com/op-rs/kona/pull/454))
- *(derive)* Channel timeout ([#437](https://github.com/op-rs/kona/pull/437))
- *(derive)* remove previous stage trait ([#423](https://github.com/op-rs/kona/pull/423))
- *(examples)* Add logs to trusted-sync ([#415](https://github.com/op-rs/kona/pull/415))
- *(derive)* refine channel frame count buckets ([#378](https://github.com/op-rs/kona/pull/378))
- *(derive)* Remove noisy batch logs ([#329](https://github.com/op-rs/kona/pull/329))
- clean up trusted sync loop ([#318](https://github.com/op-rs/kona/pull/318))
- *(docs)* Label Cleanup ([#307](https://github.com/op-rs/kona/pull/307))
- *(derive)* add targets to stage logs ([#310](https://github.com/op-rs/kona/pull/310))

## [0.0.2](https://github.com/op-rs/kona/compare/kona-derive-v0.0.1...kona-derive-v0.0.2) - 2024-06-22

### Added
- *(fjord)* fjord parameter changes ([#284](https://github.com/op-rs/kona/pull/284))
- *(client/host)* Oracle-backed Blob fetcher ([#255](https://github.com/op-rs/kona/pull/255))
- *(kona-derive)* Towards Derivation ([#243](https://github.com/op-rs/kona/pull/243))
- *(kona-derive)* Updated interface ([#230](https://github.com/op-rs/kona/pull/230))
- *(ci)* Dependabot config ([#236](https://github.com/op-rs/kona/pull/236))
- *(client)* `StatelessL2BlockExecutor` ([#210](https://github.com/op-rs/kona/pull/210))
- Pipeline Builder ([#217](https://github.com/op-rs/kona/pull/217))
- Minimal ResetProvider Implementation ([#208](https://github.com/op-rs/kona/pull/208))
- refactor the pipeline builder ([#209](https://github.com/op-rs/kona/pull/209))
- refactor reset provider ([#207](https://github.com/op-rs/kona/pull/207))
- *(preimage)* Async server components ([#183](https://github.com/op-rs/kona/pull/183))
- *(workspace)* Client programs in workspace ([#178](https://github.com/op-rs/kona/pull/178))
- *(primitives)* move attributes into primitives ([#163](https://github.com/op-rs/kona/pull/163))
- *(derive)* return the concrete online attributes queue type from the online stack constructor ([#158](https://github.com/op-rs/kona/pull/158))
- *(derive)* Abstract Alt DA out of `kona-derive` ([#156](https://github.com/op-rs/kona/pull/156))
- *(derive)* Online Data Source Factory Wiring ([#150](https://github.com/op-rs/kona/pull/150))
- *(plasma)* Implements Plasma Support for kona derive ([#152](https://github.com/op-rs/kona/pull/152))
- *(derive)* Pipeline Builder ([#127](https://github.com/op-rs/kona/pull/127))
- *(primitives)* kona-derive type refactor ([#135](https://github.com/op-rs/kona/pull/135))
- *(derive)* Span Batch Validation ([#121](https://github.com/op-rs/kona/pull/121))
- *(derive)* Use `L2ChainProvider` for system config fetching in attributes builder ([#123](https://github.com/op-rs/kona/pull/123))
- *(derive)* Online Blob Provider ([#117](https://github.com/op-rs/kona/pull/117))
- *(derive)* payload builder tests ([#106](https://github.com/op-rs/kona/pull/106))
- *(derive)* deposit derivation testing ([#115](https://github.com/op-rs/kona/pull/115))
- *(derive)* Build `L1BlockInfoTx` in payload builder ([#102](https://github.com/op-rs/kona/pull/102))
- *(derive)* `L2ChainProvider` w/ `op-alloy-consensus` ([#98](https://github.com/op-rs/kona/pull/98))
- *(derive)* Add `L1BlockInfoTx` ([#100](https://github.com/op-rs/kona/pull/100))
- *(derive)* Payload Attribute Building ([#92](https://github.com/op-rs/kona/pull/92))
- *(derive)* Online `ChainProvider` ([#93](https://github.com/op-rs/kona/pull/93))
- *(derive)* Move to `tracing` for telemetry ([#94](https://github.com/op-rs/kona/pull/94))
- *(derive)* Batch Queue Logging ([#86](https://github.com/op-rs/kona/pull/86))
- *(derive)* Add `ecrecover` trait + features ([#90](https://github.com/op-rs/kona/pull/90))
- *(derive)* Use upstream alloy ([#89](https://github.com/op-rs/kona/pull/89))
- *(derive)* add next_attributes test
- *(workspace)* Add `rustfmt.toml`
- *(derive)* `SpanBatch` type implementation WIP
- *(derive)* Reorganize modules
- *(derive)* `add_txs` function
- *(derive)* Derive raw batches, mocks
- *(derive)* Refactor serialization; `SpanBatchPayload` WIP
- *(derive)* fixed bytes and encoding
- *(derive)* raw span type refactoring
- *(types)* span batches
- *(derive)* Channel Reader Implementation ([#65](https://github.com/op-rs/kona/pull/65))
- *(derive)* share the rollup config across stages using an arc
- *(derive)* Test Utilities ([#62](https://github.com/op-rs/kona/pull/62))
- Single batch type ([#43](https://github.com/op-rs/kona/pull/43))
- *(derive)* channel bank ([#46](https://github.com/op-rs/kona/pull/46))
- Frame queue stage ([#45](https://github.com/op-rs/kona/pull/45))
- L1 retrieval ([#44](https://github.com/op-rs/kona/pull/44))
- System config update event parsing ([#42](https://github.com/op-rs/kona/pull/42))
- Add OP receipt fields ([#41](https://github.com/op-rs/kona/pull/41))
- Add `TxDeposit` type ([#40](https://github.com/op-rs/kona/pull/40))
- L1 traversal ([#39](https://github.com/op-rs/kona/pull/39))

### Fixed
- *(derive)* Fjord brotli decompression ([#298](https://github.com/op-rs/kona/pull/298))
- *(examples)* Dynamic Rollup Config Loading ([#293](https://github.com/op-rs/kona/pull/293))
- type re-exports ([#280](https://github.com/op-rs/kona/pull/280))
- *(kona-derive)* reuse upstream reqwest provider ([#229](https://github.com/op-rs/kona/pull/229))
- Derivation Pipeline ([#220](https://github.com/op-rs/kona/pull/220))
- *(derive)* Alloy EIP4844 Blob Type ([#215](https://github.com/op-rs/kona/pull/215))
- Strong Error Typing ([#187](https://github.com/op-rs/kona/pull/187))
- *(derive)* inline blob verification into the blob provider ([#175](https://github.com/op-rs/kona/pull/175))
- *(derive)* fix span batch utils read_tx_data() ([#170](https://github.com/op-rs/kona/pull/170))
- *(derive)* Ethereum Data Source ([#159](https://github.com/op-rs/kona/pull/159))
- *(derive)* remove unnecessary online feature decorator ([#160](https://github.com/op-rs/kona/pull/160))
- *(ci)* Release plz ([#145](https://github.com/op-rs/kona/pull/145))
- *(derive)* move span batch conversion to try from trait ([#142](https://github.com/op-rs/kona/pull/142))
- *(derive)* Small Fixes and Span Batch Validation Fix ([#139](https://github.com/op-rs/kona/pull/139))
- *(workspace)* Release plz ([#138](https://github.com/op-rs/kona/pull/138))
- *(workspace)* Release plz ([#137](https://github.com/op-rs/kona/pull/137))
- *(derive)* Rebase span batch validation tests ([#125](https://github.com/op-rs/kona/pull/125))
- *(derive)* Span batch bitlist encoding ([#122](https://github.com/op-rs/kona/pull/122))
- *(derive)* Doc Touchups and Telemetry ([#105](https://github.com/op-rs/kona/pull/105))
- *(derive)* Derive full `SpanBatch` in channel reader ([#97](https://github.com/op-rs/kona/pull/97))
- *(derive)* Stage Decoupling ([#88](https://github.com/op-rs/kona/pull/88))
- *(derive)* add back removed test
- *(derive)* lints
- *(derive)* extend attributes queue unit test
- *(derive)* successful payload attributes building tests
- *(derive)* error equality fixes and tests
- *(derive)* rework abstractions and attributes queue testing
- *(derive)* attributes queue
- *(derive)* hoist params
- *(derive)* merge upstream changes
- *(derive)* fix bricked arc stage param construction ([#84](https://github.com/op-rs/kona/pull/84))
- *(derive)* l1 retrieval docs ([#80](https://github.com/op-rs/kona/pull/80))
- *(derive)* clean up frame queue docs
- *(derive)* frame queue error bubbling and docs
- *(derive)* rebase
- *(derive)* merge upstream changes
- *(derive)* refactor tx enveloped
- *(derive)* refactor span batch tx types
- *(derive)* bitlist alignment
- *(derive)* span batch tx rlp
- *(derive)* span type encodings and decodings
- *(derive)* more types
- *(derive)* small l1 retrieval doc comment fix ([#61](https://github.com/op-rs/kona/pull/61))

### Other
- version dependencies ([#296](https://github.com/op-rs/kona/pull/296))
- payload decoding tests ([#287](https://github.com/op-rs/kona/pull/287))
- payload decoding tests ([#289](https://github.com/op-rs/kona/pull/289))
- re-export input types ([#279](https://github.com/op-rs/kona/pull/279))
- *(deps)* fast forward op alloy dep ([#267](https://github.com/op-rs/kona/pull/267))
- *(derive)* cleanup pipeline tracing ([#264](https://github.com/op-rs/kona/pull/264))
- *(derive)* online module touchups ([#265](https://github.com/op-rs/kona/pull/265))
- *(derive)* Sources Touchups ([#266](https://github.com/op-rs/kona/pull/266))
- *(kona-derive)* Online Pipeline Cleanup ([#241](https://github.com/op-rs/kona/pull/241))
- *(derive)* data source unit tests ([#181](https://github.com/op-rs/kona/pull/181))
- *(workspace)* Move `alloy-primitives` to workspace dependencies ([#103](https://github.com/op-rs/kona/pull/103))
- *(ci)* Fail CI on doclint failure ([#101](https://github.com/op-rs/kona/pull/101))
- *(derive)* cleanups ([#91](https://github.com/op-rs/kona/pull/91))
- Merge branch 'main' into refcell/data-sources
- Merge pull request [#87](https://github.com/op-rs/kona/pull/87) from op-rs/refcell/origin-providers
- Merge branch 'main' into refcell/channel-bank-tests
- Merge branch 'main' into refcell/payload-queue
- *(derive)* L1Traversal Doc and Test Cleanup ([#79](https://github.com/op-rs/kona/pull/79))
- Merge pull request [#67](https://github.com/op-rs/kona/pull/67) from op-rs/refcell/batch-queue
- *(derive)* Channel reader tests + fixes, batch type fixes
- *(derive)* `RawSpanBatch` diff decoding/encoding test
- *(derive)* rebase + move `alloy` module
- *(derive)* Clean up RLP encoding + use `TxType` rather than ints
- Update `derive` lint rules ([#47](https://github.com/op-rs/kona/pull/47))
- scaffold ([#37](https://github.com/op-rs/kona/pull/37))
- Make versions of packages independent ([#36](https://github.com/op-rs/kona/pull/36))
