import { networks } from '@eth-optimism/viem/chains'
import type { Chain } from 'viem'
import { useSwitchChain } from 'wagmi'

import { Badge } from '@/components/ui/badge'
import { Button, buttonVariants } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { useToast } from '@/components/ui/use-toast'
import { cn } from '@/lib/utils'
import { useConfig } from '@/stores/useConfig'

const ChainRow = ({
  sourceChain,
  chain,
}: {
  sourceChain: Chain
  chain: Chain
}) => {
  const { switchChain } = useSwitchChain()
  const { toast } = useToast()

  const copyChainId = (id: number) => {
    navigator.clipboard.writeText(id.toString())
    toast({
      description: `Copied chain ID: ${id}`,
    })
  }

  return (
    <TableRow className="hover:bg-gray-100">
      <TableCell className="font-medium">
        {chain.blockExplorers?.default ? (
          <a
            href={chain.blockExplorers!.default.url}
            target="_blank"
            rel="noreferrer"
            className={cn(
              buttonVariants({ variant: 'link' }),
              'px-0 text-primary hover:text-primary/80 underline',
            )}
          >
            {chain.name}
          </a>
        ) : (
          <span>{chain.name}</span>
        )}
      </TableCell>
      <TableCell>
        <Badge
          variant="secondary"
          className="font-mono cursor-pointer transition-transform hover:scale-105 active:scale-95 bg-gray-200 hover:bg-gray-800 hover:text-white"
          onClick={() => copyChainId(chain.id)}
        >
          {chain.id}
        </Badge>
      </TableCell>
      <TableCell>
        <Badge
          variant="outline"
          className="cursor-pointer transition-transform hover:scale-105 active:scale-95 bg-gray-200 hover:bg-gray-800 hover:text-white"
          onClick={() => copyChainId(sourceChain.id)}
        >
          {sourceChain.name} ({sourceChain.id})
        </Badge>
      </TableCell>
      <TableCell className="text-right">
        <Button
          size="sm"
          variant="outline"
          onClick={() => switchChain?.({ chainId: chain.id })}
          className="space-x-2"
        >
          <span>Add network</span>
        </Button>
      </TableCell>
    </TableRow>
  )
}

export const ChainsPage = () => {
  const { networkName } = useConfig()
  const network = networks[networkName]

  // TODO: Add ability to switch networks

  return (
    <div className="container ">
      <Card className="border-none shadow-none">
        <CardHeader>
          <CardTitle className="text-3xl font-bold tracking-tight">
            OP Stack Chains
          </CardTitle>
          <CardDescription className="flex items-center justify-between">
            <div>
              Sourced from the{' '}
              <a
                href="https://github.com/ethereum-optimism/superchain-registry"
                target="_blank"
                rel="noopener noreferrer"
                className="font-medium text-primary hover:text-primary/80 underline underline-offset-4"
              >
                Superchain Registry
              </a>
            </div>
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Chain ID</TableHead>
                <TableHead>Network</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {network.chains.map((chain) => (
                <ChainRow
                  key={chain.id}
                  sourceChain={network.sourceChain}
                  chain={chain}
                />
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  )
}
