# LibGames
[Git Source](https://github.com/ethereum-optimism/optimism/blob/eaf1cde5896035c9ff0d32731da1e103f2f1c693/src/lib/LibGames.sol)

**Author:**
refcell <https://github.com/refcell>

This library contains constants for the different game types.


## State Variables
### FaultGameType
A FaultGameType is a dispute game that uses a fault proof to verify claims.


```solidity
GameType constant FaultGameType = GameType.wrap(bytes32(abi.encodePacked("Fault")));
```


### ValidityGameType
A ValidityGameType uses a validity proof to verify claims.


```solidity
GameType constant ValidityGameType = GameType.wrap(bytes32(abi.encodePacked("Validity")));
```


### AttestationGameType
An AttestationGameType is a permissioned set of attestors who verify claims.


```solidity
GameType constant AttestationGameType = GameType.wrap(bytes32(abi.encodePacked("Attestation")));
```


