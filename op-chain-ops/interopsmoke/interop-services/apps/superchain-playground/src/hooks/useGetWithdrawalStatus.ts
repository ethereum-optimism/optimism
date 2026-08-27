import { useQuery } from '@tanstack/react-query'
import type { Chain, TransactionReceipt } from 'viem'
import { publicActionsL1 } from 'viem/op-stack'
import { usePublicClient } from 'wagmi'

export const useGetWithdrawalStatus = ({
  transactionReceipt,
  l2Chain,
}: {
  transactionReceipt?: TransactionReceipt
  l2Chain: Chain
}) => {
  const l1PublicClient = usePublicClient({ chainId: l2Chain.sourceId! })

  return useQuery({
    enabled: !!transactionReceipt && l1PublicClient !== undefined,
    queryKey: [
      'get-withdrawal-status',
      l2Chain.id,
      transactionReceipt?.transactionHash,
    ],
    queryFn: async () => {
      if (!transactionReceipt || !l1PublicClient) {
        return
      }

      const withdrawalStatus = await l1PublicClient
        .extend(publicActionsL1())
        .getWithdrawalStatus({
          receipt: transactionReceipt,
          targetChain: l2Chain as any,
        })

      return withdrawalStatus
    },

    refetchInterval: 6 * 1000,
  })
}
