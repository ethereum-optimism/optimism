# Gas repricing sensitivity

This document lists code that a gas repricing on L1 or L2 can affect. A repricing is any change
to opcode, precompile, calldata, storage, memory, refund, or intrinsic gas costs. The list
excludes code where a repricing only changes the cost to the user.

Read this document before you review or plan a fork that changes gas costs on L1 or L2. Also
read it before you add a fixed gas limit or a gas check to the code.

**Keep this document current.** When you find a new case, add it to the correct section. When
you remove or move a listed case, update or remove its entry.

## How to use this list

Each entry gives a file, the symbol, the chain whose gas schedule applies, and the risk. The
entries use symbol names, not line numbers, because line numbers change often.

The kinds of risk are:

- **Introspection:** the code reads `gasleft()`, `GAS`, gas used, or a gas price, and makes a
  decision from the value.
- **Fixed gas:** the code sets a fixed gas limit for a call, a transaction, or a deposit.
- **Copied schedule:** the code contains a copy of an L1 or L2 gas cost. Examples are 21000 and
  16 gas for each calldata byte. The copy does not change when the real schedule changes.
- **Limit bound:** the code enforces a minimum or maximum gas value.
- **Gas-scaled formula:** a fee, bond, or budget scales with gas used.

The sections are:

- **Production code:** code that runs in a live chain, node, or service. Network upgrade
  transactions that derivation runs are production code.
- **Deployment-only code:** code that runs only to deploy, upgrade, or check a chain.
- **Test and dev code:** tests, test helpers, and dev-mode code.

## Production code

### Contracts: cross-domain messages and withdrawals

| Location | Chain | Kind | Risk |
|---|---|---|---|
| `SafeCall.hasMinGas`, `SafeCall.callWithMinGas` (`packages/contracts-bedrock/src/libraries/SafeCall.sol`) | Both | Copied schedule | The check assumes that the CALL overhead is at most 40000 gas and uses the 63/64 rule. If CALL or cold access costs more, the target gets less than the minimum gas. |
| `CrossDomainMessenger.relayMessage` (`src/universal/CrossDomainMessenger.sol`) | Both | Fixed gas | The relay reserves `RELAY_RESERVED_GAS` (40000) after the call and `RELAY_GAS_CHECK_BUFFER` (5000) before the call. If the code after the call costs more, the whole relay reverts. The message is then not in `failedMessages`, so nobody can replay a message that a deposit relayed. |
| `CrossDomainMessenger.baseGas` and its constants (`RELAY_*`, `MIN_GAS_*`, `TX_BASE_GAS`, `FLOOR_CALLDATA_OVERHEAD`, `ENCODING_OVERHEAD`) | Both | Copied schedule | The constants copy the intrinsic gas, the calldata price, and the EIP-7623 floor. A change to these makes the gas limit on the other chain too low. |
| `OptimismPortal2.finalizeWithdrawalTransaction` (`src/L1/OptimismPortal2.sol`) | L1 | Fixed gas | A withdrawal runs one time with `callWithMinGas(_tx.gasLimit)`. If L1 costs increase, a withdrawal can fail, and the user cannot replay it. |
| `OptimismPortal2.minimumGasLimit` | L2 | Limit bound | The minimum deposit gas is `bytes * 40 + 21000`, which copies the L2 calldata floor. Deposits that pass this check can fail on L2 after a repricing. |
| `RECEIVE_DEFAULT_GAS_LIMIT` in `OptimismPortal2` (100000), `L2ToL1MessagePasser` (100000), and `StandardBridge` (200000) | Destination chain | Fixed gas | These are the gas limits for a plain ETH send when the user does not give one. They can become too low. |
| `L1CrossDomainMessenger.EXPIRE_MESSAGE_GAS_LIMIT` (100000) and `SET_ETH_LIQUIDITY_PAUSED_GAS_LIMIT` (200000) | L2 | Fixed gas | These admin messages fail if their L2 handlers cost more than the limit. |
| `FeeVault._WITHDRAWAL_MIN_GAS` (400000) (`src/L2/FeeVault.sol`) | L1 | Fixed gas | This is the L1 gas limit for fee vault withdrawals. |

### Contracts: introspection and other fixed gas

