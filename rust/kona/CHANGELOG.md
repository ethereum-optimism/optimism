# Changelog

All notable changes to this project will be documented in this file.

## [0.1.3] - 2025-07-31

### 🚀 Features

- *(supervisor:core)* Cyclic dependency checker on block validation (#2597)

### ⚙️ Miscellaneous Tasks

- *(bin)* Remove unused discovery node subcommand (#2609)
- *(providers)* Metrics with Claude (#2610)
- *(grafana)* Update Dashboard (#2612)
- Release 0.3.3

### Release

- *(kona-node)* 1.0.0-rc.1
- *(kona-node)* 1.0.0-rc.1

## [kona-node/v0.1.2] - 2025-07-31

### 🚀 Features

- *(node/service)* Expose standalone `healthz` endpoint (#2603)

### ⚙️ Miscellaneous Tasks

- Fix a large number of spelling issues in comments (#2592)

### Patch

- *(node/conductor)* Patch conductor bootstrap (#2589)

### Release

- *(kona-node)* 0.1.2 (#2608)

## [kona-node/v0.1.1-beta.1] - 2025-07-30

### 🚀 Features

- *(node/service)* Support sequencer recovery mode (#2460)
- *(supervisor/core)* [Safety Checker] validate interop timestamps (#2537)
- *(supervisor/core)* Integrate unsafe block reorg (#2539)
- *(node/service)* Delay sequencer's view of L1 chain (#2568)
- *(supervisor/core)* Handle invalidate blocks (#2564)
- *(supervisor/core)* [Safety Checker] executing message validation (#2570)
- *(protocol)* Add `Jovian` fork definition (#2584)
- *(node/sequencer)* Implement remote signer skeleton (#2572)
- *(node/sequencer)* Integrate remote signer in CLI (#2587)
- Introduce CLAUDE.md file (#2596)

### 🐛 Bug Fixes

- *(node/engine)* Audit engine for bugs. rename forkchoice task -> synchronize task. move block building logic to build task (#2524)
- *(supervisor/core)* Message checksum validation (#2586)

### 🧪 Testing

- *(node/engine)* Add positive engine test (#2552)

### ⚙️ Miscellaneous Tasks

- Fix broken url (#2557)
- *(bin/node)* Improve error pattern matching when pre-validating jwt (#2367)
- *(supervisor/storage)* Define `EntryNotFoundError` (#2542)
- *(docs)* Remove `RuntimeActor` mention (#2567)
- *(protocol)* Optional `OpAttributesWithParent::derived_from` field (#2569)
- *(docs)* Replaced the non-working link driver (#2576)
- *(supervisor)* Add supervisor path to log targets (#2565)
- *(supervisor)* Refactor interop validation (#2578)
- *(supervisor)* Refactor chain processor (#2590)
- *(workspace)* Add `theochap` as author (#2599)

### Patch

- *(node)* Fix macos builds (#2600)

## [kona-node/v0.1.1] - 2025-07-24

### 🚀 Features

- *(book)* Node Documentation (#2304)
- *(node/net)* Rewire unsafe block sender from engine to network (#2378)
- *(node/service)* Add L1 origin consistency check (#2373)
- *(supervisor)* Pre-interop db support (#2368)
- *(supervisor/core)* Implemented task retrier (#2386)
- *(node/engine)* Refactor engine tasks. fix sequencer startup logic (#2388)
- *(node/sequencer)* Connect unsafe head watcher from sequencer to engine (#2406)
- *(node/service)* `L1OriginSelector` advancement (#2404)
- *(supervisor/core)* Preinterop node support (#2385)
- *(node/service)* Reset engine prior to block building (#2415)
- *(node/service)* Prevent second reset on startup when sequencing (#2423)
- *(tests)* Simple `kona-node` sequencer profile (#2424)
- *(node/engine)* Fix unsafe payload signature. add large kona sequencer config (#2425)
- *(docs)* Kona Documentation with Vocs (#2398)
- *(node/sequencer)* Groundwork for op-conductor (#2405)
- *(node/service)* Wire in sequencer configuration (#2447)
- *(kona/logs)* Add logging format options (#2457)
- *(supervisor/storage)* Implemented rewinder for log storage (#2444)
- *(node/service)* Add sequencer admin RPC logs (#2472)
- *(bin/node)* L2 Chain ID Abstraction (#2394)
- *(meta/logs)* Allow logging to file, silencing stdout and more configuration (#2482)
- *(node/service)* Commit unsafe payloads to `op-conductor` (#2486)
- *(tests)* Leadership transfer test (#2493)
- *(node/rpc)* Log rpc server address (#2508)
- (supervisor/storage): rewind storage (#2484)
- *(supervisor/core)* Handle unsafe reorg (#2498)
- *(node/service)* Sequencer state metrics (#2530)
- *(docker)* Add env var to specify cluster name, docker images, grafana port (#2534)
- *(node/service)* Sequencer attributes builder duration metrics (#2531)
- *(node/service)* Sequencer block building job duration (#2532)
- *(node/service)* Conductor commitment time metric (#2533)
- *(supervisor/core)* Added invalidated block on managed node (#2541)
- *(docker)* Sequencer visualizations in `kona-node` dashboard (#2540)
- *(node/engine)* Add metrics to record task success + failure (#2527)
- *(node/engine)* Spike a dev rpc api to get inner engine state. (#2519)

### 🐛 Bug Fixes

- *(supervisor/core)* Unsafe block processing (#2357)
- *(supervisor/core)* Handle safe hash mismatch in reset (#2374)
- *(protocol)* Serialization compatibility for `RollupConfig` (#2416)
- *(node/service)* Block label metrics (#2417)
- *(node/engine)* Consolidate task transient safe chain updates (#2421)
- *(supervisor/rpc)* Making all head ref as optional in syncStatus method (#2427)
- *(bin/node)* Fix `SUPERVISOR_PORT` envvar typo (#2440)
- *(cli)* Correct `init_tracing_subscriber` behavior (#2452)
- *(docs)* Hide Landing Page Logo (#2458)
- *(docs)* Front Page (#2459)
- *(node/engine)* Fix consolidate + insert task metrics (#2450)
- *(supervisor)* Error consistency (#2461)
- *(docs)* FPP Dev Docs (#2470)
- *(docs)* Callouts and Doc Links (#2474)
- *(node/service)* Fix sequencer build ticker (#2473)
- *(node/service)* Add latest unsafe block hash to `admin_stopSequencer` response (#2475)
- *(engine)* Allow `ForkchoiceTask` to drive EL sync (#2514)
- *(docker)* Prometheus addr (#2515)
- *(node/engine)* Do not use FCU V1 (#2545)
- *(node/test)* Increase timeout for e2e test (#2547)

### 📚 Documentation

- Complete derivation documentation for kona-node (#2466)
- Complete execution engine documentation with trait abstractions and kona-node integration (#2467)
- Update shields.io badges to use crate names as labels (#2479)
- Update docker documentation to use correct kona-node targets (#2477)
- Document admin RPC methods following P2P RPC format (#2490)
- Complete rollup RPC methods documentation (#2491)
- Complete P2P RPC endpoints documentation (#2489)
- Comprehensive README for kona-node with installation and usage instructions (#2492)
- *(docker)* Add default ports to `kona-node` recipe README (#2528)

### 🧪 Testing

- *(supervisor/core)* Preinterop e2e test (#2420)
- *(supervisor)* Preinterop acceptance test (#2463)
- *(node/rpc)* Test rollup config rpc endpoint. fix rollup config metrics (#2509)

### ⚙️ Miscellaneous Tasks

- *(node/net)* Integrate the network driver inside the network actor (#2376)
- Fix 404 URL (#2393)
- *(node/engine)* Cleanup engine tasks errors. refactors the engine build task to reuse the insert task (#2399)
- Fix broken url (#2408)
- *(node/service)* Clean up engine task logs (#2422)
- *(workspace)* Remove self-referential `dev-dependencies` (#2451)
- *(docs)* Remove Mdbook (#2454)
- *(docs)* Categorize RFCs (#2455)
- *(docs)* Fix Doc Links (#2453)
- *(supervisor)* Stick to versioned op-node release (#2445)
- *(supervisor/storage)* Make `StorageError::ConflictError` type safe (#2281)
- *(docs)* Remove Supervisor Docs (#2469)
- *(node/engine)* Decouple engine and runtime actor (#2483)
- Release Some Crates (#2485)
- More Crate Releases (#2487)
- *(node/test)* Update monorepo fork (#2505)
- *(docs)* Update `kona-node` flags (#2507)
- *(docker)* Don't disable p2p scoring in `kona-node` recipe (#2510)
- Few Crate Releases (#2517)
- *(supervisor/storage)* Log improvements (#2525)
- *(supervisor)* Remove deprecated supervisor api from `kona-node` (#2024)
- *(supervisor/core)* Log improvements in managed node (#2526)

### Release

- Kona-node v0.1.1 (#2550)

## [kona-node/v0.1.0-beta.5] - 2025-07-08

### 🚀 Features

- *(node/test)* Monitor cpu usage inside test (#2292)
- *(node/test)* Add a way to retrieve RPC endpoints addresses from kurtosis services (#2293)
- Block processing metrics (#2296)
- *(supervisor)* Broadcast cross head update events to managed node (#2289)
- *(supervisor/syncnode)* Observe `ManagedModeApiClient` RPC calls (#2279)
- *(node/service)* L1 origin selector (#2240)
- *(node/service)* Sequencer actor (#2241)
- *(node/engine)* Build task modifications for sequencing (#2242)
- *(node/service)* Handle resets in `SequencerActor` (#2313)
- *(supervisor/rpc)* Map errors to spec errors (#2277)
- *(supervisor/e2e)* Interop test (#2335)
- *(supervisor/e2e)* Rpc e2e test  (#2339)
- *(supervisor/e2e)* `checkAccessList` RPC e2e (#2352)
- *(bin/node)* Sequencer key flag (#2356)
- *(bin/node)* Change `sequencer.enabled` -> `mode` (#2342)
- *(node/sequencer)* Connect build tasks to the engine (#2359)
- *(node/service)* Propagated errors raised during NodeActor::start (#2322)
- *(node/service)* Use drop guards to ensure actors are cancelled properly (#2363)
- *(node/sequencer)* Move node mode from rollup node service trait to engine actor. (#2360)

### 🐛 Bug Fixes

- *(supervisor/storage)* Chaindb metric initialization (#2297)
- *(github/codeowner)* Fix code owners list (#2300)
- *(node/service)* Invert the channel arrows in the rollup node service (#2315)
- *(supervisor/core)* Added missing metrics (#2299)
- *(supervisor)* Cross head promotion stuck (#2327)
- *(supervisor/core)* Consistency check (#2330)
- *(supervisor/rpc)* Invalid param `check_accesslist` (#2334)
- *(supervisor/storage)* Handled derivation storage corner cases (#2340)
- *(supervisor/core)* Derivation reset (#2346)
- *(supervisor/storage)* Safety head ref initialization (#2369)

### ⚙️ Miscellaneous Tasks

- *(node)* Indexed Mode Rename (#2290)
- *(node/tests)* Deprecate devnet-sdk and reactivate RPC endpoint tests (#2294)
- *(book)* Indexed + Polling Traversal Stage Docs (#2291)
- *(supervisor/core)* Cross check l1 block with config (#2268)
- Add chainsafe as codeowner for `test`, `docker` and `Cargo` (#2301)
- *(derive)* Globalize Derivation Crate Imports (#2295)
- *(node/engine)* Refactor process method (#2302)
- *(node/service)* Move actor component building process inside the actors (#2303)
- Fix some minor issues in the comments (#2309)
- *(node/sequencer)* Refactor the sequencer module using the builder pattern (#2310)
- *(node/service)* Have `EngineActor` produce reset request channel (#2312)
- *(node/rpc)* Unify RPC actor with the other actors (#2308)
- *(protocol/docs)* Update spec for gov token address constant (#2171)
- *(supervisor)* Grafana dashboard (#2326)
- *(supervisor/storage)* Remove update_safety_head_ref (#2328)
- *(supervisor/core)* Reset handling refactor (#2332)
- *(node)* `NodeMode` helpers (#2343)
- *(workspace)* Bump dependencies (#2344)
- *(node/rpc)* Remove rpc disabled field in rpc builder (#2364)
- *(supervisor)* Op-node devnet version (#2358)
- *(supervisor)* Remove l1 cache (#2349)
- *(bean/node)* Removed `--l2-engine-kind` from kona (#2321)

## [kona-node/v0.1.0-beta.4] - 2025-06-25

### 🚀 Features

- *(derive)* New Managed Pipelines (#2287)

### 🐛 Bug Fixes

- *(node/p2p)* Fix immediately resolved future (#2286)
- *(node)* Interop Mode Wiring (#2288)

### ⚙️ Miscellaneous Tasks

- *(node/service)* Refactor node actor trait. take context out of the actor (#2271)
- *(node/service)* Refactor and simplify the `RollupNodeService` trait (#2284)

## [kona-node/v0.1.0-beta.3] - 2025-06-25

### 🚀 Features

- *(node/p2p)* Handle peer score inside `opp2p_peers` (#2118)
- *(supervisor/rpc)* Implement `supervisor_localUnsafe` (#2129)
- *(supervisor/rpc)* Implement `supervisor_crossSafe` (#2131)
- *(supervisor/rpc)* Implement `supervisor_finalized` (#2139)
- *(supervisor/core)* Finalized l1 process handling (#2147)
- *(p2p)* Post Unsafe Payload (#2064)
- *(node/test-devstack)* Update devstack configuration to large networks (#2144)
- *(interop)* Managed Event (#2160)
- *(supervisor/rpc)* Implement `supervisor_finalizedL1` (#2157)
- Add predeploy_address constants in maili-genesis now kona-genesis (#2149)
- *(supervisor/core)* Finalized l1 watcher with head ref updates (#2167)
- *(node)* Supervisor Rpc Server Flags (#2168)
- *(node)* Wire up Supervisor RPC Flags + Config (#2174)
- *(node)* Supervisor Actor Setup (#2161)
- *(supervisor/rpc)* Implement supervisor_dependencySet (#2165)
- *(supervisor/core)* Remove hardcoded finalized head from reset event (#2172)
- *(supervisor/interop)* Added a `chain_id` field to ExecutingDescriptor (#2173)
- *(rpc)* Supervisor RPC Server (#2162)
- *(bin/node)* Conductor cli flags (#2193)
- *(supervisor)* Implemented `CrossChainSafetyProvider` (#2197)
- *(supervisor/rpc)* Implement `supervisor_superRootAtTimestamp` (#2180)
- *(supervisor)* Implemented cross chain safety checker (#2200)
- *(node/rpc)* Test node rpc endpoints (#2191)
- *(protocol/derive)* Provide Block Signal (#2230)
- *(supervisor/core)* Check for inconsistency and trigger reset (#2220)
- *(supervisor/storage)* Internal database metrics (#2216)
- *(node/service)* Supervisor Engine Resets (#2235)
- *(supervisor)* Implemented safety checker job (#2212)
- Chaindb metrics (#2238)
- *(derive)* Managed Traversal Stage (#2270)
- *(ci)* Introduce `zepter` checks (#2267)
- *(supervisor)* Integrated cross safety checker job (#2264)
- *(supervisor/core)* L2 finalized head update (#2253)

### 🐛 Bug Fixes

- *(node/p2p)* Fix opp2p_peers rpc handler. Only return total peer score (#2121)
- *(supervisor/rpc)* Metrics added for rpc methods (#2138)
- *(supervisor)* Expose get_superhead using trait (#2150)
- *(supervisor/rpc)* LocalUnsafe always points to genesis (#2145)
- *(supervisor)* Use raw config l2 url (#2188)
- *(interop)* Remove timestamps from the depset (#2187)
- *(protocol/interop)* Made `override_message_expiry_window` as optional (#2208)
- *(node/service)* Enable TLS in alloy providers / transports (#2222)
- *(supervisor)* Supervisor kurtosis and test (#2251)
- *(supervisor/core)* Consistency check (#2250)
- *(node/docker)* Fix kona's docker deployment (#2259)
- *(docker)* Include `ca-certificates` in final executable image (#2260)
- *(hardforks)* Enable Ecotone Selector (#2263)

### 🧪 Testing

- *(node/e2e-devstack)* Increase node coverage with devstack (#2153)
- *(node/e2e-devstack)* Deprecate p2p tests in devnet-sdk (#2159)
- *(protocol/interop)* Add log parsing test  (#2196)
- *(node/p2p)* Adding peer ban tests (#2201)

### ⚙️ Miscellaneous Tasks

- *(rpc)* Cleanup IpNet TODOs (#2120)
- *(supervisor/storage)* Add note about local unsafe block to `DerivationStorageReader::derived_to_source` (#2127)
- *(supervisor)* State e2e (#2132)
- *(node/cli)* Fix todos (#2142)
- *(node/p2p)* Merge peer infos and peerstore (#2143)
- *(supervisor)* Update `SupervisorError` conversion (#2134)
- *(supervisor/core)* Rm unused variant `ChainProcessorError::InvalidChainId` (#2152)
- *(supervisor/core)* Mv `SupervisorError` to own module (#2151)
- *(node/rpc)* Cleanup rpc config flags and launcher (#2163)
- *(node/net)* Simplify the network interface. (#2175)
- *(supervisor)* Passing l1_provider instead of l1_rpc_url (#2043)
- *(node/rpc)* Unify BaseFeeConfig to follow op-geth (#2198)
- *(bin/node)* Increase error verbosity on port binding failure (#2225)
- *(node/service)* Squash service traits (#2231)
- *(node/service)* Cleanup `RollupNodeService::start` (#2233)
- *(bin/node)* Move conductor flags to sequencer args (#2234)
- *(node/test)* Update dependencies (#2224)
- *(derive)* Metrics Mod Visibility (#2247)
- *(driver)* Code Doc References (#2248)
- *(derive)* Code Doc Comments (#2249)
- *(genesis)* Remove Deprecated HardforkConfig Alias (#2258)
- *(workspace)* Use `rustls` over `request`'s default TLS feature (#2261)
- *(bin/node)* Improve error verbosity for runtime loader failure (#2262)
- *(hardforks)* Code Comment References (#2265)
- *(node/service)* Remove process method from NodeActor trait (#2266)

### Fear

- *(supervisor/core)* Added a new method to fetch the super head in a single tx (#2140)

## [kona-node/v0.1.0-beta.2] - 2025-06-12

### 🚀 Features

- *(node/e2e-tests)* Bootstrap devstack testing suite (#2105)
- *(p2p)* Track and Return Peer Ping Latencies (#2112)
- *(node/p2p)* Impl BlockSubnet + UnblockSubnet (#2073)
- *(node/p2p)* Impl opp2p_listBlockedSubnets (#2072)
- *(node/devstack-e2e)* Convert simple tests to devstack (#2116)
- *(node/p2p)* Record peer score distribution metrics (#2117)

### 🐛 Bug Fixes

- *(node/p2p)* Reduce number of peers that need to be connected for large e2e tests to pass (#2099)
- *(p2p)* Properly Record Connectedness (#2103)

### ⚙️ Miscellaneous Tasks

- *(supervisor)* E2E workflow configuration (#2091)
- *(docker)* Update grafana dashboard (#2101)
- *(p2p)* Kona Peers (#2097)
- *(node/p2p)* Enable peer monitoring if configured (#2102)
- Update Alloy Dependencies (#2082)
- *(node)* Add L2 chain ID environment variable (#2113)
- *(supervisor/docs)* Fix docs for RPC method `supervisor_allSafeDerivedAt` (#2104)
- *(node)* Standardize envvars (#2114)
- *(node)* Add rollup config env var (#2115)

## [kona-node/v0.1.0-beta.1] - 2025-06-11

### 🚀 Features

- *(derive)* Initial Metrics (#2007)
- *(supervisor/storage)* Derivation storage initialization (#1963)
- *(supervisor/rpc)* Add RPC method `supervisor_syncStatus` (#1952)
- *(supervisor/storage)* Log storage initialization (#1979)
- *(supervisor)* Service initialisation (#1984)
- *(derive)* Frame + Channel Metrics (#2008)
- *(derive)* Max Channel Size and Timeout Metrics (#2009)
- *(derive)* Batch Metrics (#2010)
- *(derive)* Pipeline Metrics (#2021)
- *(node/p2p)* Support chain id info inside opp2p_peers (#2017)
- *(node/p2p)* Support user agent and protocol version in opp2p_peers rpc call (#2018)
- *(node/p2p)* Support gossip information inside `opp2p_peers` (#2016)
- *(derive)* System Config Update Metrics (#2029)
- *(node/p2p)* Rotate peers out of `BootStore` (#2033)
- *(derive)* Decompressed Channel Size Metric (#2031)
- *(node/service)* Introduce backpressure in DA watcher channels (#2030)
- *(node/p2p)* Add a shim handler for sync req resp protocol (#2042)
- *(rpc)* Connect Peer RPC Method (#2037)
- *(node/service)* Removed unbounded channels in kona-service actors and replaced with bounded ones (#2045)
- *(node/p2p)* Improve dial metric granularity (#2052)
- *(p2p)* Connection Gater (#2055)
- *(p2p)* Peer Blocking (#2056)
- *(supervisor/core)* Reset event handling (#2049)
- *(p2p)* Address Blocking (#2060)
- Make websocket optional (#1918)
- *(p2p)* Peer Protection (#2061)
- *(supervisor/core)* `derivationOriginUpdate` event (#2070)
- *(node/p2p)* Add a dial period to dial protections (#2075)
- *(p2p)* Peers Rpc Endpoint Progress (#2081)
- *(ci)* Improve caching of e2e jobs (#2080)
- *(supervisor/rpc)* Implement RPC method `supervisor_allSafeDerivedAt` (#2078)
- *(rpc)* Healthcheck Endpoint (#2095)

### 🐛 Bug Fixes

- *(node/p2p)* Fix op-p2p peers rpc call (#2014)
- Managed node initialisation (#2023)
- *(node/p2p)* Fixing gossip stability by limiting the number of dials to a given peer (#2025)
- *(supervisor/core)* Subscription handling (#2027)
- *(docker)* Correctly set RPC port in node recipe (#2035)
- *(node/p2p)* Reactivate e2e tests for gossip. (#2036)
- *(node/service)* Authenticate unsafe block signer updates (#2041)
- *(node/p2p)* Fix cli arguments values for node's p2p (#2051)
- *(node/p2p)* Fix large e2e kurtosis tests (#2059)
- *(node/p2p)* Raise default dial threshold (#2063)
- *(tests/deps)* Update monorepo fork dep (#2071)
- *(test/deps)* Fix go dependencies (#2076)
- *(node/e2e-tests)* Enable websockets for e2e tests (#2079)
- *(p2p)* Peer Blacklisting (#2085)
- *(node/net)* Added back the `pubpublish_rx `in the NetworkActor, adjusted the other sections (#1910)

### ⚙️ Miscellaneous Tasks

- *(docker)* Update grafana dashboard (#1991)
- Kurtosis supervisor network (#1949)
- *(derive)* Remove Prelude (#2038)
- *(docker)* Update Grafana Dashboard (#2040)
- *(derive)* Hoist Error Exports (#2039)
- *(node/service)* Display node mode in startup log (#2053)
- *(node/service)* Remove `engine_ready` flag (#2054)
- *(node/service)* Use watch channels in L1 watcher (#2057)
- Dedup GlobalArgs for log verbosity (#2066)
- *(engine)* Log finalized head updates (#2058)
- *(derive)* Flatten Exports (#2077)
- *(tests)* Add `kona-node` + `op-geth` config (#2074)
- *(supervisor)* Use devstack for e2e tests (#2062)
- *(p2p)* Banned Peer Count (#2083)
- *(p2p)* Protected Peer Stat (#2084)
- *(ci)* Don't fail fast on E2E matrix (#2098)

## [kona-host/v1.0.2] - 2025-06-04

### 🚀 Features

- *(supervisor/core)* Managed derivation data flow (#1946)
- *(node/engine)* Add `info` log for safe head promotion (#1948)
- *(node)* Override Flags (#1968)
- *(node/p2p)* Track Banned Peers (#1967)
- Unsafe block processing (#1951)
- *(supervisor/core)* Rollup config (#1953)
- *(node/p2p)* Peer score histogram (#1975)

### 🐛 Bug Fixes

- *(node/p2p)* Gossipsub connection metric keys (#1947)
- *(p2p)* Disable Peer Scoring (#1962)
- *(node)* CLI Metrics Init (#1969)

### ⚡ Performance

- Remove Duplicated L2 Block Query (#1960)

### ⚙️ Miscellaneous Tasks

- *(workspace)* Bump MSRV to `1.86` + `reth` deps (#1945)
- Update Dependencies (#1955)
- *(rpc)* Rollup Node RPC Endpoint Metrics (#1958)
- *(rpc)* Disconnect Peer RPC Request (#1964)
- *(node)* Additional CLI Option Stats (#1965)
- *(ci)* Update `monorepo` revision (#1966)
- *(node)* Hardfork Metrics (#1971)
- *(docker)* Update grafana dashboard (#1970)
- *(node)* Additional CLI Option Metrics (#1973)
- *(rpc)* Server Restarts (#1972)
- *(node)* Rollup Config Metrics (#1974)
- *(node)* Indexable hardfork activation time metrics (#1976)
- Fix some typos in comment (#1978)
- *(client/host)* Prepare for `v1.0.2` release (#1995)

## [kona-node/v0.0.1-beta.6] - 2025-06-02

### 🚀 Features

- *(node-service)* Derivation pipeline L1 origin metric (#1892)
- *(docker)* Update `kona-node` dashboard (#1894)
- *(supervisor/core)* Managed node event channel (#1887)
- *(justfile/docker)* Import docker/app justfile in main kona justfile (#1912)
- *(node/e2e-tests)* Add a restart recipe for hot reloads of the kurtosis network (#1914)
- *(supervisor/core)* Chain processor skeleton (#1906)
- Use alloy-op-hardfork constants (#1922)
- *(node/p2p)* Adding gossipsub event metrics (#1916)
- *(supervisor/core)* Logindexer implementation (#1898)
- *(node/service)* Add metric for critical derivation errors (#1938)

### 🐛 Bug Fixes

- Supervisor kurtosis devnet (#1900)
- *(node/e2e-sync)* Fix large sync test (#1899)
- *(docker)* Add fix-missing to fix docker builds (#1915)
- *(ci)* Prefix-key designation in generic setup (#1933)
- *(docker)* Use cache busting for generic app dockerfile (#1935)
- *(p2p)* Labels for Gossip Events (#1932)

### 📚 Documentation

- *(cargo-chef)* Add cargo chef to cache rust builds in docker (#1896)

### ⚡ Performance

- *(ci)* Use `mold` linker for performance (#1934)
- *(ci)* Only persist rust cache on `main` (#1936)
- *(engine)* Batch safe-head FCUs (#1937)

### 🧪 Testing

- *(node/e2e-sync)* Adding e2e sync tests for unsafe/finalized sync (#1861)

### ⚙️ Miscellaneous Tasks

- *(node-sources)* Enable `RuntimeLoader` metrics (#1893)
- *(supervisor)* Error logs consistency (#1889)
- *(supervisor)* Smol fix spacing in display string (#1903)
- *(supervisor/core)* `ManagedNode` error handling (#1904)
- *(supervisor)* Add `SubscriptionError` (#1908)
- *(supervisor)* Define `AuthenticationError` (#1911)
- *(node/supervisor)* Move MetricArgs into kona-cli  (#1888)
- Update Dependencies (#1923)
- *(protocol)* Encapsulate Magic Arithmetic (#1924)
- *(docker)* Small Grafana Doc Update (#1931)
- Update Kona Node Grafana Dashboard (#1930)
- *(ci)* Remove rust toolchain installation for e2e workflows (#1939)
- *(node)* P2P CLI Metrics (#1940)

## [kona-node/v0.0.1-beta.5] - 2025-05-28

### 🚀 Features

- *(engine)* Chain label metrics (#1741)
- *(node)* Retry engine capability handshake (#1753)
- *(node-service)* Aggressively process channels (#1756)
- *(node)* Version info metrics (#1758)
- *(node)* Mark derivation as idle when it yields (#1759)
- *(supervisor/service)* L1 watcher for supervisor (#1717)
- *(engine)* Task count metrics (#1766)
- *(engine)* Engine method call duration metrics (#1767)
- Replace SafetyLevel with op_alloy type (#1782)
- *(node)* Propagate engine reset to pipeline (#1789)
- Dependency set (#1793)
- *(node)* Add kurtosis e2e testing skeleton (#1792)
- *(engine)* Simplify EL sync startup routine (#1809)
- *(node/ci)* Add e2e tests to ci (#1798)
- *(engine)* Engine reset counter metric (#1806)
- *(node/p2p)* Enable support for the peers RPC endpoint (#1811)
- *(node/p2p)* Implements the `opp2p_peerStats` endpoint (#1812)
- *(node-service)* Handle derivation reset events (#1816)
- *(node-service)* L1 reorg metrics (#1817)
- *(supervisor/storage)* Derivation schema (#1808)
- *(node/e2e-test)* Add peer count test in Kurtosis (#1823)
- *(docker)* `kona-node` recipe (#1832)
- *(protocol/interop)* Derived ref pair (#1834)
- *(node/p2p)* Streaming engine state through websockets (#1833)
- *(supervisor/storage)* Derivation provider (#1835)
- *(node)* L2 finalization routine (#1858)
- *(supervisor/storage)* Log storage  (#1830)
- *(supervisor/storage)* Chaindb and chaindb factory (#1864)
- *(supervisor/db)* Safety head reference storage   (#1865)
- *(node-service)* Refactor `EngineActor` (#1867)
- *(docker)* Local builds + Cleanup (#1877)
- *(supervisor/core)* Implement l2 controller (#1866)
- *(workspace)* Performance release build profile (#1882)

### 🐛 Bug Fixes

- *(node/engine)* Initialize Unknowns (#1705)
- *(node/rpc)* Removed the panic and added error when building RPC actor (#1709)
- *(node/engine)* Derivation Sync Start (#1708)
- *(node-p2p)* Discovery event metric labels (#1740)
- *(engine)* Reset with inconsistent chain state (#1763)
- *(node/p2p)* Bootnodes in Bootstrap + Backoff Discv5 Restart (#1755)
- *(node/p2p)* Disable Topic Scoring (#1765)
- *(p2p)* Discv5 startup panic (#1768)
- *(bin/node)* Argument Defaults (#1769)
- *(kurtosis)* Fix kurtosis configuration (#1774)
- *(node-service)* Engine sync completion condition (#1780)
- *(node/p2p)* Async Broadcast with Backoff (#1747)
- *(node/p2p)* Default Channel Size (#1783)
- *(ci)* Use stable toolchain for lint job (#1802)
- *(node)* Zero `safe` + `finalized` hashes while EL syncs (#1801)
- *(node)* Set `local-safe` and `cross-unsafe` labels (#1803)
- *(node)* Build task + consolidate condition (#1804)
- *(ci)* Free disk space on kurtosis e2e tests (#1821)
- *(sources)* Stop sync start walkback at genesis (#1818)
- *(node/kurtosis-e2e-test)* Fixes the versioning for the e2e testing package (#1814)
- *(node/p2p)* Fix node id serialization and connectedness display (#1824)
- *(p2p/rpc)* Fix formatting for the `opp2p-peers` rpc method (#1827)
- *(node/async)* Fix cpu usage from future polling (#1848)
- *(engine)* Prevent `drain` starvation (#1853)
- *(kurtosis/config)* Fix kurtosis config to use teku instead of nimbus. (#1852)
- Several typos (#1874)

### 🧪 Testing

- *(supervisor)* Add kurtosis network params for supervisor (#1777)
- *(node/e2e)* Extend the p2p tests (#1831)
- *(node/sync-e2e)* Adding e2e safe head sync tests (#1857)
- *(supervisor/ci)* Configure supervisor workflow (#1840)
- *(supervisor)* Comment out kona-supervisor in kurtosis network params (#1873)

### ⚙️ Miscellaneous Tasks

- *(meta)* Add chainsafe as code owners of `kona-supervisor` (#1706)
- *(workspace)* Update Alloy Deps (#1732)
- *(workspace)* Update REVM Deps (#1733)
- *(engine)* Remove `skip` module (#1737)
- *(node/engine)* Remove CL Sync (#1739)
- *(node/engine)* Update Docs (#1736)
- *(node/rpc)* Remove Duplicate Protocol Versioning Code (#1744)
- *(engine)* Chain label metrics cleanup (#1751)
- *(engine)* Remove pending/backup unsafe head (#1742)
- *(bin/node)* Remove Discover Subcommand (#1748)
- *(bin/node)* Subcommand Tests (#1752)
- *(node/service)* Small Cleanup (#1743)
- Update LICENSE.md (#1764)
- *(node)* Empty cargo features metric (#1770)
- *(sources)* Refactor sync start to use alloy providers (#1773)
- *(ci)* Normalize foundry version (#1787)
- Consolidate Scripts (#1788)
- *(node/p2p)* Remove Stale Docs (#1784)
- *(supervisor)* Rm deprecated `InvalidInboxEntry` error (#1778)
- *(engine)* Update block insertion log (#1791)
- *(node/p2p)* Move P2P CLI Utilities (#1785)
- *(node-service)* Lazy initialize pipeline cursor / engine state (#1790)
- *(workspace)* Manifest Hygiene (#1795)
- *(protocol/registry)* Remove Default Hasher (#1797)
- *(workspace)* Remove + Ignore Vscode Config (#1796)
- *(workspace)* Use `alloy-eips`' `EMPTY_REQUESTS_HASH` constant (#1805)
- *(engine)* Remove `SyncStatus` (#1810)
- *(engine)* Don't zero block label gauges (#1820)
- *(supervisor)* Dedup `SupervisorApi` (#1713)
- *(workspace)* Bump dependencies (#1846)
- *(docker)* Update `kona-node` dashboard (#1859)
- *(supervisor/storage)* Rename BlockRef field `time` to `timestamp` (#1870)
- *(ci)* Consolidate workflows (#1863)
- *(workspace)* Remove kurtosis recipes (#1879)
- *(kurtosis/e2e-tests)* Revamp e2e testing to natively use kurtosis and not clone any repo (#1880)
- *(workspace)* Move `kona-macros` to `utilities` (#1884)
- *(supervisor)* Rename `ManagedNodeApi` to `ManagedModeApi` to match specs (#1875)
- *(tests)* Skip `node/p2p` e2e tests (#1885)
- *(workspace)* `Justfile` -> `justfile` (#1883)

### Patch

- *(node/p2p)* Patch the default p2p params (#1856)

## [kona-host/v1.0.1] - 2025-05-10

### 🚀 Features

- *(docker)* Enable reproducible prestate builds for interop program (#1610)
- *(interop)* Use `FpvmOpEvmFactory` in interop proof program (#1611)
- Add code hash tests (#1601)
- *(node)* Describe and Zero Metrics (#1612)
- *(genesis)* Add `is_first_fork_block` helpers to `RollupConfig` (#1646)
- *(genesis)* Add `block_number_from_timestamp` helper (#1647)
- *(interop)* Bubble up message validity errors (#1644)
- *(interop)* Support unaligned activation time (#1645)
- *(proof)* Derivation over generic L1/L2/DA providers (#1655)
- *(host)* Trace span for `L2BlockData` re-execution (#1658)
- *(node/service)* Superchain Signaling with Runtime Loading (#1662)
- *(node/engine)* Transmit the most recent version of the `EngineState` over watch channels to the engine actor (#1673)
- *(node/service)* Flush Channel on Invalid Payloads (#1675)
- *(node/service)* Update L2 Safe Head (#1677)
- *(node/service)* Re-import Deposits Only Payload (#1676)
- *(node/engine)* Set Safe Head Consolidation (#1690)
- *(executor)* Public `compute_receipts_root` method (#1686)
- *(engine/rpc)* Skeleton implementation of the engine rpc (#1664)
- *(node/l1_watcher)* Adds query handler for the l1 watcher (#1692)
- *(node/rpc)* Implement the rollup RPC endpoints (#1697)
- *(supervisor)* Boilerplate (#1700)
- Bump `kona-client` and `kona-host` versions (#1711)

### 🐛 Bug Fixes

- *(node/p2p)* Use Compat Metrics Crate (#1609)
- *(node/engine)* Incorrect Engine Method Use in Insert Task (#1630)
- *(node/engine)* Don't use Genesis in EngineState (#1635)
- *(node/service)* Wait to Kickstart Derivation (#1640)
- *(node/engine)* Attributes Matching and Sync Status (#1665)
- *(bin/node)* Runtime Config Wiring (#1669)
- *(node/engine)* Attributes Tx Mismatch (#1666)
- *(node/engine)* Round Robin Task Execution (#1667)
- *(node/service)* Mark Engine Ready on Sender (#1674)
- *(node/service)* EL Sync Only (#1691)
- *(node/service)* Check L2 Safe Head BN (#1703)
- *(node/engine)* Pre-Holocene Deposits Only (#1702)

### ⚙️ Miscellaneous Tasks

- *(client)* Loosen type constraints on `FpvmOpEvmFactory` (#1613)
- Rm redundant bounds (#1614)
- *(genesis)* Use `abi_decode_validate` (#1632)
- *(interop)* Reduce message expiry window (#1636)
- *(workspace)* Bump op-alloy Dep (#1631)
- *(bin/node)* Disable DISCV5 logging by default (#1638)
- *(node/engine)* Decrease Temporary Log Level (#1639)
- *(bin/node)* Cleans up Logs Some More (#1641)
- *(node)* Log Cleanup (#1642)
- *(interop)* Refactor `MessageGraph` test utilities (#1643)
- *(node/rpc)* Refactor p2p rpc (#1633)
- *(bin/node)* CLI Argument Unit Tests (#1660)
- *(node/engine)* Small Engine Touchups (#1657)
- *(node/rpc)* Remove no-std requirement from kona-rpc, fix imports (#1659)
- *(docker)* Update `asterisc` tag (#1661)
- *(proof)* Drop lock before `await` (#1663)
- *(hardforks)* Correct EIP-2935 source name (#1678)
- *(meta)* Add chainsafe as supervisor code owners (#1701)
- *(kona/kurtosis)* Update the link to the optimism-package repo (#1707)

## [kona-host/v1.0.0] - 2025-05-01

### 🚀 Features

- *(workspace)* Kurtosis Justfile Targets (#1594)
- *(protocol)* Interop upgrade transactions (#1597)
- *(protocol)* Interop transition batch validity rule (#1602)

### 🐛 Bug Fixes

- *(node/p2p)* Enable identify protocol. Additional small fixes (#1592)
- *(workspace)* Build Kona Node (#1598)
- *(bin/node)* Enable Metrics (#1600)
- *(node/p2p)* Block Validation (#1604)

### ⚙️ Miscellaneous Tasks

- *(workspace)* Add FPVM artifacts to `gitignore` (#1593)
- *(interop)* Update interop fork activation check for initiating messages (#1599)
- *(docker)* Update `cannon` tag (#1606)

### Release

- `kona-client` + `kona-host` v1.0.0 version bump (#1605)

## [kona-host/v0.1.0-beta.18] - 2025-04-29

### 🚀 Features

- *(node)* Redial Peers (#1477)
- Add info subcommand to kona-node (#1488)
- *(node)* Discover Command (#1481)
- *(bin/node)* Add custom bootnode list CLI arg (#1496)
- *(bin/node)* Ensure genesis matches the rollup config before fetching to the known network params list. (#1498)
- *(interop)* Add message expiry check (#1506)
- *(config/kurtosis)* Add a simple kurtosis configuration file to the `.config` folder (#1512)
- *(bin/node)* Add advertise p2p flags (#1509)
- *(bin/node)* Also advertise udp port (#1516)
- *(node/p2p)* Add simple p2p rpc endpoints (#1535)
- *(proof-interop)* Block replacement transaction (#1540)
- *(protocol)* Add `OutputRoot` type (#1544)
- *(protocol)* Complete `Predeploys` definition (#1549)
- *(ci)* Interop FPP action tests (#1546)
- *(bin/node)* Extend P2P Configurability (#1553)
- *(node/p2p)* Disable dynamic ENR updates for static IPs (#1558)
- *(std-fpvm)* Instruct kernel to lazily allocate pages (#1567)

### 🐛 Bug Fixes

- *(bin/node)* Sequencer Args Duplicate CLI Flag (#1490)
- *(node/p2p)* Metrics Tasks (#1491)
- *(node/p2p)* Peer Score Level Off (#1493)
- *(node/p2p)* Discovery Test Fixes (#1494)
- *(utilities)* CLI Verbosity Level (#1502)
- *(proof-interop)* Operate provider off of `local-safe` heads (#1503)
- *(node)* Unsafe Block Signer (#1505)
- *(bin/node)* Magic String (#1513)
- *(bin/node)* Adds Lints (#1518)
- *(node/p2p)* Add Lints (#1520)
- Use Unspecified Ipv4Addr (#1522)
- *(bin/node)* Fix the unsafe block signer to be compatible with unknown chain ids (#1523)
- *(node/p2p)* Ensure that the discovery and the gossip keys match (#1525)
- *(bin/node)* Discovery Config Unset (#1532)
- *(host-interop)* Ignore block data hint if chain hasn't progressed (#1542)
- *(node/p2p)* Use EnrValidation (#1552)
- *(node/p2p)* Wait for Swarm Address Dialing (#1554)
- *(std-fpvm)* Large file IO (#1555)
- *(node/p2p)* Fix the address list returned by `opp2p_self` (#1559)
- *(node/p2p)* Fix multiaddress translation (#1561)

### 📚 Documentation

- *(providers-alloy)* Doc touchups (#1504)

### ⚡ Performance

- *(ci)* Cache forge artifacts in action tests (#1548)
- *(std-fpvm)* Switch to `buddy_system_allocator` (#1590)

### ⚙️ Miscellaneous Tasks

- Bump alloy 0.15 (#1492)
- *(node/p2p)* Broadcast Wrapper (#1489)
- *(bin/node)* Adding more flexibility to verbosity levels (#1497)
- Bump `op-alloy` to `v0.15.1` (#1515)
- *(bin/node)* Move Runtime Loading into Sources (#1517)
- *(node/p2p)* Peer Count Metric (#1527)
- *(node/p2p)* Forward Discovery Events (#1495)
- *(ci)* Remove interop proof from codecov ignore (#1547)
- *(host)* Gate experimental `debug_payloadWitness` usage (#1591)

## [kona-node/v0.0.1-beta.1] - 2025-04-22

### 🐛 Bug Fixes

- *(node)* Fix RPC client building by adding jwt auth. (#1487)

## [kona-node/v0.0.2] - 2025-04-22

### 🚀 Features

- *(node)* Bootstore Debugging Tool (#1478)
- Function to init tracing in testing (#1467)
- Add sequencer CLI params (#1485)

### 🐛 Bug Fixes

- *(proof)* Blob preimage keys (#1473)
- *(bin/node)* Global Argument Positioning (#1482)

### ⚙️ Miscellaneous Tasks

- *(ci)* Bump timeout on Rust CI jobs (#1479)
- *(workspace)* Improve proof tracing (#1476)
- *(workspace)* Convert all tracing targets to `snake_case` format (#1484)
- *(bin/node)* Subcommand Aliases (#1483)

### Bug

- *(bin/node)* Fix metrics address flag (#1486)

## [kona-node/v0.0.1] - 2025-04-18

### 🚀 Features

- Use `alloy-evm` for stateless block building (#1400)
- *(executor)* Add example + docs for generating new test fixtures (#1438)
- Discovery Interval Configurability (#1445)
- *(node/p2p)* Refactor block validity checks (#1451)
- *(node/p2p)* Add manual block hash checks (#1453)
- *(node/p2p)* Add version specific block checks (#1454)
- *(node/p2p)* Add remaining block checks (replays + maximum block number per height) (#1455)
- *(docker)* Use generic dockerfile for all binary apps (#1465)
- *(ci)* Generic Binary (#1470)

### 🐛 Bug Fixes

- *(node/p2p)* Enr Validation (#1446)
- *(bin/node)* Break after Receiving Peer Count (#1448)
- *(node)* Configurable Bootstore Path (#1449)
- *(cli)* Missing Env Filter (#1458)
- *(node)* Local Bootstore Conflicts (#1450)
- *(protocol)* Incorrect Genesis Hash Consensus Block (#1459)

### 🧪 Testing

- *(node/p2p)* Add extensive testing for block decoding validation (#1452)

### ⚙️ Miscellaneous Tasks

- Remove useless TODO (#1418)
- Use `OpPayloadAttributes::recovered_transactions` (#1434)
- *(ci)* Add action to free up disk space in action test job (#1439)
- Use encodedwith for execute (#1441)
- *(comp)* Enable `test-utils` feature with `cfg(test)` (#1442)
- *(derive)* Enable `test-utils` feature with `cfg(test)` (#1443)
- *(protocol)* Enable `test-utils` feature with `cfg(test)` (#1444)
- *(bin/node)* Discovery Config Wiring (#1456)
- *(node/service)* P2P Rpc Module Wiring (#1469)

### Bug

- *(node/p2p)* Fix bootstrapping for enode addresses (#1435)

## [kona-host/v0.1.0-beta.16] - 2025-04-15

### 🚀 Features

- *(docker)* Reproducible `cannon` prestate (#1389)
- *(node/service)* Wire in Engine Arguments (#1383)
- *(node/service)* Init the Engine Actor (#1387)
- *(node/service)* Wire in the Engine Actor (#1390)
- *(bin/node)* Registry Subcommand (#1379)
- *(node/service)* Engine Consolidation Task (#1391)
- *(node/service)* Insert Unsafe Payload Envelope (#1392)
- *(bin/node)* Implement peer banning (#1405)
- *(node/engine)* Add transaction checks to the consolidation step (#1412)
- *(node/engine)* Check eip1559 parameters inside consolidation (#1419)
- *(bin/node)* Extend metrics configuration options (#1422)

### 🐛 Bug Fixes

- *(bin/node)* Disable P2P when Specified (#1409)
- *(node/service)* Error Bubbling and Shutdown (#1410)
- *(node/engine)* Engine State Builder Missing (#1408)

### ⚙️ Miscellaneous Tasks

- *(node/engine)* Use thiserror instead of anyhow (#1395)
- *(docs)* Edited the license link (#1403)
- *(node/service)* Engine State Builder Error (#1406)
- *(node/service)* Touchup EngineActor (#1404)
- Bump scr + monorepo (#1420)
- *(bin/node)* In-use Port Erroring (#1411)
- *(ci)* Hide action test output (#1428)
- Add `@theochap` to `CODEOWNERS` file (#1430)
- *(bin/node)* Add jwt startup check (#1421)
- Remove redundant word in comment (#1433)

## [kona-host/v0.1.0-beta.15] - 2025-04-08

### 🚀 Features

- *(node)* RPC CLI Args (#1314)
- *(node)* RPC Config (#1315)
- *(node)* Rpc Actor (#1318)
- *(node)* Wire in the RpcConfig (#1321)
- *(protocol)* Move Compression Types (#1298)
- *(node/rpc)* RpcLauncher (#1325)
- *(node/p2p)* P2P RPC Server (#1327)
- *(node)* Network Config (#1323)
- *(bin)* Network Subcommand (#1300)
- *(node/p2p)* Redial Peers (#1367)
- *(bin/node)* Peer Scoring Setup (#1376)
- *(node/p2p)* Unsafe Payload Publishing (#1359)
- *(node/service)* Unsafe Block Signer Updates (#1386)

### 🐛 Bug Fixes

- *(node/service)* Unsafe Block Signer (#1322)
- *(node/p2p)* Gossip Config (#1328)
- *(node/p2p)* Forward ENRs to the Swarm (#1337)
- *(bin)* Correct Unsafe Block Signer (#1339)
- *(node/p2p)* OP Stack ENRs (#1353)
- *(node/p2p)* Unsafe Payload Sending (#1366)
- *(bin/host)* Fix typo (#1384)

### 📚 Documentation

- Fx incorrect link reference for MSRV section (#1302)

### ⚙️ Miscellaneous Tasks

- Remove Magic 0x7E Deposit Identifier Bytes (#1292)
- *(cli)* Refactors Backtrace Init (#1293)
- *(ci)* Refactor Github Action Steps (#1294)
- *(protocol)* Remove unused L1 Tx Cost Functions (#1295)
- *(protocol)* Cleanup Utilities (#1297)
- *(protocol)* Remove Unused Frame Iterator (#1296)
- Small Manifest Cleanup (#1299)
- Update Dependencies (#1304)
- Derive std Traits (#1329)
- Derive More Core Traits (#1333)
- *(node/p2p)* Network RPC Request Handling (#1330)
- *(bin)* Rename RpcArgs (#1338)
- *(node/p2p)* Cleanup Network Driver (#1349)
- *(node/p2p)* Log Cleanup (#1348)
- *(bin/node)* P2P Config Constructor (#1350)
- *(node/service)* Cleans up the Rollup Node Service (#1352)
- Bump op-alloy Patch (#1364)
- *(node/service)* Dynamic Node Mode (#1358)
- *(bin/node)* Allow subcommands to customize telemetry (#1370)
- *(node/p2p)* Small Log Cleanup (#1369)
- *(node/service)* Remove .expect (#1381)
- *(node/service)* The RPC Launcher is Used (#1382)
- *(ci)* Bump monorepo commit (#1385)

## [kona-host/v0.1.0-beta.14] - 2025-03-24

### 🚀 Features

- *(examples)* Pulls Discovery and P2P Gossip into Examples (#1250)
- *(node)* P2P Wiring (#1246)
- *(node)* P2P Overhaul (#1260)
- *(engine)* Synchronous task queue (#1256)
- *(engine)* Block building task (#1258)
- *(node)* P2P Upgrades (#1271)
- *(interop)* Add utility trait method to `InteropTxValidator` (#1291)

### 🐛 Bug Fixes

- *(executor)* Use correct empty `sha256` hash (#1267)
- *(proof)* EIP-2935 walkback fix (#1273)

### ⚙️ Miscellaneous Tasks

- *(node)* Wire in Sync Config (#1249)
- *(node)* Simplify Node CLI (#1251)
- *(node)* P2P Secret Key (#1254)
- Remove B256 Value Parser (#1255)
- *(cli)* Remove CLI Parsers (#1259)
- *(workspace)* Fix udeps check (#1263)
- *(genesis)* Localize Import for Lints (#1265)
- Fixup Benchmark CI Job (#1274)
- *(ci)* Deprecate --all Flag (#1275)
- Cleanup and Dependency Bumps (#1235)
- *(workspace)* Remove `reth` dependency (#1279)
- *(ci)* Bump Monorepo Commit for Operator Fee Tests (#1277)
- Bump Deps before Release (#1288)
- Minor Crate Releases (#1289)
- *(node)* Further P2P Fixes (#1280)
- *(interop)* Remove new L1BlockInfo variant + deposit context (#1290)

### Refactor

- Clap attribute macros from #[clap(...)] to #[arg(...)] and #[command(...)] in v4.x (#1285)

## [kona-host/v0.1.0-beta.13] - 2025-03-11

### 🚀 Features

- *(node)* P2P CLI Args (#1242)

### ⚙️ Miscellaneous Tasks

- Allow udeps in `-Zbuild-std` lints (#1245)
- *(workspace)* Use versioned `asterisc-builder` + `cannon-builder` images (#1243)

## [kona-host/v0.1.0-beta.12] - 2025-03-11

### 🚀 Features

- *(proof)* EIP-2935 lookback (#1088)
- *(bin-utils)* Add prometheus server initializer (#1100)
- *(node)* Engine Controller (#1136)
- *(node)* Initial orchestration logic (#1166)
- *(registry)* Lookup `Chain` + `RollupConfig` by identifier (#1156)
- *(engine)* Version Providers (#1168)
- *(interop)* Dedup logic for parsing `Log` to `ExecutingMessage` (#1171)
- *(node)* Hook up `RollupNodeBuilder` to CLI (#1179)
- *(book)* Umbrella Crate RFC (#1063)
- *(node)* Derivation actor (#1180)
- *(engine)* Actor + Task Model (#1177)
- *(engine)* FCU Task Updates (#1191)
- *(protocol)* Update `RollupConfig` (#1170)
- *(engine)* Engine Task Cleanup + Insert Payload Task Stub (#1193)
- *(engine)* Insert Task Updates (#1194)
- *(providers-alloy)* Refactor `AlloyChainProvider` (#1203)
- *(providers-alloy)* Refactor `AlloyL2ChainProvider` (#1204)
- *(engine)* Insert New Payload (#1197)
- *(engine)* Wire up Insert Task (#1202)
- *(node)* Add `sync_start` module (#1207)
- *(interop)* Clean up interop validator RPC component (#1172)
- *(node)* Refactor orchestration (#1231)
- *(hardforks)* Isthmus Network Upgrade Transactions (#1080)
- *(node)* P2P Wiring (#1233)

### 🐛 Bug Fixes

- *(genesis)* System Config Tests (#1090)
- *(derive)* Use `SystemConfig` batcher key for DAP (#1106)
- *(derive)* Hardfork Deps (#1151)
- 2021 Edition Fragment Specifier (#1155)
- *(ci)* Cargo Deny Checks (#1163)
- *(engine)* Engine Client (#1169)
- *(protocol)* Use `Prague` blob fee calculation for L1 info tx (#1192)
- *(protocol)* Add optional pectra blob fee schedule fork (#1195)
- *(executor)* Dep on kona-host (#1224)

### ⚙️ Miscellaneous Tasks

- *(ci)* Dependabot Label Update (#1077)
- Crate Shields (#1078)
- *(genesis)* Rename HardForkConfiguration (#1091)
- *(genesis)* Serde Test Types (#1089)
- *(genesis)* Flatten Hardforks in Rollup Config (#1092)
- *(workspace)* Bump MSRV to `1.82` (#1097)
- *(ci)* Split doc lint + doc test jobs
- *(bin)* Split up bin utilities (#1098)
- *(nexus)* Use `kona-bin-utils` (#1099)
- *(workspace)* Adjust build recipes (#1101)
- Cleanup Crate Docs (#1116)
- *(bin)* Rework Node Binary (#1120)
- *(proof-interop)* Adjust `TRANSITION_STATE_MAX_STEPS` (#1144)
- *(rpc)* Remove L2BlockRef (#1140)
- *(book)* Book Cleanup for Node Docs (#1143)
- *(book)* Maili Rename (#1145)
- *(book)* Update Protocol Crate Docs (#1146)
- *(hardforks)* Fix Alloy Reference (#1147)
- *(book)* Cleanup Protocol Docs (#1149)
- *(interop)* Replace Interop Feat Flag (#1150)
- *(workspace)* Bump `rustc` edition to 2024 (#1152)
- *(host)* Replace anyhow with thiserror (#1093)
- *(node)* Move Engine into Crate (#1164)
- *(engine)* Sync Types (#1167)
- *(engine)* Fixup EngineClient (#1173)
- *(workspace)* Updates op-alloy Dependencies (#1174)
- *(engine)* Remove pub mod Visibility Idents (#1175)
- *(workspace)* Move `external` crates to `node` (#1182)
- *(book)* Teeny Update (#1184)
- *(executor)* Fix comments in EIP-2935 syscall module (#1181)
- *(docs)* Update `README.md` (#1186)
- *(registry)* Remove Default Hasher (#1185)
- *(preimage)* Add labels to `README.md` (#1187)
- *(executor)* Add labels to `README.md` (#1188)
- *(host)* Update `README.md` (#1189)
- *(workspace)* Update `README.md` (#1190)
- *(node)* Simplify L1 watcher (#1196)
- Scr updates (#1199)
- Bump op-alloy Deps (#1205)
- Cleanup Other Deps (#1206)
- *(protocol)* RPC Block -> L2BlockInfo (#1176)
- Fix Deny Config (#1212)
- *(node-rpc)* Delete dead code (#1213)
- *(protocol)* Update Sepolia-only fork to activate on L1 blocktime (#1210)
- *(rpc)* Rename `RollupNode` -> `RollupNodeApi`, export (#1215)
- Bump alloy 0.12 (#1208)
- *(genesis)* Update `SystemConfig` ser (#1217)
- *(net)* P2P Rename (#1221)
- Update Dependencies (#1226)
- Codecov Config (#1225)
- *(book)* Small touchups (#1230)
- *(node)* Tracing Macros (#1234)

### Release

- *(maili)* 0.2.9 (#1087)
- Maili crates one last time (#1218)
- Kona-driver (#1229)

## [kona-host/v0.1.0-beta.11] - 2025-02-21

### 🚀 Features

- *(genesis)* Deny Unknown Fields (#1060)

### 🐛 Bug Fixes

- *(registry)* Use `superchain-registry` as a submodule (#1075)
- *(workspace)* Exclude Maili Shadows (#1076)

### ⚙️ Miscellaneous Tasks

- *(workspace)* Foundry Install Target (#1074)

## [kona-host/v0.1.0-beta.10] - 2025-02-21

### 🐛 Bug Fixes

- Maili Shadows (#1071)
- Remove Maili Shadows from Workspace (#1072)

### 📚 Documentation

- Release Guide (#1067)

### ⚙️ Miscellaneous Tasks

- *(ci)* Remove Release Plz (#1068)

## [kona-hardforks-v0.1.0] - 2025-02-21

### 🚀 Features

- *(protocol)* Introduce Hardforks Crate (#1065)

## [kona-nexus-v0.1.0] - 2025-02-20

### 🚀 Features

- *(bin)* Network Component Runner (#1058)

## [kona-rpc-v0.1.0] - 2025-02-20

### 🐛 Bug Fixes

- *(ci)* Submodule Sync Crate Path (#1061)

### ⚙️ Miscellaneous Tasks

- *(book)* Move the Monorepo Doc to Archives (#1062)

### Release

- *(kona-interop)* 0.1.2 (#1066)

## [kona-serde-v0.1.0] - 2025-02-20

### 🚀 Features

- *(client)* Support cannon mips64r1 (#1054)
- *(client)* Wire up `L2PayloadWitness` hint for single-chain proof (#1034)
- Kona Optimism Monorepo (#1055)

### 🐛 Bug Fixes

- Fix type annotations (#1050)
- Exclude kona-net (#1049)
- *(docker)* `mips64` target data layout (#1056)
- *(std-fpvm)* Allow non-const fn with mut ref (#1057)

### ⚙️ Miscellaneous Tasks

- Monorepo Proposal Doc (#1036)
- *(book)* RFC and Archives Section (#1053)

## [kona-net-v0.1.0] - 2025-02-13

### ⚙️ Miscellaneous Tasks

- Bump Dependencies (#1029)
- *(interop)* Remove horizon timestamp (#1028)
- Restructure Kona to be more Extensible (#1031)
- *(host)* Expose private SingleChainHost methods (#1030)
- *(services)* Networking Crate (#1032)

## [kona-host/v0.1.0-beta.9] - 2025-02-11

### 🚀 Features

- Derive Eq/Ord/Hash for (Archived) PreimageKey(Type) (#956)
- Allow 7702 receipts after Isthmus active (#959)
- Fill eip 7702 tx env with auth list (#958)
- *(executor)* EIP-2935 Syscall Support [ISTHMUS] (#963)
- *(executor)* EIP-7002 Syscall Support [ISTHMUS] (#965)
- *(executor)* EIP-7251 Syscall Support [ISTHMUS] (#968)
- *(executor)* Export receipts (#969)
- *(client)* EIP-2537 BLS12-381 Curve Precompile Acceleration (#960)
- *(host)* Interop optimistic block re-execution hint (#983)
- *(proof-interop)* Support multiple `RollupConfigs` in boot routine (#986)
- *(host)* Re-export default CLIs (#992)
- *(proof-sdk)* Cleanup `Hint` API (#998)
- *(proof-sdk)* Optional L2 chain ID in L2-specific hints (#999)
- *(mpt)* Copy-on-hash (#1001)
- *(host)* Reintroduce `L2BlockData` hint (#1003)
- *(client)* Superchain Consolidation (#1004)
- *(ci)* Coverage for action tests (#1005)
- *(host)* Accelerate all BLS12-381 Precompiles (#1010)
- *(executor)* Sort trie keys (#1016)
- *(host)* Proactive hints (#1017)
- *(ci)* Remove support for features after MSRV (#1018)
- *(interop)* Support full timestamp invariant (#1022)
- Isthmus upgrade txs (#1025)

### 🐛 Bug Fixes

- *(executor)* Don't generate a diff when running tests (#967)
- *(executor)* Withdrawals root (#974)
- *(client)* Interop transition rules (#973)
- *(executor)* Removes EIP-7002 and EIP-7251 Pre-block Calls (#990)
- *(ci)* Action tests (#997)
- *(client)* Interop bugfixes (#1006)
- *(client)* No-op sub-transitions in Superchain STF (#1011)
- *(interop)* Check timestamp invariant against executing timestamp AND horizon timestamp (#1024)

### ⚙️ Miscellaneous Tasks

- *(docs)* Add `kailua` to the README (#955)
- Maili 0.1.9 (#964)
- *(executor)* Update SpecId with Isthmus (#962)
- *(executor)* TxEnv Stuffing (#970)
- *(executor)* De-duplicate `TrieAccount` type (#977)
- Dep Updates (#980)
- Update Maili Deps (#978)
- Update Dependencies (#988)
- *(host)* Remove `HostOrchestrator` (#994)
- Bump op-alloy dep (#996)
- *(host)* Refactor fetchers (#995)
- Maili Dependency Update (#1007)
- *(client)* Dedup MSM Required Gas Fn (#1012)
- *(client)* Precompile Run Macro (#1014)
- *(ci)* Bump `codecov-action` to v5 (#1020)
- Use Updated Maili and op-alloy Deps (#1023)
- *(book)* Adherence to devdocs (#1026)
- *(book)* Devdocs subdirectory (#1027)

## [kona-providers-alloy-v0.1.0] - 2025-01-26

### 🚀 Features

- Use empty requests hash when isthmus enabled (#951)
- *(workspace)* Re-introduce `kona-providers-alloy` (#954)

### ⚙️ Miscellaneous Tasks

- *(ci)* Improve docker releases (#952)

## [kona-client-v0.1.0-beta.8] - 2025-01-24

### 🚀 Features

- *(driver)* Multi-block derivation (#888)
- *(host)* Interop proof support (part 1) (#910)
- *(client)* Interop consolidation sub-problem (#913)
- *(host)* Modular components (#915)
- *(executor)* New static test harness (#938)
- *(build)* Migrate to `mips64r2` target for `cannon` (#943)

### 🐛 Bug Fixes

- *(ci)* Codecov (#911)

### ⚙️ Miscellaneous Tasks

- *(mpt)* Remove `anyhow` dev-dependency (#919)
- *(executor)* Remove `anyhow` dev-dependency (#937)

## [kona-proof-v0.2.3] - 2025-01-16

### 🚀 Features

- *(client)* Interop binary (#903)
- *(host)* Support multiple modes (#904)

### ⚙️ Miscellaneous Tasks

- Fix some typos in comment (#906)
- Update Maili Deps (#908)
- Release (#900)

## [kona-proof-interop-v0.1.0] - 2025-01-14

### 🚀 Features

- *(workspace)* `kona-proof-interop` crate (#902)

## [kona-interop-v0.1.0] - 2025-01-13

### 🚀 Features

- *(workspace)* `kona-interop` crate (#899)

## [kona-proof-v0.2.2] - 2025-01-13

### 🐛 Bug Fixes

- Small Spelling Issue (#893)

### 📚 Documentation

- Edited the link in the documentation (#895)

### ⚙️ Miscellaneous Tasks

- Release v0.2.2 (#891)

## [kona-client-v0.1.0-beta.7] - 2025-01-09

### ⚙️ Miscellaneous Tasks

- Remove unused function in OnlineBlobProvider (#875)
- *(derive)* Test Ignoring EIP-7702 (#887)
- Bump Maili (#894)

## [kona-std-fpvm-v0.1.2] - 2025-01-07

### 🐛 Bug Fixes

- Op-rs rename (#883)

### ⚙️ Miscellaneous Tasks

- Isthmus Withdrawals Root (#881)
- Remove redundant words in comment (#882)
- Add emhane as a codeowner (#884)
- Bump Dependencies (#880)
- Release (#885)

## [kona-client-v0.1.0-beta.6] - 2025-01-02

### 🚀 Features

- *(build)* Adjust RV target - `riscv64g` -> `riscv64ima` (#868)
- *(build)* Bump `asterisc-builder` version (#879)

### 🐛 Bug Fixes

- *(derive)* Make tests compile (#878)
- *(derive)* `BatchStream` Past batch handling (#876)

### ⚙️ Miscellaneous Tasks

- Bump alloy 0.8 (#870)

### Tooling

- Make client justfile's commands take an optional rollup_config_path (#869)

## [kona-client-v0.1.0-beta.5] - 2024-12-04

### 🚀 Features

- *(client)* Re-accelerate precompiles (#866)

## [kona-std-fpvm-v0.1.1] - 2024-12-04

### ⚙️ Miscellaneous Tasks

- Release (#837)

## [kona-client-v0.1.0-beta.4] - 2024-12-03

### 🐛 Bug Fixes

- Bump (#855)
- Bump (#865)

### ⚙️ Miscellaneous Tasks

- *(ci)* Distribute `linux/arm64` `kona-fpp` image (#860)
- Bump Other Dependencies (#856)
- Update deps and clean up misc features (#864)

## [kona-client-v0.1.0-beta.3] - 2024-12-02

### 🚀 Features

- *(workspace)* Bump MSRV (#859)

### 🐛 Bug Fixes

- Nightly lint (#858)

## [kona-client-v0.1.0-beta.2] - 2024-11-28

### 🚀 Features

- *(host)* Delete unused blob providers (#842)
- *(driver)* Refines the executor interface for the driver (#850)
- *(client)* Invalidate impossibly old claims (#852)
- *(driver)* Wait for engine (#851)

### 🐛 Bug Fixes

- Use non problematic hashmap fns (#853)

### ⚙️ Miscellaneous Tasks

- *(derive)* Remove indexed blob hash (#847)
- *(driver)* Advance with optional target (#848)
- *(host)* Hint Parsing Cleanup (#844)

## [kona-std-fpvm-v0.1.0] - 2024-11-26

### 🚀 Features

- *(workspace)* Isolate FPVM-specific platform code (#821)

### ⚙️ Miscellaneous Tasks

- *(driver)* Visibility (#834)

## [kona-proof-v0.1.0] - 2024-11-20

### ⚙️ Miscellaneous Tasks

- Minor release' (#833)

## [kona-proof-v0.0.1] - 2024-11-20

### 🚀 Features

- *(driver)* Abstract, Default Pipeline (#796)
- *(driver,client)* Pipeline Cursor Refactor (#798)
- *(mpt)* Extend `TrieProvider` in `kona-executor` (#813)
- *(preimage)* Decouple from `kona-common` (#817)
- *(workspace)* `kona-proof` (#818)

### 🐛 Bug Fixes

- *(client)* SyncStart Refactor (#797)
- Mdbook version (#810)
- *(mpt)* Remove unused collapse (#808)
- Imports (#829)

### 📚 Documentation

- Update providers.md to use new next method instead of old open_data (#809)
- Fix typo in custom-backend.md (#825)

### ⚙️ Miscellaneous Tasks

- *(ci)* Bump monorepo commit (#805)
- Dispatch book build without cache (#807)
- *(workspace)* Migrate back to `thiserror` v2 (#811)
- *(common)* Rename IO modules (#812)
- *(workspace)* Reorganize SDK (#816)
- V0.6.6 op-alloy (#804)
- *(driver)* Use tracing macros (#822)
- *(driver)* Use tracing macros (#823)
- Op-alloy 0.6.8 (#830)
- *(derive)* Remove batch reader (#826)

## [kona-driver-v0.0.0] - 2024-11-08

### 🚀 Features

- *(driver)* Introduce driver crate (#794)

### 🐛 Bug Fixes

- Remove kona-derive-alloy (#789)

### ⚙️ Miscellaneous Tasks

- *(derive)* Re-export types (#790)

## [kona-mpt-v0.0.6] - 2024-11-06

### 🚀 Features

- *(TrieProvider)* Abstract TrieNode retrieval (#787)

### 🐛 Bug Fixes

- *(derive)* Hoist types out of traits (#781)
- *(derive)* Data Availability Provider Abstraction (#782)
- *(derive-alloy)* Test coverage (#785)

### ⚙️ Miscellaneous Tasks

- Clean codecov confiv (#783)
- *(derive)* Pipeline error test coverage (#784)
- Bump alloy deps (#788)
- Release (#753)

## [kona-client-v0.1.0-alpha.7] - 2024-11-05

### 🚀 Features

- *(derive)* Sources docs (#754)
- Flush oracle cache on reorg #724 (#756)
- *(docs)* Derivation Docs (#768)
- *(client)* Remove `anyhow` (#779)
- *(derive)* `From<BlobProviderError> for PipelineErrorKind` (#780)

### 🐛 Bug Fixes

- *(derive-alloy)* Changelog (#752)
- Update monorepo (#761)
- *(derive)* Use signal value updated with system config. (#776)
- *(client)* Trace extension support (#778)

### ⚙️ Miscellaneous Tasks

- *(ci)* Use `gotestsum` for action tests (#751)
- *(derive)* Cleanup Exports (#757)
- *(derive)* Error Exports (#758)
- *(derive)* Touchup kona-derive readme (#762)
- *(derive-alloy)* Docs (#763)
- *(executor)* Rm upstream util (#755)
- *(ci)* Use `PAT_TOKEN` for automated monorepo pin update (#773)
- *(workspace)* Bump `asterisc` version (#774)
- *(ci)* Update monorepo pin to include Holocene action tests (#775)

## [kona-mpt-v0.0.5] - 2024-10-29

### 🚀 Features

- *(derive)* Remove metrics (#743)
- Update op-alloy (#745)
- *(derive)* Use upstream op-alloy batch types (#746)

### 🐛 Bug Fixes

- Tracing_subscriber problem in `kona-derive` tests (#741)
- *(client)* Don't shadow `executor` in engine retry (#750)

### ⚙️ Miscellaneous Tasks

- *(derive)* Import hygiene (#744)
- *(ci)* Don't run `online` tests in CI (#747)
- *(derive-alloy)* Remove metrics (#748)
- Release (#749)

## [kona-client-v0.1.0-alpha.6] - 2024-10-28

### 🚀 Features

- *(ci)* Bump `go` version for action tests (#730)
- Remove thiserror (#735)
- *(derive)* Sys config accessor (#722)
- *(host)* Remove `MAX_RETRIES` (#739)
- *(host)* Ensure prefetch is falliable (#740)

### 🐛 Bug Fixes

- Hashmap (#732)
- *(derive)* Holocene action tests / fixes (#733)
- Add feature for `alloy-provider`, fix `test_util` (#738)

### ⚙️ Miscellaneous Tasks

- *(workspace)* Update `asterisc` version to `1.0.3-alpha1` (#729)
- Bump op-alloy version (#731)
- Release (#715)
- *(kona-derive-alloy)* Release v0.0.1 (#736)

### Docs

- Update README (#734)

## [kona-client-v0.1.0-alpha.5] - 2024-10-22

### 🚀 Features

- *(derive)* BatchQueue Update [Holocene] (#601)
- *(derive)* Add `Signal` API (#611)
- *(derive)* Holocene flush signal (#612)
- Frame queue tests (#613)
- *(client)* Pass flush signal (#615)
- *(executor)* Use EIP-1559 parameters from payload attributes (#616)
- *(trusted-sync)* Holocene flush (#617)
- *(primitives)* Blob Test Coverage (#627)
- *(executor)* Update EIP-1559 configurability (#648)
- Codecov Shield (#652)
- Codecov sources (#657)
- Use derive more display (#675)
- *(derive)* `Past` batch validity variant (#684)
- *(derive)* Stage multiplexer (#693)
- *(derive)* Signal receiver logic (#696)
- *(derive)* Add `ChannelAssembler` size limitation (#700)
- Codecov bump threshold to 90 (#674)
- *(executor)* EIP-1559 configurability spec updates (#716)
- *(derive)* `BatchValidator` stage (#703)
- *(workspace)* Distribute pipeline, not providers (#717)
- *(executor)* Clean ups (#719)
- Frame queue test asserter (#619)
- *(derive)* Hoist stage traits (#723)
- *(derive)* `BatchProvider` multiplexed stage (#726)
- *(docker)* Update asterisc reproducible build image (#728)

### 🐛 Bug Fixes

- *(ci)* Action tests (#608)
- *(executor)* Holocene EIP-1559 params in Header (#622)
- *(codecov)* Ignore Test Utilities (#628)
- Add codeowners (#635)
- *(providers)* Remove slot derivation (#636)
- *(derive)* Remove unused online mod (#637)
- Codecov (#656)
- *(derive)* Retain L1 blocks (#683)
- Typos (#690)
- *(derive)* Holocene `SpanBatch` prefix checks (#688)
- *(derive)* SpanBatch element limit + channel RLP size limit (#692)
- *(mpt)* Empty root node case (#705)
- *(providers-alloy)* Recycle Beacon Types (#713)

### ⚙️ Miscellaneous Tasks

- Doc logos (#609)
- Delete `trusted-sync` (#621)
- Refactor test providers (#623)
- Add Test Coverage (#625)
- Test coverage for common (#629)
- *(derive)* Blob Source Test Coverage (#631)
- *(providers)* Codecov Ignore Alloy-backed Providers (#633)
- *(preimage)* Test Coverage (#634)
- Update deps (#610)
- *(derive)* Single Batch Test Coverage (#643)
- *(derive)* Pipeline Core Test Coverage (#642)
- *(providers-alloy)* Blob provider fallback tests (#644)
- *(mpt)* Account conversion tests (#647)
- *(mpt)* Mpt noop trait impls (#649)
- *(providers)* Add changelog (#653)
- *(derive)* Hoist attributes queue test utils (#654)
- *(mpt)* Codecov (#655)
- *(derive)* Test channel bank reset (#658)
- *(derive)* Test channel reader resets (#660)
- *(derive)* Adds more channel bank coverage (#659)
- *(derive)* Test channel reader flushing (#661)
- *(executor)* Test Coverage over Executor Utilities (#650)
- *(derive)* Batch Timestamp Tests (#664)
- *(client)* Improve `BootInfo` field names (#665)
- *(host)* Improve CLI flag naming (#666)
- *(derive)* Test Stage Resets and Flushes (#669)
- *(derive)* Test and Clean Batch Types (#670)
- *(ci)* Reduce monorepo auto-update frequency (#671)
- *(host)* Support environment variables for `kona-host` flags (#667)
- *(executor)* Use Upstreamed op-alloy Methods  (#651)
- *(derive)* Stage coverage (#673)
- *(executor)* Cover Builder (#676)
- *(executor)* Move todo to issue: (#680)
- *(derive)* Remove span batch todo comments (#682)
- *(providers-alloy)* Changelog (#685)
- Remove todos (#687)
- *(host)* Reduce disk<->mem KV proptest runs (#689)
- *(workspace)* Update dependencies + fix build (#702)
- *(derive)* Add tracing to `ChannelAssembler` (#701)
- *(workspace)* Removes Primitives (#638)
- Remove version types (#707)
- Hoist trait test utilities (#708)
- Erradicate anyhow (#712)
- Re-org imports (#711)
- Bump alloy dep minor (#718)

## [kona-providers-v0.0.1] - 2024-10-02

### 🚀 Features

- Large dependency update (#528)
- *(primitives)* Remove Attributes (#529)
- *(host)* Exit with client status in native mode (#530)
- *(workspace)* Action test runner (#531)
- *(ci)* Add action tests to CI (#533)
- Remove crates.io patch (#537)
- *(derive)* Typed error handling (#540)
- *(mpt)* Migrate to `thiserror` (#541)
- *(preimage/common)* Migrate to `thiserror` (#543)
- *(executor)* Migrate to `thiserror` (#544)
- *(book)* Custom backend, `kona-executor` extensions, and FPVM backend (#552)
- Remove L2 Execution Payload (#542)
- *(derive)* Latest BN (#521)
- *(derive)* Touchup Docs (#555)
- *(derive)* Hoist AttributesBuilder (#571)
- *(derive)* New BatchStream Stage for Holocene (#566)
- *(derive)* Wire up the batch span stage (#567)
- *(derive)* Holocene Activation (#574)
- *(derive)* Holocene Frame Queue (#579)
- *(derive)* Holocene Channel Bank Checks (#572)
- *(derive)* Holocene Buffer Flushing (#575)
- *(ci)* Split online/offline tests (#582)
- *(derive)* Interleaved channel tests (#585)
- *(derive)* Refactor out Online Providers (#569)
- *(derive)* BatchStreamProvider (#591)
- *(derive)* `BatchStream` buffering (#590)
- *(derive)* Span batch prefix checks (#592)
- Kona-providers (#596)
- Monorepo Pin Update (#604)
- *(derive)* Bump op-alloy dep (#605)

### 🐛 Bug Fixes

- *(derive)* Sequence window expiry (#532)
- *(preimage)* Improve error differentiation in preimage servers (#535)
- *(client)* Channel reader error handling (#539)
- *(client)* Continue derivation on execution failure (#545)
- *(derive)* Move attributes builder trait (#570)
- *(workspace)* Hoist and fix lints (#577)
- Derive pipeline params (#587)

### ⚙️ Miscellaneous Tasks

- *(host)* Make `l2-chain-id` optional if a rollup config was passed. (#534)
- *(host)* Clean up CLI (#538)
- *(workspace)* Bump MSRV to `1.81` (#546)
- *(ci)* Delete program diff job (#547)
- *(workspace)* Allow stdlib in `cfg(test)` (#548)
- *(workspace)* Bump dependencies (#550)
- *(readme)* Remove `kona-plasma` link (#551)
- Channel reader docs (#568)
- *(workspace)* `just lint` (#584)
- *(derive)* [Holocene] Drain previous channel in one iteration (#583)
- Use alloy primitives map (#586)
- *(ci)* Pin action tests monorepo rev (#603)

## [kona-client-v0.1.0-alpha.3] - 2024-09-10

### 🚀 Features

- *(host)* Add `TryFrom<DiskKeyValueStore>` for `MemoryKeyValueStore` (#512)
- Expose store (#513)
- *(ci)* Release prestate build image (#523)

### ⚙️ Miscellaneous Tasks

- *(primitives)* Rm RawTransaction (#505)
- Bumps Dependency Versions (#520)
- *(release)* Default to `amd64` platform on prestate artifacts build (#519)

## [kona-client-v0.1.0-alpha.2] - 2024-09-06

### 🚀 Features

- *(host)* Use `RocksDB` as the disk K/V store (#471)
- *(primitives)* Reuse op-alloy-protocol channel and block types (#499)

### 🐛 Bug Fixes

- *(primitives)* Re-use op-alloy frame type (#492)
- *(mpt)* Empty list walker (#493)
- *(ci)* Remove `PAT_TOKEN` ref (#494)
- *(primitives)* Use consensus hardforks (#497)

### ⚙️ Miscellaneous Tasks

- *(docker)* Update prestate builder image (#502)

## [kona-primitives-v0.0.2] - 2024-09-04

### 🚀 Features

- Increase granularity (#365)
- *(examples)* Log payload attributes on error (#371)
- *(examples)* Add metric for latest l2 reference safe head update (#375)
- *(trusted-sync)* Re-org walkback (#379)
- *(client)* Providers generic over oracles (#336)
- Add zkvm target for io (#394)
- *(derive+trusted-sync)* Online blob provider with fallback (#410)
- *(client)* Generic DerivationDriver over any BlobProvider (#412)
- *(ci)* Add scheduled FPP differential tests (#408)
- *(kdn)* Derivation Test Runner for kona-derive (#414)
- *(client+host)* Dynamic `RollupConfig` in bootloader (#439)
- *(kt)* `kdn` -> `kt`, prep for multiple test formats (#445)
- *(client)* Export `CachingOracle` (#455)
- *(primitives)* `serde` for `L1BlockInfoTx` (#460)
- Update superchain registry deps (#463)
- *(workspace)* Workspace Re-exports (#468)
- *(executor)* Expose full revm Handler (#475)
- *(client)* Granite `ecPairing` precompile limit (#479)
- Run cargo hack against workspace (#485)

### 🐛 Bug Fixes

- Trusted-sync metrics url (#363)
- Docker image metrics url set (#364)
- *(examples)* L2 safe head tracking (#373)
- *(examples)* Reduce Origin Advance to Warn (#372)
- *(actions)* Trusted sync docker publish (#376)
- Drift reset (#381)
- Drift Walkback (#382)
- *(derive)* Pipeline Reset (#383)
- Bubble up validation errors (#388)
- Pin two dependencies due to upstream semver issues (#391)
- Don't hold onto intermediate execution cache across block boundaries (#396)
- *(kona-derive)* Remove SignedRecoverable Shim (#400)
- *(deps)* Bump Alloy Dependencies (#409)
- Remove data iter option (#405)
- *(examples)* Backoff trusted-sync invalid payload retries (#411)
- *(trusted-sync)* Remove Panics (#413)
- *(kona-host)* Set explicit types (#421)
- *(derive)* Granite Hardfork Support (#420)
- *(host)* Backoff after `MAX_RETRIES` (#429)
- Fix superchain registry + primitives versions (#425)
- Broken link in readme (#432)
- Link to section (#419)
- *(kdn)* Update with Repository Rename (#441)
- *(kdn)* Updates `kdn` with op-test-vectors Generic Typing (#444)
- *(client)* Bootinfo serde (#448)
- *(derive)* Remove fpvm tests (#447)
- *(workspace)* Add Unused Dependency Lint (#453)
- Downgrade for release plz (#458)
- *(workspace)* Use published `revm` version (#459)
- *(client)* Walkback Channel Timeout (#456)
- *(client)* Break when the pipeline cannot advance (#478)
- Deprecate --all (#484)
- *(host)* Insert empty MPT root hash (#483)
- *(examples)* Revm Features (#482)

### 🧪 Testing

- *(derive)* Channel timeout (#437)

### ⚙️ Miscellaneous Tasks

- *(derive)* Refine channel frame count buckets (#378)
- *(common)* Remove need for cursors in `NativeIO` (#416)
- *(examples)* Add logs to trusted-sync (#415)
- *(derive)* Remove previous stage trait (#423)
- *(workspace)* Remove `minimal` and `simple-revm` examples (#430)
- *(client)* Ensure p256 precompile activation (#431)
- *(client)* Isolate FPVM-specific constructs (#435)
- *(common-proc)* Suppress doc warning (#436)
- *(host)* Remove TODOs (#438)
- Bump scr version (#440)
- *(workspace)* Remove `kona-plasma` (#443)
- Refactor types out of kona-derive (#454)
- *(bin)* Remove `kt` (#461)
- *(derive)* Remove udeps (#462)
- *(derive)* Reset docs (#464)
- *(workspace)* Reorg Workspace Manifest (#465)
- *(workspace)* Hoist Dependencies (#466)
- *(workspace)* Update for `anton-rs` org transfer (#474)
- *(workspace)* Fix `default-features` in workspace root (#472)
- *(workspace)* Alloy Version Bumps (#467)
- *(ci)* Configure codecov patch job (#477)
- Release (#476)

## [kona-client-v0.1.0-alpha.1] - 2024-07-09

### 🚀 Features

- *(examples)* Trusted Sync Metrics (#308)
- *(derive)* Stage Level Metrics (#309)
- *(build)* Dockerize trusted-sync (#299)
- *(examples)* Pipeline step metrics (#320)
- *(examples)* Send Logs to Loki (#321)
- *(derive)* Granular Provider Metrics (#325)
- *(derive)* More stage metrics (#326)
- *(derive)* Track the current channel size (#331)
- *(derive)* Histogram for number of channels for given frame counts (#337)
- *(executor)* Builder pattern for `StatelessL2BlockExecutor` (#339)
- *(executor)* Generic precompile overrides (#340)
- *(client)* `ecrecover` accelerated precompile (#342)
- *(client)* `ecpairing` accelerated precompile (#343)
- *(client)* KZG point evaluation accelerated precompile (#344)
- *(executor)* `StatelessL2BlockExecutor` benchmarks (#350)
- *(docker)* Reproducible `asterisc` prestate (#357)
- *(ci)* Run Host + Client natively in offline mode (#355)
- *(mpt)* `TrieNode` benchmarks (#351)
- *(ci)* Build benchmarks in CI (#352)

### 🐛 Bug Fixes

- Publish trusted-sync to GHCR (#312)
- *(ci)* Publish trusted sync docker (#314)
- *(derive)* Warnings with metrics macro (#322)
- *(examples)* Small cli fix (#323)
- *(examples)* Don't panic on validation fetch failure (#327)
- *(derive)* Prefix all metric names (#330)
- *(derive)* Bind the Pipeline trait to Iterator (#334)
- *(examples)* Reset Failed Payload Derivation Metric (#338)
- *(examples)* Justfile fixes (#341)
- *(derive)* Unused var w/o `metrics` feature (#345)
- *(examples)* Dockerfile fixes (#347)
- *(examples)* Start N Blocks Back from Tip (#349)

### ⚙️ Miscellaneous Tasks

- *(client)* Improve justfile (#305)
- *(derive)* Add targets to stage logs (#310)
- *(docs)* Label Cleanup (#307)
- Bump `superchain-registry` version (#306)
- *(derive)* Remove noisy batch logs (#329)
- *(preimage)* Remove dynamic dispatch (#354)
- *(host)* Make `exec` flag optional (#356)
- *(docker)* Pin `asterisc-builder` version in reproducible prestate builder (#362)

## [kona-primitives-v0.0.1] - 2024-06-22

### ⚙️ Miscellaneous Tasks

- Pin op-alloy-consensus (#304)

## [kona-common-v0.0.2] - 2024-06-22

### 🚀 Features

- *(precompile)* Add `precompile` key type (#179)
- *(preimage)* Async server components (#183)
- *(host)* Host program scaffold (#184)
- *(host)* Disk backed KV store (#185)
- *(workspace)* Add aliases in root `justfile` (#191)
- *(host)* Add local key value store (#189)
- *(preimage)* Async client handles (#200)
- *(mpt)* Trie node insertion (#195)
- *(mpt)* Trie DB commit (#196)
- *(mpt)* Simplify `TrieDB` (#198)
- *(book)* Add minimal program stage documentation (#202)
- *(mpt)* Block hash walkback (#199)
- Refactor reset provider (#207)
- Refactor the pipeline builder (#209)
- *(client)* `BootInfo` (#205)
- Minimal ResetProvider Implementation (#208)
- Pipeline Builder (#217)
- *(client)* `StatelessL2BlockExecutor` (#210)
- *(ci)* Add codecov (#233)
- *(ci)* Dependabot config (#236)
- *(kona-derive)* Updated interface (#230)
- *(client)* Add `current_output_root` to block executor (#225)
- *(client)* Account + Account storage hinting in `TrieDB` (#228)
- *(plasma)* Online Plasma Input Fetcher (#167)
- *(host)* More hint routes (#232)
- *(kona-derive)* Towards Derivation (#243)
- *(client)* Add `RollupConfig` to `BootInfo` (#251)
- *(client)* Oracle-backed derive traits (#252)
- *(client/host)* Oracle-backed Blob fetcher (#255)
- *(client)* Derivation integration (#257)
- *(preimage)* Add serde feature flag to preimage crate for keys (#271)
- *(fjord)* Fjord parameter changes (#284)

### 🐛 Bug Fixes

- *(ci)* Run CI on `pull_request` and `merge_group` triggers (#186)
- *(primitives)* Use decode_2718() to gracefully handle the tx type (#182)
- Strong Error Typing (#187)
- *(readme)* CI badges (#190)
- *(host)* Blocking native client program (#201)
- *(derive)* Alloy EIP4844 Blob Type (#215)
- Derivation Pipeline (#220)
- Use 2718 encoding (#231)
- *(ci)* Do not run coverage in merge queue (#239)
- *(kona-derive)* Reuse upstream reqwest provider (#229)
- Output root version to 32 bytes (#248)
- *(examples)* Clean up trusted sync logging (#263)
- Type re-exports (#280)
- *(common)* Pipe IO support (#282)
- *(examples)* Dynamic Rollup Config Loading (#293)
- Example dep feature (#297)
- *(derive)* Fjord brotli decompression (#298)
- *(mpt)* Fix extension node truncation (#300)

### ⚙️ Miscellaneous Tasks

- *(common)* Use `Box::leak` rather than `mem::forget` (#180)
- *(derive)* Data source unit tests (#181)
- *(ci)* Workflow trigger changes (#203)
- *(mpt)* Do not expose recursion vars (#197)
- Use alloy withdrawal type (#213)
- *(host)* Simplify host program (#206)
- Update README (#227)
- *(kona-derive)* Online Pipeline Cleanup (#241)
- *(workspace)* `kona-executor` (#259)
- *(derive)* Sources Touchups (#266)
- *(derive)* Online module touchups (#265)
- *(derive)* Cleanup pipeline tracing (#264)
- Update `README.md` (#269)
- Re-export input types (#279)
- *(client)* Add justfile for running client program (#283)
- *(ci)* Remove codecov from binaries (#285)
- Payload decoding tests (#289)
- Payload decoding tests (#287)
- *(workspace)* Reorganize binary example programs (#294)
- Version dependencies (#296)
- *(workspace)* Prep release (#301)
- Release (#302)

## [kona-common-proc-v0.0.1] - 2024-05-23

### 🚀 Features

- *(primitives)* Kona-derive type refactor (#135)
- *(derive)* Pipeline Builder (#127)
- *(plasma)* Implements Plasma Support for kona derive (#152)
- *(derive)* Online Data Source Factory Wiring (#150)
- *(derive)* Abstract Alt DA out of `kona-derive` (#156)
- *(derive)* Return the concrete online attributes queue type from the online stack constructor (#158)
- *(primitives)* Move attributes into primitives (#163)
- *(mpt)* Refactor `TrieNode` (#172)
- *(mpt)* `TrieNode` retrieval (#173)
- *(mpt)* `TrieCacheDB` scaffold (#174)
- *(workspace)* Client programs in workspace (#178)

### 🐛 Bug Fixes

- *(workspace)* Release plz (#138)
- *(derive)* Small Fixes and Span Batch Validation Fix (#139)
- *(derive)* Move span batch conversion to try from trait (#142)
- *(ci)* Release plz (#145)
- *(derive)* Remove unnecessary online feature decorator (#160)
- *(plasma)* Reduce plasma source generic verbosity (#165)
- *(plasma)* Plasma Data Source Cleanup (#164)
- *(derive)* Ethereum Data Source (#159)
- *(derive)* Fix span batch utils read_tx_data() (#170)
- *(derive)* Inline blob verification into the blob provider (#175)

### ⚙️ Miscellaneous Tasks

- *(workspace)* Exclude all crates except `kona-common` from cannon/asterisc lint job (#168)
- *(host)* Split CLI utilities out from binary (#169)

## [kona-mpt-v0.0.1] - 2024-04-24

### 🚀 Features

- L1 traversal (#39)
- Add `TxDeposit` type (#40)
- Add OP receipt fields (#41)
- System config update event parsing (#42)
- L1 retrieval (#44)
- Frame queue stage (#45)
- *(derive)* Channel bank (#46)
- Single batch type (#43)
- Data sources
- Clean up data sources to use concrete bytes type
- Async iterator and cleanup
- Fix async iterator issue
- *(derive)* Most of blob data source impl
- *(derive)* Fill blob pointers
- *(derive)* Blob decoding
- *(derive)* Test Utilities (#62)
- *(derive)* Share the rollup config across stages using an arc
- *(ci)* Add workflow to cycle issues (#73)
- *(derive)* Channel Reader Implementation (#65)
- *(types)* Span batches
- *(derive)* Raw span type refactoring
- *(derive)* Fixed bytes and encoding
- *(derive)* Refactor serialization; `SpanBatchPayload` WIP
- *(derive)* Derive raw batches, mocks
- *(derive)* `add_txs` function
- *(derive)* Reorganize modules
- *(derive)* `SpanBatch` type implementation WIP
- *(workspace)* Add `rustfmt.toml`
- *(derive)* Initial pass at telemetry
- *(derive)* Add signature protection check in `SpanBatchTransactions`
- *(derive)* Batch type for the channel reader
- *(derive)* Channel reader implementation with batch reader
- *(derive)* Batch queue
- *(derive)* Basic batch queue next batch derivation
- *(derive)* Finish up batch derivation
- *(derive)* Attributes queue stage
- *(derive)* Add next_attributes test
- *(derive)* Use upstream alloy (#89)
- *(derive)* Add `ecrecover` trait + features (#90)
- *(derive)* Batch Queue Logging (#86)
- *(common)* Move from `RegisterSize` to native ptr size type (#95)
- *(preimage)* `OracleServer` + `HintReader` (#96)
- *(derive)* Move to `tracing` for telemetry (#94)
- *(derive)* Online `ChainProvider` (#93)
- *(derive)* Payload Attribute Building (#92)
- *(derive)* Add `L1BlockInfoTx` (#100)
- *(derive)* `L2ChainProvider` w/ `op-alloy-consensus` (#98)
- *(derive)* Build `L1BlockInfoTx` in payload builder (#102)
- *(derive)* Deposit derivation testing (#115)
- *(derive)* Payload builder tests (#106)
- *(derive)* Online Blob Provider (#117)
- *(derive)* Use `L2ChainProvider` for system config fetching in attributes builder (#123)
- *(derive)* Span Batch Validation (#121)
- `kona-mpt` crate (#128)

### 🐛 Bug Fixes

- *(derive)* Small l1 retrieval doc comment fix (#61)
- *(derive)* Review cleanup
- *(derive)* Vec deque
- *(derive)* Remove k256 feature
- *(derive)* Async iterator type with data sources
- Result wrapping iterator item
- *(derive)* More types
- *(derive)* Span type encodings and decodings
- *(derive)* Span batch tx rlp
- *(derive)* Bitlist alignment
- *(derive)* Refactor span batch tx types
- *(derive)* Refactor tx enveloped
- *(derive)* Data sources upstream conflicts
- *(derive)* Hoist params from types
- *(derive)* Formatting
- *(derive)* Batch type lints
- *(derive)* Channel bank impl
- *(derive)* Channel reader lints
- *(derive)* Single batch validation
- *(derive)* Merge upstream changes
- *(derive)* Rebase
- *(derive)* Frame queue error bubbling and docs
- *(derive)* Clean up frame queue docs
- *(derive)* L1 retrieval docs (#80)
- *(derive)* Frame Queue Error Bubbling and Docs (#82)
- *(derive)* Fix bricked arc stage param construction (#84)
- *(derive)* Merge upstream changes
- *(derive)* Hoist params
- *(derive)* Upstream merge
- *(derive)* Clean up the channel bank and add tests
- *(derive)* Channel bank tests
- *(derive)* Channel bank testing with spinlocked primitives
- *(derive)* Rebase
- *(derive)* Omit the engine queue stage
- *(derive)* Attributes queue
- *(derive)* Rework abstractions and attributes queue testing
- *(derive)* Error equality fixes and tests
- *(derive)* Successful payload attributes building tests
- *(derive)* Extend attributes queue unit test
- *(derive)* Impl origin provider trait across stages
- *(derive)* Lints
- *(derive)* Add back removed test
- *(derive)* Stage Decoupling (#88)
- *(derive)* Derive full `SpanBatch` in channel reader (#97)
- *(derive)* Doc Touchups and Telemetry (#105)
- *(readme)* Remove blue highlights (#116)
- *(derive)* Span batch bitlist encoding (#122)
- *(derive)* Rebase span batch validation tests (#125)
- *(workspace)* Release plz (#137)

### ⚙️ Miscellaneous Tasks

- Scaffold (#37)
- *(derive)* Clean up RLP encoding + use `TxType` rather than ints
- *(derive)* Rebase + move `alloy` module
- *(derive)* Channel reader tests + fixes, batch type fixes
- *(derive)* L1Traversal Doc and Test Cleanup (#79)
- *(derive)* Cleanups (#91)
- *(workspace)* Cleanup justfiles (#104)
- *(ci)* Fail CI on doclint failure (#101)
- *(workspace)* Move `alloy-primitives` to workspace dependencies (#103)

### Dependabot

- Upgrade mio (#63)

### Wip

- *(derive)* `RawSpanBatch` diff decoding/encoding test

## [kona-preimage-v0.0.1] - 2024-02-22

### 🐛 Bug Fixes

- Specify common version (#32)

## [kona-common-v0.0.1] - 2024-02-22

### 🚀 Features

- `release-plz` release pipeline (#27)
- `release-plz` release pipeline (#29)

<!-- generated by git-cliff -->
