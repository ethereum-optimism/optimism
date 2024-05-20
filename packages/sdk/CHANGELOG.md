# @eth-optimism/sdk

## 3.3.1

### Patch Changes

- [#10593](https://github.com/ethereum-optimism/optimism/pull/10593) [`799bc898bfb207e2ccd4b2027e3fb4db4372292b`](https://github.com/ethereum-optimism/optimism/commit/799bc898bfb207e2ccd4b2027e3fb4db4372292b) Thanks [@nitaliano](https://github.com/nitaliano)! - expose FaultDisputeGame in getOEContract

## 3.3.0

### Minor Changes

- [#9951](https://github.com/ethereum-optimism/optimism/pull/9951) [`ac5b061dfce6a9817b928a8703be9252daaeeca7`](https://github.com/ethereum-optimism/optimism/commit/ac5b061dfce6a9817b928a8703be9252daaeeca7) Thanks [@smartcontracts](https://github.com/smartcontracts)! - Updates SDK for FPAC proven withdrawals mapping.

### Patch Changes

- [#9964](https://github.com/ethereum-optimism/optimism/pull/9964) [`8241220898128e1f61064f22dcb6fdd0a5f043c3`](https://github.com/ethereum-optimism/optimism/commit/8241220898128e1f61064f22dcb6fdd0a5f043c3) Thanks [@roninjin10](https://github.com/roninjin10)! - Removed only-allow command from package.json

- [#9973](https://github.com/ethereum-optimism/optimism/pull/9973) [`87093b0e9144a4709f11c7fbd631828847d891f9`](https://github.com/ethereum-optimism/optimism/commit/87093b0e9144a4709f11c7fbd631828847d891f9) Thanks [@raffaele-oplabs](https://github.com/raffaele-oplabs)! - Added support for MODE sepolia and MODE mainnet

- [#9969](https://github.com/ethereum-optimism/optimism/pull/9969) [`372bca2257764be33797d67ddca9b53c3dd3c295`](https://github.com/ethereum-optimism/optimism/commit/372bca2257764be33797d67ddca9b53c3dd3c295) Thanks [@roninjin10](https://github.com/roninjin10)! - Fixed bug where replayable transactions would fail `finalize` if they previously were marked as errors but replayable.

- Updated dependencies [[`8241220898128e1f61064f22dcb6fdd0a5f043c3`](https://github.com/ethereum-optimism/optimism/commit/8241220898128e1f61064f22dcb6fdd0a5f043c3)]:
  - @eth-optimism/contracts-bedrock@0.17.2
  - @eth-optimism/core-utils@0.13.2

## 3.2.3

### Patch Changes

- [#9907](https://github.com/ethereum-optimism/optimism/pull/9907) [`5fe797f183e502c1c7e91fc1e74dd3cc664ba22e`](https://github.com/ethereum-optimism/optimism/commit/5fe797f183e502c1c7e91fc1e74dd3cc664ba22e) Thanks [@smartcontracts](https://github.com/smartcontracts)! - Minor optimizations and improvements to FPAC functions.

- [#9919](https://github.com/ethereum-optimism/optimism/pull/9919) [`3dc129fade77ddf9d45bb4c2ecd34360d1aa838a`](https://github.com/ethereum-optimism/optimism/commit/3dc129fade77ddf9d45bb4c2ecd34360d1aa838a) Thanks [@smartcontracts](https://github.com/smartcontracts)! - Sets the address of the DisputeGameFactory contract for OP Sepolia.

## 3.2.2

### Patch Changes

- [#9805](https://github.com/ethereum-optimism/optimism/pull/9805) [`3ccd12fe5c8c4c5a6acbf370d474ffa8db816562`](https://github.com/ethereum-optimism/optimism/commit/3ccd12fe5c8c4c5a6acbf370d474ffa8db816562) Thanks [@alecananian](https://github.com/alecananian)! - Fixed an issue where Vercel builds were failing due to the `preinstall` command.

## 3.2.1

### Patch Changes

- [#9663](https://github.com/ethereum-optimism/optimism/pull/9663) [`a1329f21f33ecafe409990964d3af7bf05a8a756`](https://github.com/ethereum-optimism/optimism/commit/a1329f21f33ecafe409990964d3af7bf05a8a756) Thanks [@smartcontracts](https://github.com/smartcontracts)! - Fixes a bug in the SDK that would sometimes cause proof submission reverts.

## 3.2.0

### Minor Changes

- [#9325](https://github.com/ethereum-optimism/optimism/pull/9325) [`44a2d9cec5f3b309b723b3e4dd8d29b5b70f1cc8`](https://github.com/ethereum-optimism/optimism/commit/44a2d9cec5f3b309b723b3e4dd8d29b5b70f1cc8) Thanks [@smartcontracts](https://github.com/smartcontracts)! - Updates the SDK to support FPAC in a backwards compatible way.

### Patch Changes

- [#9367](https://github.com/ethereum-optimism/optimism/pull/9367) [`d99d425a4f73fba19ffcf180deb0ef48ff3b9a6a`](https://github.com/ethereum-optimism/optimism/commit/d99d425a4f73fba19ffcf180deb0ef48ff3b9a6a) Thanks [@smartcontracts](https://github.com/smartcontracts)! - Fixes a bug in the SDK for finalizing fpac withdrawals.

- [#9244](https://github.com/ethereum-optimism/optimism/pull/9244) [`73a748575e7c3d67c293814a12bf41eee216163c`](https://github.com/ethereum-optimism/optimism/commit/73a748575e7c3d67c293814a12bf41eee216163c) Thanks [@roninjin10](https://github.com/roninjin10)! - Added maintence mode warning to sdk

- Updated dependencies [[`79effc52e8b82d15b5eda43acf540ac6c5f8d5d7`](https://github.com/ethereum-optimism/optimism/commit/79effc52e8b82d15b5eda43acf540ac6c5f8d5d7)]:
  - @eth-optimism/contracts-bedrock@0.17.1

## 3.1.8

### Patch Changes

- [#8902](https://github.com/ethereum-optimism/optimism/pull/8902) [`18becd7e4`](https://github.com/ethereum-optimism/optimism/commit/18becd7e457577c105f6bc03597e069334cb7433) Thanks [@smartcontracts](https://github.com/smartcontracts)! - Fixes a bug in the SDK that would fail if unsupported fields were provided.

## 3.1.7

### Patch Changes

- [#8836](https://github.com/ethereum-optimism/optimism/pull/8836) [`6ec80fd19`](https://github.com/ethereum-optimism/optimism/commit/6ec80fd19d9155b17a0873672fb095d323f6e8fb) Thanks [@smartcontracts](https://github.com/smartcontracts)! - Fixes a bug in l1 gas cost estimation.

## 3.1.6

### Patch Changes

- [#8212](https://github.com/ethereum-optimism/optimism/pull/8212) [`dd0e46986`](https://github.com/ethereum-optimism/optimism/commit/dd0e46986f19dcceb304fc48f2bd410685ecd179) Thanks [@smartcontracts](https://github.com/smartcontracts)! - Simplifies getMessageStatus to use an O(1) lookup instead of an event query

## 3.1.5

### Patch Changes

- [#8155](https://github.com/ethereum-optimism/optimism/pull/8155) [`2534eabb5`](https://github.com/ethereum-optimism/optimism/commit/2534eabb50afe76f176407f83cc1f1c606e6de69) Thanks [@smartcontracts](https://github.com/smartcontracts)! - Fixed bug with tokenBridge checks throwing

## 3.1.4

### Patch Changes

- [#7450](https://github.com/ethereum-optimism/optimism/pull/7450) [`ac90e16a7`](https://github.com/ethereum-optimism/optimism/commit/ac90e16a7f85c4f73661ae6023135c3d00421c1e) Thanks [@roninjin10](https://github.com/roninjin10)! - Updated dev dependencies related to testing that is causing audit tooling to report failures

- Updated dependencies [[`ac90e16a7`](https://github.com/ethereum-optimism/optimism/commit/ac90e16a7f85c4f73661ae6023135c3d00421c1e)]:
  - @eth-optimism/contracts-bedrock@0.16.2
  - @eth-optimism/core-utils@0.13.1

## 3.1.3

### Patch Changes

- [#7244](https://github.com/ethereum-optimism/optimism/pull/7244) [`679207751`](https://github.com/ethereum-optimism/optimism/commit/6792077510fd76553c179d8b8d068262cda18db6) Thanks [@nitaliano](https://github.com/nitaliano)! - Adds Sepolia & OP Sepolia support to SDK

- Updated dependencies [[`210b2c81d`](https://github.com/ethereum-optimism/optimism/commit/210b2c81dd383bad93480aa876b283d9a0c991c2), [`2440f5e7a`](https://github.com/ethereum-optimism/optimism/commit/2440f5e7ab6577f2d2e9c8b0c78c014290dde8e7)]:
  - @eth-optimism/core-utils@0.13.0
  - @eth-optimism/contracts-bedrock@0.16.1

## 3.1.2

### Patch Changes

- [#6886](https://github.com/ethereum-optimism/optimism/pull/6886) [`9c3a03855`](https://github.com/ethereum-optimism/optimism/commit/9c3a03855dc982f0b4e1d664e83271883536632b) Thanks [@roninjin10](https://github.com/roninjin10)! - Updated npm dependencies to latest

## 3.1.1

### Patch Changes

- Updated dependencies [[`dfa309e34`](https://github.com/ethereum-optimism/optimism/commit/dfa309e3430ebc8790b932554dde120aafc4161e)]:
  - @eth-optimism/core-utils@0.12.3

## 3.1.0

### Minor Changes

- [#6053](https://github.com/ethereum-optimism/optimism/pull/6053) [`ff577455f`](https://github.com/ethereum-optimism/optimism/commit/ff577455f196b5f5b8a889339b845561ca6c538a) Thanks [@roninjin10](https://github.com/roninjin10)! - Add support for claiming multicall3 withdrawals

- [#6042](https://github.com/ethereum-optimism/optimism/pull/6042) [`89ca741a6`](https://github.com/ethereum-optimism/optimism/commit/89ca741a63c5e07f9d691bb6f7a89f7718fc49ca) Thanks [@roninjin10](https://github.com/roninjin10)! - Fixes issue with legacy withdrawal message status detection

- [#6332](https://github.com/ethereum-optimism/optimism/pull/6332) [`639163253`](https://github.com/ethereum-optimism/optimism/commit/639163253a5e2128f1c21c446b68d358d38cbd30) Thanks [@wilsoncusack](https://github.com/wilsoncusack)! - Added to and from block filters to several methods in CrossChainMessenger

### Patch Changes

- [#6254](https://github.com/ethereum-optimism/optimism/pull/6254) [`a666c4f20`](https://github.com/ethereum-optimism/optimism/commit/a666c4f2082253abbb68c0678e5a0a1ed0c00f4b) Thanks [@roninjin10](https://github.com/roninjin10)! - Fixed missing indexes for multicall support

- [#6164](https://github.com/ethereum-optimism/optimism/pull/6164) [`c11039060`](https://github.com/ethereum-optimism/optimism/commit/c11039060bc037a88916c2cba602687b6d69ad1a) Thanks [@pengin7384](https://github.com/pengin7384)! - fix typo

- [#6198](https://github.com/ethereum-optimism/optimism/pull/6198) [`77da6edc6`](https://github.com/ethereum-optimism/optimism/commit/77da6edc643e0b5e39f7b6bb41c3c7ead418a876) Thanks [@tremarkley](https://github.com/tremarkley)! - Delete dead typescript https://github.com/ethereum-optimism/optimism/pull/6148.

- [#6182](https://github.com/ethereum-optimism/optimism/pull/6182) [`3f13fd0bb`](https://github.com/ethereum-optimism/optimism/commit/3f13fd0bbea051a4550f1df6def1a53a616aa6f6) Thanks [@tremarkley](https://github.com/tremarkley)! - Update the addresses of the bridges on optimism and optimism goerli for the ECO bridge adapter

- Updated dependencies [[`c11039060`](https://github.com/ethereum-optimism/optimism/commit/c11039060bc037a88916c2cba602687b6d69ad1a), [`72d184854`](https://github.com/ethereum-optimism/optimism/commit/72d184854ebad8b2025641f126ed76573b1f0ac3), [`77da6edc6`](https://github.com/ethereum-optimism/optimism/commit/77da6edc643e0b5e39f7b6bb41c3c7ead418a876)]:
  - @eth-optimism/contracts-bedrock@0.16.0
  - @eth-optimism/core-utils@0.12.2

## 3.0.0

### Major Changes

- 119754c2f: Make optimism/sdk default to bedrock mode

### Patch Changes

- Updated dependencies [8d7dcc70c]
- Updated dependencies [d6388be4a]
- Updated dependencies [af292562f]
  - @eth-optimism/core-utils@0.12.1
  - @eth-optimism/contracts-bedrock@0.15.0

## 2.1.0

### Minor Changes

- 5063a69fb: Update sdk contract addresses for bedrock

### Patch Changes

- a1b7ff9e3: add eco bridge adapter
- 8133872ed: Fix firefox bug with getTokenPair
- afc2ab8c9: Update the migrated withdrawal gas limit for non goerli networks
- aa854bdd8: Add warning if bedrock is not turned on
- Updated dependencies [f1e867177]
- Updated dependencies [197884eae]
- Updated dependencies [6eb05430d]
- Updated dependencies [5063a69fb]
  - @eth-optimism/contracts-bedrock@0.14.0
  - @eth-optimism/contracts@0.6.0

## 2.0.2

### Patch Changes

- be3315689: Have SDK automatically create Standard and ETH bridges when L1StandardBridge is provided.
- Updated dependencies [b16067a9f]
- Updated dependencies [9a02079eb]
- Updated dependencies [98fbe9d22]
  - @eth-optimism/contracts-bedrock@0.13.2

## 2.0.1

### Patch Changes

- 66cafc00a: Update migrated withdrawal gaslimit calculation
- Updated dependencies [22c3885f5]
- Updated dependencies [f52c07529]
  - @eth-optimism/contracts-bedrock@0.13.1

## 2.0.0

### Major Changes

- cb19e2f9c: Moves `FINALIZATION_PERIOD_SECONDS` from the `OptimismPortal` to the `L2OutputOracle` & ensures the `CHALLENGER` key cannot delete finalized outputs.

### Patch Changes

- Updated dependencies [cb19e2f9c]
  - @eth-optimism/contracts-bedrock@0.13.0

## 1.10.4

### Patch Changes

- Updated dependencies [80f2271f5]
  - @eth-optimism/contracts-bedrock@0.12.1

## 1.10.3

### Patch Changes

- Updated dependencies [7c0a2cc37]
- Updated dependencies [2865dd9b4]
- Updated dependencies [efc98d261]
- Updated dependencies [388f2c25a]
  - @eth-optimism/contracts-bedrock@0.12.0

## 1.10.2

### Patch Changes

- 5372c9f5b: Remove assert node builtin from sdk
- Updated dependencies [3c22333b8]
  - @eth-optimism/contracts-bedrock@0.11.4

## 1.10.1

### Patch Changes

- Updated dependencies [4964be480]
  - @eth-optimism/contracts-bedrock@0.11.3

## 1.10.0

### Minor Changes

- 3f4b3c328: Add in goerli bedrock addresses

## 1.9.1

### Patch Changes

- Updated dependencies [8784bc0bc]
  - @eth-optimism/contracts-bedrock@0.11.2

## 1.9.0

### Minor Changes

- d1f9098f9: Removes support for Kovan

### Patch Changes

- ba8b94a60: Don't pass 0 gasLimit for migrated withdrawals
- Updated dependencies [fe80a9488]
- Updated dependencies [827fc7b04]
- Updated dependencies [a2166dcad]
- Updated dependencies [ff09ec22d]
- Updated dependencies [85dfa9fe2]
- Updated dependencies [d1f9098f9]
- Updated dependencies [0f8fc58ad]
- Updated dependencies [89f70c591]
- Updated dependencies [03940c3cb]
  - @eth-optimism/contracts-bedrock@0.11.1
  - @eth-optimism/contracts@0.5.40

## 1.8.0

### Minor Changes

- c975c9620: Add suppory for finalizing legacy withdrawals after the Bedrock migration

### Patch Changes

- 767585b07: Removes an unused variable from the SDK
- 136ea1785: Refactors the L2OutputOracle to key the l2Outputs mapping by index instead of by L2 block number.
- Updated dependencies [43f33f39f]
- Updated dependencies [237a351f1]
- Updated dependencies [1d3c749a2]
- Updated dependencies [c975c9620]
- Updated dependencies [1594678e0]
- Updated dependencies [1d3c749a2]
- Updated dependencies [136ea1785]
- Updated dependencies [4d13f0afe]
- Updated dependencies [7300a7ca7]
  - @eth-optimism/contracts-bedrock@0.11.0
  - @eth-optimism/contracts@0.5.39
  - @eth-optimism/core-utils@0.12.0

## 1.7.0

### Minor Changes

- 1bfe79f20: Adds an implementation of the Two Step Withdrawals V2 proposal

### Patch Changes

- Updated dependencies [c025a1153]
- Updated dependencies [f8697a607]
- Updated dependencies [59adcaa09]
- Updated dependencies [c71500a7e]
- Updated dependencies [f49b71d50]
- Updated dependencies [1bfe79f20]
- Updated dependencies [ccaf5bc83]
  - @eth-optimism/contracts-bedrock@0.10.0

## 1.6.11

### Patch Changes

- Updated dependencies [52079cc12]
- Updated dependencies [13bfafb21]
- Updated dependencies [eeae96941]
- Updated dependencies [427831d86]
  - @eth-optimism/contracts-bedrock@0.9.1

## 1.6.10

### Patch Changes

- Updated dependencies [1e76cdb86]
- Updated dependencies [c02831144]
- Updated dependencies [d58b0a397]
- Updated dependencies [ff860ecf3]
- Updated dependencies [cc5adbc61]
- Updated dependencies [31c91ea74]
- Updated dependencies [87702c741]
  - @eth-optimism/core-utils@0.11.0
  - @eth-optimism/contracts-bedrock@0.9.0
  - @eth-optimism/contracts@0.5.38

## 1.6.9

### Patch Changes

- Updated dependencies [db84317b]
- Updated dependencies [9b90c732]
  - @eth-optimism/contracts-bedrock@0.8.3

## 1.6.8

### Patch Changes

- Updated dependencies [7d7d9ba8]
  - @eth-optimism/contracts-bedrock@0.8.2

## 1.6.7

### Patch Changes

- b40913b1: Adds contract addresses for the Bedrock Alpha testnet
- a5e715c3: Rename the event emitted in the L2ToL1MessagePasser
- Updated dependencies [35a7bb5e]
- Updated dependencies [a5e715c3]
- Updated dependencies [d18b8aa3]
  - @eth-optimism/contracts-bedrock@0.8.1

## 1.6.6

### Patch Changes

- Updated dependencies [6ed68fa3]
- Updated dependencies [628affc7]
- Updated dependencies [3d4e8529]
- Updated dependencies [caf5dd3e]
- Updated dependencies [740e1bcc]
- Updated dependencies [a6cbfee2]
- Updated dependencies [394a26ec]
  - @eth-optimism/contracts-bedrock@0.8.0
  - @eth-optimism/contracts@0.5.37

## 1.6.5

### Patch Changes

- e2faaa8b: Update for new BedrockMessagePasser contract
- Updated dependencies [cb5fed67]
- Updated dependencies [c427f0c0]
- Updated dependencies [e2faaa8b]
- Updated dependencies [d28ad592]
- Updated dependencies [76c8ee2d]
  - @eth-optimism/contracts-bedrock@0.7.0

## 1.6.4

### Patch Changes

- 7215f4ce: Bump ethers to 5.7.0 globally
- 206f6033: Fix outdated references to 'withdrawal contract'
- d7679ca4: Add source maps
- Updated dependencies [88dde7c8]
- Updated dependencies [7215f4ce]
- Updated dependencies [249a8ed6]
- Updated dependencies [7d7c4fdf]
- Updated dependencies [e164e22e]
- Updated dependencies [0bc1be45]
- Updated dependencies [af3e56b1]
- Updated dependencies [206f6033]
- Updated dependencies [88dde7c8]
- Updated dependencies [8790156c]
- Updated dependencies [515685f4]
  - @eth-optimism/contracts-bedrock@0.6.3
  - @eth-optimism/contracts@0.5.36
  - @eth-optimism/core-utils@0.10.1

## 1.6.3

### Patch Changes

- Updated dependencies [651a2883]
  - @eth-optimism/contracts-bedrock@0.6.2

## 1.6.2

### Patch Changes

- cfa81f88: Add DAI bridge support to Goerli
- Updated dependencies [85232179]
- Updated dependencies [593f1cfb]
- Updated dependencies [334a3eb0]
- Updated dependencies [f78eb056]
  - @eth-optimism/contracts-bedrock@0.6.1
  - @eth-optimism/contracts@0.5.35

## 1.6.1

### Patch Changes

- b27d0fa7: Add wsteth support for DAI bridge to sdk
- Updated dependencies [7fdc490c]
- Updated dependencies [3d228a0e]
- Updated dependencies [dbfea116]
- Updated dependencies [63ef1949]
- Updated dependencies [299157e7]
  - @eth-optimism/contracts-bedrock@0.6.0
  - @eth-optimism/core-utils@0.10.0
  - @eth-optimism/contracts@0.5.34

## 1.6.0

### Minor Changes

- 3af9c7a9: Removes the ICrossChainMessenger interface to speed up SDK development.

### Patch Changes

- 3df66a9a: Fix eth withdrawal bug
- 8323407f: Fixes a bug in the SDK for certain bridge withdrawals.
- aa2949ef: Add eth withdrawal support
- a1a73e64: Updates the SDK to pull contract addresses from the deployments of the contracts package. Updates the Contracts package to export a function that makes it possible to pull deployed addresses.
- f53c30b9: Minor refactor to variables within the SDK package.
- Updated dependencies [a095d544]
- Updated dependencies [cdf2163e]
- Updated dependencies [791f30bc]
- Updated dependencies [193befed]
- Updated dependencies [0c2719f8]
- Updated dependencies [02420db0]
- Updated dependencies [94a8f287]
- Updated dependencies [7d03c5c0]
- Updated dependencies [fec22bfe]
- Updated dependencies [9272253e]
- Updated dependencies [a1a73e64]
- Updated dependencies [c025f418]
- Updated dependencies [329d21b6]
- Updated dependencies [35eafed0]
- Updated dependencies [3cde9205]
  - @eth-optimism/contracts-bedrock@0.5.4
  - @eth-optimism/contracts@0.5.33

## 1.5.0

### Minor Changes

- dcd715a6: Update wsteth bridge address

## 1.4.0

### Minor Changes

- f05ab6b6: Add wstETH to sdk
- dac4a9f0: Updates the SDK to be compatible with Bedrock (via the "bedrock: true" constructor param). Updates the build pipeline for contracts-bedrock to export a properly formatted dist folder that matches our other packages.

### Patch Changes

- Updated dependencies [056cb982]
- Updated dependencies [a32e68ac]
- Updated dependencies [c648d55c]
- Updated dependencies [d544f804]
- Updated dependencies [ccbfe545]
- Updated dependencies [c97ad241]
- Updated dependencies [0df744f6]
- Updated dependencies [45541553]
- Updated dependencies [3dd296e8]
- Updated dependencies [fe94b864]
- Updated dependencies [28649d64]
- Updated dependencies [898c7ac5]
- Updated dependencies [51a1595b]
- Updated dependencies [8ae39154]
- Updated dependencies [af96563a]
- Updated dependencies [dac4a9f0]
  - @eth-optimism/contracts-bedrock@0.5.3
  - @eth-optimism/core-utils@0.9.3
  - @eth-optimism/contracts@0.5.32

## 1.3.1

### Patch Changes

- 680714c1: Updates the CCM to throw a better error for missing or invalid chain IDs
- 29830750: Update the Goerli SCC's address
- Updated dependencies [0bf3b9b4]
- Updated dependencies [8d26459b]
- Updated dependencies [4477fe9f]
- Updated dependencies [1de4f48e]
  - @eth-optimism/core-utils@0.9.2
  - @eth-optimism/contracts@0.5.31

## 1.3.0

### Minor Changes

- 032f7214: Update Goerli SDK addresses for new Goerli testnet

## 1.2.1

### Patch Changes

- Updated dependencies [6e3449ba]
- Updated dependencies [f9fee446]
  - @eth-optimism/contracts@0.5.30
  - @eth-optimism/core-utils@0.9.1

## 1.2.0

### Minor Changes

- 977493bc: Have SDK use L2 chain ID as the source of truth.

### Patch Changes

- Updated dependencies [700dcbb0]
  - @eth-optimism/core-utils@0.9.0
  - @eth-optimism/contracts@0.5.29

## 1.1.9

### Patch Changes

- 29ff7462: Revert es target back to 2017
- Updated dependencies [27234f68]
- Updated dependencies [29ff7462]
  - @eth-optimism/contracts@0.5.28
  - @eth-optimism/core-utils@0.8.7

## 1.1.8

### Patch Changes

- Updated dependencies [7c5ac36f]
- Updated dependencies [3d4d988c]
  - @eth-optimism/contracts@0.5.27

## 1.1.7

### Patch Changes

- Updated dependencies [17962ca9]
  - @eth-optimism/core-utils@0.8.6
  - @eth-optimism/contracts@0.5.26

## 1.1.6

### Patch Changes

- d18ae135: Updates all ethers versions in response to BN.js bug
- Updated dependencies [d18ae135]
  - @eth-optimism/contracts@0.5.25
  - @eth-optimism/core-utils@0.8.5

## 1.1.5

### Patch Changes

- 86901552: Fixes a bug in the SDK which would cause the SDK to throw if no tx nonce is provided

## 1.1.4

### Patch Changes

- Updated dependencies [b7a04acf]
  - @eth-optimism/contracts@0.5.24

## 1.1.3

### Patch Changes

- Updated dependencies [412688d5]
  - @eth-optimism/contracts@0.5.23

## 1.1.2

### Patch Changes

- Updated dependencies [51adb389]
- Updated dependencies [5cb3a5f7]
- Updated dependencies [6b9fc055]
  - @eth-optimism/contracts@0.5.22
  - @eth-optimism/core-utils@0.8.4

## 1.1.1

### Patch Changes

- 1338135c: Fixes a bug where the wrong Overrides type was being used for gas estimation functions

## 1.1.0

### Minor Changes

- a9f8e577: New isL2Provider helper function. Internal cleanups.

### Patch Changes

- Updated dependencies [5818decb]
  - @eth-optimism/contracts@0.5.21

## 1.0.4

### Patch Changes

- b57014d1: Update to typescript@4.6.2
- Updated dependencies [d040a8d9]
- Updated dependencies [b57014d1]
  - @eth-optimism/contracts@0.5.20
  - @eth-optimism/core-utils@0.8.3

## 1.0.3

### Patch Changes

- c1957126: Update Dockerfile to use Alpine
- d9a51154: Bump to hardhat@2.9.1
- Updated dependencies [c1957126]
- Updated dependencies [d9a51154]
  - @eth-optimism/contracts@0.5.19
  - @eth-optimism/core-utils@0.8.2

## 1.0.2

### Patch Changes

- d49feca1: Comment out non-functional getMessagesByAddress function
- Updated dependencies [88601cb7]
  - @eth-optimism/contracts@0.5.18

## 1.0.1

### Patch Changes

- 7ae1c67f: Update package json to include correct repo link
- 47e5d118: Tighten type restriction on ProviderLike
- Updated dependencies [175ae0bf]
  - @eth-optimism/contracts@0.5.17

## 1.0.0

### Major Changes

- 84f63c49: Update README and bump SDK to 1.0.0

### Patch Changes

- 42227d69: Fix typo in constructor docstring

## 0.2.5

### Patch Changes

- b66e3131: Add a function for waiting for a particular message status
- Updated dependencies [962f36e4]
- Updated dependencies [f2179e37]
- Updated dependencies [b6a4fa4b]
- Updated dependencies [b7c0a5ca]
- Updated dependencies [5a6f539c]
- Updated dependencies [27d8942e]
  - @eth-optimism/contracts@0.5.16
  - @eth-optimism/core-utils@0.8.1

## 0.2.4

### Patch Changes

- 44420939: 1. Fix a bug in `L2Provider.getL1GasPrice()` 2. Make it easier to get correct estimates from `L2Provider.estimateL1Gas()` and `L2.estimateL2GasCost`.

## 0.2.3

### Patch Changes

- f37c283c: Have SDK properly handle case when no batches are submitted yet
- 3f4d3c13: Have SDK wait for transactions in getMessagesByTransaction
- 0c54e60e: Add approval functions to the SDK
- Updated dependencies [0b4453f7]
- Updated dependencies [78298782]
  - @eth-optimism/core-utils@0.8.0
  - @eth-optimism/contracts@0.5.15

## 0.2.2

### Patch Changes

- fd6ea3ee: Adds support for depositing or withdrawing to a target address
- 5ffb5fcf: Removes the getTokenBridgeMessagesByAddress function
- dd4b2055: This update implements the asL2Provider function
- f08c06a8: Updates the SDK to include default bridges for the local Optimism network (31337)
- da53dc64: Have SDK sort deposits/withdrawals descending by block number
- Updated dependencies [b4165299]
- Updated dependencies [3c2acd91]
  - @eth-optimism/core-utils@0.7.7
  - @eth-optimism/contracts@0.5.14

## 0.2.1

### Patch Changes

- Updated dependencies [438bc78a]
  - @eth-optimism/contracts@0.5.13

## 0.2.0

### Minor Changes

- dd9683bb: Correctly export SDK contents

## 0.1.0

### Minor Changes

- cb65f3d8: Beta release of the Optimism SDK

### Patch Changes

- ba14c59d: Updates various ethers dependencies to their latest versions
- 64e746b6: Have SDK include ethers as a peer dependency
- Updated dependencies [ba14c59d]
  - @eth-optimism/contracts@0.5.12
  - @eth-optimism/core-utils@0.7.6

## 0.0.7

### Patch Changes

- Updated dependencies [e631c39c]
  - @eth-optimism/contracts@0.5.11

## 0.0.6

### Patch Changes

- Updated dependencies [ad94b9d1]
  - @eth-optimism/core-utils@0.7.5
  - @eth-optimism/contracts@0.5.10

## 0.0.5

### Patch Changes

- Updated dependencies [ba96a455]
- Updated dependencies [c3e85fef]
  - @eth-optimism/core-utils@0.7.4
  - @eth-optimism/contracts@0.5.9

## 0.0.4

### Patch Changes

- Updated dependencies [b3efb8b7]
- Updated dependencies [279603e5]
- Updated dependencies [b6040bb3]
  - @eth-optimism/contracts@0.5.8

## 0.0.3

### Patch Changes

- Updated dependencies [b6f89fad]
  - @eth-optimism/contracts@0.5.7

## 0.0.2

### Patch Changes

- Updated dependencies [bbd42e03]
- Updated dependencies [453f0774]
  - @eth-optimism/contracts@0.5.6
