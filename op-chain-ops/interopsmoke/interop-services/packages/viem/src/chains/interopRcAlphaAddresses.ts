import type { AddressSet } from '@/addressSet.js'

export const interopRcAlpha0Addresses = {
  OpChainProxyAdmin: '0x8a2df05608b2ae0eb75809b210527dd1d2705e31',
  AddressManager: '0x436adaadea096e6c8f61ef3dcf76f83604a4495d',
  L1ERC721BridgeProxy: '0xb1eb4f07d7ef21952eb7736ab0f048d3dff34011',
  SystemConfigProxy: '0x38cfb302cda19fd376be2237d220d35c404a36ba',
  OptimismMintableERC20FactoryProxy:
    '0x35d43f9e909105168917b004726edb7dcbd6f275',
  L1StandardBridgeProxy: '0x6be35d2af0ad0e6955145da1214e16bbffca74d4',
  L1CrossDomainMessengerProxy: '0x3393ffdc32d1b4eb4eed15ea076af4941ca20870',
  OptimismPortalProxy: '0xbd80b66b60a6c6580aa0a92783bdb4c42b1405c4',
  DisputeGameFactoryProxy: '0x64b013223986389e1594369e01c9cd01667c4aac',
  AnchorStateRegistryProxy: '0x04b4318d950d669669520609be753d31bd4de88d',
  FaultDisputeGame: '0x0000000000000000000000000000000000000000',
  PermissionedDisputeGame: '0xf5dbdcbfea04961249425c7127fb9779e930daee',
  DelayedWETHPermissionedGameProxy:
    '0x652c61e8a59c218f8facfdf23379b0e851aacc56',
  DelayedWETHPermissionlessGameProxy:
    '0x0000000000000000000000000000000000000000',
} as const satisfies AddressSet

export const interopRcAlpha1Addresses = {
  OpChainProxyAdmin: '0x6220fd818ea70803b65dfaba7deb8e72c8fb396e',
  AddressManager: '0xed401f324b7b8d1bbbf453402969de37c2d20f6a',
  L1ERC721BridgeProxy: '0x879c7072c856a0f0efb71a6b8222da5edf993061',
  SystemConfigProxy: '0xce1da8571d67d139a8040eba35bef8cfd34a0f2f',
  OptimismMintableERC20FactoryProxy:
    '0xe7e22560b1945120705c50b573960ecb427ff2ee',
  L1StandardBridgeProxy: '0x690194141578fffc38598d60ddbeb24716c5a049',
  L1CrossDomainMessengerProxy: '0xd40ea45998b8abae23541249d2522ef97bbb604e',
  OptimismPortalProxy: '0x92b3b2d4032492cd177fefa20e67c64826ccbc70',
  DisputeGameFactoryProxy: '0xe0ff87008d8b6e77de390092f523b8fa412828bd',
  AnchorStateRegistryProxy: '0x352264750cad1dee3b91045150d4efecd2f36ede',
  FaultDisputeGame: '0x0000000000000000000000000000000000000000',
  PermissionedDisputeGame: '0x9a77e6f08585e5aec9fc64f1aae2469eea7ff420',
  DelayedWETHPermissionedGameProxy:
    '0x908f99cd4c0383db228033be08bac37a2c2c1953',
  DelayedWETHPermissionlessGameProxy:
    '0x0000000000000000000000000000000000000000',
} as const satisfies AddressSet
