# ConfigurableCaller
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/CommonTest.t.sol)


## State Variables
### doRevert

```solidity
bool doRevert = true;
```


### target

```solidity
address target;
```


### payload

```solidity
bytes payload;
```


## Functions
### call

Call the configured target with the configured payload OR revert.


```solidity
function call() external;
```

### setDoRevert

Set whether or not to have `call` revert.


```solidity
function setDoRevert(bool _doRevert) external;
```

### setTarget

Set the target for the call made in `call`.


```solidity
function setTarget(address _target) external;
```

### setPayload

Set the payload for the call made in `call`.


```solidity
function setPayload(bytes calldata _payload) external;
```

### fallback

Fallback function that reverts if `doRevert` is true.
Otherwise, it does nothing.


```solidity
fallback() external;
```

## Events
### WhatHappened

```solidity
event WhatHappened(bool success, bytes returndata);
```

