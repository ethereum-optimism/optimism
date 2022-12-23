import { HandleBlock, createBlockEvent } from 'forta-agent'
import { BigNumber, utils } from 'ethers'
import { expect } from 'chai'

import agent from './agent'

describe('minimum balance agent', async () => {
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
              return utils.parseEther('1001')
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

  describe('handleBlock', async () => {
    it('returns empty findings if balance is above threshold', async () => {
      mockEthersProvider = mockEthersProviderByCase('safe')
      handleBlock = agent.provideHandleBlock(mockEthersProvider)
      const findings = await handleBlock(blockEvent)

      expect(findings).to.deep.equal([])
    })
  })
})
