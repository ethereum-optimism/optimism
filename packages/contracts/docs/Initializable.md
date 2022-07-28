# Initializable







*This is a base contract to aid in writing upgradeable contracts, or any kind of contract that will be deployed behind a proxy. Since proxied contracts do not make use of a constructor, it&#39;s common to move constructor logic to an external initializer function, usually called `initialize`. It then becomes necessary to protect this initializer function so it can only be called once. The {initializer} modifier provided by this contract will have this effect. The initialization functions use a version number. Once a version number is used, it is consumed and cannot be reused. This mechanism prevents re-execution of each &quot;step&quot; but allows the creation of new initialization steps in case an upgrade adds a module that needs to be initialized. For example: [.hljs-theme-light.nopadding] ``` contract MyToken is ERC20Upgradeable {     function initialize() initializer public {         __ERC20_init(&quot;MyToken&quot;, &quot;MTK&quot;);     } } contract MyTokenV2 is MyToken, ERC20PermitUpgradeable {     function initializeV2() reinitializer(2) public {         __ERC20Permit_init(&quot;MyToken&quot;);     } } ``` TIP: To avoid leaving the proxy in an uninitialized state, the initializer function should be called as early as possible by providing the encoded function call as the `_data` argument to {ERC1967Proxy-constructor}. CAUTION: When used with inheritance, manual care must be taken to not invoke a parent initializer twice, or to ensure that all initializers are idempotent. This is not verified automatically as constructors are by Solidity. [CAUTION] ==== Avoid leaving a contract uninitialized. An uninitialized contract can be taken over by an attacker. This applies to both a proxy and its implementation contract, which may impact the proxy. To prevent the implementation contract from being used, you should invoke the {_disableInitializers} function in the constructor to automatically lock it when it is deployed: [.hljs-theme-light.nopadding] ```*


## Events

### Initialized

```solidity
event Initialized(uint8 version)
```



*Triggered when the contract has been initialized or reinitialized.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| version  | uint8 | undefined |



