# Portal_Initializer
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/CommonTest.t.sol)

**Inherits:**
[L2OutputOracle_Initializer](/contracts/test/CommonTest.t.sol/contract.L2OutputOracle_Initializer.md)


## State Variables
### opImpl

```solidity
OptimismPortal internal opImpl;
```


### op

```solidity
OptimismPortal internal op;
```


### systemConfig

```solidity
SystemConfig systemConfig;
```


## Functions
### setUp


```solidity
function setUp() public virtual override;
```

## Events
### WithdrawalFinalized

```solidity
event WithdrawalFinalized(bytes32 indexed withdrawalHash, bool success);
```

### WithdrawalProven

```solidity
event WithdrawalProven(bytes32 indexed withdrawalHash, address indexed from, address indexed to);
```

