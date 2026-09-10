import { contracts } from '@eth-optimism/viem'
import { l2ToL2CrossDomainMessengerAbi } from '@eth-optimism/viem/abis'
import type { RelayCrossDomainMessageParameters } from '@eth-optimism/viem/actions/interop'
import { useCallback } from 'react'
import { useConfig, useWriteContract } from 'wagmi'

export const useRelayL2ToL2Message = () => {
  const config = useConfig()
  const { writeContractAsync, isError, isPending, isSuccess } =
    useWriteContract({ config })

  const relayMessage = useCallback(
    (params: RelayCrossDomainMessageParameters) => {
      const {
        id,
        payload,
        nonce,
        maxFeePerGas,
        maxPriorityFeePerGas,
        chain,
        gas,
      } = params

      return writeContractAsync({
        abi: l2ToL2CrossDomainMessengerAbi,
        address: contracts.l2ToL2CrossDomainMessenger.address,
        functionName: 'relayMessage',
        args: [id, payload],
        chain,
        gas: gas ? gas : undefined,
        nonce,
        maxFeePerGas,
        maxPriorityFeePerGas,
      })
    },
    [writeContractAsync],
  )

  return { relayMessage, isError, isPending, isSuccess }
}
