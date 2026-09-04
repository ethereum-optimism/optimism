import type { NetworkName } from '@eth-optimism/viem/chains'
import { networks } from '@eth-optimism/viem/chains'

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useConfig } from '@/stores/useConfig'

export const SelectNetwork = () => {
  const { networkName, setNetworkName } = useConfig()

  return (
    <Select
      value={networkName}
      onValueChange={(value) => setNetworkName(value as NetworkName)}
    >
      <SelectTrigger className="select-none">
        <SelectValue>
          <span className="font-bold">Network:</span> {networkName}
        </SelectValue>
      </SelectTrigger>
      <SelectContent className="flex">
        {Object.values(networks).map((network) => (
          <SelectItem
            key={network.name}
            value={network.name}
            className="flex-1 w-full"
          >
            <div>{network.name}</div>
            <div className="text-muted-foreground text-xs">
              Source ChainID: {network.sourceChain.id}
            </div>
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}
