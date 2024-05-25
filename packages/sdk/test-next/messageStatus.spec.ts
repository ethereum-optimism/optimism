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

describe('getMessageStatus', () => {
  it(`should be able to correctly find a finalized withdrawal`, async () => {
    /**
     * Tx hash of a withdrawal
     *
     * @see https://goerli-optimism.etherscan.io/tx/0x8fb235a61079f3fa87da66e78c9da075281bc4ba5f1af4b95197dd9480e03bb5
     */
    const txWithdrawalHash =
      '0x8fb235a61079f3fa87da66e78c9da075281bc4ba5f1af4b95197dd9480e03bb5'

    const txReceipt = await l2Provider.getTransactionReceipt(txWithdrawalHash)

    expect(txReceipt).toBeDefined()

    expect(
      await crossChainMessenger.getMessageStatus(
        txWithdrawalHash,
        0,
        9370789 - 1000,
        9370789
      )
    ).toBe(MessageStatus.RELAYED)
  }, 20_000)

  it(`should return READY_FOR_RELAY if not in block range`, async () => {
    const txWithdrawalHash =
      '0x8fb235a61079f3fa87da66e78c9da075281bc4ba5f1af4b95197dd9480e03bb5'

    const txReceipt = await l2Provider.getTransactionReceipt(txWithdrawalHash)

    expect(txReceipt).toBeDefined()

    expect(
      await crossChainMessenger.getMessageStatus(txWithdrawalHash, 0, 0, 0)
    ).toBe(MessageStatus.READY_FOR_RELAY)
  }, 20_000)
})
