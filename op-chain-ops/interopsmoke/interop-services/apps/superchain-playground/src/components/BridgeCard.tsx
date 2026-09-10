import type { Network } from '@eth-optimism/viem/chains'
import { ArrowRight } from 'lucide-react'
import { useState } from 'react'
import type { Chain, ChainContract, Hex } from 'viem'
import { encodeFunctionData, formatEther, parseEther, toHex } from 'viem'
import {
  useAccount,
  useBalance,
  useSimulateContract,
  useSwitchChain,
  useWaitForTransactionReceipt,
  useWriteContract,
} from 'wagmi'

import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import { Switch } from '@/components/ui/switch'
import { ToastAction } from '@/components/ui/toast'
import { useToast } from '@/components/ui/use-toast'
import { l1StandardBridgeAbi } from '@/constants/l1StandardBridgeAbi'
import { multicall3Abi } from '@/constants/multicall3Abi'
import { truncateHash } from '@/lib/truncateHash'
import { cn } from '@/lib/utils'
import { truncateDecimal } from '@/utils/truncateDecimal'

const NetworkSwitch = ({
  chain,
  onChange,
  isSelected,
}: {
  chain: Chain
  onChange: (isSelected: boolean) => void
  isSelected: boolean
}) => {
  const { address } = useAccount()
  const { data } = useBalance({
    address: address!,
    chainId: chain.id,
    query: { enabled: !!address },
  })
  const { formatted: balance, symbol } = data ?? { formatted: '-', symbol: '' }

  return (
    <div
      className={cn(
        'w-full flex rounded-lg py-2 px-2 items-center',
        data
          ? 'cursor-pointer hover:bg-muted/50'
          : 'opacity-50 cursor-not-allowed',
      )}
      onClick={(e) => {
        if (!data) return
        e.stopPropagation()
        onChange(!isSelected)
      }}
    >
      <Switch
        id={`${chain.id}-network-switch`}
        checked={isSelected}
        disabled={!data}
        className="self-center data-[state=checked]:bg-primary"
        onCheckedChange={(checked) => {
          if (!data) return
          onChange(checked)
        }}
      />
      <div
        className={cn(
          'pl-2 w-full flex items-center justify-between select-none pr-2 text-sm font-medium leading-none',
        )}
      >
        <div className="flex items-center gap-2">{chain.name}</div>
        <div className="text-muted-foreground text-sm">
          {truncateDecimal(balance)} {symbol}
        </div>
      </div>
    </div>
  )
}

const NetworkSwitches = ({
  chains,
  isSelectedByChainId,
  setIsSelected,
}: {
  chains: Chain[]
  isSelectedByChainId: { [key: number]: boolean }
  setIsSelected: (chainId: number, isSelected: boolean) => void
}) => {
  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Label>Networks</Label>
        </div>
        <div className="text-sm text-muted-foreground">Current balance</div>
      </div>
      <div id="network-switch-input" className="flex flex-col">
        {chains.map((chain) => {
          return (
            <NetworkSwitch
              key={chain.id}
              chain={chain}
              isSelected={isSelectedByChainId[chain.id] ?? false}
              onChange={(isChecked) => setIsSelected(chain.id, isChecked)}
            />
          )
        })}
      </div>
    </div>
  )
}

const AmountInput = ({
  amount,
  setAmount,
}: {
  amount: bigint
  setAmount: (amount: bigint) => void
}) => {
  const quickInputAmounts = [
    parseEther('10'),
    parseEther('1'),
    parseEther('0.1'),
    parseEther('0.01'),
  ]

  return (
    <div className="flex flex-col gap-2">
      <Label htmlFor="bridge-amount-input">Amount</Label>
      <Input
        id="bridge-amount-input"
        type="number"
        placeholder="0.0"
        value={formatEther(amount)}
        onChange={(e) => setAmount(parseEther(e.target.value))}
      />
      <div className="flex gap-2">
        {quickInputAmounts.map((quickInputAmount) => {
          return (
            <Button
              variant="outline"
              size="sm"
              onClick={() => setAmount(quickInputAmount)}
              className={cn(
                'flex-1 transition-all',
                quickInputAmount === amount
                  ? 'border-primary/90 bg-primary text-primary-foreground hover:bg-primary/90'
                  : 'hover:border-primary/50',
              )}
            >
              {formatEther(quickInputAmount)} ETH
            </Button>
          )
        })}
      </div>
    </div>
  )
}

