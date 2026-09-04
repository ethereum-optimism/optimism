import { describe, expect, it } from 'vitest'

import { supersimL2B } from '@/chains/supersim.js'
import {
  publicClientA,
  publicClientB,
  testAccount,
  walletClientA,
  walletClientB,
} from '@/test/clients.js'

const AMOUNT_TO_SEND = 10n

describe('sendETH', () => {
  describe('write contract', () => {
    it('should return expected request', async () => {
      const startingBalance = await publicClientB.getBalance({
        address: testAccount.address,
      })

      const hash = await walletClientA.interop.sendETH({
        to: testAccount.address,
        value: AMOUNT_TO_SEND,
        chainId: supersimL2B.id,
      })

      const receipt = await publicClientA.waitForTransactionReceipt({ hash })
      const messages = await publicClientA.interop.getCrossDomainMessages({
        logs: receipt.logs,
      })
      expect(messages).length(1)

      const params = await publicClientA.interop.buildExecutingMessage({
        log: messages[0].log,
      })
      const relayTxHash = await walletClientB.interop.relayCrossDomainMessage(
        params,
      )
      expect(relayTxHash).toBeDefined()

      await publicClientB.waitForTransactionReceipt({ hash: relayTxHash })
      const status = await publicClientB.interop.getCrossDomainMessageStatus({
        message: messages[0],
      })
      expect(status).toEqual('relayed')

      const endingBalance = await publicClientB.getBalance({
        address: testAccount.address,
      })

      // TODO: fix after restructuring supersim
      expect(endingBalance > startingBalance)
    })
  })

  describe('estimate gas', () => {
    it('should estimate gas', async () => {
      const gas = await publicClientA.interop.estimateSendETHGas({
        account: testAccount.address,
        to: testAccount.address,
        value: AMOUNT_TO_SEND,
        chainId: supersimL2B.id,
      })

      expect(gas).toBeDefined()
    })
  })

  describe('simulate', () => {
    it('should simulate', async () => {
      await expect(
        publicClientA.interop.simulateSendETH({
          account: testAccount.address,
          to: testAccount.address,
          value: AMOUNT_TO_SEND,
          chainId: supersimL2B.id,
        }),
      ).resolves.not.toThrow()
    })
  })
})
