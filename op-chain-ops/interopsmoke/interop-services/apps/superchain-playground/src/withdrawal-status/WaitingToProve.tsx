import { Clock } from 'lucide-react'
import type { Chain, Hash } from 'viem'

import { RemainingTimeDisplay } from '@/components/RemainingTimeDisplay'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { useGetTimeToProve } from '@/hooks/useGetTimeToProve'

const EstimatedRemainingTime = ({
  transactionHash,
  l2Chain,
}: {
  transactionHash: Hash
  l2Chain: Chain
}) => {
  const { data, isLoading } = useGetTimeToProve({
    transactionHash,
    l2Chain,
  })

  if (!data || isLoading) {
    return (
      <div className="space-y-2">
        <Skeleton className="h-4 w-[200px]" />
        <Skeleton className="h-6 w-[100px]" />
      </div>
    )
  }

  return (
    <div className="space-y-1">
      <div className="text-sm text-muted-foreground">
        Estimated remaining time
      </div>
      <RemainingTimeDisplay seconds={data.seconds} />
    </div>
  )
}

export const WaitingToProve = ({
  transactionHash,
  l2Chain,
}: {
  transactionHash: Hash
  l2Chain: Chain
}) => {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Clock className="h-5 w-5" />
          Waiting to prove
        </CardTitle>
      </CardHeader>
      <CardContent>
        <EstimatedRemainingTime
          transactionHash={transactionHash}
          l2Chain={l2Chain}
        />
      </CardContent>
    </Card>
  )
}
