# MerkleTrie
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/libraries/trie/MerkleTrie.sol)

MerkleTrie is a small library for verifying standard Ethereum Merkle-Patricia trie
inclusion proofs. By default, this library assumes a hexary trie. One can change the
trie radix constant to support other trie radixes.


## State Variables
### TREE_RADIX
Determines the number of elements per branch node.


```solidity
uint256 internal constant TREE_RADIX = 16;
```


### BRANCH_NODE_LENGTH
Branch nodes have TREE_RADIX elements and one value element.


```solidity
uint256 internal constant BRANCH_NODE_LENGTH = TREE_RADIX + 1;
```


### LEAF_OR_EXTENSION_NODE_LENGTH
Leaf nodes and extension nodes have two elements, a `path` and a `value`.


```solidity
uint256 internal constant LEAF_OR_EXTENSION_NODE_LENGTH = 2;
```


### PREFIX_EXTENSION_EVEN
Prefix for even-nibbled extension node paths.


```solidity
uint8 internal constant PREFIX_EXTENSION_EVEN = 0;
```


### PREFIX_EXTENSION_ODD
Prefix for odd-nibbled extension node paths.


```solidity
uint8 internal constant PREFIX_EXTENSION_ODD = 1;
```


### PREFIX_LEAF_EVEN
Prefix for even-nibbled leaf node paths.


```solidity
uint8 internal constant PREFIX_LEAF_EVEN = 2;
```


### PREFIX_LEAF_ODD
Prefix for odd-nibbled leaf node paths.


```solidity
uint8 internal constant PREFIX_LEAF_ODD = 3;
```


## Functions
### verifyInclusionProof

Verifies a proof that a given key/value pair is present in the trie.


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


### _parseProof

Parses an array of proof elements into a new array that contains both the original
encoded element and the RLP-decoded element.


```solidity
function _parseProof(bytes[] memory _proof) private pure returns (TrieNode[] memory);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_proof`|`bytes[]`|Array of proof elements to parse.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`TrieNode[]`|Proof parsed into easily accessible structs.|


### _getNodeID

Picks out the ID for a node. Node ID is referred to as the "hash" within the
specification, but nodes < 32 bytes are not actually hashed.


```solidity
function _getNodeID(RLPReader.RLPItem memory _node) private pure returns (bytes memory);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_node`|`RLPItem.RLPReader`|Node to pull an ID for.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bytes`|ID for the node, depending on the size of its contents.|


### _getNodePath

Gets the path for a leaf or extension node.


```solidity
function _getNodePath(TrieNode memory _node) private pure returns (bytes memory);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_node`|`TrieNode`|Node to get a path for.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bytes`|Node path, converted to an array of nibbles.|


### _getSharedNibbleLength

Utility; determines the number of nibbles shared between two nibble arrays.


```solidity
function _getSharedNibbleLength(bytes memory _a, bytes memory _b) private pure returns (uint256);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_a`|`bytes`|First nibble array.|
|`_b`|`bytes`|Second nibble array.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`uint256`|Number of shared nibbles.|


## Structs
### TrieNode
Struct representing a node in the trie.


```solidity
struct TrieNode {
    bytes encoded;
    RLPReader.RLPItem[] decoded;
}
```

