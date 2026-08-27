import { encodeFunctionData } from 'viem'
import { describe, expect, it } from 'vitest'

import { supersimL2B } from '@/chains/supersim.js'
import { publicClientA, testAccount, walletClientA } from '@/test/clients.js'
import { ticTacToeAbi, ticTacToeAddress } from '@/test/setupTicTacToe.js'

describe('sendCrossDomainMessage', () => {
  const calldata = encodeFunctionData({
    abi: ticTacToeAbi,
    functionName: 'createGame',
    args: [testAccount.address],
  })

  describe('write contract', () => {
    it('should return expected request', async () => {
      const txHash = await walletClientA.interop.sendCrossDomainMessage({
        account: testAccount.address,
        destinationChainId: supersimL2B.id,
        target: ticTacToeAddress,
        message: calldata,
      })

      expect(txHash).toBeDefined()

      // SentMessage event
      const receipt = await publicClientA.waitForTransactionReceipt({
        hash: txHash,
      })
      const messages = await publicClientA.interop.getCrossDomainMessages({
        logs: receipt.logs,
      })
      expect(messages).length(1)

      // verify cross chain message
      const { destination, sender, target, message } = messages[0]
      expect(destination).toEqual(BigInt(supersimL2B.id))
      expect(sender).toEqual(testAccount.address)
      expect(target).toEqual(ticTacToeAddress)
      expect(message).toEqual(calldata)
    })
  })

  describe('estimate gas', () => {
    it('should estimate gas', async () => {
      const gas = await publicClientA.interop.estimateSendCrossDomainMessageGas(
        {
          account: testAccount.address,
          target: ticTacToeAddress,
          destinationChainId: supersimL2B.id,
          message: calldata,
        },
      )

      expect(gas).toBeDefined()
    })
  })

  describe('simulate', () => {
    it('should simulate', async () => {
      expect(() =>
        publicClientA.interop.simulateSendCrossDomainMessage({
          account: testAccount.address,
          destinationChainId: supersimL2B.id,
          target: ticTacToeAddress,
          message: calldata,
        }),
      ).not.throw()
    })
  })
})
