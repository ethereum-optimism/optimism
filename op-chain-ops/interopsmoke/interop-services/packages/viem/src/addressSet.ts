import type { Address } from 'viem'

export type AddressSet = {
  OpChainProxyAdmin: Address
  AddressManager: Address
  L1ERC721BridgeProxy: Address
  SystemConfigProxy: Address
  OptimismMintableERC20FactoryProxy: Address
  L1StandardBridgeProxy: Address
  L1CrossDomainMessengerProxy: Address
  OptimismPortalProxy: Address
  DisputeGameFactoryProxy: Address
  AnchorStateRegistryProxy: Address
  FaultDisputeGame: Address
  PermissionedDisputeGame: Address
  DelayedWETHPermissionedGameProxy: Address
  DelayedWETHPermissionlessGameProxy: Address
}

export const addressesToViemContractConstant = (
  addressSet: AddressSet,
  sourceId: number,
) => {
  return {
    opChainProxyAdmin: addressForChain(addressSet.OpChainProxyAdmin, sourceId),
    addressManager: addressForChain(addressSet.AddressManager, sourceId),
    l1Erc721BridgeProxy: addressForChain(
      addressSet.L1ERC721BridgeProxy,
      sourceId,
    ),
    systemConfig: addressForChain(addressSet.SystemConfigProxy, sourceId),
    optimismMintableErc20FactoryProxy: addressForChain(
      addressSet.OptimismMintableERC20FactoryProxy,
      sourceId,
    ),
    l1StandardBridge: addressForChain(
      addressSet.L1StandardBridgeProxy,
      sourceId,
    ),
    l1CrossDomainMessenger: addressForChain(
      addressSet.L1CrossDomainMessengerProxy,
      sourceId,
    ),
    optimismPortal: addressForChain(addressSet.OptimismPortalProxy, sourceId),
    disputeGameFactory: addressForChain(
      addressSet.DisputeGameFactoryProxy,
      sourceId,
    ),
    anchorStateRegistry: addressForChain(
      addressSet.AnchorStateRegistryProxy,
      sourceId,
    ),
    faultDisputeGame: addressForChain(addressSet.FaultDisputeGame, sourceId),
    permissionedDisputeGame: addressForChain(
      addressSet.PermissionedDisputeGame,
      sourceId,
    ),
  } as const
}

const addressForChain = (address: Address, chainId: number) => {
  return {
    [chainId]: { address },
  }
}
