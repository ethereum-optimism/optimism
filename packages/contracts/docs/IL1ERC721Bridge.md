# IL1ERC721Bridge



> IL1ERC721Bridge





## Methods

### depositERC721

```solidity
function depositERC721(address _l1Token, address _l2Token, uint256 _tokenId, uint32 _l2Gas, bytes _data) external nonpayable
```



*deposit the ERC721 token to the caller on L2.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token | address | Address of the L1 ERC721 we are depositing
| _l2Token | address | Address of the L1 respective L2 ERC721
| _tokenId | uint256 | Token ID of the ERC721 to deposit
| _l2Gas | uint32 | Gas limit required to complete the deposit on L2.
| _data | bytes | Optional data to forward to L2. This data is provided        solely as a convenience for external contracts. Aside from enforcing a maximum        length, these contracts provide no guarantees about its content.

### depositERC721To

```solidity
function depositERC721To(address _l1Token, address _l2Token, address _to, uint256 _tokenId, uint32 _l2Gas, bytes _data) external nonpayable
```



*deposit the ERC721 token to a recipient on L2.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token | address | Address of the L1 ERC721 we are depositing
| _l2Token | address | Address of the L1 respective L2 ERC721
| _to | address | L2 address to credit the withdrawal to.
| _tokenId | uint256 | Token ID of the ERC721 to deposit.
| _l2Gas | uint32 | Gas limit required to complete the deposit on L2.
| _data | bytes | Optional data to forward to L2. This data is provided        solely as a convenience for external contracts. Aside from enforcing a maximum        length, these contracts provide no guarantees about its content.

### finalizeERC721Withdrawal

```solidity
function finalizeERC721Withdrawal(address _l1Token, address _l2Token, address _from, address _to, uint256 _tokenId, bytes _data) external nonpayable
```



*Complete a withdrawal from L2 to L1, and send the ERC721 token to the recipient on L1 This call will fail if the initialized withdrawal from L2 has not been finalized.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token | address | Address of L1 token to finalizeWithdrawal for.
| _l2Token | address | Address of L2 token where withdrawal was initiated.
| _from | address | L2 address initiating the transfer.
| _to | address | L1 address to credit the withdrawal to.
| _tokenId | uint256 | Token ID of the ERC721 to deposit.
| _data | bytes | Data provided by the sender on L2. This data is provided   solely as a convenience for external contracts. Aside from enforcing a maximum   length, these contracts provide no guarantees about its content.

### l2ERC721Bridge

```solidity
function l2ERC721Bridge() external nonpayable returns (address)
```



*get the address of the corresponding L2 bridge contract.*


#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | Address of the corresponding L2 bridge contract.



## Events

### ERC721DepositInitiated

```solidity
event ERC721DepositInitiated(address indexed _l1Token, address indexed _l2Token, address indexed _from, address _to, uint256 _tokenId, bytes _data)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token `indexed` | address | undefined |
| _l2Token `indexed` | address | undefined |
| _from `indexed` | address | undefined |
| _to  | address | undefined |
| _tokenId  | uint256 | undefined |
| _data  | bytes | undefined |

### ERC721WithdrawalFinalized

```solidity
event ERC721WithdrawalFinalized(address indexed _l1Token, address indexed _l2Token, address indexed _from, address _to, uint256 _tokenId, bytes _data)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token `indexed` | address | undefined |
| _l2Token `indexed` | address | undefined |
| _from `indexed` | address | undefined |
| _to  | address | undefined |
| _tokenId  | uint256 | undefined |
| _data  | bytes | undefined |



