import { createWalletClient, custom } from 'viem'
import { mainnet } from 'viem/chains'
import { describe, expect } from 'vitest'
import { opViemWalletExtension } from 'index'
import { optimistABI, optimistAddress } from '@eth-optimism/contracts-ts'


describe('opViemWalletExtension', async () => {
  // User makes their viem wallet as normal
  // @see https://viem.sh/docs/clients/wallet.html
  const client = createWalletClient({
    chain: mainnet,
    // TODO replace this with a real transport that will work in test
    // this code is currently just to show the API
    transport: custom((window as any).ethereum)
  })

  // To add extra OP functionality they then add our extension
  const extendedClient = client.extend(opViemWalletExtension)

  const myAddress = '0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045'
  const theirAddress = '0x420A6BF26964aF9D7eEd9e03E53415D37aA96420'

  // Now they can call a contract on l2 from l1 for example using normal viem apis
  // this api is the exact same api as the normal viem function `writeContract`
  const l2TxHash = await extendedClient.bridgeWriteContract({
    // Interacting with the optimist abi
    abi: optimistABI,
    // It's an SBT but pretend it isn't
    functionName: 'transferFrom',
    args: [myAddress, theirAddress, BigInt(1)],
    address: optimistAddress[10],
    account: myAddress,
    // TODO modify api here make it so the chain is the destination chain instead of origin chain
    chain: mainnet,
  })

  console.log(l2TxHash)

  expect(/0x.+/.test(l2TxHash)).toBe(true)

})
