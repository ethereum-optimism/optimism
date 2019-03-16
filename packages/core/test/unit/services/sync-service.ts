import '../../setup'

/* External Imports */
import BigNum = require('bn.js')
import { capture } from 'ts-mockito'

/* Internal Imports */
import { PlasmaBlock, Deposit, Exit } from '../../../src/models/chain'
import { SyncService } from '../../../src/services'

/*
describe('SyncService', () => {
  const sync = new SyncService()

  it('should react to new deposits', () => {
    const depositEvent = new DepositEvent({
      token: new BigNum(0),
      start: new BigNum(0),
      end: new BigNum(100),
      block: new BigNum(0),
      owner: '0x123',
    })
    const deposit = depositEvent.toDeposit()
    eventHandler.emit('event:Deposit', [depositEvent])

    const callArgs = capture(mockChainService.addDeposits).last()
    callArgs[0].should.deep.equal([deposit])
  })

  it('should react to new blocks', () => {
    const blockSubmittedEvent = new BlockSubmittedEvent({
      number: 0,
      hash: '0x0',
    })
    const block = blockSubmittedEvent.toBlock()
    eventHandler.emit('event:BlockSubmitted', [blockSubmittedEvent])

    const callArgs = capture(mockChainDB.addBlockHeaders).last()
    callArgs[0].should.deep.equal([block])
  })

  it('should react to new exits', () => {
    const exitStartedEvent = new ExitStartedEvent({
      token: new BigNum(0),
      start: new BigNum(0),
      end: new BigNum(100),
      block: new BigNum(0),
      id: new BigNum(0),
      owner: '0x123',
    })
    const exit = exitStartedEvent.toExit()
    eventHandler.emit('event:ExitStarted', [exitStartedEvent])

    const callArgs = capture(mockChainService.addExit).last()
    callArgs[0].should.deep.equal(exit)
  })
})
*/
