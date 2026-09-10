import { Clock, Loader2 } from 'lucide-react'
import type { Chain, Hash } from 'viem'

import { RemainingTimeDisplay } from '@/components/RemainingTimeDisplay'
import { Card, CardContent } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { useGetTimeToFinalize } from '@/hooks/useGetTimeToFinalize'

const EstimatedRemainingTime = ({
  transactionHash,
  l2Chain,
}: {
  transactionHash: Hash
  l2Chain: Chain
}) => {
  const { data, isLoading } = useGetTimeToFinalize({
    transactionHash,
    l2Chain,
  })

  if (isLoading) {
    return (
      <div className="space-y-2">
        <Skeleton className="h-4 w-[200px]" />
        <Skeleton className="h-4 w-[100px]" />
      </div>
    )
  }

  if (!data) return null

  return (
    <div className="space-y-1">
      <div className="text-sm text-muted-foreground flex items-center gap-2">
        <Clock className="h-4 w-4" />
        Estimated remaining time
      </div>
      <RemainingTimeDisplay seconds={data.seconds} />
    </div>
  )
}

export const WaitingToFinalize = ({
  transactionHash,
  l2Chain,
}: {
  transactionHash: Hash
  l2Chain: Chain
}) => {
  return (
    <Card>
      <CardContent className="pt-6 space-y-4">
        <div className="flex items-center gap-2 text-muted-foreground">
          <Loader2 className="h-4 w-4 animate-spin" />
          <span>Waiting to finalize...</span>
        </div>
        <EstimatedRemainingTime
          transactionHash={transactionHash}
          l2Chain={l2Chain}
        />
      </CardContent>
    </Card>
  )
}
