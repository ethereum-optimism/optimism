

### OVM_DeployerWhitelist

The Deployer Whitelist is a temporary predeploy used to provide additional safety during the initial phases of our mainnet roll out. It is owned by the Optimism team, and defines accounts which are allowed to deploy contracts on Layer2. The Execution Manager will only allow an  ovmCREATE or ovmCREATE2 operation to proceed if the deployer's address whitelisted.

### OVM_ETH

The ETH predeploy provides an ERC20 interface for ETH deposited to Layer 2. Note that unlike on Layer 1, Layer 2 accounts do not have a balance field.

### OVM_L1MessageSender

The L1MessageSender is a predeploy contract running on L2. During the execution of cross  domain transaction from L1 to L2, it returns the address of the L1 account (either an EOA or contract) which sent the message to L2 via the Canonical Transaction Chain's `enqueue()`  function.

This contract exclusively serves as a getter for the ovmL1TXORIGIN operation. This is necessary  because there is no corresponding operation in the EVM which the the optimistic solidity compiler  can be replaced with a call to the ExecutionManager's ovmL1TXORIGIN() function.

### OVM_L2ToL1MessagePasser

### OVM_ProxySequencerEntrypoint



### OVM_SequencerEntrypoint

Handles


### ERC1820Registry

- Broken, needs to be removed or ovm compiled.
