import { contracts } from '@eth-optimism/viem'
import type { NetworkName } from '@eth-optimism/viem/chains'
import { networks } from '@eth-optimism/viem/chains'
import { CheckCircle2, Loader2, Send } from 'lucide-react'
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import type { Address } from 'viem'
import { parseUnits } from 'viem'
import {
  useAccount,
  useSimulateContract,
  useSwitchChain,
  useWaitForTransactionReceipt,
  useWriteContract,
} from 'wagmi'

import { NetworkPicker } from '@/components/NetworkPicker'
import { SupportedNetworks } from '@/components/SupportedNetworks'
import { Alert, AlertDescription } from '@/components/ui/alert'
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
import { superchainTokenBridgeAbi } from '@/constants/superchainTokenBridgeAbi'
import { useTokenInfo } from '@/hooks/useTokenInfo'
import { useConfig } from '@/stores/useConfig'
import { useTransactionStore } from '@/stores/useTransactionStore'

const supportedNetworkNames: NetworkName[] = ['supersim', 'interop-alpha']

export const SuperchainTokenBridgePage = () => {
  const { address } = useAccount()
  const { networkName } = useConfig()
  const [amount, setAmount] = useState('')

  const network = networks[networkName]
  const [fromChainId, setFromChain] = useState<number>(
    network.chains[0]?.id || 0,
  )

  const [toChainId, setToChain] = useState<number>(network.chains[1]?.id || 0)
  const [tokenAddress, setTokenAddress] = useState<Address>(
    '0xAaA2b0D6295b91505500B7630e9E36a461ceAd1b',
  )

  const { addTransaction } = useTransactionStore()

  const {
    symbol,
    decimals = 18,
    name,
    isLoading: isTokenLoading,
  } = useTokenInfo({
    address: tokenAddress as Address,
    chainId: fromChainId,
  })

  const amountUnits = parseUnits(amount, decimals)

  const { switchChain } = useSwitchChain()

  const simulationResult = useSimulateContract({
    abi: superchainTokenBridgeAbi,
    address: contracts.superchainTokenBridge.address,
    functionName: 'sendERC20',
    args: [tokenAddress, address!, amountUnits, BigInt(toChainId)],
    chainId: fromChainId,
    query: {
      enabled: Boolean(fromChainId && toChainId && address && amount),
    },
  })

  const {
    data: hash,
    isPending: isSendPending,
    writeContract,
    reset,
  } = useWriteContract({
    mutation: {
      onSuccess: (hash) => {
        addTransaction({
          hash,
          chainId: fromChainId,
        })
      },
    },
  })

  const handleFromChainChange = async (chainId: string) => {
    const numChainId = parseInt(chainId)
    setFromChain(numChainId)
    if (numChainId === toChainId) {
      const availableChains = network.chains.filter(
        (chain) => chain.id !== numChainId,
      )

      setToChain(availableChains[0]?.id || 0)
    }
    reset()
  }

  const { isLoading: isReceiptLoading, isSuccess: isReceiptSuccess } =
    useWaitForTransactionReceipt({
      hash,
    })

  const isLoading =
    isSendPending || isReceiptLoading || !simulationResult.data?.request

  const isButtonDisabled =
    !address ||
    !amount ||
    !fromChainId ||
    !toChainId ||
    isLoading ||
    !simulationResult.data?.request

  const getButtonContent = () => {
    if (isSendPending) {
      return (
        <>
          <span className="mr-2">Sending transaction...</span>
          <Send className="h-4 w-4 animate-pulse" />
        </>
      )
    }
    if (isReceiptLoading) {
      return (
        <>
          <span className="mr-2">Waiting for confirmation...</span>
          <Loader2 className="h-4 w-4 animate-spin" />
        </>
      )
    }
    return 'Bridge'
  }

  const navigate = useNavigate()

  // Add handler for navigation
  const handleViewRelayer = () => {
    navigate(
      `/superchain-message-relayer?chainId=${fromChainId}&txHash=${hash}`,
    )
  }

  return (
    <div className="flex flex-col gap-8 w-full max-w-3xl mx-auto px-4 sm:px-6 py-8">
      <SupportedNetworks networks={supportedNetworkNames}>
        <Card>
          <CardHeader className="space-y-1">
            <CardTitle className="text-lg sm:text-xl">
              Bridge SuperchainERC20 between Superchain L2s
            </CardTitle>
            <CardDescription className="text-sm">
              Transfer assets between networks
            </CardDescription>
          </CardHeader>

          <CardContent className="space-y-4 sm:space-y-6">
            <div className="space-y-4">
              <NetworkPicker networks={supportedNetworkNames} />

              <div className="space-y-2">
                <div className="flex flex-col sm:flex-row sm:justify-between sm:items-center gap-2">
                  <Label>Token Address</Label>
                  {fromChainId !== 0 &&
                    (isTokenLoading ? (
                      <div className="flex items-center gap-2 text-xs sm:text-sm text-muted-foreground">
                        <Loader2 className="h-4 w-4 animate-spin" />
                        Loading...
                      </div>
                    ) : (
                      <div className="text-xs sm:text-sm text-muted-foreground sm:text-right">
                        {name && `${name} • `}
                        {symbol && `${symbol} • `}
                        {decimals && `${decimals} decimals`}
                      </div>
                    ))}
                </div>
                <Input
                  type="text"
                  placeholder="0x..."
                  value={tokenAddress}
                  onChange={(e) => setTokenAddress(e.target.value as Address)}
                  className="font-mono text-sm"
                />
              </div>

              <div className="space-y-2">
                <Label>Amount</Label>
                <Input
                  type="number"
                  placeholder="0.0"
                  value={amount}
                  onChange={(e) => setAmount(e.target.value)}
                  className="text-sm"
                />
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label>From Chain</Label>
                  <Select
                    onValueChange={handleFromChainChange}
                    value={fromChainId.toString()}
                    disabled={!networkName}
                  >
                    <SelectTrigger className="text-sm">
                      <SelectValue placeholder="Select chain" />
                    </SelectTrigger>
                    <SelectContent>
                      {network.chains.map((chain) => (
                        <SelectItem
                          key={chain.id}
                          value={chain.id.toString()}
                          className="text-sm"
                        >
                          {chain.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>

                <div className="space-y-2">
                  <Label>To Chain</Label>
                  <Select
                    onValueChange={(value) => setToChain(parseInt(value))}
                    disabled={!fromChainId}
                    value={toChainId.toString()}
                  >
                    <SelectTrigger className="text-sm">
                      <SelectValue placeholder="Select chain" />
                    </SelectTrigger>
                    <SelectContent>
                      {network.chains
                        .filter((chain) => chain.id !== fromChainId)
                        .map((chain) => (
                          <SelectItem
                            key={chain.id}
                            value={chain.id.toString()}
                            className="text-sm"
                          >
                            {chain.name}
                          </SelectItem>
                        ))}
                    </SelectContent>
                  </Select>
                </div>
              </div>
            </div>

            <Button
              className="w-full text-sm"
              size="lg"
              disabled={isButtonDisabled}
              onClick={async () => {
                await switchChain({ chainId: fromChainId })
                writeContract(simulationResult.data!.request)
              }}
            >
              {getButtonContent()}
            </Button>
          </CardContent>
        </Card>

        {isReceiptSuccess && (
          <Alert className="border-green-500/50 bg-green-500/10">
            <AlertDescription className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
              <div className="flex items-center gap-2">
                <CheckCircle2 className="h-4 w-4 text-green-500 flex-shrink-0" />
                <span className="text-sm">
                  Transaction confirmed successfully!
                </span>
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={handleViewRelayer}
                className="w-full sm:w-auto text-sm"
              >
                View in Message Relayer
              </Button>
            </AlertDescription>
          </Alert>
        )}
      </SupportedNetworks>
    </div>
  )
}
