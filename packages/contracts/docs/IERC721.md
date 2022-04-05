# IERC721







*Required interface of an ERC721 compliant contract.*

## Methods

### approve

```solidity
function approve(address to, uint256 tokenId) external nonpayable
```



*Gives permission to `to` to transfer `tokenId` token to another account. The approval is cleared when the token is transferred. Only a single account can be approved at a time, so approving the zero address clears previous approvals. Requirements: - The caller must own the token or be an approved operator. - `tokenId` must exist. Emits an {Approval} event.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| to | address | undefined
| tokenId | uint256 | undefined

### balanceOf

```solidity
function balanceOf(address owner) external view returns (uint256 balance)
```



*Returns the number of tokens in ``owner``&#39;s account.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| owner | address | undefined

#### Returns

| Name | Type | Description |
|---|---|---|
| balance | uint256 | undefined

### getApproved

```solidity
function getApproved(uint256 tokenId) external view returns (address operator)
```



*Returns the account approved for `tokenId` token. Requirements: - `tokenId` must exist.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| tokenId | uint256 | undefined

#### Returns

| Name | Type | Description |
|---|---|---|
| operator | address | undefined

### isApprovedForAll

```solidity
function isApprovedForAll(address owner, address operator) external view returns (bool)
```



*Returns if the `operator` is allowed to manage all of the assets of `owner`. See {setApprovalForAll}*

#### Parameters

| Name | Type | Description |
|---|---|---|
| owner | address | undefined
| operator | address | undefined

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bool | undefined

### ownerOf

```solidity
function ownerOf(uint256 tokenId) external view returns (address owner)
```



*Returns the owner of the `tokenId` token. Requirements: - `tokenId` must exist.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| tokenId | uint256 | undefined

#### Returns

| Name | Type | Description |
|---|---|---|
| owner | address | undefined

### safeTransferFrom

```solidity
function safeTransferFrom(address from, address to, uint256 tokenId, bytes data) external nonpayable
```



*Safely transfers `tokenId` token from `from` to `to`. Requirements: - `from` cannot be the zero address. - `to` cannot be the zero address. - `tokenId` token must exist and be owned by `from`. - If the caller is not `from`, it must be approved to move this token by either {approve} or {setApprovalForAll}. - If `to` refers to a smart contract, it must implement {IERC721Receiver-onERC721Received}, which is called upon a safe transfer. Emits a {Transfer} event.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| from | address | undefined
| to | address | undefined
| tokenId | uint256 | undefined
| data | bytes | undefined

### setApprovalForAll

```solidity
function setApprovalForAll(address operator, bool _approved) external nonpayable
```



*Approve or remove `operator` as an operator for the caller. Operators can call {transferFrom} or {safeTransferFrom} for any token owned by the caller. Requirements: - The `operator` cannot be the caller. Emits an {ApprovalForAll} event.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| operator | address | undefined
| _approved | bool | undefined

### supportsInterface

```solidity
function supportsInterface(bytes4 interfaceId) external view returns (bool)
```



*Returns true if this contract implements the interface defined by `interfaceId`. See the corresponding https://eips.ethereum.org/EIPS/eip-165#how-interfaces-are-identified[EIP section] to learn more about how these ids are created. This function call must use less than 30 000 gas.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| interfaceId | bytes4 | undefined

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bool | undefined

### transferFrom

```solidity
function transferFrom(address from, address to, uint256 tokenId) external nonpayable
```



*Transfers `tokenId` token from `from` to `to`. WARNING: Usage of this method is discouraged, use {safeTransferFrom} whenever possible. Requirements: - `from` cannot be the zero address. - `to` cannot be the zero address. - `tokenId` token must be owned by `from`. - If the caller is not `from`, it must be approved to move this token by either {approve} or {setApprovalForAll}. Emits a {Transfer} event.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| from | address | undefined
| to | address | undefined
| tokenId | uint256 | undefined



## Events

### Approval

```solidity
event Approval(address indexed owner, address indexed approved, uint256 indexed tokenId)
```



*Emitted when `owner` enables `approved` to manage the `tokenId` token.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| owner `indexed` | address | undefined |
| approved `indexed` | address | undefined |
| tokenId `indexed` | uint256 | undefined |

### ApprovalForAll

```solidity
event ApprovalForAll(address indexed owner, address indexed operator, bool approved)
```



*Emitted when `owner` enables or disables (`approved`) `operator` to manage all of its assets.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| owner `indexed` | address | undefined |
| operator `indexed` | address | undefined |
| approved  | bool | undefined |

### Transfer

```solidity
event Transfer(address indexed from, address indexed to, uint256 indexed tokenId)
```



*Emitted when `tokenId` token is transferred from `from` to `to`.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| from `indexed` | address | undefined |
| to `indexed` | address | undefined |
| tokenId `indexed` | uint256 | undefined |



