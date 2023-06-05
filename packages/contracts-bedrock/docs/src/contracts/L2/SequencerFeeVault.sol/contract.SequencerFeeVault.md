# SequencerFeeVault
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/L2/SequencerFeeVault.sol)

**Inherits:**
[FeeVault](/contracts/universal/FeeVault.sol/abstract.FeeVault.md), [Semver](/contracts/universal/Semver.sol/contract.Semver.md)

The SequencerFeeVault is the contract that holds any fees paid to the Sequencer during
transaction processing and block production.


## Functions
### constructor


```solidity
constructor(address _recipient) FeeVault(_recipient, 10 ether) Semver(1, 1, 0);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_recipient`|`address`|Address that will receive the accumulated fees.|


### l1FeeWallet

Legacy getter for the recipient address.


```solidity
function l1FeeWallet() public view returns (address);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`address`|The recipient address.|