| Location | Chain | Kind | Risk |
|---|---|---|---|
| `CrossL2Inbox._isWarm` (`src/L2/CrossL2Inbox.sol`) | L2 | Introspection | It measures one `SLOAD` and compares the cost with `_WARM_READ_THRESHOLD` (1000). A change to warm or cold read prices gives the wrong result, and message validation breaks. |
| `CrossL2Inbox` deposit check | L2 | Introspection | It finds deposits with `block.basefee > 0 && tx.gasprice == 0`. A change to fee rules can break this check. |
| `Burn.gas` (`src/libraries/Burn.sol`) | L1 | Introspection | The loop checks `gasleft()`. Opcode prices set the step size and the overshoot. |
| `WETH98.withdraw` (`src/universal/WETH98.sol`), also called by `DelayedWETH.withdraw` (`src/dispute/DelayedWETH.sol`) | Both | Fixed gas | `transfer` gives a 2300 gas stipend. On L1, `DelayedWETH` sends ETH to the dispute game this way. A receiver can fail if its opcodes cost more. |
| `FaultDisputeGame.closeGame`, `SuperFaultDisputeGame.closeGame` | L1 | Introspection | The 63/64 rule applies to `try setAnchorState {} catch {}`, and the catch hides an out-of-gas failure. If the registry costs more, a caller can make the update fail on purpose. |
| `Faucet.drip` (`src/periphery/faucet/Faucet.sol`) | Both | Fixed gas | It calls the recipient with a gas limit that the caller gives. |

### Contracts: limits and gas-scaled formulas

| Location | Chain | Kind | Risk |
|---|---|---|---|
| `SystemConfig` gas limit bounds (`MAX_GAS_LIMIT`, `minimumGasLimit`) (`src/L1/SystemConfig.sol`) | L2 | Limit bound | The maximum is 500M. The minimum is `maxResourceLimit + systemTxMaxGas`. A repricing changes how much work these limits allow. |
| `GasPriceOracle` L1 fee functions (`src/L2/GasPriceOracle.sol`) | L1 | Copied schedule | The functions use 4 and 16 gas for each calldata byte, a 68-byte signature allowance, and `* 16` in the Ecotone and Fjord formulas. A change to L1 calldata pricing makes them disagree with the real L1 cost. They must also match the client formulas in `op-core/fees` and `op-revm`. |
| `GasPriceOracle.getOperatorFee` | L2 | Gas-scaled formula | The operator fee scales with gas used. |
| `DataAvailabilityChallenge` resolution cost (`fixedResolutionCost`, `variableResolutionCost`) | L1 | Copied schedule | The values come from measured gas. They set the refund and the burn of bonds. |
| `FaultDisputeGame.getRequiredBond`, `SuperFaultDisputeGame.getRequiredBond` | L1 | Copied schedule | The bond uses an assumed base fee of 200 gwei and gas values of 400k and 300M. If move or step costs change, the bonds do not cover the cost to respond. |
| `PreimageOracle.MIN_BOND_SIZE` and `addLeavesLPP` (`src/cannon/PreimageOracle.sol`) | L1 | Copied schedule | The bond must cover the cost of a challenge. A change to calldata, memory, or LOG costs changes the cost of large preimages against the bond. |
| `PreimageOracle.loadPrecompilePreimagePart` | L1 | Introspection | It requires `gas * 63 >= requiredGas * 64 + 100000 * 63`, and `requiredGas` is part of the preimage key. A precompile repricing on L1 can make the oracle and the fault proof program disagree. |

### op-node and kona: deposits and the block gas limit

op-node and kona must agree on each item in this section.

