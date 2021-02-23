# Changelog

## v0.1.9

Standardized ETH and ERC20 Gateways.

- Add ETH deposit contract.
- Add standard deposit/withdrawal interfaces.

## v0.1.5

Various cleanup and maintenance tasks.

- Improving comments and some names (#211)
- Add descriptive comments above the contract declaration for all 'non-abstract contracts' (#200)
- Add generic mock xdomain messenger (#209)
- Move everything over to hardhat (#208)
- Add comment to document v argument (#199)
- Add security related comments (#191)

## v0.1.4

Fix single contract redeployment & state dump script for
mainnet.

## v0.1.3

Add events to fraud proof initialization and finalization.

## v0.1.2

Npm publish integrity.

## v0.1.1

Audit fixes, deployment fixes & final parameterization.

- Add build mainnet command to package.json (#186)
- revert chain ID 422 -> 420 (#185)
- add `AddressSet` event (#184)
- Add mint & burn to L2 ETH (#178)
- Wait for deploy transactions (#180)
- Final Parameterization of Constants (#176)
- re-enable monotonicity tests (#177)
- make ovmSETNONCE notStatic (#179)
- Add reentry protection to ExecutionManager.run() (#175)
- Add nonReentrant to `relayMessage()` (#172)
- ctc: public getters, remove dead variable (#174)
- fix tainted memory bug in `Lib_BytesUtils.slice` (#171)

## v0.1.0

Initial Release
