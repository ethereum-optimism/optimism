import {
  Finding,
  FindingSeverity,
  FindingType,
  HandleBlock,
  createBlockEvent,
} from 'forta-agent'
import { BigNumber, utils } from 'ethers'
import { expect } from 'chai'

import agent, { accounts } from './agent'
import { describeFinding } from './utils'

describe('minimum balance agent', () => {
  let handleBlock: HandleBlock
  let mockEthersProvider
  const blockEvent = createBlockEvent({
    block: { hash: '0xa', number: 1 } as any,
  })

  // A function which returns a mock provider to give us values based on the case we want
  // to test.
  const mockEthersProviderByCase = (severity: string) => {
    switch (severity) {
      case 'safe':
        return {
          getBalance: async (addr: string): Promise<BigNumber> => {
            if (addr === '0xabba') {
              return utils.parseEther('101')
            }
            if (addr === '0xacdc') {
              return utils.parseEther('2001')
            }
          },
        } as any
      case 'danger':
        return {
          getBalance: async (addr: string): Promise<BigNumber> => {
            if (addr === '0xabba') {
              return utils.parseEther('99') // below danger threshold
            }
            if (addr === '0xacdc') {
              return utils.parseEther('2001')
            }
          },
        } as any
      default:
        break
    }
  }

  before(() => {
    handleBlock = agent.provideHandleBlock(mockEthersProvider)
  })

  describe('handleBlock', () => {
    it('returns empty findings if balance is above threshold', async () => {
      mockEthersProvider = mockEthersProviderByCase('safe')
      handleBlock = agent.provideHandleBlock(mockEthersProvider)

      const findings = await handleBlock(blockEvent)

      expect(findings).to.deep.equal([])
    })

    it('returns high severity finding if balance is below danger threshold', async () => {
      mockEthersProvider = mockEthersProviderByCase('danger')
      handleBlock = agent.provideHandleBlock(mockEthersProvider)

      const balance = await mockEthersProvider.getBalance('0xabba')
      const findings = await handleBlock(blockEvent)

      // Take the second alert in the list, as the first is a warning
      expect(findings).to.deep.equal([
        Finding.fromObject({
          name: 'Minimum Account Balance',
          description: describeFinding(
            accounts[0].address,
            balance,
            accounts[0].thresholds.danger
          ),
          alertId: 'OPTIMISM-BALANCE-DANGER-Sequencer',
          severity: FindingSeverity.High,
          type: FindingType.Info,
          metadata: {
            balance: balance.toString(),
          },
        }),
      ])
    })
  })
})