| Location | Chain | Kind | Risk |
|---|---|---|---|
| `RegolithSystemTxGas` (`op-node/rollup/derive/l1_block_info.go`) and the gas of the L1 info deposit (`rust/kona/crates/protocol/protocol/src/info/variant.rs`) | L2 | Fixed gas | The L1 info deposit has 1,000,000 gas. If `L1Block` storage writes cost more, the deposit can fail in every block. |
| Gas limits of the Ecotone, Fjord, Isthmus, and Jovian upgrade deposits (`op-node/rollup/derive/*_upgrade_transactions.go`, `rust/kona/crates/protocol/hardforks/src/`) | L2 | Fixed gas | These forks are active, so a future repricing does not change them. A new fork that copies this pattern must measure its limits under the new costs. |
| Gas limits in the NUT bundles (`op-core/nuts/bundles/*_nut_bundle.json`), read by `op-node/rollup/derive/upgrade_transaction.go`, `rust/kona/crates/protocol/hardforks/build.rs`, and `rust/op-alloy/crates/consensus/src/nuts/` | L2 | Fixed gas | Each upgrade deployment has a fixed gas limit. If CREATE, code deposit, SSTORE, or calldata costs increase in the same fork, a deployment can fail in the activation block. |
| Lagoon activation deposits (`op-node/rollup/derive/lagoon_activation_transactions.go`, `rust/kona/crates/protocol/hardforks/src/lagoon.rs`) | L2 | Fixed gas | `setFeature` has 100000 gas and the liquidity funding has 50000 gas. |
| Gas limit of the upgrade block (`op-node/rollup/derive/attributes.go`, `rust/kona/crates/protocol/derive/src/attributes/stateful.rs`) | L2 | Limit bound | The activation block gas limit is the system config gas limit plus the bundle gas total. The bundle limits therefore also set the block size. |
| Upgrade gas removal (`op-node/rollup/derive/payload_util.go`, `rust/kona/crates/protocol/protocol/src/utils.rs`) | L2 | Limit bound | The node subtracts the upgrade gas to rebuild the system config from an L2 block. If the bundle gas changes after activation, the rebuild gives a wrong value. |
| `L2ContractsManager` and `L2ContractsManagerUtils` try/catch reads (`packages/contracts-bedrock/src/L2/`, `packages/contracts-bedrock/src/libraries/`) | L2 | Introspection | These run in upgrade deposits with a fixed gas limit. An out-of-gas failure in the try call makes the catch use a default value. |

### Execution layer: op-revm, op-reth, alloy-op-evm

| Location | Chain | Kind | Risk |
|---|---|---|---|
| OP fork to L1 fork mapping (`rust/op-revm/src/spec.rs`, `rust/alloy-op-hardforks/src/lib.rs`, `rust/alloy-op-evm/src/env.rs`) | L2 | Copied schedule | Each OP fork takes the gas schedule of its mapped L1 fork. An L1 repricing reaches L2 only through this mapping. |
| Precompile sets and input limits (`rust/op-revm/src/precompiles.rs`) | Both | Limit bound | The bn254 and BLS12-381 input limits exist so that L1 can prove a maximum-size input through `PreimageOracle`. A change to L1 calldata or precompile prices can make a maximum-size input too expensive to prove. The pairing variants also hard-code the Istanbul pairing gas. |
| Deposit validation in `rust/op-revm/src/handler.rs` | L2 | Limit bound | Deposits get the intrinsic gas and EIP-7623 floor checks. Deposits do not get the EIP-7825 transaction gas cap. A failed deposit uses all its gas limit. |
| State gas refund in `rust/op-revm/src/handler.rs` | L2 | Copied schedule | The handler refunds the EIP-2780 and EIP-8037 state gas on the first frame. New state gas prices change this refund. |
| L1 fee formulas (`rust/op-revm/src/l1block.rs`, `rust/op-revm/src/constants.rs`) | L1 | Copied schedule | `NON_ZERO_BYTE_COST` is 16. The Ecotone and Fjord fees use 16 gas for each L1 calldata byte and ignore the L1 calldata floor. |
| Operator fee (`rust/op-revm/src/l1block.rs`) | L2 | Gas-scaled formula | The operator fee and its refund scale with gas used. |
| DA footprint (`rust/op-revm/src/transaction.rs`, `rust/alloy-op-evm/src/block/mod.rs`) | L2 | Limit bound | The DA footprint goes in `blobGasUsed` and competes with the block gas limit. A repricing changes how gas used compares with the DA footprint. |
| Jovian base fee (`rust/op-reth/crates/chainspec/src/basefee.rs`, `rust/kona/crates/proof/executor/src/builder/env.rs`) | L2 | Introspection | The base fee uses the larger of gas used and blob gas used. A repricing changes which value is larger. |
| SDM refunds (`rust/alloy-op-evm/src/block/mod.rs`, `rust/alloy-op-evm/src/post_exec/inspector.rs`) | L2 | Copied schedule | The rebates refund the EIP-2929 cold access surcharges. The refund cap and the EIP-7623 floor also limit them. A new access price changes the correct rebate. |
| Block gas checks (`rust/alloy-op-evm/src/block/mod.rs`, `rust/op-reth/crates/payload/src/builder.rs`) | L2 | Limit bound | The executor and builder check each transaction gas limit against the remaining block gas. `validate_block_gas` does not apply the EIP-7825 cap. |
| `L1_INFO_GAS_OVERHEAD` (70000) (`rust/op-reth/crates/txpool/src/validator.rs`) | L2 | Fixed gas | The pool reserves this gas for the L1 info deposit. If the deposit uses more gas, the pool can accept transactions that no block can include. |
| `l1GasUsed` receipt field (`rust/op-reth/crates/rpc/src/eth/receipt.rs`, `rust/op-reth/crates/evm/src/l1.rs`) | L1 | Copied schedule | The field uses 16 gas for each calldata byte. |

