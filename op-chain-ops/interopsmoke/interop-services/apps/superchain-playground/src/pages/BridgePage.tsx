import type { NetworkName } from '@eth-optimism/viem/chains'
import { networks } from '@eth-optimism/viem/chains'

import { BridgeCard } from '@/components/BridgeCard'
import { SupportedNetworks } from '@/components/SupportedNetworks'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useConfig } from '@/stores/useConfig'

const supportedNetworkNames: NetworkName[] = [
  'sepolia',
  'interop-alpha',
  'supersim',
]

export const BridgePage = () => {
  const { networkName, setNetworkName } = useConfig()
  const supportedNetworks = supportedNetworkNames.map((name) => networks[name])

  return (
    <div className="flex flex-col gap-4 max-w-2xl mx-auto">
      <SupportedNetworks networks={supportedNetworkNames}>
        <Tabs defaultValue={networkName} value={networkName}>
          <TabsList className="w-full flex">
            {supportedNetworks.map((network) => (
              <TabsTrigger
                onClick={() => setNetworkName(network.name)}
                className="flex-1 relative"
                key={network.name}
                value={network.name}
              >
                {network.name}
              </TabsTrigger>
            ))}
          </TabsList>

          {supportedNetworks.map((network) => {
            return (
              <TabsContent key={network.name} value={network.name}>
                <BridgeCard network={network} />
              </TabsContent>
            )
          })}
        </Tabs>
      </SupportedNetworks>
    </div>
  )
}
