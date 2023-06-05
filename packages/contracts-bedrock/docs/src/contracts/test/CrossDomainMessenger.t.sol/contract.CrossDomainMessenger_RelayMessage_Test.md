# CrossDomainMessenger_RelayMessage_Test
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/CrossDomainMessenger.t.sol)

**Inherits:**
[Messenger_Initializer](/contracts/test/CommonTest.t.sol/contract.Messenger_Initializer.md)

Fuzz tests re-entrancy into the CrossDomainMessenger relayMessage function.


## State Variables
### senderSlotIndex

```solidity
uint256 constant senderSlotIndex = 50;
```


### er

```solidity
ExternalRelay public er;
```


## Functions
### setUp


```solidity
function setUp() public override;
```

### testFuzz_relayMessageReenter_succeeds

*This test mocks an OptimismPortal call to the L1CrossDomainMessenger via
the relayMessage function. The relayMessage function will then use SafeCall's
callWithMinGas to call the target with call data packed in the callMessage.
For this test, the callWithMinGas will call the mock ExternalRelay test contract
defined above, executing the externalCallWithMinGas function which will try to
re-enter the CrossDomainMessenger's relayMessage function, resulting in that message
being recorded as failed.*


```solidity
function testFuzz_relayMessageReenter_succeeds(address _sender, uint256 _gasLimit) external;
```

