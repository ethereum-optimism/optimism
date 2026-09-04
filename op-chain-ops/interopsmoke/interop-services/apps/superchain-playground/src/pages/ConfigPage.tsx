import type { NetworkName } from '@eth-optimism/viem/chains'
import { networks } from '@eth-optimism/viem/chains'
import { Wrench, X } from 'lucide-react'
import { useState } from 'react'
import { z } from 'zod'

import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useConfig } from '@/stores/useConfig'

// Add URL schema
const urlSchema = z.string().url().or(z.literal(''))

interface UrlInputProps {
  id: string
  value: string
  onChange: (value: string) => void
  placeholder?: string
  onClear?: () => void
}

const UrlInput = ({
  id,
  value,
  onChange,
  placeholder,
  onClear,
}: UrlInputProps) => {
  const [error, setError] = useState<boolean>(false)

  const handleChange = (newValue: string) => {
    const result = urlSchema.safeParse(newValue)
    setError(!result.success && newValue !== '')
    onChange(newValue)
  }

  return (
    <div className="flex gap-2 items-center">
      <Input
        id={id}
        value={value}
        onChange={(e) => handleChange(e.target.value)}
        placeholder={placeholder}
        className={`font-mono text-sm ${error ? 'border-red-500' : ''}`}
      />
      {value && onClear && (
        <Button
          variant="ghost"
          size="icon"
          onClick={onClear}
          className="h-10 w-10 shrink-0"
        >
          <X className="h-4 w-4" />
        </Button>
      )}
    </div>
  )
}

export const ConfigPage = () => {
  const {
    networkName,
    setNetworkName,
    rpcOverrideByChainId,
    setRpcOverrideByChainId,
  } = useConfig()

  const network = networks[networkName]

  const handleClearOverride = (chainId: number) => {
    setRpcOverrideByChainId(chainId, '')
  }

  return (
    <Card className="border-none shadow-none">
      <CardHeader className="pb-2">
        <CardTitle className="text-4xl font-bold">RPC Overrides</CardTitle>
        <CardDescription className="text-sm leading-relaxed text-muted-foreground">
          Override the default RPC URLs here if needed. Some default RPC URLs
          have rate limits or don't support calls like{' '}
          <span className="font-mono bg-slate-100 font-2xs px-1.5 py-0.5 rounded-md text-slate-700">
            debug_traceCall
          </span>
          . All changes are <b>only stored locally in your browser</b>.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-6 mt-4">
        <div className="flex items-center gap-4">
          <Label
            htmlFor="source-chain"
            className="text-sm text-muted-foreground w-32"
          >
            Network
          </Label>
          <Select
            value={networkName}
            onValueChange={(value) => setNetworkName(value as NetworkName)}
          >
            <SelectTrigger id="source-chain" className="w-[240px]">
              <SelectValue placeholder="Select source network" />
            </SelectTrigger>
            <SelectContent>
              {Object.keys(networks).map((networkName) => {
                return (
                  <SelectItem key={networkName} value={networkName}>
                    <div className="w-[180px]  flex-1 flex items-center justify-between">
                      <div>{networkName}</div>
                    </div>
                  </SelectItem>
                )
              })}
            </SelectContent>
          </Select>
        </div>
        <div key={network.sourceChain.id} className="space-y-3">
          <div className="grid grid-cols-[200px,1fr] gap-4 items-center mb-4">
            <Label
              htmlFor={`chain-${network.sourceChain.id}`}
              className="text-sm text-muted-foreground flex items-center gap-2"
            >
              {network.sourceChain.name} ({network.sourceChain.id})
              {rpcOverrideByChainId[network.sourceChain.id] && (
                <Wrench className="h-3 w-3" />
              )}
            </Label>
            <UrlInput
              id={`chain-${network.sourceChain.id}`}
              placeholder={network.sourceChain.rpcUrls.default.http[0]}
              value={rpcOverrideByChainId[network.sourceChain.id] || ''}
              onChange={(value) =>
                setRpcOverrideByChainId(network.sourceChain.id, value)
              }
              onClear={() => handleClearOverride(network.sourceChain.id)}
            />
          </div>
          <hr className="my-4 border-t border-gray-300" />
          <div className="space-y-2">
            {network.chains.map((chain) => (
              <div
                key={chain.id}
                className="grid grid-cols-[200px,1fr] gap-4 items-center"
              >
                <Label
                  htmlFor={`chain-${chain.id}`}
                  className="text-sm text-muted-foreground flex items-center gap-2"
                >
                  {chain.name} ({chain.id})
                  {rpcOverrideByChainId[chain.id] && (
                    <Wrench className="h-3 w-3" />
                  )}
                </Label>
                <UrlInput
                  id={`chain-${chain.id}`}
                  placeholder={chain.rpcUrls.default.http[0]}
                  value={rpcOverrideByChainId[chain.id] || ''}
                  onChange={(value) => setRpcOverrideByChainId(chain.id, value)}
                  onClear={() => handleClearOverride(chain.id)}
                />
              </div>
            ))}
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
