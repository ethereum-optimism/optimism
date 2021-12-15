# TestLib_SecureMerkleTrie



> TestLib_SecureMerkleTrie





## Methods

### get

```solidity
function get(bytes _key, bytes _proof, bytes32 _root) external pure returns (bool, bytes)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _key | bytes | undefined
| _proof | bytes | undefined
| _root | bytes32 | undefined

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bool | undefined
| _1 | bytes | undefined

### getSingleNodeRootHash

```solidity
function getSingleNodeRootHash(bytes _key, bytes _value) external pure returns (bytes32)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _key | bytes | undefined
| _value | bytes | undefined

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bytes32 | undefined

### update

```solidity
function update(bytes _key, bytes _value, bytes _proof, bytes32 _root) external pure returns (bytes32)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _key | bytes | undefined
| _value | bytes | undefined
| _proof | bytes | undefined
| _root | bytes32 | undefined

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bytes32 | undefined

### verifyInclusionProof

```solidity
function verifyInclusionProof(bytes _key, bytes _value, bytes _proof, bytes32 _root) external pure returns (bool)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _key | bytes | undefined
| _value | bytes | undefined
| _proof | bytes | undefined
| _root | bytes32 | undefined

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bool | undefined




