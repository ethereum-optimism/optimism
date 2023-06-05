# StandardBridgeTester
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/StandardBridge.t.sol)

**Inherits:**
[StandardBridge](/contracts/universal/StandardBridge.sol/abstract.StandardBridge.md)

Simple wrapper around the StandardBridge contract that exposes
internal functions so they can be more easily tested directly.


## Functions
### constructor


```solidity
constructor(address payable _messenger, address payable _otherBridge) StandardBridge(_messenger, _otherBridge);
```

### isOptimismMintableERC20


```solidity
function isOptimismMintableERC20(address _token) external view returns (bool);
```

### isCorrectTokenPair


```solidity
function isCorrectTokenPair(address _mintableToken, address _otherToken) external view returns (bool);
```

### receive


```solidity
receive() external payable override;
```

