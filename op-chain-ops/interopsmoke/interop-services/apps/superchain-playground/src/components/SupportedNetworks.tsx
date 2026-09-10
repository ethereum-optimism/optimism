import type { NetworkName } from '@eth-optimism/viem/chains'
import { AlertTriangle } from 'lucide-react'
import type { ReactNode } from 'react'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { useConfig } from '@/stores/useConfig'

import { NetworkPicker } from './NetworkPicker'

interface SupportedNetworksProps {
  networks: NetworkName[]
  children: ReactNode
}

export const SupportedNetworks = ({
  networks,
  children,
}: SupportedNetworksProps) => {
  const { networkName } = useConfig()
  if (networks.includes(networkName)) {
    return <>{children}</>
  }

  return (
    <Alert variant="destructive" className="space-y-4">
      <div className="flex items-start gap-2">
        <AlertTriangle className="h-4 w-4 mt-0.5" />
        <div>
          <AlertTitle>Unsupported Network</AlertTitle>
          <AlertDescription>
            Select a supported network to continue
          </AlertDescription>
        </div>
      </div>
      <div className="text-foreground">
        <NetworkPicker networks={networks} />
      </div>
    </Alert>
  )
}
