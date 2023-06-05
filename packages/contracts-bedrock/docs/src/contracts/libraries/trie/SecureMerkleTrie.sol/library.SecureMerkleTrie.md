# SecureMerkleTrie
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/libraries/trie/SecureMerkleTrie.sol)

SecureMerkleTrie is a thin wrapper around the MerkleTrie library that hashes the input
keys. Ethereum's state trie hashes input keys before storing them.


## Functions
### verifyInclusionProof

Verifies a proof that a given key/value pair is present in the Merkle trie.


```solidity
function verifyInclusionProof(bytes memory _key, bytes memory _value, bytes[] memory _proof, bytes32 _root)
    internal
    pure
    returns (bool);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_key`|`bytes`|  Key of the node to search for, as a hex string.|
|`_value`|`bytes`|Value of the node to search for, as a hex string.|
|`_proof`|`bytes[]`|Merkle trie inclusion proof for the desired node. Unlike traditional Merkle trees, this proof is executed top-down and consists of a list of RLP-encoded nodes that make a path down to the target node.|
|`_root`|`bytes32`| Known root of the Merkle trie. Used to verify that the included proof is correctly constructed.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bool`|Whether or not the proof is valid.|


### get

Retrieves the value associated with a given key.


```solidity
function get(bytes memory _key, bytes[] memory _proof, bytes32 _root) internal pure returns (bytes memory);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_key`|`bytes`|  Key to search for, as hex bytes.|
|`_proof`|`bytes[]`|Merkle trie inclusion proof for the key.|
|`_root`|`bytes32`| Known root of the Merkle trie.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bytes`|Value of the key if it exists.|


### _getSecureKey

Computes the hashed version of the input key.


```solidity
function _getSecureKey(bytes memory _key) private pure returns (bytes memory);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_key`|`bytes`|Key to hash.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bytes`|Hashed version of the key.|