### Fault proofs

| Location | Chain | Kind | Risk |
|---|---|---|---|
| Accelerated precompiles (`rust/kona/bin/client/src/fpvm_evm/precompiles/`) | Both | Copied schedule | Some L2 charges are local constants, for example ECRECOVER 3000 and KZG point evaluation 50000. The L1 oracle gas values are also fixed. Examples are KZG 89363, bn254 pairing 34103 for each pair, and BLS12-381 G1 and G2 add 643 and 765. A repricing on either chain causes a mismatch. |
| BLS12-381 MSM, pairing, and map precompiles in the same directory | L1 | Copied schedule | The L1 oracle gas is the L2 required gas. If L1 costs more, the L1 staticcall fails. |
| Precompile input limits in the same directory | L2 | Limit bound | The limits copy `rust/op-revm/src/precompiles.rs`. Both places must change together. |
| Precompile set selection (`rust/kona/bin/client/src/fpvm_evm/precompiles/provider.rs`) | L2 | Copied schedule | A new L2 precompile price needs a new accelerated variant here. |
| Host precompile execution (`rust/kona/bin/host/src/eth/precompiles.rs`) | L1 | Fixed gas | The host runs the precompile with the gas that the client gives. If that gas is too low, the host returns a failure. |
| `requiredGas` in the precompile preimage upload (`op-challenger/game/fault/contracts/oracle.go`) | L1 | Fixed gas | The value comes from the oracle data. An L1 precompile repricing can make it differ from the real cost. |
| `MaxBlocksPerChunk` (300) (`op-challenger/game/fault/preimages/large.go`) | L1 | Limit bound | Each large preimage chunk is about 40 KB. Higher calldata or memory costs, or the EIP-7825 cap, can make one chunk transaction too large. |
| Receipt gas bound (`rust/kona/sp1/crates/proposer/src/log_proofs/scan.rs`) | L2 | Copied schedule | The proposer multiplies receipt gas by 5/4. This assumes that refunds are at most 1/5 of gas used. |
| `REPLACEMENT_GAS` (`rust/kona/sp1/crates/super-range-executor/src/cost.rs`) | L2 | Gas-scaled formula | The proving cost estimate uses 11000 proving units for each unit of deposit gas. A repricing changes the proving work for one unit of gas. |

### op-batcher

| Location | Chain | Kind | Risk |
|---|---|---|---|
| Gas limit of batch transactions (`op-batcher/batcher/driver.go`) | L1 | Fixed gas | The batcher sets the gas limit from a floor formula and does not estimate. `amsterdamTxBaseGas` and `amsterdamCalldataFloorGasPerByte` copy draft EIP-2780 and EIP-7976 values. |
| `maxL1CalldataSize` (`op-batcher/batcher/config.go`) | L1 | Limit bound | The maximum `MaxL1TxSize` comes from `MaxTxGas` and the Amsterdam constants. |
| Default size of calldata frames (120000 bytes) (`op-batcher/flags/flags.go`, `op-batcher/batcher/service.go`) | L1 | Limit bound | A higher calldata floor can push one frame above the EIP-7825 cap. |
| Blob or calldata choice (`op-batcher/batcher/channel_config_provider.go`) | L1 | Copied schedule | The choice uses `TxGas`, the calldata floors, and `BlobTxBlobGasPerBlob`. The batcher finds Amsterdam from the block access list hash, not from the L1 chain config. |

