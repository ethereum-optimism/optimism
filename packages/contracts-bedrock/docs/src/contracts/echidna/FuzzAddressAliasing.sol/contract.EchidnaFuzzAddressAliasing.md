# EchidnaFuzzAddressAliasing
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/echidna/FuzzAddressAliasing.sol)


## State Variables
### failedRoundtrip

```solidity
bool internal failedRoundtrip;
```


## Functions
### testRoundTrip

Takes an address to be aliased with AddressAliasHelper and then unaliased
and updates the test contract's state indicating if the round trip encoding
failed.


```solidity
function testRoundTrip(address addr) public;
```

### echidna_round_trip_aliasing


```solidity
function echidna_round_trip_aliasing() public view returns (bool);
```

