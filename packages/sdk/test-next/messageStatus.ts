import { describe, expect, it } from 'vitest'

import { CrossChainMessenger, MessageStatus } from '../src'
import { l1Provider, l2Provider } from './testUtils/ethersProviders'

const crossChainMessenger = new CrossChainMessenger({
  l1SignerOrProvider: l1Provider,
  l2SignerOrProvider: l2Provider,
  l1ChainId: 5,
  l2ChainId: 420,
  bedrock: true,
})

describe('prove message', () => {
  it(`should be able to correctly find a finalized withdrawal`, async () => {
    /**
     * Tx hash of legacy withdrawal that was claimed
     *
     * @see https://goerli-optimism.etherscan.io/tx/0xda9e9c8dfc7718bc1499e1e64d8df6cddbabc46e819475a6c755db286a41b9fa
     */
    const txWithdrawalHash =
      '0xda9e9c8dfc7718bc1499e1e64d8df6cddbabc46e819475a6c755db286a41b9fa'

    const txReceipt = await l2Provider.getTransactionReceipt(txWithdrawalHash)

    expect(txReceipt).toBeDefined()

    expect(await crossChainMessenger.getMessageStatus(txWithdrawalHash)).toBe(
      MessageStatus.RELAYED
    )
  }, 20_000)
})
