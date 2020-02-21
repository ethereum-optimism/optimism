import '../setup'
/* External Imports */
import { add0x, getLogger, remove0x } from '@eth-optimism/core-utils'

/* Internal Imports */
import { TestWeb3Handler } from '../../src/app/test-handler'
import { Web3RpcMethods } from '../../src/types'

const log = getLogger('test-web3-handler', true)

const secondsSinceEopch = (): number => {
  return Math.round(Date.now() / 1000)
}

describe('TestHandler', () => {
  let testHandler: TestWeb3Handler

  beforeEach(async () => {
    testHandler = await TestWeb3Handler.create()
  })

  const assertGetTimestampNoOverride = async () => {
    const currentTime = secondsSinceEopch()
    const res: string = await testHandler.handleRequest(
      Web3RpcMethods.getTimestamp,
      []
    )
    const timeAfter = secondsSinceEopch()

    const timestamp: number = parseInt(remove0x(res), 16)
    timestamp.should.be.gte(currentTime, 'Timestamp out of range')
    timestamp.should.be.lte(timeAfter, 'Timestamp out of range')
  }

  describe('Timestamps', () => {
    it('should get timestamp', async () => {
      await assertGetTimestampNoOverride()
    })

    it('should set and clear timestamp', async () => {
      const timestamp: number = 9999
      const setRes: string = await testHandler.handleRequest(
        Web3RpcMethods.setTimestamp,
        [add0x(timestamp.toString(16))]
      )
      setRes.should.equal(
        TestWeb3Handler.successString,
        'Should set timestamp!'
      )

      const fetched: string = await testHandler.handleRequest(
        Web3RpcMethods.getTimestamp,
        []
      )
      const fetchedTimestamp: number = parseInt(remove0x(fetched), 16)
      fetchedTimestamp.should.equal(timestamp, 'Timestamp was not set!')

      const clearRes: string = await testHandler.handleRequest(
        Web3RpcMethods.clearTimestamp,
        []
      )
      clearRes.should.equal(
        TestWeb3Handler.successString,
        'Should clear timestamp!'
      )

      await assertGetTimestampNoOverride()
    })
  })
})
