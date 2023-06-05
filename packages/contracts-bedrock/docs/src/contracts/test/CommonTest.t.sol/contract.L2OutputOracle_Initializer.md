# L2OutputOracle_Initializer
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/CommonTest.t.sol)

**Inherits:**
[CommonTest](/contracts/test/CommonTest.t.sol/contract.CommonTest.md)


## State Variables
### oracle

```solidity
L2OutputOracle oracle;
```


### oracleImpl

```solidity
L2OutputOracle oracleImpl;
```


### messagePasser

```solidity
L2ToL1MessagePasser messagePasser = L2ToL1MessagePasser(payable(Predeploys.L2_TO_L1_MESSAGE_PASSER));
```


### proposer

```solidity
address internal proposer = 0x000000000000000000000000000000000000AbBa;
```


### owner

```solidity
address internal owner = 0x000000000000000000000000000000000000ACDC;
```


### submissionInterval

```solidity
uint256 internal submissionInterval = 1800;
```


### l2BlockTime

```solidity
uint256 internal l2BlockTime = 2;
```


### startingBlockNumber

```solidity
uint256 internal startingBlockNumber = 200;
```


### startingTimestamp

```solidity
uint256 internal startingTimestamp = 1000;
```


### guardian

```solidity
address guardian;
```


### initL1Time

```solidity
uint256 initL1Time;
```


## Functions
### warpToProposeTime


```solidity
function warpToProposeTime(uint256 _nextBlockNumber) public;
```

### setUp


```solidity
function setUp() public virtual override;
```

## Events
### OutputProposed

```solidity
event OutputProposed(
    bytes32 indexed outputRoot, uint256 indexed l2OutputIndex, uint256 indexed l2BlockNumber, uint256 l1Timestamp
);
```

### OutputsDeleted

```solidity
event OutputsDeleted(uint256 indexed prevNextOutputIndex, uint256 indexed newNextOutputIndex);
```

