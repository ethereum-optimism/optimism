import type { AddressSet } from '@/addressSet.js'

// Sourced from ethereum-optimism/devnets dev/interop-jnt-v3/<chain>/chain.yaml.
// Upstream publishes *Impl addresses for OpChainProxyAdmin, AddressManager,
// FaultDisputeGame and PermissionedDisputeGame (no proxies) — those impl
// addresses are used here.

export const interopJntV3_0Addresses = {
  OpChainProxyAdmin: '0xd110472E846252409E6C3930319a9F2d8679e259',
  AddressManager: '0xEF897c77CC436917C3e258F8583d2e0234150eEe',
  L1ERC721BridgeProxy: '0x438C4e3F0e94779d8559FF6c1BaDa9fF152c42Bf',
  SystemConfigProxy: '0xb5f0810dCb1cCAA57499667f81B06143D9d57a05',
  OptimismMintableERC20FactoryProxy:
    '0xAeC9e1cAC87D2B9E0d5EB8AD4490840b1674bf6e',
  L1StandardBridgeProxy: '0xAF7090724c9F26e020CEeE97febD3b47001a2428',
  L1CrossDomainMessengerProxy: '0x1690C542967C5BAf4D03294A89F67f9B6d2A6614',
  OptimismPortalProxy: '0x0055E3B713A81852691025d42F2a204716878431',
  DisputeGameFactoryProxy: '0x722d9ef06A80C4ff1c5Eec472BbAe03ad4339991',
  AnchorStateRegistryProxy: '0xE7C0a5477dD3C66997F7Bf3274BfAd9635407f3c',
  FaultDisputeGame: '0x2DDA3584b51eF5236f7726Dea5A0FB6B3cA94AeC',
  PermissionedDisputeGame: '0xe1dFFCBE4e22B813F26d2106D943C102e7cAb87e',
  DelayedWETHPermissionedGameProxy:
    '0x7fAE29dFe61abA809E49fBA88aA9f1094F20A5Ad',
  DelayedWETHPermissionlessGameProxy:
    '0x7fAE29dFe61abA809E49fBA88aA9f1094F20A5Ad',
} as const satisfies AddressSet

export const interopJntV3_1Addresses = {
  OpChainProxyAdmin: '0x4cAfc18CBDDb8E4555f379B4C0C6316FD4766719',
  AddressManager: '0xDc97789D0E9762Bc5073FD51b722B3583f7104d0',
  L1ERC721BridgeProxy: '0x2f788AEF04f452A7135198fc36099458e1C2985E',
  SystemConfigProxy: '0x18c6ED43C97D4ed8bF21EE338C82016d1D5ef8f1',
  OptimismMintableERC20FactoryProxy:
    '0x180741D013c7566Aaa66cC9034d7DdcF437C6eaB',
  L1StandardBridgeProxy: '0x03Bc772A6418A5A9878ca72D301640c97eD9B12d',
  L1CrossDomainMessengerProxy: '0x7f2A729f63da15F2250c64aCB4B31Bc803Cc56dB',
  OptimismPortalProxy: '0xcAc9F00Dee6CF6D3585B8485f5b262D005fc7e46',
  DisputeGameFactoryProxy: '0xC346bc6A73787A3273594106291997bC6B08ffd4',
  AnchorStateRegistryProxy: '0x10fE2674Fac3524711f93518c541C790746Fe94c',
  FaultDisputeGame: '0x2DDA3584b51eF5236f7726Dea5A0FB6B3cA94AeC',
  PermissionedDisputeGame: '0xe1dFFCBE4e22B813F26d2106D943C102e7cAb87e',
  DelayedWETHPermissionedGameProxy:
    '0xA6Ee8cD362eCa08151a4D1b18d046DAd9d41369C',
  DelayedWETHPermissionlessGameProxy:
    '0xA6Ee8cD362eCa08151a4D1b18d046DAd9d41369C',
} as const satisfies AddressSet
