# Encoding_Converter
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/invariants/Encoding.t.sol)


## State Variables
### failedRoundtripAToB

```solidity
bool public failedRoundtripAToB;
```


### failedRoundtripBToA

```solidity
bool public failedRoundtripBToA;
```


## Functions
### convertRoundTripAToB

Takes a pair of integers to be encoded into a versioned nonce with the
Encoding library and then decoded and updates the test contract's state
indicating if the round trip encoding failed.


```solidity
function convertRoundTripAToB(uint240 _nonce, uint16 _version) external;
```

### convertRoundTripBToA

Takes an integer representing a packed version and nonce and attempts
to decode them using the Encoding library before re-encoding and updates
the test contract's state indicating if the round trip encoding failed.


```solidity
function convertRoundTripBToA(uint256 _versionedNonce) external;
```

