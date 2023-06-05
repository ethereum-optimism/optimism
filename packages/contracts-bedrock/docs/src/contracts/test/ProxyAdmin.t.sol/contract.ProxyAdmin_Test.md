# ProxyAdmin_Test
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/ProxyAdmin.t.sol)

**Inherits:**
Test


## State Variables
### alice

```solidity
address alice = address(64);
```


### proxy

```solidity
Proxy proxy;
```


### chugsplash

```solidity
L1ChugSplashProxy chugsplash;
```


### resolved

```solidity
ResolvedDelegateProxy resolved;
```


### addressManager

```solidity
AddressManager addressManager;
```


### admin

```solidity
ProxyAdmin admin;
```


### implementation

```solidity
SimpleStorage implementation;
```


## Functions
### setUp


```solidity
function setUp() external;
```

### test_setImplementationName_succeeds


```solidity
function test_setImplementationName_succeeds() external;
```

### test_setAddressManager_notOwner_reverts


```solidity
function test_setAddressManager_notOwner_reverts() external;
```

### test_setImplementationName_notOwner_reverts


```solidity
function test_setImplementationName_notOwner_reverts() external;
```

### test_setProxyType_notOwner_reverts


```solidity
function test_setProxyType_notOwner_reverts() external;
```

### test_owner_succeeds


```solidity
function test_owner_succeeds() external;
```

### test_proxyType_succeeds


```solidity
function test_proxyType_succeeds() external;
```

### test_erc1967GetProxyImplementation_succeeds


```solidity
function test_erc1967GetProxyImplementation_succeeds() external;
```

### test_chugsplashGetProxyImplementation_succeeds


```solidity
function test_chugsplashGetProxyImplementation_succeeds() external;
```

### test_delegateResolvedGetProxyImplementation_succeeds


```solidity
function test_delegateResolvedGetProxyImplementation_succeeds() external;
```

### getProxyImplementation


```solidity
function getProxyImplementation(address payable _proxy) internal;
```

### test_erc1967GetProxyAdmin_succeeds


```solidity
function test_erc1967GetProxyAdmin_succeeds() external;
```

### test_chugsplashGetProxyAdmin_succeeds


```solidity
function test_chugsplashGetProxyAdmin_succeeds() external;
```

### test_delegateResolvedGetProxyAdmin_succeeds


```solidity
function test_delegateResolvedGetProxyAdmin_succeeds() external;
```

### getProxyAdmin


```solidity
function getProxyAdmin(address payable _proxy) internal;
```

### test_erc1967ChangeProxyAdmin_succeeds


```solidity
function test_erc1967ChangeProxyAdmin_succeeds() external;
```

### test_chugsplashChangeProxyAdmin_succeeds


```solidity
function test_chugsplashChangeProxyAdmin_succeeds() external;
```

### test_delegateResolvedChangeProxyAdmin_succeeds


```solidity
function test_delegateResolvedChangeProxyAdmin_succeeds() external;
```

### changeProxyAdmin


```solidity
function changeProxyAdmin(address payable _proxy) internal;
```

### test_erc1967Upgrade_succeeds


```solidity
function test_erc1967Upgrade_succeeds() external;
```

### test_chugsplashUpgrade_succeeds


```solidity
function test_chugsplashUpgrade_succeeds() external;
```

### test_delegateResolvedUpgrade_succeeds


```solidity
function test_delegateResolvedUpgrade_succeeds() external;
```

### upgrade


```solidity
function upgrade(address payable _proxy) internal;
```

### test_erc1967UpgradeAndCall_succeeds


```solidity
function test_erc1967UpgradeAndCall_succeeds() external;
```

### test_chugsplashUpgradeAndCall_succeeds


```solidity
function test_chugsplashUpgradeAndCall_succeeds() external;
```

### test_delegateResolvedUpgradeAndCall_succeeds


```solidity
function test_delegateResolvedUpgradeAndCall_succeeds() external;
```

### upgradeAndCall


```solidity
function upgradeAndCall(address payable _proxy) internal;
```

### test_onlyOwner_notOwner_reverts


```solidity
function test_onlyOwner_notOwner_reverts() external;
```

### test_isUpgrading_succeeds


```solidity
function test_isUpgrading_succeeds() external;
```

