import { useMutation } from '@tanstack/react-query'
import { getWalletClient, switchChain } from '@wagmi/core'
import { CheckCircle, Loader2 } from 'lucide-react'
import type { Chain, Hash } from 'viem'
import type { BuildProveWithdrawalReturnType } from 'viem/op-stack'
import { walletActionsL1 } from 'viem/op-stack'
import { useTransactionReceipt, useWaitForTransactionReceipt } from 'wagmi'

import { Button } from '@/components/ui/button'
import { useBuildProveWithdrawal } from '@/hooks/useBuildProveWithdrawal'
import { useGetL2Output } from '@/hooks/useGetL2Output'
import { useWithdrawalMessage } from '@/hooks/useWithdrawalMessage'
import { config } from '@/lib/wagmi'

const useProveWithdrawalParams = ({
  transactionHash,
  l2Chain,
}: {
  transactionHash?: Hash
  l2Chain: Chain
}) => {
  const { data: receipt } = useTransactionReceipt({
    hash: transactionHash,
    chainId: l2Chain.id,
  })

  const { data: l2Output } = useGetL2Output({
    l2BlockNumber: receipt?.blockNumber,
    l2Chain,
  })

  const withdrawal = useWithdrawalMessage(receipt)

  return useBuildProveWithdrawal({
    withdrawal,
    output: l2Output,
    l2Chain,
  })
}

const useWriteProveWithdrawal = () => {
  return useMutation({
    mutationFn: async ({
      l2Chain,
      proveWithdrawalParams,
    }: {
      l2Chain: Chain
      proveWithdrawalParams: BuildProveWithdrawalReturnType<
        undefined,
        undefined,
        Chain
      >
    }) => {
      if (!proveWithdrawalParams) {
        return
      }

      await switchChain(config, {
        chainId: l2Chain.sourceId!,
      })

      const walletClient = await getWalletClient(config, {
        chainId: l2Chain.sourceId!,
      })

      return await walletClient
        .extend(walletActionsL1())
        .proveWithdrawal(proveWithdrawalParams as any)
    },
  })
}

export const ReadyToProve = ({
  transactionHash,
  l2Chain,
}: {
  transactionHash: Hash
  l2Chain: Chain
}) => {
  const { data: proveWithdrawalParams, isLoading } = useProveWithdrawalParams({
    transactionHash,
    l2Chain,
  })

  const { mutate, isPending, error, data: txHash } = useWriteProveWithdrawal()

  if (error) {
    console.error(error)
  }

  const { isLoading: isWaitingForTx, data: proveReceipt } =
    useWaitForTransactionReceipt({
      hash: txHash,
      chainId: l2Chain.sourceId,
    })

  const isButtonDisabled =
    !proveWithdrawalParams || isLoading || isPending || isWaitingForTx

  if (proveReceipt) {
    return (
      <div>
        <Loader2 className="h-4 w-4 animate-spin" />
      </div>
    )
  }

  return (
    <div>
      <Button
        disabled={isButtonDisabled}
        onClick={() => {
          if (!proveWithdrawalParams) {
            return
          }
          mutate({ l2Chain, proveWithdrawalParams })
        }}
      >
        {isPending ? (
          <>
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            Sending transaction...
          </>
        ) : isWaitingForTx ? (
          <>
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            Waiting for confirmation...
          </>
        ) : isLoading ? (
          <>
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            Loading...
          </>
        ) : (
          <>
            <CheckCircle className="mr-2 h-4 w-4" />
            Prove Withdrawal
          </>
        )}
      </Button>
    </div>
  )
}
