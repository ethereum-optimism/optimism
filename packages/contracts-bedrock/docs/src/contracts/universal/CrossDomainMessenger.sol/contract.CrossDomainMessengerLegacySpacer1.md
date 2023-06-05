# CrossDomainMessengerLegacySpacer1
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/universal/CrossDomainMessenger.sol)

Contract only exists to add a spacer to the CrossDomainMessenger where the
PausableUpgradable and OwnableUpgradeable variables used to exist. Must be
the third contract in the inheritance tree of the CrossDomainMessenger.


## State Variables
### spacer_1_0_1600
Spacer for backwards compatibility. Comes from OpenZeppelin
ContextUpgradable.


```solidity
uint256[50] private spacer_1_0_1600;
```


### spacer_51_0_20
Spacer for backwards compatibility.
Come from OpenZeppelin OwnableUpgradeable.


```solidity
address private spacer_51_0_20;
```


### spacer_52_0_1568
Spacer for backwards compatibility. Comes from OpenZeppelin
OwnableUpgradeable.


```solidity
uint256[49] private spacer_52_0_1568;
```


### spacer_101_0_1
Spacer for backwards compatibility. Comes from OpenZeppelin
PausableUpgradable.


```solidity
bool private spacer_101_0_1;
```


### spacer_102_0_1568
Spacer for backwards compatibility. Comes from OpenZeppelin
PausableUpgradable.


```solidity
uint256[49] private spacer_102_0_1568;
```


### spacer_151_0_32
Spacer for backwards compatibility.


```solidity
uint256 private spacer_151_0_32;
```


### spacer_152_0_1568
Spacer for backwards compatibility.


```solidity
uint256[49] private spacer_152_0_1568;
```


### spacer_201_0_32
Spacer for backwards compatibility.


```solidity
mapping(bytes32 => bool) private spacer_201_0_32;
```


### spacer_202_0_32
Spacer for backwards compatibility.


```solidity
mapping(bytes32 => bool) private spacer_202_0_32;
```


