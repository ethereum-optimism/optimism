import { useQuery } from '@tanstack/react-query'
import type { Chain, Hash } from 'viem'
import { publicActionsL1 } from 'viem/op-stack'
import { usePublicClient, useTransactionReceipt } from 'wagmi'

export const useGetTimeToProve = ({
  transactionHash,
  l2Chain,
}: {
  transactionHash: Hash
  l2Chain: Chain
}) => {
  const { data: receipt } = useTransactionReceipt({
    hash: transactionHash,
    chainId: l2Chain.id,
  })

  const l1PublicClient = usePublicClient({ chainId: l2Chain.sourceId! })

  return useQuery({
    enabled: !!receipt && !!l1PublicClient,
    queryKey: ['get-time-to-prove', l2Chain.id, transactionHash],
    queryFn: async () => {
      if (!receipt || !l1PublicClient) return

      return await l1PublicClient.extend(publicActionsL1()).getTimeToProve({
        receipt,
        targetChain: l2Chain as any,
      })
    },
    refetchInterval: 6 * 1000,
  })
}
