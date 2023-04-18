# AttestationDisputeGame
[Git Source](https://github.com/ethereum-optimism/optimism/blob/c6ae546047e96fbfd2d0f78febba2885aab34f5f/src/AttestationDisputeGame.sol)

**Inherits:**
[IAttestationDisputeGame](/src/interfaces/IAttestationDisputeGame.sol/interface.IAttestationDisputeGame.md), [Clone](/src/util/Clone.sol/contract.Clone.md), [Initializable](/src/util/Initializable.sol/abstract.Initializable.md), EIP712

**Authors:**
clabby <https://github.com/clabby>, refcell <https://github.com/refcell>

A contract for disputing the validity of a claim via permissioned attestations.


## State Variables
### DISPUTE_TYPE_HASH
The EIP-712 type hash for the `Dispute` struct.


```solidity
Hash public constant DISPUTE_TYPE_HASH = Hash.wrap(keccak256("Dispute(bytes32 outputRoot,uint256 l2BlockNumber)"));
```


### bondManager
The BondManager contract that is used to manage the bonds for this game.


```solidity
IBondManager public immutable bondManager;
```


### systemConfig
The L1's SystemConfig contract.


```solidity
SystemConfig public immutable systemConfig;
```


### l2OutputOracle
The L2OutputOracle contract.


```solidity
L2OutputOracle public immutable l2OutputOracle;
```


### createdAt
The timestamp that the DisputeGame contract was created at.


```solidity
Timestamp public createdAt;
```


### status
The current status of the game.


```solidity
GameStatus public status;
```


### attestationSubmitters
An array of addresses that have submitted positive attestations for the `rootClaim`.


```solidity
address[] public attestationSubmitters;
```


### challenges
A mapping of addresses from the `signerSet` to booleans signifying whether
or not they support the `rootClaim` being the valid output for `l2BlockNumber`.


```solidity
mapping(address => bool) public challenges;
```


## Functions
### constructor


```solidity
constructor(IBondManager _bondmanager, SystemConfig _systemConfig, L2OutputOracle _l2OutputOracle) EIP712;
```

### signerSet

The signer set consists of authorized public keys that may challenge the `rootClaim`.


```solidity
function signerSet() external pure override returns (address[] memory _signers);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`_signers`|`address[]`|An array of authorized signers.|


### challenge

Challenge the `rootClaim`.

*- If the `ecrecover`ed address that created the signature is not a part of the
signer set returned by `signerSet`, this function should revert.
- If the signature provided is the signature that breaches the signature threshold,
the function should call the `resolve` function to resolve the game as `CHALLENGER_WINS`.
- When the game resolves, the bond attached to the root claim should be distributed among
the signers who participated in challenging the invalid claim.*


```solidity
function challenge(bytes calldata signature) external;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`signature`|`bytes`|An EIP-712 signature committing to the `rootClaim` and `l2BlockNumber` (within the `extraData`) from a key that exists within the `signerSet`.|


### signatureThreshold

The amount of signatures required to successfully challenge the `rootClaim`
output proposal. Once this threshold is met by members of the `signerSet`
calling `challenge`, the game will be resolved to `CHALLENGER_WINS`.


```solidity
function signatureThreshold() public pure returns (uint16 _signatureThreshold);
```

### getTypedDataHash

Returns an Ethereum Signed Typed Data hash, as defined in EIP-712, for the
`Dispute` struct. This hash is signed by members of the `signerSet` to
issue a positive attestation for the `rootClaim`.


```solidity
function getTypedDataHash() public view returns (Hash _typedDataHash);
```

### initialize

Initializes the `DisputeGame_Fault` contract.


```solidity
function initialize() external initializer;
```

### version

Returns the semantic version of the DisputeGame contract.

*Current version: 0.0.1*


```solidity
function version() external pure override returns (string memory);
```

### resolve

If all necessary information has been gathered, this function should mark the game
status as either `CHALLENGER_WINS` or `DEFENDER_WINS` and return the status of
the resolved game. It is at this stage that the bonds should be awarded to the
necessary parties.

*May only be called if the `status` is `IN_PROGRESS`.*


```solidity
function resolve() public returns (GameStatus _status);
```

### gameType

Returns the type of proof system being used for the AttestationDisputeGame.

*The reference impl should be entirely different depending on the type (fault, validity)
i.e. The game type should indicate the security model.*


```solidity
function gameType() public pure override returns (GameType);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`GameType`|_gameType The type of proof system being used.|


### _domainNameAndVersion

Solady EIP712 override.


```solidity
function _domainNameAndVersion() internal pure override returns (string memory _name, string memory _version);
```

### extraData

Getter for the extra data. In the case of the AttestationDisputeGame, this is
just the L2 block number that the root claim commits to.

*`clones-with-immutable-args` argument #3*


```solidity
function extraData() external pure returns (bytes memory _extraData);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`_extraData`|`bytes`|Any extra data supplied to the dispute game contract by the creator.|


### rootClaim

Fetches the root claim from the calldata appended by the CWIA proxy.

*`clones-with-immutable-args` argument #2*


```solidity
function rootClaim() public pure returns (Claim _rootClaim);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`_rootClaim`|`Claim`|The root claim of the DisputeGame.|


### l2BlockNumber

Returns the L2 Block Number that the `rootClaim` commits to. Exists within the `extraData`.


```solidity
function l2BlockNumber() public pure returns (uint256 _l2BlockNumber);
```