### op-service and op-core

| Location | Chain | Kind | Risk |
|---|---|---|---|
| `txmgr` gas estimation (`op-service/txmgr/txmgr.go`) | Both | Fixed gas | A candidate with a gas limit gets no estimation. A gas bump estimates again but never lowers the limit. op-challenger, op-proposer, op-dispute-mon, op-conductor, and op-supernode use estimation and set no fixed gas. |
| Blob base fee (`op-service/eth/block_info.go`) | L1 | Copied schedule | The fee uses `eip4844.CalcBlobFee` and the chain config. An old blob schedule gives a wrong fee. |
| L1 cost formulas (`op-core/fees/fees.go`) | L1 | Copied schedule | The formulas use 4 and 16 gas for each calldata byte and ignore the L1 calldata floor. They must match `op-revm` and `GasPriceOracle`. |
| Operator fee and DA footprint (`op-core/fees/fees.go`, `op-core/fees/dafootprint.go`) | L2 | Gas-scaled formula | The operator fee scales with gas used. The DA footprint competes with the block gas limit. |

## Deployment-only code

### Contracts and scripts

| Location | Chain | Kind | Risk |
|---|---|---|---|
| `deployGasLimit` for each predeploy (`src/libraries/Predeploys.sol`) | L2 | Fixed gas | These limits go into the NUT bundles. If deployment costs increase, the upgrade deposits can run out of gas. |
| `OPContractsManagerUtils` try/catch reads (`src/L1/opcm/OPContractsManagerUtils.sol`) | L1 | Introspection | An out-of-gas failure in the try call makes the catch use a default value. |
| `OPContractsManagerStandardValidator` checks (`src/L1/OPContractsManagerStandardValidator.sol`) | L1 | Limit bound | It requires `gasLimit <= 500M` and `systemTxMaxGas == 1_000_000`. |
| `Constants.DEFAULT_RESOURCE_CONFIG` (`src/libraries/Constants.sol`) | L2 | Fixed gas | `systemTxMaxGas` is 1,000,000. The L1Block system transaction must fit in this gas. |
| `GenerateNUTBundle` limits (`scripts/upgrade/GenerateNUTBundle.s.sol`) | L2 | Limit bound | Each gas limit must be between the EIP-7623 floor and the 2^24 cap. |
| `UpgradeUtils` gas values (`scripts/libraries/UpgradeUtils.sol`) | L2 | Copied schedule | It has fixed limits of 8.9M and 4M. It also computes intrinsic gas and the floor with copies of the geth formulas and a 1.5x margin. |
| `ExecuteNUTBundle` (`scripts/upgrade/ExecuteNUTBundle.s.sol`) | L2 | Copied schedule | It calls with `gasLimit - intrinsicGas` and adds EIP-8037 state gas. The intrinsic gas formula must match the client. |
| `DeployUtils` (`scripts/libraries/DeployUtils.sol`) | Both | Introspection | It requires `gasleft() > 200_000` before a try/catch call. |
| `DeployConfig` defaults (`scripts/deploy/DeployConfig.s.sol`) | L2 | Fixed gas | The default genesis gas limit is 25M. |

### op-deployer and op-chain-ops

