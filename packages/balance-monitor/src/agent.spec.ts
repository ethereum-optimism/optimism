import {
  Finding,
  FindingSeverity,
  FindingType,
  HandleBlock,
  createBlockEvent,
} from 'forta-agent'
import { expect } from 'chai'
import * as ethers from 'ethers'

import agent, { accounts } from './agent'

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
          getBalance: (addr: string) => {
            if (addr == '0xabba') {
              return '1001' + '0'.repeat(18)
            }
            if (addr == '0xacdc') {
              return '2001' + '0'.repeat(18)
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
  })
})
