import { useQuery } from '@tanstack/react-query'
import type { Chain } from 'viem'
import { publicActionsL1 } from 'viem/op-stack'
import { usePublicClient } from 'wagmi'

export const useGetL2Output = ({
  l2BlockNumber,
  l2Chain,
}: {
  l2BlockNumber?: bigint
  l2Chain: Chain
}) => {
  const publicClientL1 = usePublicClient({ chainId: l2Chain.sourceId! })
  return useQuery({
    enabled: !!publicClientL1 && !!l2BlockNumber,
    queryKey: ['get-l2-output', l2Chain.id, l2BlockNumber?.toString()],
    queryFn: async () => {
      if (!publicClientL1 || l2BlockNumber === undefined) {
        return
      }
      return await publicClientL1.extend(publicActionsL1()).getL2Output({
        l2BlockNumber,
        targetChain: l2Chain as any,
      })
    },
  })
}
