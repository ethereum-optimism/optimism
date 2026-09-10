import type { Address } from 'viem'
import { erc20Abi } from 'viem'
import { useReadContracts } from 'wagmi'

export const useTokenInfo = ({
  address,
  chainId,
}: {
  address: Address
  chainId: number
}) => {
  const result = useReadContracts({
    contracts: [
      {
        address,
        abi: erc20Abi,
        functionName: 'symbol',
        chainId,
      },
      {
        address,
        abi: erc20Abi,
        functionName: 'decimals',
        chainId,
      },
      {
        address,
        abi: erc20Abi,
        functionName: 'name',
        chainId,
      },
    ],
  })
  const [symbol, decimals, name] = result.data || []

  return {
    isLoading: result.isLoading,
    symbol: symbol?.result,
    decimals: decimals?.result,
    name: name?.result,
  }
}