| Location | Chain | Kind | Risk |
|---|---|---|---|
| `defaultGasLimit` (30M) (`op-deployer/pkg/deployer/broadcaster/calldata.go`) | Both | Fixed gas | The calldata dump gives each transaction 30M gas. This is above the EIP-7825 cap, so tools that send the dump can get rejections. |
| Standard gas limit (60M) (`op-deployer/pkg/deployer/standard/standard.go`) | L2 | Limit bound | Standard chains must use this value. |
| Genesis gas limit check (`op-chain-ops/genesis/config.go`) | L2 | Limit bound | The genesis gas limit must be at least `MaxResourceLimit` plus `SystemTxMaxGas`. |
| Default gas limits of genesis blocks (`op-chain-ops/genesis/genesis.go`, `op-chain-ops/interopgen/recipe.go`) | Both | Fixed gas | The defaults are 30M and 60M. |
| Script host gas (`op-chain-ops/script/`, `op-deployer/pkg/deployer/opcm/standard_validator.go`) | In-process EVM | Fixed gas | Script calls use `int64.max` gas. A repricing changes the reported gas used only. |
| `check-karst` probes (`op-chain-ops/cmd/check-karst/karsttest/checks.go`) | Both | Fixed gas | The probes use exact gas budgets around precompile costs. Some assume 21000 intrinsic gas. |
| `check-lagoon` checks (`op-chain-ops/cmd/check-lagoon/`) | L2 | Fixed gas | The relay check uses 300000 gas. The SDM check uses 21000 and 2,000,000 gas. |
| `check-jovian` and SDM conformance (`op-chain-ops/cmd/check-jovian/main.go`, `op-chain-ops/pkg/sdm/conformance.go`) | L2 | Introspection | They assert exact gas used and DA footprint totals. |
| SDM workload sender (`op-chain-ops/pkg/sdm/workload.go`, `op-chain-ops/cmd/sdm-workload/main.go`) | Both | Fixed gas | The sender uses the caller's gas and does not estimate. The command uses fixed 21000, 200000, and 2,000,000 gas. |

### op-service transaction tools

`op-deployer`, the `op-chain-ops` check tools, and the test frameworks use these packages. No
live service uses them.

| Location | Chain | Kind | Risk |
|---|---|---|---|
| `txplan` defaults (`op-service/txplan/txplan.go`) | Both | Fixed gas | The default `GasRatio` is 1.0. A plan with no estimation step uses `params.TxGas` (21000). |
| Resubmitter errors (`op-service/txinclude/resubmitter.go`) | Both | Introspection | `ErrGasLimit`, `ErrIntrinsicGas`, and `ErrFloorDataGas` are fatal. A transaction with a fixed limit stops after a repricing and does not retry. |
| Transaction budget (`op-service/txinclude/cost_oracle.go`, `op-service/txinclude/txbudget.go`) | L2 | Gas-scaled formula | The budget uses the L1 cost and operator cost formulas with gas used. |

## Test and dev code

Many tests use a fixed gas limit or expect an exact gas result. A repricing makes these tests
fail. Search for them when you implement a repricing.

### Contracts (`packages/contracts-bedrock/test`)

- **Exact gas values:**
  - `libraries/SafeCall.t.sol`
  - `L2/GasPriceOracle.t.sol`
  - `legacy/L1ChugSplashProxy.t.sol`
  - `L1/OptimismPortal2.t.sol` (`minimumGasLimit(0) == 21000`)
  - `libraries/Constants.t.sol`
- **Upper bounds on gas used:**
  - `universal/BenchmarkTest.t.sol` (L1Block system transaction)
  - `L1/opcm/OPContractsManagerV2.t.sol` (fractions of the 2^24 cap)
  - `safe/DeputyPauseModule.t.sol`
  - `L2/fork/L2ForkUpgrade.t.sol` (upgrade transactions against bundle limits)
- **Hand-computed intrinsic gas or floor:**
  - `L1/L1CrossDomainMessenger.t.sol`
  - `L2/L2CrossDomainMessenger.t.sol`
  - `universal/CrossDomainMessenger.t.sol`
- **Fixed call gas:**
  - `L2/L2ToL2CrossDomainMessenger.t.sol`
  - `L2/ETHLiquidity.t.sol`
  - `cannon/PreimageOracle.t.sol`
  - `dispute/FaultDisputeGame.t.sol`
  - `invariants/CrossDomainMessenger.t.sol`
  - `periphery/faucet/`
  - `setup/PastNUTBundles.t.sol`
  - `scripts/ExecuteNUTBundle.t.sol`
- **Introspection:**
  - `invariants/Burn.Gas.t.sol`
  - `L1/ResourceMetering.t.sol`
  - `L1/DataAvailabilityChallenge.t.sol`
  - `L2/CrossL2Inbox.t.sol` (the warm read tests depend on read prices)
- **Test helpers:**
  - `mocks/GasBurner.sol` assumes a 500 gas overhead.

### Go unit tests and helpers

