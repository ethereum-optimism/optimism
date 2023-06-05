# Bytes_slice_Test
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/Bytes.t.sol)

**Inherits:**
Test


## Functions
### test_slice_fromZeroIdx_works

Tests that the `slice` function works as expected when starting from index 0.


```solidity
function test_slice_fromZeroIdx_works() public;
```

### test_slice_fromNonZeroIdx_works

Tests that the `slice` function works as expected when starting from indices [1, 9]
with lengths [1, 9], in reverse order.


```solidity
function test_slice_fromNonZeroIdx_works() public;
```

### test_slice_acrossWords_works

Tests that the `slice` function works as expected when slicing between multiple words
in memory. In this case, we test that a 2 byte slice between the 32nd byte of the
first word and the 1st byte of the second word is correct.


```solidity
function test_slice_acrossWords_works() public;
```

### test_slice_acrossMultipleWords_works

Tests that the `slice` function works as expected when slicing between multiple
words in memory. In this case, we test that a 34 byte slice between 3 separate words
returns the correct result.


```solidity
function test_slice_acrossMultipleWords_works() public;
```

### testFuzz_slice_outOfBounds_reverts

Tests that, when given an input bytes array of length `n`, the `slice` function will
always revert if `_start + _length > n`.


```solidity
function testFuzz_slice_outOfBounds_reverts(bytes memory _input, uint256 _start, uint256 _length) public;
```

### testFuzz_slice_lengthOverflows_reverts

Tests that, when given a length `n` that is greater than `type(uint256).max - 31`,
the `slice` function reverts.


```solidity
function testFuzz_slice_lengthOverflows_reverts(bytes memory _input, uint256 _start, uint256 _length) public;
```

### testFuzz_slice_rangeOverflows_reverts

Tests that, when given a start index `n` that is greater than
`type(uint256).max - n`, the `slice` function reverts.


```solidity
function testFuzz_slice_rangeOverflows_reverts(bytes memory _input, uint256 _start, uint256 _length) public;
```

### testFuzz_slice_memorySafety_succeeds

Tests that the `slice` function correctly updates the free memory pointer depending
on the length of the slice.


```solidity
function testFuzz_slice_memorySafety_succeeds(bytes memory _input, uint256 _start, uint256 _length) public;
```

