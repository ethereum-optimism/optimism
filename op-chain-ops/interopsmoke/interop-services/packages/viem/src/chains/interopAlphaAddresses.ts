import type { AddressSet } from '@/addressSet.js'

export const interopAlpha0Addresses = {
  OpChainProxyAdmin: '0x025742b2546fc412e0777425c1e7c3cc64950043',
  AddressManager: '0xc42e6a86a0eff5efac331f53f92aed6ac2f32743',
  L1ERC721BridgeProxy: '0x3e7ad9192ded22e23dd5571f08e474c4308c279c',
  SystemConfigProxy: '0xf8ad68d102824e41e319afa79968e6ec1d538fa9',
  OptimismMintableERC20FactoryProxy:
    '0x08bfba2b29020b492112eddd321b570e7316da35',
  L1StandardBridgeProxy: '0xffec7459c6eb449492b2522f0dbf7afef5572002',
  L1CrossDomainMessengerProxy: '0x98a51b7137e27e7a79306cff193baa29166eef9e',
  OptimismPortalProxy: '0x7385d89d38ab79984e7c84fab9ce5e6f4815468a',
  DisputeGameFactoryProxy: '0xd3a4e353b2e2ef27889ec74a4a9439e9e7ee500f',
  AnchorStateRegistryProxy: '0x532731c54cbe660af2a551e4f174d1965597740f',
  FaultDisputeGame: '0x0000000000000000000000000000000000000000',
  PermissionedDisputeGame: '0xd12c7aba48a5dd9a2dea94319c460f7264bfd08b',
  DelayedWETHPermissionedGameProxy:
    '0x0111e03f511d304b0ff925eea07aef99a4f112e5',
  DelayedWETHPermissionlessGameProxy:
    '0x0000000000000000000000000000000000000000',
} as const satisfies AddressSet

export const interopAlpha1Addresses = {
  OpChainProxyAdmin: '0xd0e4d4e988615435448fff53922dd30c05d6763c',
  AddressManager: '0x29ab61ab9f749e138513b787e4fbe42077193bd2',
  L1ERC721BridgeProxy: '0x86ea8a1e7b4f5e819adeca9d0a89f321881426ee',
  SystemConfigProxy: '0x92ec84285039a152b105c40163aabe6719dce7c1',
  OptimismMintableERC20FactoryProxy:
    '0xb9d97f3e754048c5ff7e058ff537b04545eb21de',
  L1StandardBridgeProxy: '0x58b317164459e63fdc2cc7ef8d458384ab05314a',
  L1CrossDomainMessengerProxy: '0xe235aed0943f1a4eeb5de2fe2cb571710885e6b8',
  OptimismPortalProxy: '0x55f5c4653dbcde7d1254f9c690a5d761b315500c',
  DisputeGameFactoryProxy: '0xae9cbb584e1d7c245189b66058d2e5a67817c0df',
  AnchorStateRegistryProxy: '0x076cc832efd3540d353a4fe1bbb5d95aa62c1f97',
  FaultDisputeGame: '0x0000000000000000000000000000000000000000',
  PermissionedDisputeGame: '0x87a5b25f625f59f3786429a35217c182303119a0',
  DelayedWETHPermissionedGameProxy:
    '0x4587b4ed6b86c69b4a5cb6d7a52484c13cde180a',
  DelayedWETHPermissionlessGameProxy:
    '0x0000000000000000000000000000000000000000',
} as const satisfies AddressSet
