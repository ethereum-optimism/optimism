# IERC721Receiver



> ERC721 token receiver interface



*Interface for any contract that wants to support safeTransfers from ERC721 asset contracts.*

## Methods

### onERC721Received

```solidity
function onERC721Received(address operator, address from, uint256 tokenId, bytes data) external nonpayable returns (bytes4)
```



*Whenever an {IERC721} `tokenId` token is transferred to this contract via {IERC721-safeTransferFrom} by `operator` from `from`, this function is called. It must return its Solidity selector to confirm the token transfer. If any other value is returned or the interface is not implemented by the recipient, the transfer will be reverted. The selector can be obtained in Solidity with `IERC721.onERC721Received.selector`.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| operator | address | undefined
| from | address | undefined
| tokenId | uint256 | undefined
| data | bytes | undefined

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bytes4 | undefined




