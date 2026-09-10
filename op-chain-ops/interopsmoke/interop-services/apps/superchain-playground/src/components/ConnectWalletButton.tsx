import {
  Activity,
  ArrowRight,
  ChevronRight,
  Copy,
  Power,
  WalletIcon,
} from 'lucide-react'
import * as React from 'react'
import type { Connector } from 'wagmi'
import { useAccount, useConnect, useDisconnect } from 'wagmi'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useToast } from '@/hooks/use-toast'

function WalletOptions() {
  const { connectors, connect } = useConnect()

  return (
    <div className="grid gap-4">
      <DialogDescription className="text-muted-foreground">
        Choose your preferred wallet to connect.
      </DialogDescription>

      <div className="grid gap-2">
        {connectors.map((connector) => (
          <WalletOption
            key={connector.uid}
            connector={connector}
            onClick={() => connect({ connector })}
          />
        ))}
      </div>
    </div>
  )
}

function WalletOption({
  connector,
  onClick,
}: {
  connector: Connector
  onClick: () => void
}) {
  const [ready, setReady] = React.useState(false)

  React.useEffect(() => {
    ;(async () => {
      const provider = await connector.getProvider()
      setReady(!!provider)
    })()
  }, [connector])

  return (
    <Card className="overflow-hidden border-0 bg-muted/30">
      <button
        className="relative flex w-full items-center gap-3 p-4 disabled:opacity-50 group hover:bg-muted/50 transition-colors cursor-pointer"
        disabled={!ready}
        onClick={onClick}
      >
        <div className="flex h-10 w-10 items-center justify-center rounded-md bg-background shadow-sm ring-1 ring-border">
          {connector.icon ? (
            <img
              src={connector.icon}
              alt={`${connector.name} icon`}
              className="h-6 w-6"
            />
          ) : (
            <WalletIcon className="h-6 w-6" />
          )}
        </div>
        <div className="flex flex-col items-start gap-1">
          <span className="text-sm font-medium">{connector.name}</span>
          {!ready && (
            <Badge variant="outline" className="text-xs text-yellow-500">
              <Activity className="h-3 w-3 mr-1" />
              Not Installed
            </Badge>
          )}
        </div>
        <ArrowRight className="h-5 w-5 text-muted-foreground ml-auto group-hover:translate-x-1 transition-transform" />
      </button>
    </Card>
  )
}

export function Account() {
  const { address, connector } = useAccount()
  const { disconnect } = useDisconnect()
  const { toast } = useToast()

  const copyAddress = () => {
    navigator.clipboard.writeText(address || '')
    toast({
      description: 'Address copied to clipboard',
      duration: 2000,
    })
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" size="sm" className="gap-2 cursor-pointer">
          <div className="flex items-center gap-2">
            <div className="rounded-full p-1 bg-muted">
              {connector?.icon ? (
                <img
                  src={connector.icon}
                  alt={connector.name}
                  className="h-3 w-3"
                />
              ) : (
                <WalletIcon className="h-3 w-3" />
              )}
            </div>
            <span className="hidden sm:inline text-sm font-medium">
              {address?.slice(0, 6)}...{address?.slice(-4)}
            </span>
            <span className="sm:hidden text-sm">{address?.slice(0, 4)}...</span>
          </div>
          <ChevronRight className="h-4 w-4 text-muted-foreground" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-56">
        <DropdownMenuLabel>{connector?.name || 'Wallet'}</DropdownMenuLabel>
        <DropdownMenuSeparator />
        <DropdownMenuItem
          onClick={copyAddress}
          className="gap-2 cursor-pointer"
        >
          <Copy className="h-4 w-4" />
          <span>Copy Address</span>
        </DropdownMenuItem>
        <DropdownMenuItem
          onClick={() => disconnect()}
          className="gap-2 text-destructive focus:text-destructive cursor-pointer"
        >
          <Power className="h-4 w-4" />
          <span>Disconnect</span>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

export const ConnectWalletButton = () => {
  const { isConnected } = useAccount()

  if (isConnected) return <Account />

  return (
    <Dialog>
      <DialogTrigger asChild>
        <Button variant="outline" className="gap-2 cursor-pointer">
          <WalletIcon className="h-4 w-4" />
          Connect Wallet
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-[425px] p-6">
        <DialogHeader>
          <DialogTitle>Connect Wallet</DialogTitle>
        </DialogHeader>

        <WalletOptions />
      </DialogContent>
    </Dialog>
  )
}
