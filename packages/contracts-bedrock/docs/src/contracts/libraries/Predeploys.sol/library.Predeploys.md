# Predeploys
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/libraries/Predeploys.sol)

Contains constant addresses for contracts that are pre-deployed to the L2 system.


## State Variables
### L2_TO_L1_MESSAGE_PASSER
Address of the L2ToL1MessagePasser predeploy.


```solidity
address internal constant L2_TO_L1_MESSAGE_PASSER = 0x4200000000000000000000000000000000000016;
```


### L2_CROSS_DOMAIN_MESSENGER
Address of the L2CrossDomainMessenger predeploy.


```solidity
address internal constant L2_CROSS_DOMAIN_MESSENGER = 0x4200000000000000000000000000000000000007;
```


### L2_STANDARD_BRIDGE
Address of the L2StandardBridge predeploy.


```solidity
address internal constant L2_STANDARD_BRIDGE = 0x4200000000000000000000000000000000000010;
```


### L2_ERC721_BRIDGE
Address of the L2ERC721Bridge predeploy.


```solidity
address internal constant L2_ERC721_BRIDGE = 0x4200000000000000000000000000000000000014;
```


### SEQUENCER_FEE_WALLET
Address of the SequencerFeeWallet predeploy.


```solidity
address internal constant SEQUENCER_FEE_WALLET = 0x4200000000000000000000000000000000000011;
```


### OPTIMISM_MINTABLE_ERC20_FACTORY
Address of the OptimismMintableERC20Factory predeploy.


```solidity
address internal constant OPTIMISM_MINTABLE_ERC20_FACTORY = 0x4200000000000000000000000000000000000012;
```


### OPTIMISM_MINTABLE_ERC721_FACTORY
Address of the OptimismMintableERC721Factory predeploy.


```solidity
address internal constant OPTIMISM_MINTABLE_ERC721_FACTORY = 0x4200000000000000000000000000000000000017;
```


### L1_BLOCK_ATTRIBUTES
Address of the L1Block predeploy.


```solidity
address internal constant L1_BLOCK_ATTRIBUTES = 0x4200000000000000000000000000000000000015;
```


### GAS_PRICE_ORACLE
Address of the GasPriceOracle predeploy. Includes fee information
and helpers for computing the L1 portion of the transaction fee.


```solidity
address internal constant GAS_PRICE_ORACLE = 0x420000000000000000000000000000000000000F;
```


### L1_MESSAGE_SENDER
Address of the L1MessageSender predeploy. Deprecated. Use L2CrossDomainMessenger
or access tx.origin (or msg.sender) in a L1 to L2 transaction instead.


```solidity
address internal constant L1_MESSAGE_SENDER = 0x4200000000000000000000000000000000000001;
```


### DEPLOYER_WHITELIST
Address of the DeployerWhitelist predeploy. No longer active.


```solidity
address internal constant DEPLOYER_WHITELIST = 0x4200000000000000000000000000000000000002;
```


### LEGACY_ERC20_ETH
Address of the LegacyERC20ETH predeploy. Deprecated. Balances are migrated to the
state trie as of the Bedrock upgrade. Contract has been locked and write functions
can no longer be accessed.


```solidity
address internal constant LEGACY_ERC20_ETH = 0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000;
```


### L1_BLOCK_NUMBER
Address of the L1BlockNumber predeploy. Deprecated. Use the L1Block predeploy
instead, which exposes more information about the L1 state.


```solidity
address internal constant L1_BLOCK_NUMBER = 0x4200000000000000000000000000000000000013;
```


### LEGACY_MESSAGE_PASSER
Address of the LegacyMessagePasser predeploy. Deprecate. Use the updated
L2ToL1MessagePasser contract instead.


```solidity
address internal constant LEGACY_MESSAGE_PASSER = 0x4200000000000000000000000000000000000000;
```


### PROXY_ADMIN
Address of the ProxyAdmin predeploy.


```solidity
address internal constant PROXY_ADMIN = 0x4200000000000000000000000000000000000018;
```


### BASE_FEE_VAULT
Address of the BaseFeeVault predeploy.


```solidity
address internal constant BASE_FEE_VAULT = 0x4200000000000000000000000000000000000019;
```


### L1_FEE_VAULT
Address of the L1FeeVault predeploy.


```solidity
address internal constant L1_FEE_VAULT = 0x420000000000000000000000000000000000001A;
```


### GOVERNANCE_TOKEN
Address of the GovernanceToken predeploy.


```solidity
address internal constant GOVERNANCE_TOKEN = 0x4200000000000000000000000000000000000042;
```


