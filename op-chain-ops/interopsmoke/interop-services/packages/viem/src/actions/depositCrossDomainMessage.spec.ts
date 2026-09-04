import { createPublicClient, http, parseEventLogs } from 'viem'
import { extractTransactionDepositedLogs } from 'viem/op-stack'
import { describe, expect, it } from 'vitest'

import { crossDomainMessengerAbi } from '@/abis/index.js'
import {
  depositCrossDomainMessage,
  estimateDepositCrossDomainMessageGas,
  simulateDepositCrossDomainMessage,
} from '@/actions/depositCrossDomainMessage.js'
import { supersimL1 } from '@/chains/supersim.js'
import { testAccount } from '@/test/clients.js'
import { supersimL2AL1CrossDomainMessengerAddress } from '@/test/testConstants.js'

describe('depositCrossDomainMessage', () => {
  // Hardcoded since we don't have a good way to pull the L1 contracts for supersim yet.
  const l1CrossDomainMessengerAddress = supersimL2AL1CrossDomainMessengerAddress

  const publicL1Client = createPublicClient({
    chain: supersimL1,
    transport: http(supersimL1.rpcUrls.default.http[0]),
  })

  describe('write contract', () => {
    it('should return expected result', async () => {
      const hash = await depositCrossDomainMessage(publicL1Client, {
        account: testAccount,
        target: '0x0000000000000000000000000000000000000000',
        message: '0xabcd',
        l1CrossDomainMessengerAddress,
        value: 10n,
      })

      const { logs } = await publicL1Client.waitForTransactionReceipt({ hash })
      const depositLog = extractTransactionDepositedLogs({ logs })
      expect(depositLog).toBeDefined()
      expect(depositLog!.length).toBe(1)

      const sentMessages = parseEventLogs({
        abi: crossDomainMessengerAbi,
        eventName: 'SentMessage',
        logs,
      })
      expect(sentMessages).toBeDefined()
      expect(sentMessages!.length).toBe(1)
      expect(sentMessages![0].args.message).toBe('0xabcd')
      expect(sentMessages![0].args.sender).toBe(testAccount.address)
      expect(sentMessages![0].args.target).toBe(
        '0x0000000000000000000000000000000000000000',
      )

      const extension = parseEventLogs({
        abi: crossDomainMessengerAbi,
        eventName: 'SentMessageExtension1',
        logs,
      })
      expect(extension).toBeDefined()

      // TODO: This is a bug in viem which includes the `SentMessage` event in the extension logs.
      // Latest version of viem has a fix but we need to update the entire workspace
      expect(extension!.length).toBe(2)
      expect(extension![1].args.value).toBe(10n)
    })
  })

  describe('estimate gas', () => {
    it('should estimate gas', async () => {
      const gas = await estimateDepositCrossDomainMessageGas(publicL1Client, {
        account: testAccount,
        target: '0x0000000000000000000000000000000000000000',
        message: '0xabcd',
        l1CrossDomainMessengerAddress,
        value: 10n,
      })

      expect(gas).toBeDefined()
    })
  })

  describe('simulate', () => {
    it('should simulate', async () => {
      expect(
        async () =>
          await simulateDepositCrossDomainMessage(publicL1Client, {
            account: testAccount,
            target: '0x0000000000000000000000000000000000000000',
            message: '0xabcd',
            l1CrossDomainMessengerAddress,
            value: 10n,
          }),
      ).not.throw()
    })
  })
})
