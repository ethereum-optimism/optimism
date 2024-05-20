# ERC1155 NFT Bridging

BOBA ERC1155 bridges consists of two bridge contracts. The [L1ERC1155Bridge](https://github.com/bobanetwork/boba\_legacy/blob/add-ERC1155-bridge/packages/boba/contracts/contracts/ERC1155Bridges/L1ERC1155Bridge.sol) contract is deployed on L1 and the [L2ERC1155Bridge](https://github.com/bobanetwork/boba\_legacy/blob/add-ERC1155-bridge/packages/boba/contracts/contracts/ERC1155Bridges/L2ERC1155Bridge.sol) contract is deployed on L2. It supports **native L1 ERC1155 tokens** and **native L2 ERC1155 tokens** to be moved back and forth.

* Native L1 ERC1155 token: the original token contract was deployed on L1
* Native L2 ERC1155 token: the original token contract was deployed on L2

Bridging a token to Boba takes several minutes, and bridging a token from Boba to Ethereum takes 7 days. **Not all tokens are bridgeable - developers must use specialized token contracts (e.g. L2StandardERC1155.sol) to enable this functionality.**

When deploying your L2StandardERC1155, please take caution if you extend the contract with more features, as an incorrect implementation may result in loss of tokens. For instance, do not add a method that would allow updating the corresponding 'l1Contract' address for an L2StandardERC1155. An update in between operation would deem the previous tokens to be locked on the bridge. Furthermore, The ERC1155Bridge contracts use the information at the time of registration to obtain the l1Token information and send messages between the bridges.

<figure><img src="../../../assets/native l1 erc1155.png" alt=""><figcaption></figcaption></figure>

Assuming you have already deployed an ERC1155 token contract on L1, and you wish to transfer those tokens to L2, please make sure that your L1 ERC1155 token contract is [ERC1155](https://github.com/ethereum/EIPs/blob/master/EIPS/eip-1155.md) compatible. Your contract must implement `ERC165` and `ERC721` interfaces. We will check the interface before registering your token contracts to our bridges.

```solidity
bytes4 erc1155 = 0xd9b67a26;
require(ERC165Checker.supportsInterface(_l1Contract, erc1155), "L1 token is not ERC1155 compatible");
```

After verifying the interface, please deploy [L2StandardERC1155](https://github.com/bobanetwork/boba\_legacy/blob/add-ERC1155-bridge/packages/boba/contracts/contracts/standards/L2StandardERC1155.sol) on Boba. The `L1_ERC1155_TOKEN_CONTRACT_ADDRESS` is the address of your token on Ethereum.

```js
const Factory__L2StandardERC1155 = new ethers.ContractFactory(
  L2StandardERC1155.abi,
  L2StandardERC1155.bytecode,
  L2Wallet
)
const L2StandardERC1155 = await Factory__L2StandardERC1155.deploy(
  L2_ERC1155_BRIDGE_ADDRESS,   // L2 ERC1155 Bridge Address
  L1_ERC1155_TOKEN_CONTRACT_ADDRESS, // Your L1 Token Address
  URI
)
await L2StandardERC1155.deployTransaction.wait()
```

If you want to deploy your own L2 ERC1155 token contract, please follow requirements:

* Your L2 token contract must be [ERC1155](https://github.com/ethereum/EIPs/blob/master/EIPS/eip-1155.md) compatible and implemented `ERC165` and `ERC1155` interfaces.
*   The following functions in your L2 ERC1155 contract should be overriden by

    ```solidity
    function mint(address _to, uint256 _tokenId, uint256 _amount, bytes memory _data) public virtual override onlyL2Bridge {
      _mint(_to, _tokenId, _amount, _data);
      emit Mint(_to, _tokenId, _amount);
    }
    
    function mintBatch(address _to, uint256[] memory _tokenIds, uint256[] memory _amounts, bytes memory _data) public virtual override onlyL2Bridge {
      _mintBatch(_to, _tokenIds, _amounts, _data);
      emit MintBatch(_to, _tokenIds, _amounts);
    }
    
    function burn(address _from, uint256 _tokenId, uint256 _amount) public virtual override onlyL2Bridge {
      _burn(_from, _tokenId, _amount);
      emit Burn(_from, _tokenId, _amount);
    }
    
    function burnBatch(address _from, uint256[] memory _tokenIds, uint256[] memory _amounts) public virtual override onlyL2Bridge {
      _burnBatch(_from, _tokenIds, _amounts);
      emit BurnBatch(_from, _tokenIds, _amounts);
    }
    ```
* In your L2 ERC1155 token contract, you must add the following code to bypass our interface check in our bridge

```solidity
pragma solidity >0.7.5;

import "@openzeppelin/contracts/token/ERC1155/ERC1155.sol";
import "./IL2StandardERC1155.sol";

contract L2StandardERC1155 is IL2StandardERC1155, ERC1155 {
  // [This is mandatory]
  // This is your L1 token contract address. Only one pair of your token contracts can be registered in the bridges.
  address public override l1Contract;
  // [This is not mandatory] You can use other names or other ways to only allow l2 bridge to mint and burn tokens
  // This is L2 brigde contract address
  address public l2Bridge;

  // [This is not mandatory]
  // Only l2Brigde (L2 bridge) can mint or burn tokens
  modifier onlyL2Bridge {
    require(msg.sender == l2Bridge, "Only L2 Bridge can mint and burn");
    _;
  }

  // [This is mandatory]
  // You must export this interface
  function supportsInterface(bytes4 _interfaceId) public view override(IERC165, ERC1155) returns (bool) {
    bytes4 bridgingSupportedInterface = IL2StandardERC1155.l1Contract.selector
      ^ IL2StandardERC1155.mint.selector
      ^ IL2StandardERC1155.burn.selector
      ^ IL2StandardERC1155.mintBatch.selector
      ^ IL2StandardERC1155.burnBatch.selector;
    return _interfaceId == bridgingSupportedInterface || super.supportsInterface(_interfaceId);
  }

  // [The input is mandatory] The input must be `address _to, uint256 _tokenId, uint256 _amount, bytes memory _data`
  // [SECURITY] Make sure that only L2 ERC1155 bridge can mint tokens
  function mint(address _to, uint256 _tokenId, uint256 _amount, bytes memory _data) public virtual override onlyL2Bridge {
    _mint(_to, _tokenId, _amount, _data);

    emit Mint(_to, _tokenId, _amount);
  }

  // [The input is mandatory] The input must be `address _to, uint256[] memory _tokenIds, uint256[] memory _amounts, bytes memory _data`
  // [SECURITY] Make sure that only L2 ERC1155 bridge can mint tokens
  function mintBatch(address _to, uint256[] memory _tokenIds, uint256[] memory _amounts, bytes memory _data) public virtual override onlyL2Bridge {
    _mintBatch(_to, _tokenIds, _amounts, _data);

    emit MintBatch(_to, _tokenIds, _amounts);
  }

  // [The input is mandatory] The input must be `address _from, uint256 _tokenId, uint256 _amount`
  // [SECURITY] Make sure that only L2 ERC1155 bridge can burn tokens
  function burn(address _from, uint256 _tokenId, uint256 _amount) public virtual override onlyL2Bridge {
    _burn(_from, _tokenId, _amount);

   emit Burn(_from, _tokenId, _amount);
  }

  // [The input is mandatory] The input must be `address _from, uint256[] memory _tokenIds, uint256[] memory _amounts`
  // [SECURITY] Make sure that only L2 ERC1155 bridge can burn tokens
  function burnBatch(address _from, uint256[] memory _tokenIds, uint256[] memory _amounts) public virtual override onlyL2Bridge {
    _burnBatch(_from, _tokenIds, _amounts);

    emit BurnBatch(_from, _tokenIds, _amounts);
  }
}
```

> NOTE: Once you have your L2 ERC1155 token contract address, please contact us so we can register that address in the L1 and L2 bridges.

<figure><img src="../../../assets/native l2 erc115.png" alt=""><figcaption></figcaption></figure>

Deploy your ERC115 token on Boba and then deploy [L1StandardERC1155](https://github.com/bobanetwork/boba\_legacy/blob/add-ERC1155-bridge/packages/boba/contracts/contracts/standards/L1StandardERC1155.sol) on Ethereum. The `L1_ERC1155_TOKEN_CONTRACT_ADDRESS` is the address of your token on Boba.

```js
const Factory__L1StandardERC1155 = new ethers.ContractFactory(
  L1StandardERC1155.abi,
  L1StandardERC1155.bytecode,
  L1Wallet
)
const L1StandardERC1155 = await Factory__L1StandardERC1155.deploy(
  L1_ERC1155_BRIDGE_ADDRESS,   // L1 ERC1155 Bridge Address
  L2_ERC1155_TOKEN_CONTRACT_ADDRESS, // Your L2 Token Address
  URI
)
await L1StandardERC1155.deployTransaction.wait()
```

If you want to deploy your own L1 ERC1155 token contract, please follow requirements:

* Your L1 token contract must be [ERC1155](https://github.com/ethereum/EIPs/blob/master/EIPS/eip-1155.md) compatible and implemented `ERC165` and `ERC1155` interfaces.
*   The following functions in your L1 ERC1155 contract should be overriden by

    ```solidity
    function mint(address _to, uint256 _tokenId, uint256 _amount, bytes memory _data) public virtual override onlyL1Bridge {
      _mint(_to, _tokenId, _amount, _data);
      emit Mint(_to, _tokenId, _amount);
    }
    
    function mintBatch(address _to, uint256[] memory _tokenIds, uint256[] memory _amounts, bytes memory _data) public virtual override onlyL1Bridge {
      _mintBatch(_to, _tokenIds, _amounts, _data);
      emit MintBatch(_to, _tokenIds, _amounts);
    }
    
    function burn(address _from, uint256 _tokenId, uint256 _amount) public virtual override onlyL1Bridge {
      _burn(_from, _tokenId, _amount);
      emit Burn(_from, _tokenId, _amount);
    }
    
    function burnBatch(address _from, uint256[] memory _tokenIds, uint256[] memory _amounts) public virtual override onlyL1Bridge {
      _burnBatch(_from, _tokenIds, _amounts);
      emit BurnBatch(_from, _tokenIds, _amounts);
    }
    ```
* In your L1 ERC1155 token contract, you must add the following code to bypass our interface check in our bridge

```solidity
pragma solidity >0.7.5;

import "@openzeppelin/contracts/token/ERC1155/ERC1155.sol";
import "./IL1StandardERC1155.sol";

contract L1StandardERC1155 is IL1StandardERC1155, ERC1155 {
  // [This is mandatory]
  // This is your L2 token contract address. Only one pair of your token contracts can be registered in the bridges.
  address public override l2Contract;
  // [This is not mandatory] You can use other names or other ways to only allow l2 bridge to mint and burn tokens
  // This is L1 brigde contract address
  address public l1Bridge;

  // [This is not mandatory]
  // Only l1Brigde (L1 bridge) can mint or burn tokens
  modifier onlyL1Bridge {
    require(msg.sender == l1Bridge, "Only L1 Bridge can mint and burn");
    _;
  }

  // [This is mandatory]
  // You must export this interface
  function supportsInterface(bytes4 _interfaceId) public view override(IERC165, ERC1155) returns (bool) {
    bytes4 bridgingSupportedInterface = IL1StandardERC1155.l1Contract.selector
      ^ IL1StandardERC1155.mint.selector
      ^ IL1StandardERC1155.burn.selector
      ^ IL1StandardERC1155.mintBatch.selector
      ^ IL1StandardERC1155.burnBatch.selector;
    return _interfaceId == bridgingSupportedInterface || super.supportsInterface(_interfaceId);
  }

  // [The input is mandatory] The input must be `address _to, uint256 _tokenId, uint256 _amount, bytes memory _data`
  // [SECURITY] Make sure that only L1 ERC1155 bridge can mint tokens
  function mint(address _to, uint256 _tokenId, uint256 _amount, bytes memory _data) public virtual override onlyL1Bridge {
    _mint(_to, _tokenId, _amount, _data);

    emit Mint(_to, _tokenId, _amount);
  }

  // [The input is mandatory] The input must be `address _to, uint256[] memory _tokenIds, uint256[] memory _amounts, bytes memory _data`
  // [SECURITY] Make sure that only L1 ERC1155 bridge can mint tokens
  function mintBatch(address _to, uint256[] memory _tokenIds, uint256[] memory _amounts, bytes memory _data) public virtual override onlyL1Bridge {
    _mintBatch(_to, _tokenIds, _amounts, _data);

    emit MintBatch(_to, _tokenIds, _amounts);
  }

  // [The input is mandatory] The input must be `address _from, uint256 _tokenId, uint256 _amount`
  // [SECURITY] Make sure that only L1 ERC1155 bridge can burn tokens
  function burn(address _from, uint256 _tokenId, uint256 _amount) public virtual override onlyL1Bridge {
    _burn(_from, _tokenId, _amount);

   emit Burn(_from, _tokenId, _amount);
  }

  // [The input is mandatory] The input must be `address _from, uint256[] memory _tokenIds, uint256[] memory _amounts`
  // [SECURITY] Make sure that only L1 ERC1155 bridge can burn tokens
  function burnBatch(address _from, uint256[] memory _tokenIds, uint256[] memory _amounts) public virtual override onlyL1Bridge {
    _burnBatch(_from, _tokenIds, _amounts);

    emit BurnBatch(_from, _tokenIds, _amounts);
  }
}
```

> NOTE: Once you have your L1 token contract address, please contact us so we can register that address in the L1 and L2 bridges.

<figure><img src="../../../assets/how to bridge erc1155.png" alt=""><figcaption></figcaption></figure>

### CASE 1 - Native L1 token - Bridge tokens from Ethereum to Boba

First, users transfer their token to the L1 Bridge, starting with an approval.

```js
const approveTx = await L1Token.setApprovalForAll(L1_ERC1155_BRIDGE_ADDRESS, true)
await approveTx.wait()
```

Users then call the `deposit` or `depositTo` function to deposit token to L2. The token arrives on L2 after L1 conf blocks.

```js
const tx = await L1ERC1155Brige.deposit(
  L1_ERC1155_TOKEN_CONTRACT_ADDRESS,
  TOKEN_ID,
  TOKEN_AMOUNT,
  DATA, // event data - you can pass `0x` if you don't want to emit any data in the events
  9999999 // L2 gas
)
await tx.wait()
```

### CASE 2 - Native L1 token - Bridge tokens from Boba to Ethereum

Prior to the exit, the L2 Bridge burns the L2 tokens, so the first step is for the user to approve the transaction.

```js
const approveTx = await L2Token.setApprovalForAll(L1_ERC1155_BRIDGE_ADDRESS, true)
await approveTx.wait()
```

Users have to approve the Boba for the exit fee next. They then call the `withdraw` or `withdrawTo` function to exit the tokens from Boba to Ethereum. The token will arrive on L1 after the seven days.

```js
const exitFee = await BOBABillingContract.exitFee()
const approveBOBATx = await L2BOBAToken.approve(
  L2ERC1155Brige.address,
  exitFee
)
await approveBOBATx.wait()
const tx = await L1ERC1155Brige.withdraw(
  L2_ERC1155_TOKEN_CONTRACT_ADDRESS,
  TOKEN_ID,
  TOKEN_AMOUNT,
  DATA, // event data - you can pass `0x` if you don't want to emit any data in the events
  9999999 // L2 gas
)
await tx.wait()
```

### CASE 3 - Native L2 token - Bridge tokens from Boba to Ethereum

Users have to transfer their tokens to the L2 Bridge, so they start by approving the transaction.

```js
const approveTx = await L2Token.setApprovalForAll(L2_ERC1155_BRIDGE_ADDRESS, TOKEN_ID)
await approveTx.wait()
```

Users have to approve the Boba for the exit fee next. They then call the `withdraw` or `withdrawTo` function to exit tokens from L2. The token will arrive on L1 after the seven days.

```js
const exitFee = await BOBABillingContract.exitFee()
const approveBOBATx = await L2BOBAToken.approve(
  L2ERC1155Brige.address,
  exitFee
)
await approveBOBATx.wait()
const tx = await L2ERC1155Brige.withdraw(
  L2_ERC1155_TOKEN_CONTRACT_ADDRESS,
  TOKEN_ID,
  TOKEN_AMOUNT,
  DATA, // event data - you can pass `0x` if you don't want to emit any data in the events
  9999999 // L2 gas
)
await tx.wait()
```

### CASE 4 - Native L2 token - Bridge tokens from Ethereum to Boba

The L1 Bridge has to burn the L1 tokens, so the user needs to approve the transaction first.

```js
const approveTx = await L2BOBAToken.setApprovalForAll(L1_ERC1155_BRIDGE_ADDRESS, true)
await approveTx.wait()
```

Users then call the `deposit` or `depositTo` function to deposit tokens to L2. The token arrives on L2 after L1 conf blocks.

```js
const tx = await L1ERC1155Brige.deposit(
  L1_ERC1155_TOKEN_CONTRACT_ADDRESS,
  TOKEN_ID,
  TOKEN_AMOUNT,
  DATA, // event data - you can pass `0x` if you don't want to emit any data in the events
  9999999 // L2 gas
)
await tx.wait()
```

### NOTE

To bridge tokens from Alt L2s to Alt L1, you need to add the BOBA as the value to cover the exit fee.

```javascript
const exitFee = await BOBABillingContract.exitFee()
const tx = await L2ERC1155Brige.withdraw(
  L2_ERC1155_TOKEN_CONTRACT_ADDRESS,
  TOKEN_ID,
  TOKEN_AMOUNT,
  DATA, // event data - you can pass `0x` if you don't want to emit any data in the events
  9999999, // L2 gas
  {value: exitFee} // exit fee
)
await tx.wait()
```

<figure><img src="../../../assets/erc1155 bridge addresses.png" alt=""><figcaption></figcaption></figure>

### Mainnet

#### Ethereum

| Layer | Contract Name            | Contract Address                           |
| ----- | ------------------------ | ------------------------------------------ |
| L1    | Proxy\_\_L1ERC1155Bridge | 0x1dF39152AC0e81aB100341cACC4dE4c372A550cb |
| L2    | Proxy\_\_L2ERC1155Bridge | 0x1dF39152AC0e81aB100341cACC4dE4c372A550cb |

#### BNB Mainnet

| Layer | Contract Name            | Contract Address                           |
| ----- | ------------------------ | ------------------------------------------ |
| L1    | Proxy\_\_L1ERC1155Bridge | 0x1dF39152AC0e81aB100341cACC4dE4c372A550cb |
| L2    | Proxy\_\_L2ERC1155Bridge | 0x1dF39152AC0e81aB100341cACC4dE4c372A550cb |

### Testnet

#### BNB Testnet

| Layer | Contract Name            | Contract Address                           |
| ----- | ------------------------ | ------------------------------------------ |
| L1    | Proxy\_\_L1ERC1155Bridge | 0x1dF39152AC0e81aB100341cACC4dE4c372A550cb |
| L2    | Proxy\_\_L2ERC1155Bridge | 0x1dF39152AC0e81aB100341cACC4dE4c372A550cb |
