# Initializable







*This is a base contract to aid in writing upgradeable contracts, or any kind of contract that will be deployed behind a proxy. Since a proxied contract can&#39;t have a constructor, it&#39;s common to move constructor logic to an external initializer function, usually called `initialize`. It then becomes necessary to protect this initializer function so it can only be called once. The {initializer} modifier provided by this contract will have this effect. TIP: To avoid leaving the proxy in an uninitialized state, the initializer function should be called as early as possible by providing the encoded function call as the `_data` argument to {ERC1967Proxy-constructor}. CAUTION: When used with inheritance, manual care must be taken to not invoke a parent initializer twice, or to ensure that all initializers are idempotent. This is not verified automatically as constructors are by Solidity. [CAUTION] ==== Avoid leaving a contract uninitialized. An uninitialized contract can be taken over by an attacker. This applies to both a proxy and its implementation contract, which may impact the proxy. To initialize the implementation contract, you can either invoke the initializer manually, or you can include a constructor to automatically mark it as initialized when it is deployed: [.hljs-theme-light.nopadding] ```*



