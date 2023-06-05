# EchidnaFuzzOptimismPortal
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/echidna/FuzzOptimismPortal.sol)


## State Variables
### portal

```solidity
OptimismPortal internal portal;
```


### failedToComplete

```solidity
bool internal failedToComplete;
```


## Functions
### constructor


```solidity
constructor();
```

### testDepositTransactionCompletes


```solidity
function testDepositTransactionCompletes(
    address _to,
    uint256 _mint,
    uint256 _value,
    uint64 _gasLimit,
    bool _isCreation,
    bytes memory _data
) public payable;
```

### echidna_deposit_completes


```solidity
function echidna_deposit_completes() public view returns (bool);
```

