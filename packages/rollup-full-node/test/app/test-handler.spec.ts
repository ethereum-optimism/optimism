import '../setup'
/* External Imports */
import { add0x, getLogger, remove0x } from '@eth-optimism/core-utils'

/* Internal Imports */
import { Web3RpcMethods, TestWeb3Handler } from '../../src'

const log = getLogger('test-web3-handler', true)

const secondsSinceEopch = (): number => {
  return Math.round(Date.now() / 1000)
}

describe('TestHandler', () => {
  let testHandler: TestWeb3Handler

  beforeEach(async () => {
    testHandler = await TestWeb3Handler.create()
  })

  describe('Timestamps', () => {
    it('should get timestamp', async () => {
      const currentTime = secondsSinceEopch()
      const res: string = await testHandler.handleRequest(
        Web3RpcMethods.getTimestamp,
        []
      )
      const timeAfter = secondsSinceEopch()

      const timestamp: number = parseInt(remove0x(res), 16)
      timestamp.should.be.gte(currentTime, 'Timestamp out of range')
      timestamp.should.be.lte(timeAfter, 'Timestamp out of range')
    })

    it('should increase timestamp', async () => {
      const previous: string = await testHandler.handleRequest(
        Web3RpcMethods.getTimestamp,
        []
      )
      const previousTimestamp: number = parseInt(remove0x(previous), 16)

      const increase: number = 9999
      const setRes: string = await testHandler.handleRequest(
        Web3RpcMethods.increaseTimestamp,
        [add0x(increase.toString(16))]
      )
      setRes.should.equal(
        TestWeb3Handler.successString,
        'Should increase timestamp!'
      )

      const fetched: string = await testHandler.handleRequest(
        Web3RpcMethods.getTimestamp,
        []
      )
      const fetchedTimestamp: number = parseInt(remove0x(fetched), 16)
      fetchedTimestamp.should.be.gte(
        previousTimestamp + increase,
        'Timestamp was not increased properly!'
      )
    })
  })
})
