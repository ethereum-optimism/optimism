# IChainStorageContainer



> IChainStorageContainer





## Methods

### deleteElementsAfterInclusive

```solidity
function deleteElementsAfterInclusive(uint256 _index) external nonpayable
```

Removes all objects after and including a given index. Also allows setting the global metadata field.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _index | uint256 | Object index to delete from.

### get

```solidity
function get(uint256 _index) external view returns (bytes32)
```

Retrieves an object from the container.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _index | uint256 | Index of the particular object to access.

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bytes32 | 32 byte object value.

### getGlobalMetadata

```solidity
function getGlobalMetadata() external view returns (bytes27)
```

Retrieves the container&#39;s global metadata field.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bytes27 | Container global metadata field.

### length

```solidity
function length() external view returns (uint256)
```

Retrieves the number of objects stored in the container.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | Number of objects in the container.

### push

```solidity
function push(bytes32 _object) external nonpayable
```

Pushes an object into the container. Function allows setting the global metadata since we&#39;ll need to touch the &quot;length&quot; storage slot anyway, which also contains the global metadata (it&#39;s an optimization).



#### Parameters

| Name | Type | Description |
|---|---|---|
| _object | bytes32 | A 32 byte value to insert into the container.

### setGlobalMetadata

```solidity
function setGlobalMetadata(bytes27 _globalMetadata) external nonpayable
```

Sets the container&#39;s global metadata field. We&#39;re using `bytes27` here because we use five bytes to maintain the length of the underlying data structure, meaning we have an extra 27 bytes to store arbitrary data.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _globalMetadata | bytes27 | New global metadata to set.




