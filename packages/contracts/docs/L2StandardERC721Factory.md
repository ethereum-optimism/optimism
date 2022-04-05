# L2StandardERC721Factory



> L2StandardERC721Factory



*Factory contract for creating standard L2 ERC721 representations of L1 ERC721s compatible with and working on the NFT bridge.*

## Methods

### createStandardL2ERC721

```solidity
function createStandardL2ERC721(address _l1Token, string _name, string _symbol) external nonpayable
```



*Creates an instance of the standard ERC721 token on L2.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token | address | Address of the corresponding L1 token.
| _name | string | ERC721 name.
| _symbol | string | ERC721 symbol.

### isStandardERC721

```solidity
function isStandardERC721(address) external view returns (bool)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bool | undefined

### standardERC721Mapping

```solidity
function standardERC721Mapping(address) external view returns (address)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined



## Events

### StandardL2ERC721Created

```solidity
event StandardL2ERC721Created(address indexed _l1Token, address indexed _l2Token)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token `indexed` | address | undefined |
| _l2Token `indexed` | address | undefined |



