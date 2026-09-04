import { formatDuration } from '@/utils/formatDuration'

export const RemainingTimeDisplay = ({ seconds }: { seconds: number }) => {
  const formattedTime = formatDuration(seconds)

  return <div className="text-lg font-medium">{formattedTime}</div>
}