- **Test helpers in production packages:**
  - `op-batcher/batcher/test_batch_submitter.go` uses only the pre-Amsterdam floor.
  - `op-service/txmgr/test_txmgr.go` uses the candidate gas limit and does not estimate.
- **Exact gas assertions:**
  - `op-batcher/batcher/driver_test.go`
  - `op-node/rollup/derive/l1_block_info_test.go`
  - `op-node/rollup/derive/lagoon_activation_transactions_test.go`
  - `op-node/rollup/derive/upgrade_transaction_test.go`
  - `op-core/fees/fees_test.go`
  - `op-core/fees/dafootprint_test.go`
  - `op-service/txinclude/*_test.go`
  - `op-service/txmgr/txmgr_test.go`
- **Fixture gas only:** many other `*_test.go` files hold gas values in fixtures, but do not check costs. Expect little effect.

### op-e2e

- `op-e2e/opgeth/op_geth_test.go` has fixed deposit gas, expects exactly 21000 gas used, and compares SSTORE refunds.
- `op-e2e/actions/upgrades/fjord_fork_test.go` expects an exact `getL1GasUsed` result.
- Many `op-e2e/actions/` tests send transfers with `Gas: params.TxGas`.
- `op-e2e/actions/helpers/l2_batcher.go` uses `TxGas` or 1,000,000 gas for batch transactions.
- `op-e2e/actions/helpers/enginetest/sequencer_smoke_test.go` expects a transfer to use exactly `TxGas`.
- `op-e2e/system/da/da_throttling_test.go` computes gas as `21000 + len * 16`.
- `op-e2e/system/helpers/`, `op-e2e/system/bridge/`, `op-e2e/system/fees/`, and `op-e2e/e2eutils/` use fixed gas limits.
- `op-e2e/faultproofs/precompile_test.go` and `op-e2e/faultproofs/cannon_benchmark_test.go` use gas limits above the EIP-7825 cap.
- `op-e2e/e2eutils/disputegame/preimage/preimage_helper.go` uses 2,000,000 gas for large preimages.

### op-acceptance-tests and op-devstack

- `op-acceptance-tests/tests/isthmus/pectra/pectra_features_test.go` computes the EIP-7623 gas by hand and asserts it exactly.
- `op-acceptance-tests/tests/glamsterdam_test.go` and `op-acceptance-tests/tests/osaka_on_l2_test.go` use fixed limits. One limit is close to the EIP-7825 cap.
- `op-acceptance-tests/tests/sdm/` asserts exact raw and canonical gas used.
- Fixed gas limits also occur in these directories under `op-acceptance-tests/tests/`:
  - `interop/loadtest/`
  - `interop/filter/`
  - `interop/expiry/`
  - `superfaultproofs/`
  - `batcher/throttling/`
- `op-devstack/dsl/eoa.go` uses a fixed 75000 gas. Many acceptance tests use this helper.
- `op-devstack/sysgo/deployer.go` sets the devnet L1 genesis gas limit to 60M.

### Rust tests and dev mode

- **op-revm:**
  - Tests in `rust/op-revm/src/handler.rs`, `l1block.rs`, and `precompiles.rs` use exact intrinsic gas, calldata gas, and precompile gas.
  - `rust/revm-ee-tests/` snapshot files store exact gas results.
- **alloy-op-evm:** refund and gas cap tests are in `rust/alloy-op-evm/src/tests.rs`, `rust/alloy-op-evm/src/block/tests*`, and `rust/alloy-op-evm/src/env.rs`.
- **op-reth:**
  - The txpool, payload builder, EVM, consensus, and base fee crates have tests that use 21000 gas or exact fee values.
  - `rust/op-reth-test-engine/` expects every transfer to use exactly 21000 gas.
- **kona:**
  - Each file in `rust/kona/bin/client/src/fpvm_evm/precompiles/` has tests that check exact L2 charges and oracle gas.
  - The hardfork tests check upgrade gas totals, for example 55,370,657 for Karst.
  - `rust/kona/tests/proofs/` uses fixed deposit gas, `TxGas + 6`, and 40 gas for each calldata byte.
- **op-reth dev mode:** `rust/op-reth/crates/node/src/node.rs` uses a fixed L1 info transaction with 1,000,000 gas.
