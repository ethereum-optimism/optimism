import {
  type Chain,
  type Transport,
  type Account,
  WalletClient,
  encodeFunctionData,
  EncodeFunctionDataParameters
} from 'viem'
import { l1CrossDomainMessengerABI, l1CrossDomainMessengerAddress } from '@eth-optimism/contracts-ts'

type OptimismExtended = {
  bridgeWriteContract: WalletClient['writeContract']
}

export const opViemWalletExtension = <
  transport extends Transport = Transport,
  chain extends Chain | undefined = Chain | undefined,
  account extends Account | undefined = Account | undefined,
>(client: WalletClient<transport, chain, account>) => ({
  bridgeWriteContract: async (args) => {
    // TODO don't hardcode this
    const minGasLimit = 200_000
    const message = encodeFunctionData({
      abi: args.abi,
      functionName: args.functionName,
      args: args.args,
    } as unknown as EncodeFunctionDataParameters<typeof args.abi, typeof args.functionName>)
    const l1TxHash = await client.writeContract({
      abi: l1CrossDomainMessengerABI,
      // TODO currently hardcoded for OP
      address: l1CrossDomainMessengerAddress[1],
      functionName: 'sendMessage' as any,
      value: args.value as any,
      args: [
        args.address,
        message,
        minGasLimit,
      ] as any,
    })
    //TODO derive me using new tx hash method from core utils
    const l2TxHash = l1TxHash
    // TODO return both l1 and l2 tx hash instead of only l2
    return l2TxHash
  },
} satisfies OptimismExtended)

