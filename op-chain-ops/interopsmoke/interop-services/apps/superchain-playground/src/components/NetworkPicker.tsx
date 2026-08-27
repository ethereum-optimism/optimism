import type { NetworkName } from '@eth-optimism/viem/chains'

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useConfig } from '@/stores/useConfig'

interface NetworkPickerProps {
  networks: NetworkName[]
}

export const NetworkPicker = ({ networks }: NetworkPickerProps) => {
  const { networkName, setNetworkName } = useConfig()

  return (
    <div className="flex-1 space-y-2">
      <label className="text-sm font-medium">Select Network</label>
      <Select
        value={networkName}
        onValueChange={(value) => setNetworkName(value as NetworkName)}
      >
        <SelectTrigger>
          <SelectValue placeholder="Select a network" />
        </SelectTrigger>
        <SelectContent>
          {networks.map((network) => (
            <SelectItem key={network} value={network}>
              {network}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}
