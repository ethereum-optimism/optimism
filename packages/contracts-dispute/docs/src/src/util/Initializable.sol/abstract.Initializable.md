# Initializable
[Git Source](https://github.com/ethereum-optimism/optimism/blob/c6ae546047e96fbfd2d0f78febba2885aab34f5f/src/util/Initializable.sol)

**Inherits:**
[IInitializable](/src/interfaces/IInitializable.sol/interface.IInitializable.md)

**Author:**
clabby <https://github.com/clabby>

Enables a contract to have an initializer function that may only be ran once.


## State Variables
### initialized
Flag that designates whether or not the contract has been initialized.


```solidity
bool public initialized;
```


## Functions
### initializer

Only allows this contract to be initialized once.


```solidity
modifier initializer();
```

## Events
### Initialized
Emitted upon initialization of the contract.


```solidity
event Initialized();
```

## Errors
### AlreadyInitialized
Thrown when the contract has already been initialized.


```solidity
error AlreadyInitialized();
```

