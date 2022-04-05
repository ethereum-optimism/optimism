# L1ERC721Bridge



> L1ERC721Bridge



*The L1 ERC721 Bridge is a contract which stores deposited L1 NFTs that are in use on L2. It synchronizes a corresponding L2 Bridge, informing it of deposits and listening to it for newly finalized withdrawals.*

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

### deposits

```solidity
function deposits(address, address, uint256) external view returns (bool)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined
| _1 | address | undefined
| _2 | uint256 | undefined

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bool | undefined

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

### initialize

```solidity
function initialize(address _l1messenger, address _l2ERC721Bridge) external nonpayable
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1messenger | address | L1 Messenger address being used for cross-chain communications.
| _l2ERC721Bridge | address | L2 ERC721 bridge address.

### l2ERC721Bridge

```solidity
function l2ERC721Bridge() external view returns (address)
```



*get the address of the corresponding L2 bridge contract.*


#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | Address of the corresponding L2 bridge contract.

### messenger

```solidity
function messenger() external view returns (address)
```






#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined



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



