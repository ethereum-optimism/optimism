import '../../helpers/setup'
import { ethereum } from '../../helpers/ethereum'

import { DefaultEthProvider } from '../../../src/eth-provider/default-eth-provider'
import { DummyContract } from '../../helpers/contract'

describe('DefaultEthProvider', () => {
  const eth = new DefaultEthProvider()

  before(async () => {
    await ethereum.start()
  })

  after(async () => {
    await ethereum.stop()
  })

  describe('connected', () => {
    it('should tell us if the node is connected', async () => {
      const connected = await eth.connected()
      connected.should.be.true
    })
  })

  describe('getCurrentBlock', () => {
    it('should return the current block', async () => {
      const block = await eth.getCurrentBlock()
      block.should.equal(0)
    })
  })

  describe('getEvents', () => {
    const dummy = new DummyContract()

    before(async () => {
      await dummy.deploy()
      await dummy.createEvents(10)
    })

    it('should return all events for a contract', async () => {
      const events = await eth.getEvents({
        event: 'TestEvent',
        address: dummy.address,
        abi: dummy.abi,
        fromBlock: 1,
        toBlock: 11,
      })

      events.should.have.lengthOf(10)
    })

    it('should return only events between given blocks', async () => {
      const events = await eth.getEvents({
        event: 'TestEvent',
        address: dummy.address,
        abi: dummy.abi,
        fromBlock: 1,
        toBlock: 6,
      })

      events.should.have.lengthOf(5)
    })
  })
})
