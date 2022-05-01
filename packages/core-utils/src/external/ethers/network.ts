import { Provider } from '@ethersproject/abstract-provider'

export const getChainId = async (provider: Provider): Promise<number> => {
  const network = await provider.getNetwork()
  return network.chainId
}
