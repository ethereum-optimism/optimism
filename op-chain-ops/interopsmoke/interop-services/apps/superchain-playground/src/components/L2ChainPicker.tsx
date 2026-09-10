import { networks } from '@eth-optimism/viem/chains'

import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useConfig } from '@/stores/useConfig'

interface ChainPickerProps {
  label?: string
  chainId: number | undefined
  onChange: (chainId: number) => void
  placeholder?: string
}

export const L2ChainPicker = ({
  label,
  chainId,
  onChange,
  placeholder = 'Select Chain',
}: ChainPickerProps) => {
  const { networkName } = useConfig()
  const network = networks[networkName]

  return (
    <div className="space-y-2 flex-1">
      {label && <Label>{label}</Label>}
      <Select
        value={chainId?.toString()}
        onValueChange={(value) => onChange(Number(value))}
      >
        <SelectTrigger className="">
          <SelectValue placeholder={placeholder} />
        </SelectTrigger>
        <SelectContent>
          {network.chains.map((chain) => (
            <SelectItem key={chain.id} value={chain.id.toString()}>
              {chain.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}
