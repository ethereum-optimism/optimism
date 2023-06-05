# Constants
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/libraries/Constants.sol)

Constants is a library for storing constants. Simple! Don't put everything in here, just
the stuff used in multiple contracts. Constants that only apply to a single contract
should be defined in that contract instead.


## State Variables
### ESTIMATION_ADDRESS
Special address to be used as the tx origin for gas estimation calls in the
OptimismPortal and CrossDomainMessenger calls. You only need to use this address if
the minimum gas limit specified by the user is not actually enough to execute the
given message and you're attempting to estimate the actual necessary gas limit. We
use address(1) because it's the ecrecover precompile and therefore guaranteed to
never have any code on any EVM chain.


```solidity
address internal constant ESTIMATION_ADDRESS = address(1);
```


### DEFAULT_L2_SENDER
Value used for the L2 sender storage slot in both the OptimismPortal and the
CrossDomainMessenger contracts before an actual sender is set. This value is
non-zero to reduce the gas cost of message passing transactions.


```solidity
address internal constant DEFAULT_L2_SENDER = 0x000000000000000000000000000000000000dEaD;
```


## Functions
### DEFAULT_RESOURCE_CONFIG

Returns the default values for the ResourceConfig. These are the recommended values
for a production network.


```solidity
function DEFAULT_RESOURCE_CONFIG() internal pure returns (ResourceMetering.ResourceConfig memory);
```

