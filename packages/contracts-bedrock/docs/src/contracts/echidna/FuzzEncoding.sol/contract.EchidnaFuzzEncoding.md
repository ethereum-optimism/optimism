# EchidnaFuzzEncoding
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/echidna/FuzzEncoding.sol)


## State Variables
### failedRoundtripAToB

```solidity
bool internal failedRoundtripAToB;
```


### failedRoundtripBToA

```solidity
bool internal failedRoundtripBToA;
```


## Functions
### testRoundTripAToB

Takes a pair of integers to be encoded into a versioned nonce with the
Encoding library and then decoded and updates the test contract's state
indicating if the round trip encoding failed.


```solidity
function testRoundTripAToB(uint240 _nonce, uint16 _version) public;
```

### testRoundTripBToA

Takes an integer representing a packed version and nonce and attempts
to decode them using the Encoding library before re-encoding and updates
the test contract's state indicating if the round trip encoding failed.


```solidity
function testRoundTripBToA(uint256 _versionedNonce) public;
```

### echidna_round_trip_encoding_AToB


```solidity
function echidna_round_trip_encoding_AToB() public view returns (bool);
```

### echidna_round_trip_encoding_BToA


```solidity
function echidna_round_trip_encoding_BToA() public view returns (bool);
```

