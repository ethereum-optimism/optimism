# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.23.1](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.23.1) - 2025-12-10

### Features

- Add `OpReceipt::map_logs` ([#615](https://github.com/alloy-rs/op-alloy/issues/615))

## [0.23.0](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.23.0) - 2025-12-08

### Bug Fixes

- Correct `OpReceipt` rlp ([#612](https://github.com/alloy-rs/op-alloy/issues/612))

### Features

- [consensus] Add L2ToL1MessagePasser predeploy address ([#614](https://github.com/alloy-rs/op-alloy/issues/614))
- Add helpers for recovering txs from flashblocks ([#611](https://github.com/alloy-rs/op-alloy/issues/611))
- Use `OpReceipt` as `ReceiptEnvelope` for OP network ([#610](https://github.com/alloy-rs/op-alloy/issues/610))

### Miscellaneous Tasks

- Release 0.23.0
- [rpc-types-engine] Upstream `payloadID` computation to `op-alloy` ([#613](https://github.com/alloy-rs/op-alloy/issues/613))

## [0.22.3](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.22.3) - 2025-11-19

### Miscellaneous Tasks

- Release 0.22.3
- [op-alloy] Update blob gas used serialization method ([#609](https://github.com/alloy-rs/op-alloy/issues/609))

## [0.22.2](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.22.2) - 2025-11-18

### Features

- Add bincode compat to OpReceipt ([#608](https://github.com/alloy-rs/op-alloy/issues/608))

### Miscellaneous Tasks

- Release 0.22.2

## [0.22.1](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.22.1) - 2025-11-11

### Dependencies

- Bump msrv ([#607](https://github.com/alloy-rs/op-alloy/issues/607))

### Features

- [flashblock] Commit shared flashblock data structures ([#606](https://github.com/alloy-rs/op-alloy/issues/606))

### Miscellaneous Tasks

- Release 0.22.1

## [0.22.0](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.22.0) - 2025-10-29

### Features

- [rpc-jsonrpsee] Add setGasLimit RPC to MinerApiExt ([#605](https://github.com/alloy-rs/op-alloy/issues/605))
- Generate `OpTypedTransaction` via macro ([#600](https://github.com/alloy-rs/op-alloy/issues/600))

### Miscellaneous Tasks

- Release 0.22.0

## [0.21.0](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.21.0) - 2025-10-14

### Bug Fixes

- Serde default for attribute fields ([#603](https://github.com/alloy-rs/op-alloy/issues/603))

### Features

- [jovian] Update da footprint types ([#599](https://github.com/alloy-rs/op-alloy/issues/599))
- [rpc-types-engine] `HeaderInfo` helper for OpExecutionPayload ([#602](https://github.com/alloy-rs/op-alloy/issues/602))
- Add missing signature helper fn ([#598](https://github.com/alloy-rs/op-alloy/issues/598))
- Add transaction helper methods to OpExecutionPayload ([#597](https://github.com/alloy-rs/op-alloy/issues/597))

### Miscellaneous Tasks

- Release 0.21.0
- Remove doc_auto_cfg from docsrs feature ([#604](https://github.com/alloy-rs/op-alloy/issues/604))

## [0.20.0](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.20.0) - 2025-09-12

### Bug Fixes

- [consensus/holocene] Fix the extra data length for holocene ([#596](https://github.com/alloy-rs/op-alloy/issues/596))

### Miscellaneous Tasks

- Release 0.20.0
- Missing typed conversion ([#595](https://github.com/alloy-rs/op-alloy/issues/595))
- Add into_envelope helper ([#594](https://github.com/alloy-rs/op-alloy/issues/594))
- `missing-const-for-fn` lint back to warn. ([#593](https://github.com/alloy-rs/op-alloy/issues/593))

### Other

- [jovian] extend `extraData` to include minBaseFee in EIP1559Params ([#580](https://github.com/alloy-rs/op-alloy/issues/580))

## [0.19.1](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.19.1) - 2025-08-26

### Dependencies

- Bump alloy 1.0.26 ([#591](https://github.com/alloy-rs/op-alloy/issues/591))

### Miscellaneous Tasks

- Release 0.19.1
- Ignore changelog in typos ([#592](https://github.com/alloy-rs/op-alloy/issues/592))

## [0.19.0](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.19.0) - 2025-08-25

### Dependencies

- Bump jsonrpsee dependencies to version 0.26 ([#589](https://github.com/alloy-rs/op-alloy/issues/589))

### Features

- [rpc] Add superchain reorg error variants ([#590](https://github.com/alloy-rs/op-alloy/issues/590))

### Miscellaneous Tasks

- Release 0.19.0
- Fix typo in comment ([#583](https://github.com/alloy-rs/op-alloy/issues/583))
- Fix grammatical typos ([#576](https://github.com/alloy-rs/op-alloy/issues/576))

### Other

- Add spellcheck workflow with typos ([#585](https://github.com/alloy-rs/op-alloy/issues/585))
- Migrate workflows to checkout v5 ([#586](https://github.com/alloy-rs/op-alloy/issues/586))

## [0.18.14](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.18.14) - 2025-07-31

### Dependencies

- Bump msrv 1.86 ([#579](https://github.com/alloy-rs/op-alloy/issues/579))

### Features

- Add `jovian_time` to `OpGenesisInfo` ([#581](https://github.com/alloy-rs/op-alloy/issues/581))

### Miscellaneous Tasks

- Release 0.18.14

## [0.18.13](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.18.13) - 2025-07-22

### Bug Fixes

- [rpc-types-engine] `OpExecutionPayloadEnvelope` serialization fix ([#578](https://github.com/alloy-rs/op-alloy/issues/578))

### Miscellaneous Tasks

- Release 0.18.13

## [0.18.12](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.18.12) - 2025-07-21

### Bug Fixes

- Generalize `TransactionResponse` impl ([#577](https://github.com/alloy-rs/op-alloy/issues/577))

### Features

- Add deposit receipt conversion ([#575](https://github.com/alloy-rs/op-alloy/issues/575))

### Miscellaneous Tasks

- Release 0.18.12

## [0.18.11](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.18.11) - 2025-07-17

### Features

- Add into_block_raw methods to OpExecutionPayload types ([#570](https://github.com/alloy-rs/op-alloy/issues/570))
- Add map logs helpers ([#574](https://github.com/alloy-rs/op-alloy/issues/574))

### Miscellaneous Tasks

- Release 0.18.11

### Other

- [rpc-types-engine/ssz] Fix ssz encoding for blocks of v1 and v2 ([#573](https://github.com/alloy-rs/op-alloy/issues/573))

## [0.18.10](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.18.10) - 2025-07-17

### Bug Fixes

- [rpc-types-engine] Fix `OpExecutionPayloadEnvelope::payload_hash` ([#572](https://github.com/alloy-rs/op-alloy/issues/572))
- Simplify OpExecutionPayload numeric field deserialization ([#568](https://github.com/alloy-rs/op-alloy/issues/568))
- Fix typo in test function name ([#566](https://github.com/alloy-rs/op-alloy/issues/566))

### Features

- Add try_into_block_with methods to OpExecutionPayload ([#563](https://github.com/alloy-rs/op-alloy/issues/563))

### Miscellaneous Tasks

- Release 0.18.10
- Add transaction getters ([#571](https://github.com/alloy-rs/op-alloy/issues/571))
- Add receipt helpers ([#567](https://github.com/alloy-rs/op-alloy/issues/567))

### Other

- Update broken OpAttributesWithParent docs.rs link to crate root in engine.md ([#565](https://github.com/alloy-rs/op-alloy/issues/565))

## [0.18.9](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.18.9) - 2025-06-27

### Dependencies

- Bump alloy ([#562](https://github.com/alloy-rs/op-alloy/issues/562))

### Miscellaneous Tasks

- Release 0.18.9

## [0.18.8](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.18.8) - 2025-06-27

### Bug Fixes

- Fix links.md ([#559](https://github.com/alloy-rs/op-alloy/issues/559))

### Dependencies

- Upgrade `alloy` version `1.0.12` => `1.0.14` ([#560](https://github.com/alloy-rs/op-alloy/issues/560))

### Features

- Override recover_signer_unchecked_with_buf ([#561](https://github.com/alloy-rs/op-alloy/issues/561))

### Miscellaneous Tasks

- Release 0.18.8

## [0.18.7](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.18.7) - 2025-06-25

### Bug Fixes

- Use `TransactionRequest` in `Network` impl ([#525](https://github.com/alloy-rs/op-alloy/issues/525))
- Correct typos in comments ([#554](https://github.com/alloy-rs/op-alloy/issues/554))

### Documentation

- Fix typo in payload comment ([#555](https://github.com/alloy-rs/op-alloy/issues/555))

### Features

- [rpc] Add `OpTransactionRequest` and associate it with `TransactionRequest` of `Optimism` ([#557](https://github.com/alloy-rs/op-alloy/issues/557))
- Derive `TransactionEnvelope` for `OpPooledTransaction` ([#556](https://github.com/alloy-rs/op-alloy/issues/556))

### Miscellaneous Tasks

- Release 0.18.7
- Add funding.json

## [0.18.6](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.18.6) - 2025-06-18

### Dependencies

- Bump alloy ([#551](https://github.com/alloy-rs/op-alloy/issues/551))

### Miscellaneous Tasks

- Release 0.18.6
- Release 0.18.5

## [0.18.4](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.18.4) - 2025-06-17

### Bug Fixes

- Move `Transaction::from_transaction` ([#552](https://github.com/alloy-rs/op-alloy/issues/552))

### Miscellaneous Tasks

- Release 0.18.4

## [0.18.3](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.18.3) - 2025-06-16

### Features

- [rpc] Convert into RPC transaction from generic `OpTransaction` ([#550](https://github.com/alloy-rs/op-alloy/issues/550))

### Miscellaneous Tasks

- Release 0.18.3

## [0.18.2](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.18.2) - 2025-06-16

### Bug Fixes

- Redirect to op-alloy-provider crate ([#540](https://github.com/alloy-rs/op-alloy/issues/540))
- Fix link to starting.md ([#533](https://github.com/alloy-rs/op-alloy/issues/533))

### Documentation

- Grammar fixes and a cleanup of a broken documentation link ([#537](https://github.com/alloy-rs/op-alloy/issues/537))
- Docs (README.md): create tables for crateses ([#534](https://github.com/alloy-rs/op-alloy/issues/534))

### Features

- [rpc] Replace wrapped `OpTxEnvelope` with a generic in the `Transaction` RPC response ([#542](https://github.com/alloy-rs/op-alloy/issues/542))
- [consensus] Add `as_deposit` into `OpTransaction` trait ([#549](https://github.com/alloy-rs/op-alloy/issues/549))
- [consensus] Add `OpTransaction` trait ([#548](https://github.com/alloy-rs/op-alloy/issues/548))
- [rpc-types-engine] SSZ Encoding ([#544](https://github.com/alloy-rs/op-alloy/issues/544))
- [rpc-types-engine] OpExecutionPayload Wrapper ([#539](https://github.com/alloy-rs/op-alloy/issues/539))
- [interop] Rename `InvalidInboxEntry`->`SuperchainDAError` ([#535](https://github.com/alloy-rs/op-alloy/issues/535))

### Miscellaneous Tasks

- Release 0.18.2
- [consensus] Rename `SafetyLevel` variants `Unsafe`->`LocalUnsafe` and `Safe`->`CrossSafe` ([#545](https://github.com/alloy-rs/op-alloy/issues/545))
- Add smol conversion helper ([#536](https://github.com/alloy-rs/op-alloy/issues/536))
- Add receipt helpers ([#531](https://github.com/alloy-rs/op-alloy/issues/531))

### Other

- 0.18.1 ([#547](https://github.com/alloy-rs/op-alloy/issues/547))
- 0.18.0 ([#541](https://github.com/alloy-rs/op-alloy/issues/541))
- Fix link to Enum OpTxType ([#538](https://github.com/alloy-rs/op-alloy/issues/538))
- Change `InvalidInboxEntry` repr `i64`->`i32` for `jsonrpsee` error compat ([#532](https://github.com/alloy-rs/op-alloy/issues/532))

## [0.17.2](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.17.2) - 2025-05-26

### Features

- Add Extended conversions for OpPooledTransaction, OpTxEnvelope ([#530](https://github.com/alloy-rs/op-alloy/issues/530))
- Added Transaction conversion from consensus for rpc  ([#529](https://github.com/alloy-rs/op-alloy/issues/529))
- Add meta helpers ([#527](https://github.com/alloy-rs/op-alloy/issues/527))

### Miscellaneous Tasks

- Release 0.17.2

## [0.17.1](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.17.1) - 2025-05-23

### Bug Fixes

- [rpc-types-engine] Rename op payload sidecar field ([#497](https://github.com/alloy-rs/op-alloy/issues/497))

### Documentation

- Added a new badge to the license ([#494](https://github.com/alloy-rs/op-alloy/issues/494))

### Features

- Made TxDeposit's mint field non-optional ([#514](https://github.com/alloy-rs/op-alloy/issues/514))
- [`network`] Impl RecommendedFillers for Optimism ([#524](https://github.com/alloy-rs/op-alloy/issues/524))

### Miscellaneous Tasks

- Release 0.17.1
- Release 0.17.0
- Relax conversion ([#523](https://github.com/alloy-rs/op-alloy/issues/523))

## [0.16.0](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.16.0) - 2025-05-13

### Bug Fixes

- [rpc-types-engine] Error Implementation ([#519](https://github.com/alloy-rs/op-alloy/issues/519))

### Dependencies

- Bump alloy 1.0.0 ([#521](https://github.com/alloy-rs/op-alloy/issues/521))

### Miscellaneous Tasks

- Release 0.16.0
- Rm native recover fn ([#522](https://github.com/alloy-rs/op-alloy/issues/522))
- [super-consensus] Migrate `InvalidInboxEntry` from reth ([#518](https://github.com/alloy-rs/op-alloy/issues/518))

### Other

- 0.15.7 ([#520](https://github.com/alloy-rs/op-alloy/issues/520))

## [0.15.6](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.15.6) - 2025-05-12

### Features

- Add SingerRecoverable impl ([#516](https://github.com/alloy-rs/op-alloy/issues/516))

### Miscellaneous Tasks

- Release 0.15.6

## [0.15.5](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.15.5) - 2025-05-09

### Bug Fixes

- Ensure all bytes consumed ([#515](https://github.com/alloy-rs/op-alloy/issues/515))

### Documentation

- Fix dead link to starting.md `starting.md` ([#511](https://github.com/alloy-rs/op-alloy/issues/511))

### Features

- [engine] Superchain Signal ([#512](https://github.com/alloy-rs/op-alloy/issues/512))

### Miscellaneous Tasks

- Release 0.15.5

## [0.15.4](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.15.4) - 2025-05-05

### Miscellaneous Tasks

- Release 0.15.4
- Add istyped support ([#510](https://github.com/alloy-rs/op-alloy/issues/510))

## [0.15.3](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.15.3) - 2025-05-05

### Miscellaneous Tasks

- Release 0.15.3

## [0.15.2](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.15.2) - 2025-05-02

### Bug Fixes

- [engine] Use OpExecutionPayloadV4 ([#509](https://github.com/alloy-rs/op-alloy/issues/509))
- Remove broken Examples links ([#507](https://github.com/alloy-rs/op-alloy/issues/507))

### Dependencies

- Rm unused deps ([#503](https://github.com/alloy-rs/op-alloy/issues/503))

### Features

- [consensus] Add interop block replacement deposit source ([#505](https://github.com/alloy-rs/op-alloy/issues/505))

### Miscellaneous Tasks

- Release 0.15.2
- Release 0.15.1 ([#506](https://github.com/alloy-rs/op-alloy/issues/506))
- Add new_unchecked ([#504](https://github.com/alloy-rs/op-alloy/issues/504))

### Other

- Update OpTxEnvelope documentation link path ([#508](https://github.com/alloy-rs/op-alloy/issues/508))

## [0.15.0](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.15.0) - 2025-04-23

### Dependencies

- Bump alloy 0.15 ([#502](https://github.com/alloy-rs/op-alloy/issues/502))

### Documentation

- Fix broken links in `building` section ([#501](https://github.com/alloy-rs/op-alloy/issues/501))

### Miscellaneous Tasks

- Release 0.15.0
- Add is_deposit to optxtype ([#500](https://github.com/alloy-rs/op-alloy/issues/500))

## [0.14.1](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.14.1) - 2025-04-16

### Features

- Add OpTxEnvelope::new_unhashed ([#499](https://github.com/alloy-rs/op-alloy/issues/499))
- Added recover helpers for OpPayloadAttributes ([#489](https://github.com/alloy-rs/op-alloy/issues/489))
- Added OpTxEnvelope::try_into_recovered ([#496](https://github.com/alloy-rs/op-alloy/issues/496))

### Miscellaneous Tasks

- Release 0.14.1
- Release 0.14.0 ([#498](https://github.com/alloy-rs/op-alloy/issues/498))

## [0.13.0](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.13.0) - 2025-04-09

### Dependencies

- Alloy 1.0 ([#493](https://github.com/alloy-rs/op-alloy/issues/493))
- [deps] Bincode 2.0 ([#491](https://github.com/alloy-rs/op-alloy/issues/491))

### Features

- Add input mut ([#478](https://github.com/alloy-rs/op-alloy/issues/478))
- Add missing conversions ([#492](https://github.com/alloy-rs/op-alloy/issues/492))

### Miscellaneous Tasks

- Release 0.13.0

## [0.12.2](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.12.2) - 2025-04-09

### Miscellaneous Tasks

- Release 0.12.2
- Rm borrow attr ([#490](https://github.com/alloy-rs/op-alloy/issues/490))

## [0.12.1](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.12.1) - 2025-04-09

### Bug Fixes

- [rpc-types-engine] Fix `PayloadHash` for V3 + V4 topic ([#482](https://github.com/alloy-rs/op-alloy/issues/482))

### Features

- [rpc-types-engine] SSZ Payload Encoding ([#485](https://github.com/alloy-rs/op-alloy/issues/485))
- Add hash ref function ([#483](https://github.com/alloy-rs/op-alloy/issues/483))

### Miscellaneous Tasks

- Release 0.12.1
- Clippy happy ([#487](https://github.com/alloy-rs/op-alloy/issues/487))
- Implement serde_bincode_compat for envelope ([#486](https://github.com/alloy-rs/op-alloy/issues/486))
- Remove op-alloy-flz crate and dep ([#484](https://github.com/alloy-rs/op-alloy/issues/484))

## [0.12.0](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.12.0) - 2025-03-28

### Dependencies

- Bump alloy 0.13 ([#481](https://github.com/alloy-rs/op-alloy/issues/481))

### Miscellaneous Tasks

- Release 0.12.0

## [0.11.3](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.11.3) - 2025-03-26

### Bug Fixes

- [l2-withdrawals-root] `OpExecutionPayloadEnvelopeV4` missing v4 payload ([#472](https://github.com/alloy-rs/op-alloy/issues/472))

### Dependencies

- Update Dependencies ([#474](https://github.com/alloy-rs/op-alloy/issues/474))
- Bump edition ([#458](https://github.com/alloy-rs/op-alloy/issues/458))
- Bump msrv to 1.85 ([#457](https://github.com/alloy-rs/op-alloy/issues/457))

### Features

- Move safety level ([#477](https://github.com/alloy-rs/op-alloy/issues/477))
- [op-alloy-rpc-types-engine] Add `superchain` mod ([#476](https://github.com/alloy-rs/op-alloy/issues/476))
- [rpc-types-engine] Support V4 network payload ([#471](https://github.com/alloy-rs/op-alloy/issues/471))
- Derive hash for envelope ([#470](https://github.com/alloy-rs/op-alloy/issues/470))

### Miscellaneous Tasks

- Release 0.11.3
- Remove deposit context source ([#475](https://github.com/alloy-rs/op-alloy/issues/475))

### Other

- 0.11.2 ([#473](https://github.com/alloy-rs/op-alloy/issues/473))

## [0.11.1](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.11.1) - 2025-03-12

### Miscellaneous Tasks

- Release 0.11.1
- Remove associated constant from RlpEcdsaEncodableTx ([#469](https://github.com/alloy-rs/op-alloy/issues/469))

## [0.11.0](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.11.0) - 2025-03-07

### Dependencies

- [deps] Alloy 0.12 ([#466](https://github.com/alloy-rs/op-alloy/issues/466))

### Miscellaneous Tasks

- Release 0.11.0
- [consensus] AsRef<OpTxEnvelope> ([#464](https://github.com/alloy-rs/op-alloy/issues/464))

### Other

- 0.10.9 ([#465](https://github.com/alloy-rs/op-alloy/issues/465))

## [0.10.8](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.10.8) - 2025-03-06

### Bug Fixes

- [engine] Empty requests hash ([#463](https://github.com/alloy-rs/op-alloy/issues/463))

### Features

- Add signabletx impl for typedtx ([#462](https://github.com/alloy-rs/op-alloy/issues/462))

### Miscellaneous Tasks

- Release 0.10.8

## [0.10.7](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.10.7) - 2025-02-28

### Dependencies

- Bump msrv to 1.82 ([#459](https://github.com/alloy-rs/op-alloy/issues/459))

### Miscellaneous Tasks

- Release 0.10.7

### Other

- Remove redundant method for v4 payload ([#461](https://github.com/alloy-rs/op-alloy/issues/461))
- Add conversions from block to payload ([#460](https://github.com/alloy-rs/op-alloy/issues/460))

## [0.10.6](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.10.6) - 2025-02-26

### Features

- [l2-withdrawals-root] Add conversions for `OpExecutionData` ([#455](https://github.com/alloy-rs/op-alloy/issues/455))
- Impl AnyRpcTransaction for OpTxEnvelope ([#454](https://github.com/alloy-rs/op-alloy/issues/454))
- Added helpers for opExecutionData ([#451](https://github.com/alloy-rs/op-alloy/issues/451))

### Miscellaneous Tasks

- Release 0.10.6
- Fix imports ([#452](https://github.com/alloy-rs/op-alloy/issues/452))
- [consensus] Remove Hardforks ([#448](https://github.com/alloy-rs/op-alloy/issues/448))

### Testing

- Fix flaky bincode compat rountrip test ([#453](https://github.com/alloy-rs/op-alloy/issues/453))

## [0.10.5](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.10.5) - 2025-02-19

### Features

- [l2-withdrawals] Add methods for prague payload fields to `OpExecutionPayloadSidecar` ([#445](https://github.com/alloy-rs/op-alloy/issues/445))

### Miscellaneous Tasks

- Release 0.10.5

### Other

- Add interop time to genesis ([#447](https://github.com/alloy-rs/op-alloy/issues/447))

## [0.10.4](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.10.4) - 2025-02-19

### Bug Fixes

- Std leakage ([#432](https://github.com/alloy-rs/op-alloy/issues/432))
- [consensus] Ecotone Upgrade Txs ([#412](https://github.com/alloy-rs/op-alloy/issues/412))
- [consensus] L1BlockInfo datas ([#408](https://github.com/alloy-rs/op-alloy/issues/408))

### Features

- [l2-withdrawals] Add `OpPayloadError` variants for blob transactions and l1 withdrawals ([#442](https://github.com/alloy-rs/op-alloy/issues/442))
- Add signed conversion ([#443](https://github.com/alloy-rs/op-alloy/issues/443))
- [l2-withdrawals] Impl conversion payload + sidecar into block  ([#441](https://github.com/alloy-rs/op-alloy/issues/441))
- Add bincode compat support for depositreceipts ([#440](https://github.com/alloy-rs/op-alloy/issues/440))
- Add fn for signature hash ([#438](https://github.com/alloy-rs/op-alloy/issues/438))
- Impl OpExecutionData ([#429](https://github.com/alloy-rs/op-alloy/issues/429))
- Add tryfrom envelope conversions ([#433](https://github.com/alloy-rs/op-alloy/issues/433))
- [l2-withdrawals] Impl conversion to block for `OpExecutionPayloadV4` ([#435](https://github.com/alloy-rs/op-alloy/issues/435))
- Add additional compat impls ([#427](https://github.com/alloy-rs/op-alloy/issues/427))
- Remove IsthmusPayloadFields ([#431](https://github.com/alloy-rs/op-alloy/issues/431))
- Add is-deposit helper ([#422](https://github.com/alloy-rs/op-alloy/issues/422))
- [isthmus] Define `OpExecutionData` ([#418](https://github.com/alloy-rs/op-alloy/issues/418))
- Add OpExecutionPayloadV4 ([#414](https://github.com/alloy-rs/op-alloy/issues/414))
- [isthmus] Define `IsthmusPayloadFields` ([#410](https://github.com/alloy-rs/op-alloy/issues/410))
- [consensus] Isthmus Network Upgrade Txs ([#405](https://github.com/alloy-rs/op-alloy/issues/405))

### Miscellaneous Tasks

- Release 0.10.4
- Additional envelope conversion ([#437](https://github.com/alloy-rs/op-alloy/issues/437))
- Make test compile ([#434](https://github.com/alloy-rs/op-alloy/issues/434))
- Rm bad non_exhaustive ([#404](https://github.com/alloy-rs/op-alloy/issues/404))

### Other

- Custom deserialize impl for OpExecutionPayload ([#436](https://github.com/alloy-rs/op-alloy/issues/436))
- 0.10.3 ([#426](https://github.com/alloy-rs/op-alloy/issues/426))
- Add operator fee to rpc l1 block ([#420](https://github.com/alloy-rs/op-alloy/issues/420))
- Define `OpExecutionPayload` ([#416](https://github.com/alloy-rs/op-alloy/issues/416))
- 0.10.2 ([#413](https://github.com/alloy-rs/op-alloy/issues/413))
- 0.10.1 ([#409](https://github.com/alloy-rs/op-alloy/issues/409))

## [0.10.0](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.10.0) - 2025-01-31

### Dependencies

- Bump alloy 0.11 ([#403](https://github.com/alloy-rs/op-alloy/issues/403))

### Features

- [ci] Add feature propagation checks ([#402](https://github.com/alloy-rs/op-alloy/issues/402))

### Miscellaneous Tasks

- Release 0.10.0
- Rm execution requests from v4 payload fn ([#401](https://github.com/alloy-rs/op-alloy/issues/401))

## [0.9.6](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.9.6) - 2025-01-22

### Bug Fixes

- Op-alloy-provider ([#390](https://github.com/alloy-rs/op-alloy/issues/390))

### Dependencies

- Revert maili deps ([#399](https://github.com/alloy-rs/op-alloy/issues/399))
- Bump `maili-*` to `0.1.6` ([#393](https://github.com/alloy-rs/op-alloy/issues/393))
- [deps] Bump `maili` to 0.1.5 ([#385](https://github.com/alloy-rs/op-alloy/issues/385))
- Quick Version Bumps ([#362](https://github.com/alloy-rs/op-alloy/issues/362))

### Features

- Add flz ([#400](https://github.com/alloy-rs/op-alloy/issues/400))
- [consensus] Add is_deposit to OpTxEnvelope ([#396](https://github.com/alloy-rs/op-alloy/issues/396))
- [interop] Define `ExecutingMessage` wrapper ([#361](https://github.com/alloy-rs/op-alloy/issues/361))
- [protocol] Compressors with Mocked Brotli Streaming ([#335](https://github.com/alloy-rs/op-alloy/issues/335))
- [protocol] Interop Types ([#352](https://github.com/alloy-rs/op-alloy/issues/352))

### Miscellaneous Tasks

- Release 0.9.6
- Add serde-bincode-compat for re-export features ([#397](https://github.com/alloy-rs/op-alloy/issues/397))
- [docs] Update readme ([#392](https://github.com/alloy-rs/op-alloy/issues/392))
- [consensus] Migrate deposit tx behaviour to `maili` ([#383](https://github.com/alloy-rs/op-alloy/issues/383))
- [genesis] Migrate `op-alloy-genesis`->`maili-genesis` ([#381](https://github.com/alloy-rs/op-alloy/issues/381))
- [provider] Revert [#365](https://github.com/alloy-rs/op-alloy/issues/365) remove `OpEngineApi` ([#379](https://github.com/alloy-rs/op-alloy/issues/379))
- [consensus] Migrate deposit source to `maili-common` ([#377](https://github.com/alloy-rs/op-alloy/issues/377))
- [rpc] Migrate rpc types to maili ([#378](https://github.com/alloy-rs/op-alloy/issues/378))
- [docs] Remove `op-alloy-protocol` from docs ([#380](https://github.com/alloy-rs/op-alloy/issues/380))
- [genesis] Add `interop_time` to `RollupConfig` + `HardForkConfiguration` ([#382](https://github.com/alloy-rs/op-alloy/issues/382))
- Remove rpc-jsonrpsee Crate ([#376](https://github.com/alloy-rs/op-alloy/issues/376))
- [protocol] Remove Protocol Crate ([#371](https://github.com/alloy-rs/op-alloy/issues/371))
- [registry] Remove the Registry Crate ([#372](https://github.com/alloy-rs/op-alloy/issues/372))
- [provider] Remove Provider Crate ([#373](https://github.com/alloy-rs/op-alloy/issues/373))
- [rpc-types-engine] Migrate `op_alloy_rpc_types_engine::superchain`->`op-rs/maili-rpc-types-engine` ([#367](https://github.com/alloy-rs/op-alloy/issues/367))
- [registry] Migrate `op-alloy-registry`->`op-rs/maili-registry` ([#366](https://github.com/alloy-rs/op-alloy/issues/366))
- [provider] Migrate `op-alloy-provider`->`op-rs/maili-provider` ([#365](https://github.com/alloy-rs/op-alloy/issues/365))
- [protocol] Migrate `op-alloy-protocol`->`op-rs/maili-protocol` ([#364](https://github.com/alloy-rs/op-alloy/issues/364))
- [ci] Check wasm compilation for `op-alloy-rpc-types` in CI ([#357](https://github.com/alloy-rs/op-alloy/issues/357))
- [ci] Update target `wasm32-wasi` to `wasm32-wasip1` for ci ([#354](https://github.com/alloy-rs/op-alloy/issues/354))
- [rpc] `no_std` support `op-alloy-rpc-jsonrpsee` ([#356](https://github.com/alloy-rs/op-alloy/issues/356))

### Other

- 0.9.5 ([#394](https://github.com/alloy-rs/op-alloy/issues/394))
- 0.9.4 ([#389](https://github.com/alloy-rs/op-alloy/issues/389))
- 0.9.3 ([#384](https://github.com/alloy-rs/op-alloy/issues/384))
- 0.9.2 ([#368](https://github.com/alloy-rs/op-alloy/issues/368))
- 0.9.1 ([#363](https://github.com/alloy-rs/op-alloy/issues/363))
- Define supervisor API ([#359](https://github.com/alloy-rs/op-alloy/issues/359))

## [0.9.0](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.9.0) - 2024-12-30

### Dependencies

- Bump alloy 0.9 ([#350](https://github.com/alloy-rs/op-alloy/issues/350))

### Miscellaneous Tasks

- Release 0.9.0
- Make clippy happy ([#349](https://github.com/alloy-rs/op-alloy/issues/349))

## [0.8.5](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.8.5) - 2024-12-19

### Features

- Impl From<TxEip7702> for OpTypedTransaction ([#348](https://github.com/alloy-rs/op-alloy/issues/348))

### Miscellaneous Tasks

- Release 0.8.5

### Other

- [Feature] Use Upstream Forkchoice Version ([#347](https://github.com/alloy-rs/op-alloy/issues/347))

## [0.8.4](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.8.4) - 2024-12-17

### Dependencies

- Bump alloy 081 ([#344](https://github.com/alloy-rs/op-alloy/issues/344))

### Miscellaneous Tasks

- Release 0.8.4

### Other

- [Bug] miner_setMaxDASize should return bool type ([#346](https://github.com/alloy-rs/op-alloy/issues/346))

## [0.8.3](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.8.3) - 2024-12-14

### Documentation

- Fix docs ([#343](https://github.com/alloy-rs/op-alloy/issues/343))

### Features

- Add OpPooledTransaction ([#341](https://github.com/alloy-rs/op-alloy/issues/341))

### Miscellaneous Tasks

- Release 0.8.3
- Reorder impl fns ([#342](https://github.com/alloy-rs/op-alloy/issues/342))

## [0.8.2](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.8.2) - 2024-12-12

### Features

- Upstream decode extradata fn ([#340](https://github.com/alloy-rs/op-alloy/issues/340))

### Miscellaneous Tasks

- Release 0.8.2

## [0.8.1](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.8.1) - 2024-12-12

### Features

- Add serde for OpTxType ([#317](https://github.com/alloy-rs/op-alloy/issues/317))

### Miscellaneous Tasks

- Release 0.8.1
- Reuse methods for receipt rlp ([#339](https://github.com/alloy-rs/op-alloy/issues/339))

## [0.8.0](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.8.0) - 2024-12-10

### Dependencies

- Bump alloy ([#338](https://github.com/alloy-rs/op-alloy/issues/338))
- [registry] Bump superchain-registry commit ([#336](https://github.com/alloy-rs/op-alloy/issues/336))
- Bump alloy to 0.7.3 ([#334](https://github.com/alloy-rs/op-alloy/issues/334))
- Enable alloy-primitives/arbitrary in dev-deps ([#329](https://github.com/alloy-rs/op-alloy/issues/329))

### Features

- [consensus] Tx envelope tx hash ([#324](https://github.com/alloy-rs/op-alloy/issues/324))
- Add miner extension trait ([#325](https://github.com/alloy-rs/op-alloy/issues/325))
- [engine] FCU Version ([#321](https://github.com/alloy-rs/op-alloy/issues/321))
- Add typed 2718 for txtype ([#323](https://github.com/alloy-rs/op-alloy/issues/323))

### Miscellaneous Tasks

- Release 0.8.0
- [registry] Update SCR ([#327](https://github.com/alloy-rs/op-alloy/issues/327))

### Other

- 0.7.3 ([#333](https://github.com/alloy-rs/op-alloy/issues/333))
- Add placeholder for isthmus time to genesis ([#331](https://github.com/alloy-rs/op-alloy/issues/331))
- Propagate arbitrary ([#330](https://github.com/alloy-rs/op-alloy/issues/330))

## [0.7.2](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.7.2) - 2024-12-02

### Features

- Bump alloy ([#322](https://github.com/alloy-rs/op-alloy/issues/322))

### Miscellaneous Tasks

- Release 0.7.2
- Release 0.7.2

## [0.7.1](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.7.1) - 2024-11-28

### Bug Fixes

- [op-alloy] Add Missing Registry Crate ([#311](https://github.com/alloy-rs/op-alloy/issues/311))
- [protocol] Remove panic in brotli compress method ([#296](https://github.com/alloy-rs/op-alloy/issues/296))
- [genesis] Base Fee Params ([#292](https://github.com/alloy-rs/op-alloy/issues/292))
- Protected bits handling ([#270](https://github.com/alloy-rs/op-alloy/issues/270))
- [book] Batch over SingleBatch ([#260](https://github.com/alloy-rs/op-alloy/issues/260))
- [book] Getting Start Links ([#256](https://github.com/alloy-rs/op-alloy/issues/256))
- [book] Broken Mdbook Version ([#250](https://github.com/alloy-rs/op-alloy/issues/250))

### Features

- Bump alloy ([#314](https://github.com/alloy-rs/op-alloy/issues/314))
- [protocol] Compressors ([#299](https://github.com/alloy-rs/op-alloy/issues/299))
- [book] Hardfork Change Example ([#306](https://github.com/alloy-rs/op-alloy/issues/306))
- Introduce op-alloy-registry ([#290](https://github.com/alloy-rs/op-alloy/issues/290))
- [genesis] Holocene Timestamps on Sepolia ([#285](https://github.com/alloy-rs/op-alloy/issues/285))
- Add missing txtype tryfroms ([#272](https://github.com/alloy-rs/op-alloy/issues/272))
- [protocol] Batch Reader ([#265](https://github.com/alloy-rs/op-alloy/issues/265))
- [protocol] ZLIB Compression ([#264](https://github.com/alloy-rs/op-alloy/issues/264))
- [protocol] Brotli Compression behind `std` ([#263](https://github.com/alloy-rs/op-alloy/issues/263))
- [protocol] Batch Encoding ([#259](https://github.com/alloy-rs/op-alloy/issues/259))
- Add missing OpTxType trait impls ([#258](https://github.com/alloy-rs/op-alloy/issues/258))
- [book] Frames ([#226](https://github.com/alloy-rs/op-alloy/issues/226))
- [book] Add Badges for Crates ([#253](https://github.com/alloy-rs/op-alloy/issues/253))

### Miscellaneous Tasks

- Release 0.7.1
- [workspace] Remove Deprecated Methods ([#313](https://github.com/alloy-rs/op-alloy/issues/313))
- Release 0.7.0
- [registry] Dogfood Test Rollup Config ([#308](https://github.com/alloy-rs/op-alloy/issues/308))
- [workspace] Remove Hand-rolled Display Error Impls ([#312](https://github.com/alloy-rs/op-alloy/issues/312))
- [workspace] Touchup crate docs with badges ([#309](https://github.com/alloy-rs/op-alloy/issues/309))
- [registry] Small Cleanup ([#307](https://github.com/alloy-rs/op-alloy/issues/307))
- [ci] Add missing no_std crates ([#310](https://github.com/alloy-rs/op-alloy/issues/310))
- [consensus] EIP-2718 Encoding Trait Impls ([#300](https://github.com/alloy-rs/op-alloy/issues/300))
- [protocol] Refactor Block Info Txs ([#303](https://github.com/alloy-rs/op-alloy/issues/303))
- [readme] Add op-alloy-registry ([#301](https://github.com/alloy-rs/op-alloy/issues/301))
- Issue Template Update ([#304](https://github.com/alloy-rs/op-alloy/issues/304))
- [protocol] Move and Extend Brotli Compression ([#298](https://github.com/alloy-rs/op-alloy/issues/298))
- [ci] Run examples in CI ([#297](https://github.com/alloy-rs/op-alloy/issues/297))
- Add default for txtype ([#295](https://github.com/alloy-rs/op-alloy/issues/295))
- [consensus] Trait Abstracted Hardforks ([#289](https://github.com/alloy-rs/op-alloy/issues/289))
- [genesis] Remove hardcoded configs ([#291](https://github.com/alloy-rs/op-alloy/issues/291))
- [consensus] Cleanup Hardforks ([#288](https://github.com/alloy-rs/op-alloy/issues/288))
- [consensus] Re-export and Hardfork Cleanup ([#274](https://github.com/alloy-rs/op-alloy/issues/274))
- [consensus] Signature Definitions ([#281](https://github.com/alloy-rs/op-alloy/issues/281))
- [consensus] OpTxType Conversion ([#283](https://github.com/alloy-rs/op-alloy/issues/283))
- [protocol] Batch Transaction Mod ([#284](https://github.com/alloy-rs/op-alloy/issues/284))
- [consensus] Move OpTxType and add tests ([#282](https://github.com/alloy-rs/op-alloy/issues/282))
- [protocol] Cleanup Examples ([#278](https://github.com/alloy-rs/op-alloy/issues/278))
- [op-alloy] Docs ([#277](https://github.com/alloy-rs/op-alloy/issues/277))
- [genesis] Remove Re-exports ([#276](https://github.com/alloy-rs/op-alloy/issues/276))
- Remove Error Impls ([#273](https://github.com/alloy-rs/op-alloy/issues/273))
- [workspace] Use thiserror for Error Types ([#269](https://github.com/alloy-rs/op-alloy/issues/269))
- [protocol] Remove TryFrom ([#268](https://github.com/alloy-rs/op-alloy/issues/268))
- [protocol] Re-organizes Modules and Errors ([#261](https://github.com/alloy-rs/op-alloy/issues/261))
- [book] Building Docs ([#257](https://github.com/alloy-rs/op-alloy/issues/257))
- [book] Frames to Batches Example ([#232](https://github.com/alloy-rs/op-alloy/issues/232))
- [book] Missing Sections and Enhancements ([#255](https://github.com/alloy-rs/op-alloy/issues/255))
- [book] Touchup Introduction ([#254](https://github.com/alloy-rs/op-alloy/issues/254))

### Other

- 0.6.8 ([#294](https://github.com/alloy-rs/op-alloy/issues/294))
- 0.6.7 ([#287](https://github.com/alloy-rs/op-alloy/issues/287))
- V0.6.6 ([#271](https://github.com/alloy-rs/op-alloy/issues/271))

## [0.6.5](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.6.5) - 2024-11-12

### Dependencies

- Bump alloy 064 ([#249](https://github.com/alloy-rs/op-alloy/issues/249))

### Features

- Wrap `TxDeposit` into `Sealed` in `OpTxEnvelope` ([#247](https://github.com/alloy-rs/op-alloy/issues/247))
- Add nonce to RPC transaction ([#246](https://github.com/alloy-rs/op-alloy/issues/246))

### Miscellaneous Tasks

- Release 0.6.5
- Add deserde test ([#248](https://github.com/alloy-rs/op-alloy/issues/248))

## [0.6.4](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.6.4) - 2024-11-12

### Bug Fixes

- [consensus] Add conversion for `OpTxType::Eip7702` ([#244](https://github.com/alloy-rs/op-alloy/issues/244))
- [consensus] Fix arbitrary impl for `OpTxType` ([#242](https://github.com/alloy-rs/op-alloy/issues/242))

### Miscellaneous Tasks

- Release 0.6.4
- Add is dynamic fee ([#245](https://github.com/alloy-rs/op-alloy/issues/245))

## [0.6.3](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.6.3) - 2024-11-08

### Dependencies

- Bump Alloy Deps ([#239](https://github.com/alloy-rs/op-alloy/issues/239))

### Features

- Bump alloy ([#240](https://github.com/alloy-rs/op-alloy/issues/240))

### Miscellaneous Tasks

- Release 0.6.3

## [0.6.2](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.6.2) - 2024-11-06

### Bug Fixes

- [protocol] Batch Decoding ([#235](https://github.com/alloy-rs/op-alloy/issues/235))
- [book] Links Imports ([#227](https://github.com/alloy-rs/op-alloy/issues/227))

### Features

- Add fn for decoded 1559 params ([#236](https://github.com/alloy-rs/op-alloy/issues/236))
- [book] Engine RPC Types ([#229](https://github.com/alloy-rs/op-alloy/issues/229))

### Miscellaneous Tasks

- Release 0.6.2
- Move eip1559 impls ([#237](https://github.com/alloy-rs/op-alloy/issues/237))
- [rpc-types] Clean up Exports ([#231](https://github.com/alloy-rs/op-alloy/issues/231))
- [book] Consolidate Links ([#230](https://github.com/alloy-rs/op-alloy/issues/230))
- [book] RPC Types ([#228](https://github.com/alloy-rs/op-alloy/issues/228))
- [book] Protocol Docs ([#225](https://github.com/alloy-rs/op-alloy/issues/225))

### Other

- V0.6.1 ([#238](https://github.com/alloy-rs/op-alloy/issues/238))

## [0.6.0](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.6.0) - 2024-11-06

### Bug Fixes

- [book] Small Book Touchups ([#220](https://github.com/alloy-rs/op-alloy/issues/220))
- [ci] Remove Docs gh-page publish ([#216](https://github.com/alloy-rs/op-alloy/issues/216))
- Ci powerset ([#214](https://github.com/alloy-rs/op-alloy/issues/214))
- [book] Missing READMEs ([#213](https://github.com/alloy-rs/op-alloy/issues/213))

### Dependencies

- [wip] feat: bump alloy ([#205](https://github.com/alloy-rs/op-alloy/issues/205))
- [workspace] Import Touchups ([#199](https://github.com/alloy-rs/op-alloy/issues/199))
- Bump alloy ([#178](https://github.com/alloy-rs/op-alloy/issues/178))

### Features

- Add holocene extradata fn ([#233](https://github.com/alloy-rs/op-alloy/issues/233))
- Add jsonrpsee trait for SuperchainSignal ([#217](https://github.com/alloy-rs/op-alloy/issues/217))
- `OpTransactionRequest` ([#215](https://github.com/alloy-rs/op-alloy/issues/215))
- [book] Consensus ([#212](https://github.com/alloy-rs/op-alloy/issues/212))
- [book] Genesis - System Config ([#211](https://github.com/alloy-rs/op-alloy/issues/211))
- `op-alloy` meta crate ([#210](https://github.com/alloy-rs/op-alloy/issues/210))
- [book] Genesis - Rollup Config ([#209](https://github.com/alloy-rs/op-alloy/issues/209))
- Book Setup ([#208](https://github.com/alloy-rs/op-alloy/issues/208))
- README ([#207](https://github.com/alloy-rs/op-alloy/issues/207))
- Book ([#206](https://github.com/alloy-rs/op-alloy/issues/206))
- [protocol] Batch ([#200](https://github.com/alloy-rs/op-alloy/issues/200))
- [protocol] Span Batch Validity Checks ([#198](https://github.com/alloy-rs/op-alloy/issues/198))
- [protocol] Span Batch Type ([#197](https://github.com/alloy-rs/op-alloy/issues/197))
- [protocol] Span Batch Transactions ([#196](https://github.com/alloy-rs/op-alloy/issues/196))
- [protocol] Batch TX Data ([#195](https://github.com/alloy-rs/op-alloy/issues/195))
- [protocol] Span Batch Bits ([#194](https://github.com/alloy-rs/op-alloy/issues/194))
- [protocol] Span Batch Element ([#193](https://github.com/alloy-rs/op-alloy/issues/193))
- [protocol] Batch Utilities ([#191](https://github.com/alloy-rs/op-alloy/issues/191))
- [protocol] Batch Error Types ([#190](https://github.com/alloy-rs/op-alloy/issues/190))
- [protocol] BatchValidationProvider ([#189](https://github.com/alloy-rs/op-alloy/issues/189))
- [protocol] SingleBatch Type ([#188](https://github.com/alloy-rs/op-alloy/issues/188))
- [protocol] Batch Validity ([#187](https://github.com/alloy-rs/op-alloy/issues/187))
- [protocol] Batch Type ([#186](https://github.com/alloy-rs/op-alloy/issues/186))
- [rpc-types] `{Try}From` impl for `OpTransactionReceipt` + `Transaction` -> consensus types ([#183](https://github.com/alloy-rs/op-alloy/issues/183))
- [genesis] EIP 1559 System Config Accessor ([#179](https://github.com/alloy-rs/op-alloy/issues/179))

### Miscellaneous Tasks

- Release 0.6.0
- [book] Load Rollup Config Example ([#224](https://github.com/alloy-rs/op-alloy/issues/224))
- [book] Genesis Docs ([#223](https://github.com/alloy-rs/op-alloy/issues/223))
- [book] Consensus Docs ([#222](https://github.com/alloy-rs/op-alloy/issues/222))
- [ci] Use Justfile Targets in Github Actions ([#219](https://github.com/alloy-rs/op-alloy/issues/219))
- [book] Fix Doc Links ([#218](https://github.com/alloy-rs/op-alloy/issues/218))
- Release 0.5.2 ([#201](https://github.com/alloy-rs/op-alloy/issues/201))
- [consensus] Upstream Receipt Constructor ([#165](https://github.com/alloy-rs/op-alloy/issues/165))
- Release 0.5.1 ([#184](https://github.com/alloy-rs/op-alloy/issues/184))
- [consensus] Small Cleanup ([#180](https://github.com/alloy-rs/op-alloy/issues/180))
- Dependency Updates ([#177](https://github.com/alloy-rs/op-alloy/issues/177))

### Other

- Add arbitrary attr ([#182](https://github.com/alloy-rs/op-alloy/issues/182))

## [0.5.0](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.5.0) - 2024-10-18

### Dependencies

- Bump alloy and remove `OpExecutionPayloadV4` ([#176](https://github.com/alloy-rs/op-alloy/issues/176))

### Features

- Add signature function to TxDeposit ([#174](https://github.com/alloy-rs/op-alloy/issues/174))
- Add depositTransaction trait ([#171](https://github.com/alloy-rs/op-alloy/issues/171))
- Op network execution payload envelope decoding ([#149](https://github.com/alloy-rs/op-alloy/issues/149))
- [rollup] Backward-activate forks in `RollupConfig` ([#170](https://github.com/alloy-rs/op-alloy/issues/170))
- [envelope] Add missing `From<Signed<TxEip7702>>` ([#168](https://github.com/alloy-rs/op-alloy/issues/168))

### Miscellaneous Tasks

- Release 0.5.0

## [0.4.0](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.4.0) - 2024-10-09

### Bug Fixes

- Alloy Updates ([#166](https://github.com/alloy-rs/op-alloy/issues/166))
- Op Prefix ([#164](https://github.com/alloy-rs/op-alloy/issues/164))
- [genesis] Op Prefix Naming Convention ([#161](https://github.com/alloy-rs/op-alloy/issues/161))
- [rpc-types-engine] Op Prefix Naming Convention ([#163](https://github.com/alloy-rs/op-alloy/issues/163))
- [rpc-types] Op Prefix Naming Convention ([#162](https://github.com/alloy-rs/op-alloy/issues/162))
- Elide Lifetimes ([#160](https://github.com/alloy-rs/op-alloy/issues/160))
- Safeheadresponse field types ([#156](https://github.com/alloy-rs/op-alloy/issues/156))
- Genesis l1 l2 field types ([#157](https://github.com/alloy-rs/op-alloy/issues/157))
- Remove 4844 transaction type ([#151](https://github.com/alloy-rs/op-alloy/issues/151))
- Reverts 13d0c2 - impl SignableTransaction for Deposit ([#153](https://github.com/alloy-rs/op-alloy/issues/153))
- [genesis] BaseFeeParams Arbitrary Bounds ([#147](https://github.com/alloy-rs/op-alloy/issues/147))

### Features

- Add 7702 ([#167](https://github.com/alloy-rs/op-alloy/issues/167))
- [consensus] Transaction for OpTxEnvelope ([#159](https://github.com/alloy-rs/op-alloy/issues/159))
- [consensus] System Transaction ([#154](https://github.com/alloy-rs/op-alloy/issues/154))
- [`consensus`] Impl `SignableTx` for `TxDeposit` ([#152](https://github.com/alloy-rs/op-alloy/issues/152))
- Codeowner Updates ([#148](https://github.com/alloy-rs/op-alloy/issues/148))
- [protocol] Arbitrary Block Info Types ([#145](https://github.com/alloy-rs/op-alloy/issues/145))
- [genesis] Arbitrary Support ([#144](https://github.com/alloy-rs/op-alloy/issues/144))
- [protocol] Add Frame Iterator ([#141](https://github.com/alloy-rs/op-alloy/issues/141))
- Justfile for my sanity ([#142](https://github.com/alloy-rs/op-alloy/issues/142))
- [rpc-types-engine] EIP-1559 parameters in `OptimismPayloadAttributes` ([#138](https://github.com/alloy-rs/op-alloy/issues/138))
- [genesis] `SystemConfig` holocene updates ([#139](https://github.com/alloy-rs/op-alloy/issues/139))
- [protocol] SystemConfig Conversion Utility ([#135](https://github.com/alloy-rs/op-alloy/issues/135))

### Miscellaneous Tasks

- Release 0.4.0
- Cleanup Arbitrary Tests ([#146](https://github.com/alloy-rs/op-alloy/issues/146))
- Cleanup Workspace Manifest ([#143](https://github.com/alloy-rs/op-alloy/issues/143))
- V0.3.3 ([#140](https://github.com/alloy-rs/op-alloy/issues/140))
- Cleanup Workspace Documentation ([#129](https://github.com/alloy-rs/op-alloy/issues/129))
- [protocol] Remove `L1BlockInfoTx::Holocene` variant ([#137](https://github.com/alloy-rs/op-alloy/issues/137))
- [protocol] Payload Conversion Utilities ([#136](https://github.com/alloy-rs/op-alloy/issues/136))

### Other

- Adding fee computation functions to l1BlockInfoTx ([#134](https://github.com/alloy-rs/op-alloy/issues/134))

## [0.3.2](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.3.2) - 2024-09-30

### Features

- [consensus] Bincode compatibility ([#131](https://github.com/alloy-rs/op-alloy/issues/131))

### Miscellaneous Tasks

- Release 0.3.2 ([#133](https://github.com/alloy-rs/op-alloy/issues/133))
- [genesis] Small README Update ([#128](https://github.com/alloy-rs/op-alloy/issues/128))

## [0.3.1](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.3.1) - 2024-09-30

### Bug Fixes

- HashMap default

### Miscellaneous Tasks

- Release 0.3.1

## [0.3.0](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.3.0) - 2024-09-30

### Bug Fixes

- Clean up protocol std feat flagging ([#119](https://github.com/alloy-rs/op-alloy/issues/119))
- [protocol] Functional Batch Transaction ([#88](https://github.com/alloy-rs/op-alloy/issues/88))
- L1Origin -> l1origin during deser of L2BlockRef ([#116](https://github.com/alloy-rs/op-alloy/issues/116))
- [engine] Missing Error Source ([#114](https://github.com/alloy-rs/op-alloy/issues/114))

### Dependencies

- Bump alloy 0.4 ([#127](https://github.com/alloy-rs/op-alloy/issues/127))
- Use alloy map ([#126](https://github.com/alloy-rs/op-alloy/issues/126))

### Features

- [consensus] OpBlock Type ([#105](https://github.com/alloy-rs/op-alloy/issues/105))
- [workspace] Use Workspace Level Lints ([#125](https://github.com/alloy-rs/op-alloy/issues/125))
- [genesis] Simplify Log Updates in System Config ([#123](https://github.com/alloy-rs/op-alloy/issues/123))
- [genesis] Optimism Base Fee Params ([#122](https://github.com/alloy-rs/op-alloy/issues/122))
- [protocol] Holocene Support ([#118](https://github.com/alloy-rs/op-alloy/issues/118))
- [provider] OP engine api trait ext + superchain signal type ([#117](https://github.com/alloy-rs/op-alloy/issues/117))
- [engine] Deprecate RollupConfig Argument ([#112](https://github.com/alloy-rs/op-alloy/issues/112))
- Exec payload v4 serde test ([#113](https://github.com/alloy-rs/op-alloy/issues/113))

### Miscellaneous Tasks

- Release 0.3.0
- [protocol] Cleanup block info block hash retrieval ([#120](https://github.com/alloy-rs/op-alloy/issues/120))

### Other

- Replace u8 direction field with Direction type ([#90](https://github.com/alloy-rs/op-alloy/issues/90))
- Add holocene time to genesis ([#115](https://github.com/alloy-rs/op-alloy/issues/115))

## [0.2.12](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.2.12) - 2024-09-18

### Bug Fixes

- No_std for op-alloy-rpc-types-engine ([#109](https://github.com/alloy-rs/op-alloy/issues/109))
- [protocol] Invalid Frame Data Length ([#108](https://github.com/alloy-rs/op-alloy/issues/108))

### Dependencies

- Bump alloy 0.3.6 ([#111](https://github.com/alloy-rs/op-alloy/issues/111))
- Bump msrv 1.81 ([#106](https://github.com/alloy-rs/op-alloy/issues/106))

### Features

- [engine] Payload Conversion Utilities ([#110](https://github.com/alloy-rs/op-alloy/issues/110))
- Remove the superchain primitives dependency ([#100](https://github.com/alloy-rs/op-alloy/issues/100))
- [rpc-types-engine] No_std Support ([#104](https://github.com/alloy-rs/op-alloy/issues/104))
- [rpc-types] No_std Support ([#103](https://github.com/alloy-rs/op-alloy/issues/103))
- Remove std flag over alloc ([#101](https://github.com/alloy-rs/op-alloy/issues/101))

### Miscellaneous Tasks

- Release 0.2.12
- Re-export module items ([#102](https://github.com/alloy-rs/op-alloy/issues/102))

## [0.2.11](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.2.11) - 2024-09-13

### Bug Fixes

- Remove Block ID ([#94](https://github.com/alloy-rs/op-alloy/issues/94))
- Issue Template ([#96](https://github.com/alloy-rs/op-alloy/issues/96))

### Features

- Genesis Types ([#97](https://github.com/alloy-rs/op-alloy/issues/97))
- Attributes with parent ([#95](https://github.com/alloy-rs/op-alloy/issues/95))

### Miscellaneous Tasks

- Release 0.2.11

### Other

- Make `l1_origin` in `L2BlockRef` a struct instead of an enum ([#91](https://github.com/alloy-rs/op-alloy/issues/91))

## [0.2.10](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.2.10) - 2024-09-13

### Dependencies

- Bump alloy ([#98](https://github.com/alloy-rs/op-alloy/issues/98))

### Features

- [rpc-types] Replace u8 with Connectedness Enum ([#84](https://github.com/alloy-rs/op-alloy/issues/84))
- Feat(protocol) add block information module ([#82](https://github.com/alloy-rs/op-alloy/issues/82))

### Miscellaneous Tasks

- Release 0.2.10

## [0.2.9](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.2.9) - 2024-09-09

### Bug Fixes

- Use no_std workflow ([#78](https://github.com/alloy-rs/op-alloy/issues/78))
- Alloy-protocols utils fix ([#80](https://github.com/alloy-rs/op-alloy/issues/80))
- Alloy-rs/core update ([#75](https://github.com/alloy-rs/op-alloy/issues/75))
- [protocol] Native u64 ([#73](https://github.com/alloy-rs/op-alloy/issues/73))

### Dependencies

- Bump alloy 0.3.2 ([#86](https://github.com/alloy-rs/op-alloy/issues/86))

### Documentation

- [rpc-type] Add reference to peerdump ([#83](https://github.com/alloy-rs/op-alloy/issues/83))

### Features

- [op-alloy-protocol] Add deposit module ([#81](https://github.com/alloy-rs/op-alloy/issues/81))
- Bump superchain-primitives ([#79](https://github.com/alloy-rs/op-alloy/issues/79))
- [protocol] Deposit Tx Utility ([#74](https://github.com/alloy-rs/op-alloy/issues/74))
- Feature Powerset Job ([#72](https://github.com/alloy-rs/op-alloy/issues/72))
- [protocol] Exports Frame Constants ([#71](https://github.com/alloy-rs/op-alloy/issues/71))

### Miscellaneous Tasks

- Release 0.2.9
- Cleanup depositerror ([#87](https://github.com/alloy-rs/op-alloy/issues/87))

## [0.2.8](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.2.8) - 2024-09-04

### Bug Fixes

- [docs] L1 gas used deprecated since Fjord not Ecotone ([#67](https://github.com/alloy-rs/op-alloy/issues/67))

### Dependencies

- Bump MSRV ([#66](https://github.com/alloy-rs/op-alloy/issues/66))

### Features

- [protocol] Batch Transaction ([#70](https://github.com/alloy-rs/op-alloy/issues/70))

### Miscellaneous Tasks

- Release 0.2.8

### Other

- Make decode_fields pub for TxDeposit ([#68](https://github.com/alloy-rs/op-alloy/issues/68))
- Add encode methods for `TxDeposit` ([#69](https://github.com/alloy-rs/op-alloy/issues/69))

## [0.2.7](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.2.7) - 2024-09-02

### Miscellaneous Tasks

- Release 0.2.7

### Other

- Derive arbitrary for TxDeposit ([#65](https://github.com/alloy-rs/op-alloy/issues/65))

## [0.2.6](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.2.6) - 2024-09-02

### Bug Fixes

- Derive_more dep ([#63](https://github.com/alloy-rs/op-alloy/issues/63))
- [rpc] Add l1 block info to OpTransactionReceipt ([#62](https://github.com/alloy-rs/op-alloy/issues/62))

### Features

- Workflow to validate no_std Compatibility ([#64](https://github.com/alloy-rs/op-alloy/issues/64))
- [consensus] Hardfork Transaction Builders ([#55](https://github.com/alloy-rs/op-alloy/issues/55))

### Miscellaneous Tasks

- Release 0.2.6
- Clean up components used in the feature form ([#60](https://github.com/alloy-rs/op-alloy/issues/60))
- Remove ethers-rs contact link ([#61](https://github.com/alloy-rs/op-alloy/issues/61))

## [0.2.2](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.2.2) - 2024-08-29

### Features

- [protocol] Core Protocol Types ([#56](https://github.com/alloy-rs/op-alloy/issues/56))

### Miscellaneous Tasks

- Release 0.2.2

### Other

- Add ecotone support to `op_alloy_rpc_types::OptimismTransactionReceiptFileds` ([#58](https://github.com/alloy-rs/op-alloy/issues/58))

## [0.2.1](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.2.1) - 2024-08-28

### Bug Fixes

- Some serde fixes ([#51](https://github.com/alloy-rs/op-alloy/issues/51))

### Miscellaneous Tasks

- Release 0.2.1
- Release 0.2.1
- Add missing envelope fns ([#52](https://github.com/alloy-rs/op-alloy/issues/52))

### Other

- Add emhane to CODEOWNERS ([#50](https://github.com/alloy-rs/op-alloy/issues/50))

## [0.2.0](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.2.0) - 2024-08-28

### Bug Fixes

- [rpc] Add Missing Safe Head Endpoint ([#47](https://github.com/alloy-rs/op-alloy/issues/47))

### Dependencies

- [deps] Use latest alloy ([#45](https://github.com/alloy-rs/op-alloy/issues/45))

### Features

- Op-alloy-rpc-types-engine ([#49](https://github.com/alloy-rs/op-alloy/issues/49))
- Add other op endpoints ([#46](https://github.com/alloy-rs/op-alloy/issues/46))
- [rpc-client] Introduce rpc-jsonrpsee Crate ([#37](https://github.com/alloy-rs/op-alloy/issues/37))
- Add rollup and other config types ([#42](https://github.com/alloy-rs/op-alloy/issues/42))
- Added sync file with types from reth ([#35](https://github.com/alloy-rs/op-alloy/issues/35))
- [rpc-types] P2p net types ([#39](https://github.com/alloy-rs/op-alloy/issues/39))

### Miscellaneous Tasks

- Release 0.2.0

### Other

- Set op_alloy_rpc_types::Transaction as Optimism::TransactionResponse ([#33](https://github.com/alloy-rs/op-alloy/issues/33))

## [0.1.5](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.1.5) - 2024-08-08

### Bug Fixes

- Fix arbitrary impl for OpTxType to include deposit tx

### Miscellaneous Tasks

- Release 0.1.5
- Clippy happy ([#30](https://github.com/alloy-rs/op-alloy/issues/30))
- Codeowners
- Downgrad clippy all

### Other

- Add granite_time to OptimismGenesisInfo ([#31](https://github.com/alloy-rs/op-alloy/issues/31))
- Merge pull request [#26](https://github.com/alloy-rs/op-alloy/issues/26) from alloy-rs/matt/codeowners1
- Merge pull request [#23](https://github.com/alloy-rs/op-alloy/issues/23) from alloy-rs/emhane/op-alloy-tx-type
- Replace TxType with OpTxType in Network impl for Optimism
- Implement display for OpTxType
- Merge pull request [#25](https://github.com/alloy-rs/op-alloy/issues/25) from alloy-rs/emhane/fix-arbitrary-op-tx-ty
- Merge pull request [#24](https://github.com/alloy-rs/op-alloy/issues/24) from alloy-rs/matt/downgrade-all-clippy

## [0.1.4](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.1.4) - 2024-07-16

### Dependencies

- Bump alloy

### Miscellaneous Tasks

- Release 0.1.4

## [0.1.3](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.1.3) - 2024-07-13

### Bug Fixes

- Op alloy rpc tx receipt

### Miscellaneous Tasks

- Release 0.1.3
- Use serde::quantity
- Rename mod

### Other

- Merge pull request [#21](https://github.com/alloy-rs/op-alloy/issues/21) from alloy-rs/matt/op-alloy-rpc-receipt
- Merge pull request [#20](https://github.com/alloy-rs/op-alloy/issues/20) from alloy-rs/matt/use-serde-quantity
- Merge pull request [#19](https://github.com/alloy-rs/op-alloy/issues/19) from alloy-rs/matt/rename-mod

## [0.1.2](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.1.2) - 2024-07-08

### Miscellaneous Tasks

- Release 0.1.2
- Update alloy
- Update changelog

## [0.1.1](https://github.com/alloy-rs/op-alloy
/releases/tag/v0.1.1) - 2024-07-03

### Bug Fixes

- Cliff typo
- Fix test
- Fix identifier
- Fix feature
- U128 conversion
- Doc comments
- Receipt type name and flattening
- Receipt trait
- Receipt.rs imports are fixed.

### Dependencies

- Bump alloy version
- Bump alloy version.
- Add serde and alloy_primitives to the dependencies

### Documentation

- Remove outdated documentation.

### Features

- Extract optimism genesis info
- Add genesis types
- Add OP network
- [op-consensus] Trim and complete OP modifications
- [op-consensus] Add optimism deposit tx type
- [consensus] Op-consensus
- [consensus] Op-consensus
- Use generics, remove unnecessary types.
- Review changes.
- Re-export all eth types.
- Add filters.rs
- Fix imports, add TODO comments, organize the code.
- Add pubsub.rs
- Add call.rs and update visibility of transaction requests, types, and receipts.
- Add op-consensus and receiptEnvelope
- Add transaction, and request types. Adjust block to use the crate's transaction and alloy's header.
- Add txType as a separate file under transactions and update receipt.rs accordingly.
- Add block.
- Add txtype, deposit nonce, and receipt version.
- Add log
- Add transaction receipt type without tests + several dependencies.

### Miscellaneous Tasks

- Release 0.1.1
- Add cliff support
- Use alloy from crates
- Rename crate
- Rename crates

### Other

- Merge pull request [#17](https://github.com/alloy-rs/op-alloy/issues/17) from Vid201/feat/op_genesis
- Merge pull request [#16](https://github.com/alloy-rs/op-alloy/issues/16) from alloy-rs/matt/add-genesis-types
- Merge pull request [#15](https://github.com/alloy-rs/op-alloy/issues/15) from alloy-rs/matt/add-cliff-support
- Merge pull request [#14](https://github.com/alloy-rs/op-alloy/issues/14) from alloy-rs/matt/alloy-crates
- Merge pull request [#12](https://github.com/alloy-rs/op-alloy/issues/12) from alloy-rs/matt/add-network-crates
- Exclude wasm
- Merge pull request [#11](https://github.com/alloy-rs/op-alloy/issues/11) from alloy-rs/matt/rename-crates
- Merge pull request [#8](https://github.com/alloy-rs/op-alloy/issues/8) from alloy-rs/feat/op-alloy-consensus
- Reuse exiting receipt
- Make it compile
- Cleanup tx type
- Cleanup tx type
- Inherit `TxReceipt` trait
- Use upstream alloy
- `deposit` fn in `OpTypedTransaction`
- Use upstream Ethereum transaction types from `alloy-consensus`
- Add deposit receipt roundtrip RLP tests
- Use upstreamed `Signed` + `SignableTransaction`
- Merge pull request [#7](https://github.com/alloy-rs/op-alloy/issues/7) from alloy-rs/matt/add-default
- Add missing default
- Merge pull request [#6](https://github.com/alloy-rs/op-alloy/issues/6) from alloy-rs/matt/add-tx-rpc-type
- Allow
- Allow git
- Some cleanup
- Initial commit

### Refactor

- Use native types
- Re-import instead of redefining.
- Update optimism specific fields and their (de)serialization methods in receipt.rs

### Styling

- Fmt
- Cargo fmt
- Cargo fmt.

<!-- generated by git-cliff -->
