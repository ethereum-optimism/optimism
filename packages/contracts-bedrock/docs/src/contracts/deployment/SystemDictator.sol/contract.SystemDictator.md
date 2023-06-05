# SystemDictator
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/deployment/SystemDictator.sol)

**Inherits:**
OwnableUpgradeable

The SystemDictator is responsible for coordinating the deployment of a full Bedrock
system. The SystemDictator is designed to support both fresh network deployments and
upgrades to existing pre-Bedrock systems.


## State Variables
### EXIT_1_NO_RETURN_STEP
Step after which exit 1 can no longer be used.


```solidity
uint8 public constant EXIT_1_NO_RETURN_STEP = 3;
```


### PROXY_TRANSFER_STEP
Step where proxy ownership is transferred.


```solidity
uint8 public constant PROXY_TRANSFER_STEP = 4;
```


### config
System configuration.


```solidity
DeployConfig public config;
```


### l2OutputOracleDynamicConfig
Dynamic configuration for the L2OutputOracle.


```solidity
L2OutputOracleDynamicConfig public l2OutputOracleDynamicConfig;
```


### optimismPortalDynamicConfig
Dynamic configuration for the OptimismPortal. Determines
if the system should be paused when initialized.


```solidity
bool public optimismPortalDynamicConfig;
```


### currentStep
Current step;


```solidity
uint8 public currentStep;
```


### dynamicConfigSet
Whether or not dynamic config has been set.


```solidity
bool public dynamicConfigSet;
```


### finalized
Whether or not the deployment is finalized.


```solidity
bool public finalized;
```


### exited
Whether or not the deployment has been exited.


```solidity
bool public exited;
```


### oldL1CrossDomainMessenger
Address of the old L1CrossDomainMessenger implementation.


```solidity
address public oldL1CrossDomainMessenger;
```


## Functions
### step

Checks that the current step is the expected step, then bumps the current step.


```solidity
modifier step(uint8 _step);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_step`|`uint8`|Current step.|


### constructor

Constructor required to ensure that the implementation of the SystemDictator is
initialized upon deployment.


```solidity
constructor();
```

### initialize


```solidity
function initialize(DeployConfig memory _config) public initializer;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_config`|`DeployConfig`|System configuration.|


### updateDynamicConfig

Allows the owner to update dynamic config.


```solidity
function updateDynamicConfig(
    L2OutputOracleDynamicConfig memory _l2OutputOracleDynamicConfig,
    bool _optimismPortalDynamicConfig
) external onlyOwner;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_l2OutputOracleDynamicConfig`|`L2OutputOracleDynamicConfig`|Dynamic L2OutputOracle config.|
|`_optimismPortalDynamicConfig`|`bool`|Dynamic OptimismPortal config.|


### step1

Configures the ProxyAdmin contract.


```solidity
function step1() public onlyOwner step(1);
```

### step2

Pauses the system by shutting down the L1CrossDomainMessenger and setting the
deposit halt flag to tell the Sequencer's DTL to stop accepting deposits.


```solidity
function step2() public onlyOwner step(2);
```

### step3

Removes deprecated addresses from the AddressManager.


```solidity
function step3() public onlyOwner step(EXIT_1_NO_RETURN_STEP);
```

### step4

Transfers system ownership to the ProxyAdmin.


```solidity
function step4() public onlyOwner step(PROXY_TRANSFER_STEP);
```

### step5

Upgrades and initializes proxy contracts.


```solidity
function step5() public onlyOwner step(5);
```

### phase1

Calls the first 2 steps of the migration process.


```solidity
function phase1() external onlyOwner;
```

### phase2

Calls the remaining steps of the migration process, and finalizes.


```solidity
function phase2() external onlyOwner;
```

### finalize

Tranfers admin ownership to the final owner.


```solidity
function finalize() public onlyOwner;
```

### exit1

First exit point, can only be called before step 3 is executed.


```solidity
function exit1() external onlyOwner;
```

## Structs
### GlobalConfig
Basic system configuration.


```solidity
struct GlobalConfig {
    AddressManager addressManager;
    ProxyAdmin proxyAdmin;
    address controller;
    address finalOwner;
}
```

### ProxyAddressConfig
Set of proxy addresses.


```solidity
struct ProxyAddressConfig {
    address l2OutputOracleProxy;
    address optimismPortalProxy;
    address l1CrossDomainMessengerProxy;
    address l1StandardBridgeProxy;
    address optimismMintableERC20FactoryProxy;
    address l1ERC721BridgeProxy;
    address systemConfigProxy;
}
```

### ImplementationAddressConfig
Set of implementation addresses.


```solidity
struct ImplementationAddressConfig {
    L2OutputOracle l2OutputOracleImpl;
    OptimismPortal optimismPortalImpl;
    L1CrossDomainMessenger l1CrossDomainMessengerImpl;
    L1StandardBridge l1StandardBridgeImpl;
    OptimismMintableERC20Factory optimismMintableERC20FactoryImpl;
    L1ERC721Bridge l1ERC721BridgeImpl;
    PortalSender portalSenderImpl;
    SystemConfig systemConfigImpl;
}
```

### L2OutputOracleDynamicConfig
Dynamic L2OutputOracle config.


```solidity
struct L2OutputOracleDynamicConfig {
    uint256 l2OutputOracleStartingBlockNumber;
    uint256 l2OutputOracleStartingTimestamp;
}
```

### SystemConfigConfig
Values for the system config contract.


```solidity
struct SystemConfigConfig {
    address owner;
    uint256 overhead;
    uint256 scalar;
    bytes32 batcherHash;
    uint64 gasLimit;
    address unsafeBlockSigner;
    ResourceMetering.ResourceConfig resourceConfig;
}
```

### DeployConfig
Combined system configuration.


```solidity
struct DeployConfig {
    GlobalConfig globalConfig;
    ProxyAddressConfig proxyAddressConfig;
    ImplementationAddressConfig implementationAddressConfig;
    SystemConfigConfig systemConfigConfig;
}
```

