import { useQueryClient } from '@tanstack/react-query'

export const useInvalidateGetWithdrawalStatus = () => {
  const queryClient = useQueryClient()
  return {
    invalidate: (chainPairId: string, transactionHash: string) => {
      queryClient.invalidateQueries({
        queryKey: ['get-withdrawal-status', chainPairId, transactionHash],
      })
    },
  }
}
