# ERC721 NFT Bridging

BOBA NFT bridges consists of two bridge contracts. The [L1NFTBridge](https://github.com/bobanetwork/boba_legacy/blob/develop/packages/boba/contracts/contracts/ERC721Bridges/L1NFTBridge.sol) contract is deployed on L1 and the [L2NFTBridge](https://github.com/bobanetwork/boba_legacy/blob/develop/packages/boba/contracts/contracts/ERC721Bridges/L2NFTBridge.sol) contract is deployed on L2. It supports **native L1 NFTs** and **native L2 NFTs** to be moved back and forth.

* Native L1 NFT: the original NFT contract was deployed on L1
* Native L2 NFT: the original NFT contract was deployed on L2

Bridging an NFT to Boba takes several minutes, and bridging an NFT from Boba to Ethereum takes 7 days. **Not all NFTs are bridgeable - developers must use specialized NFT contracts (e.g. L2StandardERC721.sol) to enable this functionality.**

When deploying your L2StandardERC721, please take caution if you extend the contract with more features, as an incorrect implementation may result in loss of tokens. For instance, do not add a method that would allow updating the corresponding 'l1Contract' address for an L2StandardERC721. An update in between operation would deem the previous tokens to be locked on the bridge. Furthermore, The NFTBridge contracts use the information at the time of registration to obtain the l1Token information and send messages between the bridges.

<figure><img src="../../../assets/native l1 nft.png" alt=""><figcaption></figcaption></figure>

Assuming you have already deployed an NFT contract on L1, and you wish to transfer those NFTs to L2, please make sure that your L1 NFT contract is [ERC721](https://github.com/ethereum/EIPs/blob/master/EIPS/eip-721.md) compatible. Your contract must implement `ERC165` and `ERC721` interfaces. We will check the interface before registering your NFT contracts to our bridges.

```solidity
bytes4 erc721 = 0x80ac58cd;
require(ERC165Checker.supportsInterface(_l1Contract, erc721), "L1 NFT is not ERC721 compatible");
```

After verifying the interface, please deploy [L2StandardERC721](https://github.com/bobanetwork/boba/blob/release/v0.2.2/packages/boba/contracts/contracts/standards/L2StandardERC721.sol) on Boba. The `L1_NFT_CONTRACT_ADDRESS` is the address of your NFT on Ethereum.

```js
const Factory__L2StandardERC721 = new ethers.ContractFactory(
  L2StandardERC721.abi,
  L2StandardERC721.bytecode,
  L2Wallet
)
const L2StandardERC721 = await Factory__L2StandardERC721.deploy(
  L2_NFT_BRIDGE_ADDRESS,   // L2 NFT Bridge Address
  L1_NFT_CONTRACT_ADDRESS, // Your L1 NFT Address
  NFT_NAME,
  NFT_SYMBOL,
  BASE_URI
)
await L2StandardERC721.deployTransaction.wait()
```

If you want to deploy your own L2 NFT contract, please follow requirements:

* Your L2 NFT contract must be [ERC721](https://github.com/ethereum/EIPs/blob/master/EIPS/eip-721.md) compatible and implemented `ERC165` and `ERC721` interfaces.
*   The `mint` function in your L2 NFT contract should be overriden by.

    ```solidity
    function mint(address _to, uint256 _tokenId, bytes memory _data) public virtual override onlyL2Bridge {}
    ```

    The input must be `address _to, uint256 _tokenId, bytes memory _data`, even though you might need them all.
* In your L2 NFT contract, you must add the following code to bypass our interface check in our NFT bridge

```solidity
pragma solidity >0.7.5;

import "@openzeppelin/contracts/token/ERC721/ERC721.sol";
import "./IL2StandardERC721.sol";

contract L2StandardERC721 is IL2StandardERC721, ERC721 {
  // [This is mandatory]
  // This is your L1 NFT contract address. Only one pair of your NFT contracts can be registered in the NFT bridges.
  address public override l1Contract;
  // [This is not mandatory] You can use other names or other ways to only allow l2 NFT bridge to mint and burn tokens
  // This is L2 NFT brigde contract address
  address public l2Bridge;

  // [This is not mandatory]
  // Only l2Brigde (L2 NFT bridge) can mint or burn NFTs
  modifier onlyL2Bridge {
    require(msg.sender == l2Bridge, "Only L2 Bridge can mint and burn");
    _;
  }

  // [This is mandatory]
  // You must export this interface `IL2StandardERC721.l1Contract.selector`
  function supportsInterface(bytes4 _interfaceId) public view override(IERC165, ERC721) returns (bool) {
    bytes4 bridgingSupportedInterface = IL2StandardERC721.l1Contract.selector
      ^ IL2StandardERC721.mint.selector
      ^ IL2StandardERC721.burn.selector;
    return _interfaceId == bridgingSupportedInterface || super.supportsInterface(_interfaceId);
  }

  // [The input is mandatory] The input must be `address _to, uint256 _tokenId, bytes memory _data`
  // [SECURITY] Make sure that only L2 NFT bridge can mint tokens
  function mint(address _to, uint256 _tokenId, bytes memory _data) public virtual override onlyL2Bridge {
    _mint(_to, _tokenId);

    emit Mint(_to, _tokenId);
  }

  // [SECURITY] Make sure that only L2 NFT bridge can burn tokens
  function burn(uint256 _tokenId) public virtual override onlyL2Bridge {
  _burn(_tokenId);

  emit Burn(_tokenId);
  }
}
```

> NOTE: Once you have your L2 NFT contract address, please contact us so we can register that address in the L1 and L2 NFT bridges.

<figure><img src="../../../assets/native l2 nft.png" alt=""><figcaption></figcaption></figure>

Deploy your NFT on Boba and then deploy [L1StandardERC721](https://github.com/bobanetwork/boba\_legacy/blob/release/v0.2.2/packages/boba/contracts/contracts/standards/L1StandardERC721.sol) on Ethereum. The `L2_NFT_CONTRACT_ADDRESS` is the address of your NFT on Boba.

```js
const Factory__L1StandardERC721 = new ethers.ContractFactory(
  L1StandardERC721.abi,
  L1StandardERC721.bytecode,
  L1Wallet
)
const L1StandardERC721 = await Factory__L1StandardERC721.deploy(
  L1_NFT_BRIDGE_ADDRESS, // L1 NFT Bridge Address
  L2_NFT_CONTRACT_ADDRESS, // Your L2 NFT Address
  NFT_NAME,
  NFT_SYMBOL,
  BASE_URI
)
await L2StandardERC721.deployTransaction.wait()
```

If you want to deploy your own L1 NFT contract, please follow requirements:

* Your L1 NFT contract must be [ERC721](https://github.com/ethereum/EIPs/blob/master/EIPS/eip-721.md) compatible and implemented `ERC165` and `ERC721` interfaces.
*   The `mint` function in your L1 NFT contract should be overriden by.

    ```solidity
    function mint(address _to, uint256 _tokenId, bytes memory _data) public virtual override onlyL1Bridge {}
    ```

    The input must be `address _to, uint256 _tokenId, bytes memory _data`, even though you might need them all.
* In your L1 NFT contract, you must add the following code to bypass our interface check in our NFT bridge

```solidity
pragma solidity >0.7.5;

import "@openzeppelin/contracts/token/ERC721/ERC721.sol";
import "./IL1StandardERC721.sol";

contract L1StandardERC721 is IL1StandardERC721, ERC721 {
  // [This is mandatory]
  // This is your L1 NFT contract address. Only one pair of your NFT contracts can be registered in the NFT bridges.
  address public override l2Contract;
  // [This is not mandatory] You can use other names or other ways to only allow l1 NFT bridge to mint and burn tokens
  // This is L1 NFT brigde contract address
  address public l1Bridge;

  // [This is not mandatory]
  // Only l1Brigde (L1 NFT bridge) can mint or burn NFTs
  modifier onlyL1Bridge {
    require(msg.sender == l1Bridge, "Only L1 Bridge can mint and burn");
    _;
  }

  // [This is mandatory]
  // You must export this interface `IL1StandardERC721.l2Contract.selector`
  function supportsInterface(bytes4 _interfaceId) public view override(IERC165, ERC721) returns (bool) {
    bytes4 bridgingSupportedInterface = IL1StandardERC721.l2Contract.selector
      ^ IL1StandardERC721.mint.selector
      ^ IL1StandardERC721.burn.selector;

    return _interfaceId == bridgingSupportedInterface || super.supportsInterface(_interfaceId);
  }

  // [SECURITY] Make sure that only L1 NFT bridge can mint tokens
  // The input must be `address _to, uint256 _tokenId, bytes memory _data`
  function mint(address _to, uint256 _tokenId, bytes memory _data) public virtual override onlyL1Bridge {
    _mint(_to, _tokenId);

    emit Mint(_to, _tokenId);
  }

  // [The input is mandatory] The input must be `address _to, uint256 _tokenId, bytes memory _data`
  // [SECURITY] Make sure that only L1 NFT bridge can mint tokens
  function burn(uint256 _tokenId) public virtual override onlyL1Bridge {
    _burn(_tokenId);

    emit Burn(_tokenId);
  }
}
```

> NOTE: Once you have your L1 NFT contract address, please contact us so we can register that address in the L1 and L2 NFT bridges.

<figure><img src="../../../assets/how to bridge nfts.png" alt=""><figcaption></figcaption></figure>

### CASE 1 - Native L1 NFT - Bridge NFTs from Ethereum to Boba

First, users transfer their NFT to the L1 NFT Bridge, starting with an approval.

```js
const approveTx = await L1NFT.approve(L1_NFT_BRIDGE_ADDRESS, TOKEN_ID)
await approveTx.wait()
```

Users then call the `depositNFT` or `depositNFTTo` function to deposit NFT to L2. The NFT arrives on L2 after L1 conf blocks.

```js
const tx = await L1NFTBrige.depositNFT(
  L1_NFT_CONTRACT_ADDRESS,
  TOKEN_ID,
  9999999 // L2 gas
)
await tx.wait()
```

### CASE 2 - Native L1 NFT - Bridge NFTs from Boba to Ethereum

Prior to the exit, the L2 NFT Bridge burns the L2 NFT, so the first step is for the user to approve the transaction.

```js
const approveTx = await L2NFT.approve(L2_NFT_BRIDGE_ADDRESS, TOKEN_ID)
await approveTx.wait()
```

Users have to approve the Boba for the exit fee next. They then call the `withdraw` or `withdrawTo` function to exit the NFT from Boba to Ethereum. The NFT will arrive on L1 after the seven days.

```js
const exitFee = await BOBABillingContract.exitFee()
const approveBOBATx = await L2BOBAToken.approve(
  L2NFTBrige.address,
  exitFee
)
await approveBOBATx.wait()
const tx = await L2NFTBrige.withdraw(
  L2_NFT_CONTRACT_ADDRESS,
  TOKEN_ID,
  9999999 // L2 gas
)
await tx.wait()
```

### CASE 3 - Native L2 NFT - Bridge NFTs from Boba to Ethereum

Users have to transfer their NFTs to the L2 NFT Bridge, so they start by approving the transaction.

```js
const approveTx = await L2NFT.approve(L2_NFT_BRIDGE_ADDRESS, TOKEN_ID)
await approveTx.wait()
```

Users have to approve the Boba for the exit fee next. They then call the `withdraw` or `withdrawTo` function to exit NFT from L2. The NFT will arrive on L1 after the seven days.

```js
const exitFee = await BOBABillingContract.exitFee()
const approveBOBATx = await L2BOBAToken.approve(
  L2NFTBrige.address,
  exitFee
)
await approveBOBATx.wait()
const tx = await L2NFTBrige.withdraw(
  L2_NFT_CONTRACT_ADDRESS,
  TOKEN_ID,
  9999999 // L2 gas
)
await tx.wait()
```

### CASE 4 - Native L2 NFT - Bridge NFTs from Ethereum to Boba

The L1 NFT Bridge has to burn the L1 NFT, so the user needs to approve the transaction first.

```js
const approveTx = await L1NFT.approve(L1_NFT_BRIDGE_ADDRESS, TOKEN_ID)
await approveTx.wait()
```

Users then call the `depositNFT` or `depositNFTTo` function to deposit NFT to L2. The NFT arrives on L2 after L1 conf blocks.

```js
const tx = await L1NFTBrige.depositNFT(
  L1_NFT_CONTRACT_ADDRESS,
  TOKEN_ID,
  9999999 // L2 gas
)
await tx.wait()
```

<figure><img src="../../../assets/more advanced options.png" alt=""><figcaption></figcaption></figure>

Attempting to categorize ERC721s on the basis of metadata, we have:

1. ERC721 with derivable metaData (more common)

The general ERC721(like the one in the example) has the tokenURI in the form = 'baseURI' + 'tokenId' or is completely derivable on-chain from the tokenId

In this case, you don't really need to worry about transporting metadata between layers and hence you are already at best.

2. ERC721 with no metadata (non- ERC721Metadata)

Some ERC721 do not have metadata associated, in which case you surely do not need to worry about transporting metadata

3. ERC721 with non-derivable metaData (unrecoverable context)

These are the "special NFTs" that require some form of transportation of metadata between layers

The NFT Bridge provides with a special method 'withdrawWithExtraData' in comparision to 'withdraw' for the usual bridging to allow transporting the metadata when bridging the NFT to the other layer

_What metadata are actually bridged?_

When you chose to withdraw through the aforementioned method - the `tokenURI()` data will be encoded and passed on to the L1StandardERC721 for it to receive and handle it

### Optimisations

Bridging the tokenURI data as a whole might not be ecnomical always and depends on the size of the tokenURI. For example, bridging the tokenURI data for on-chain NFTs could be very costly. The NFTBridge, asks for a special method for this - `bridgeExtraData`, if this method is implemented on your native ERC721 contracts and returns some data that you would need to bridge to the other layer, the bridge will prioritize this over `tokenURI` potentially allowing to bridge seed data for generation on the L1 side.

### Making your ERC721 bridge extra data

To enable the bridge to pick up the exposed extra data that you would want to bridge

1. expose the method `bridgeExtraData()` on your contract. This can encode one/many unique seed data to be transported over to the other layer while bridging. For example, an on-chain contract that requires three unique integers to be transported for each tokenId can expose the data in a way similar to:

```javascript
function bridgeExtraData(uint256 tokenId) public view returns(bytes memory) {
  return abi.encode(data_1[tokenId], data_2[tokenId], data_3[tokenId]);
}
```

2. Modify ERC165 supportsInterface to return the bridgeExtraData(uint256) selector

For example,

```javascript
function supportsInterface(bytes4 _interfaceId) public view virtual override returns (bool) {
  bytes4 bridgingSupportedInterface = IL1StandardERC721.l2Contract.selector
    ^ IL1StandardERC721.mint.selector
    ^ IL1StandardERC721.burn.selector;

  return _interfaceId == this.bridgeExtraData.selector || super.supportsInterface(_interfaceId) || bridgingSupportedInterface;
}
```

On the other side, our NFT bridge mints the NFT using the function

```solidity
IL2StandardERC721(_l2Contract).mint(_to, _tokenId, _data);
```

so your NFT contract should decode `_data` and write into the smart contract.

To bridge the NFT with the extra data, you need to call these functions of NFT bridges:

```javascript
// This example is for the L1 native NFT. L2 native NFT is similar to this one.

// Approve L1 NFT contract first
const approveL1Tx = await L1NFT.approve(L1_NFT_BRIDGE_ADDRESS, TOKEN_ID)
await approveL1Tx.wait()

// Deposit with extra data
const depositTx = await L1NFTBrige.depositNFTWithExtraData(
  L1_NFT_CONTRACT_ADDRESS,
  TOKEN_ID,
  9999999 // L2 gas
)
await depositTx.wait()

// Or deposit to another wallet
const depositToTx = await L1NFTBrige.depositNFTWithExtraDataTo(
  L1_NFT_CONTRACT_ADDRESS,
  TARGET_WALLET_ADDRESS,
  TOKEN_ID,
  9999999 // L2 gas
)
await depositToTx.wait()

// Approve L2 NFT contract first
const approveL2Tx = await L2NFT.approve(L2_NFT_BRIDGE_ADDRESS, TOKEN_ID)
await approveL2Tx.wait()

// You can use the standard function to withdraw
const withdrawTx = await L2NFTBrige.withdraw(
  L2_NFT_CONTRACT_ADDRESS,
  TOKEN_ID,
  9999999 // L2 gas
)
await withdrawTx.wait()

// Or withdraw to another wallet
const withdrawToTx = await L2NFTBrige.withdrawTo(
  L2_NFT_CONTRACT_ADDRESS,
  TARGET_WALLET_ADDRESS,
  TOKEN_ID,
  9999999 // L2 gas
)
await withdrawToTx.wait()
```

<figure><img src="../../../assets/nft bridges addresses.png" alt=""><figcaption></figcaption></figure>

### Mainnet

#### Ethereum

| Layer | Contract Name        | Contract Address                           |
| ----- | -------------------- | ------------------------------------------ |
| L1    | Proxy\_\_L1NFTBridge | 0xC891F466e53f40603250837282eAE4e22aD5b088 |
| L2    | Proxy\_\_L2NFTBridge | 0xFB823b65D0Dc219fdC0d759172D1E098dA32f9eb |

#### BNB Mainnet

| Layer | Contract Name        | Contract Address                           |
| ----- | -------------------- | ------------------------------------------ |
| L1    | Proxy\_\_L1NFTBridge | 0x76bD545C03391d4e6E3d5cC2B5617c94C6038c86 |
| L2    | Proxy\_\_L2NFTBridge | 0xA774C3f4572C5BA93F75D802ea7Dc6F93228e5cc |

### Testnet

#### BNB Testnet

| Layer | Contract Name        | Contract Address                           |
| ----- | -------------------- | ------------------------------------------ |
| L1    | Proxy\_\_L1NFTBridge | 0x4c3f621d01c22658F711c70a12662ECDfCA5916A |
| L2    | Proxy\_\_L2NFTBridge | 0x6fA80303E479Ea2d705F4f241Ef162aA2F793e71 |