const Preview = ({
  selectedChains,
  amount,
}: {
  selectedChains: Chain[]
  amount: bigint
}) => {
  return (
    <div className="flex flex-col gap-4 p-4 bg-muted/30 rounded-lg">
      <div className="flex justify-between items-center">
        <div className="text-sm text-muted-foreground">Total amount</div>
        <div className="font-medium text-lg">
          {formatEther(BigInt(selectedChains.length) * amount)} ETH
        </div>
      </div>
      {selectedChains.length > 0 && (
        <div className="flex flex-col gap-2">
          <div className="text-sm text-muted-foreground">Bridging to:</div>
          <div className="flex flex-wrap gap-2">
            {selectedChains.map((chain) => (
              <div
                key={chain.id}
                className="bg-background px-3 py-1 rounded-full text-sm border"
              >
                {chain.name}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

export const BridgeCard = ({ network }: { network: Network }) => {
  const sourceChainId = network.sourceChain.id
  const chains = network.chains

  const { toast } = useToast()
  const { address, chainId } = useAccount()
  const { switchChain, isPending: switchChainPending } = useSwitchChain()

  const { data } = useBalance({
    address,
    chainId: sourceChainId,
    query: { enabled: !!address },
  })
  const { formatted: balance, symbol } = data ?? { formatted: '-', symbol: '' }

  const [amount, setAmount] = useState(0n)
  const [isSelectedByChainId, setIsSelectedByChainId] = useState(
    chains.reduce<{ [key: number]: boolean }>((acc, chain) => {
      acc[chain.id] = false
      return acc
    }, {}),
  )

  const selectedChains = chains.filter((chain) => isSelectedByChainId[chain.id])
  const setIsSelected = (chainId: number, isSelected: boolean) => {
    setIsSelectedByChainId({ ...isSelectedByChainId, [chainId]: isSelected })
  }

  const { data: simulatedData, error } = useSimulateContract({
    abi: multicall3Abi,
    functionName: 'aggregate3Value',
    address: network.sourceChain.contracts?.multicall3?.address,
    value: amount * BigInt(selectedChains.length),
    args: [
      selectedChains.map((selectedChain) => {
        const l1StandardBridge = selectedChain.contracts?.l1StandardBridge as {
          [key: number]: ChainContract
        }

        const calldata = encodeFunctionData({
          abi: l1StandardBridgeAbi,
          functionName: 'bridgeETHTo',
          args: [address ?? '0x', 0, toHex('')],
        })

        return {
          target: l1StandardBridge[sourceChainId]!.address,
          allowFailure: false,
          callData: calldata,
          value: amount,
        }
      }),
    ],
  })

  if (error) {
    console.error(`FAILED TO SIMULATE L1 BRIDGE TRANSACTION: ${error}`)
  }

  const {
    writeContract,
    isPending,
    data: hash,
  } = useWriteContract({
    mutation: {
      onSuccess: (hash: Hex) => {
        toast({
          title: 'L1 transaction sent',
          description: `${truncateHash(hash)}`,
          action: (
            <ToastAction
              altText="View on explorer"
              onClick={() => {
                window.open(
                  `${
                    network.sourceChain.blockExplorers!.default.url
                  }/tx/${hash}`,
                  '_blank',
                )
              }}
            >
              View on explorer
            </ToastAction>
          ),
        })
      },
    },
  })

  const { isLoading } = useWaitForTransactionReceipt({ hash })

  return (
    <Card className="flex-1">
      <CardHeader>
        <div className="flex items-center justify-between h-8">
          <CardTitle>{network.sourceChain.name}</CardTitle>
          <div className="flex items-center gap-2">
            <span className="text-sm font-normal text-muted-foreground">
              Balance:
            </span>
            <span className="text-md font-medium">
              {truncateDecimal(balance)} {symbol}
            </span>
          </div>
        </div>
        <CardDescription className="mt-2">
          Bridge ETH to OP Stack chains
        </CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-8">
        <NetworkSwitches
          chains={chains}
          isSelectedByChainId={isSelectedByChainId}
          setIsSelected={setIsSelected}
        />
        <AmountInput amount={amount} setAmount={setAmount} />
        <Separator className="" />
        <Preview selectedChains={selectedChains} amount={amount} />
      </CardContent>
      <CardFooter className="flex gap-2">
        {chainId !== sourceChainId ? (
          <Button
            className="w-full"
            disabled={!switchChain || switchChainPending}
            onClick={() => {
              switchChain?.({ chainId: sourceChainId })
            }}
          >
            Switch network to {network.sourceChain.name}
          </Button>
        ) : (
          <Button
            className="w-full gap-2"
            onClick={() => writeContract(simulatedData!.request)}
            disabled={
              isPending ||
              isLoading ||
              selectedChains.length === 0 ||
              amount === 0n ||
              !writeContract ||
              !simulatedData?.request
            }
          >
            Bridge <ArrowRight className="w-4 h-4" />
          </Button>
        )}
      </CardFooter>
    </Card>
  )
}
