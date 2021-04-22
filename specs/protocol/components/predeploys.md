### OVM_DeployerWhitelist

The Deployer Whitelist is a temporary predeploy used to provide additional safety during the initial phases of our mainnet roll out. It is owned by the Optimism team, and defines accounts which are allowed to deploy contracts on Layer2. The Execution Manager will only allow an ovmCREATE or ovmCREATE2 operation to proceed if the deployer's address whitelisted.

### OVM_ETH

The ETH predeploy provides an ERC20 interface for ETH deposited to Layer 2. Note that unlike on Layer 1, Layer 2 accounts do not have a balance field.

### OVM_L1MessageSender

The L1MessageSender is a predeploy contract running on L2. During the execution of cross-domain transaction from L1 to L2, it returns the address of the L1 account (either an EOA or contract) which sent the message to L2 via the Canonical Transaction Chain's `enqueue()` function.
This contract exclusively serves as a getter for the `ovmL1TXORIGIN` operation. This is necessary because there is no corresponding operation in the EVM which the the optimistic solidity compiler can be replaced with a call to the ExecutionManager's `ovmL1TXORIGIN()` function.

### OVM_L2ToL1MessagePasser

The L2 to L1 Message Passer is a utility contract which facilitate an L1 proof of the
of a message on L2. The L1 Cross Domain Messenger performs this proof in its
\_verifyStorageProof function, which verifies the existence of the transaction hash in this
contract's `sentMessages` mapping.

### OVM_ProxySequencerEntrypoint

The Proxy Sequencer Entrypoint is a predeployed proxy to the implementation of the
Sequencer Entrypoint. This will enable the Optimism team to upgrade the Sequencer Entrypoint
contract.

### OVM_SequencerEntrypoint

It accepts a more efficient compressed calldata format, which it decompresses and encodes to the standard EIP155 transaction format.
This contract is the implementation referenced by the Proxy Sequencer Entrypoint, thus enabling the Optimism team to upgrade the decompression of calldata from the Sequencer.

### ERC1820Registry

This contract has been included as a popular standard which MUST be deployed at a specific address using CREATE2. This is not achievable in the OVM as the bytecode will not be a perfect match.
See EIP-1820 for more information.
