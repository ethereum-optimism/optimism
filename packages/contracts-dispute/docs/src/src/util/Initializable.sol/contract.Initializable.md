# Initializable
[Git Source](https://github.com/ethereum-optimism/cannon-v2-contracts/blob/896a9e7a2e7769b1273deb0b0a9ed4c533f56f75/src/util/Initializable.sol)

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

