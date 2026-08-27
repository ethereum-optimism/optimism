import { contracts } from '@eth-optimism/viem'
import { l2ToL2CrossDomainMessengerAbi } from '@eth-optimism/viem/abis'
import type { SendCrossDomainMessageParameters } from '@eth-optimism/viem/actions/interop'
import { useCallback } from 'react'
import { useConfig, useWriteContract } from 'wagmi'

export const useSendL2ToL2Message = () => {
  const config = useConfig()
  const { writeContractAsync, isError, isPending, isSuccess } =
    useWriteContract({ config })

  const sendMessage = useCallback(
    (params: SendCrossDomainMessageParameters) => {
      const { destinationChainId, target, message } = params

      return writeContractAsync({
        abi: l2ToL2CrossDomainMessengerAbi,
        address: contracts.l2ToL2CrossDomainMessenger.address,
        functionName: 'sendMessage',
        args: [BigInt(destinationChainId), target, message],
      })
    },
    [writeContractAsync],
  )

  return { sendMessage, isError, isPending, isSuccess }
}
