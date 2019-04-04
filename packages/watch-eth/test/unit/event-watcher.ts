import { should } from '../helpers/setup'
import * as mockEth from '../helpers/mock/eth-provider'
import * as mockDB from '../helpers/mock/event-db'

import { EventWatcher } from '../../src/event-watcher'
import { EventLog } from '../../src/models'
import { sleep } from '../../src/utils'
import { capture } from 'ts-mockito'

/**
 * Small class for spying on listeners.
 */
class ListenerSpy {
  public args: EventLog[]
  public listener(args: EventLog[]): void {
    this.args = args
  }
}

/**
 * Helper function for creating a watcher instance.
 * @param finalityDepth Number of blocks before the event is final.
 * @returns the watcher instance.
 */
const createWatcher = (finalityDepth = 0) => {
  return new EventWatcher({
    address: '0x0',
    abi: [
      {
        anonymous: false,
        inputs: [
          {
            indexed: false,
            name: '_value',
            type: 'uint256',
          },
        ],
        name: 'TestEvent',
        type: 'event',
      },
    ],
    finalityDepth,
    pollInterval: 0,
    eth: mockEth.eth,
    db: mockDB.db,
  })
}

describe('EventWatcher', () => {
  let watcher: EventWatcher

  beforeEach(() => {
    watcher = createWatcher()
  })

  afterEach(() => {
    watcher.stopPolling()
    mockEth.reset()
  })

  describe('subscribe', () => {
    it('should allow a user to subscribe to an event', () => {
      const filter = 'TestEvent'
      const listener = () => {
        return
      }

      watcher.subscribe(filter, listener)
      watcher.isPolling.should.be.true
    })

    it('should allow a user to subscribe twice with the same listener', () => {
      const filter = 'TestEvent'
      const listener = () => {
        return
      }

      should.not.Throw(() => {
        watcher.subscribe(filter, listener)
        watcher.subscribe(filter, listener)
      })
    })
  })

  describe('unsubscribe', () => {
    it('should allow a user to unsubscribe from an event', () => {
      const filter = 'TestEvent'
      const listener = () => {
        return
      }

      watcher.subscribe(filter, listener)
      watcher.unsubscribe(filter, listener)
      watcher.isPolling.should.be.false
    })

    it('should allow a user to unsubscribe even if not subscribed', () => {
      const filter = 'TestEvent'
      const listener = () => {
        return
      }

      should.not.Throw(() => {
        watcher.unsubscribe(filter, listener)
      })
    })

    it('should still be polling if other listeners exist', () => {
      const filter = 'TestEvent'
      const listener1 = () => {
        return true
      }
      const listener2 = () => {
        return false
      }

      watcher.subscribe(filter, listener1)
      watcher.subscribe(filter, listener2)
      watcher.unsubscribe(filter, listener1)
      watcher.isPolling.should.be.true
    })
  })

  describe('events', () => {
    it('should alert a listener when it hears an event', async () => {
      const filter = 'TestEvent'
      const spy = new ListenerSpy()

      // Mock out the events that will be returned.
      const event: EventLog = new EventLog({
        transactionHash: '0x123',
        logIndex: 0,
      })
      mockEth.setEvents([event])

      // Subscribe for new events.
      watcher.subscribe(filter, spy.listener.bind(spy))

      // Wait for events to be detected.
      await sleep(10)

      spy.args.should.deep.equal([event])
      capture(mockDB.dbSpy.setEventSeen)
        .last()[0]
        .should.deep.equal(event.hash)
    })

    it('should alert multiple listeners on the same event', async () => {
      const filter = 'TestEvent'
      const spy1 = new ListenerSpy()
      const spy2 = new ListenerSpy()

      // Mock out the events that will be returned.
      const event: EventLog = new EventLog({
        transactionHash: '0x123',
        logIndex: 0,
      })
      mockEth.setEvents([event])

      // Subscribe for new events.
      watcher.subscribe(filter, spy1.listener.bind(spy1))
      watcher.subscribe(filter, spy2.listener.bind(spy2))

      // Wait for events to be detected.
      await sleep(10)

      spy1.args.should.deep.equal([event])
      spy2.args.should.deep.equal([event])
      capture(mockDB.dbSpy.setEventSeen)
        .last()[0]
        .should.deep.equal(event.hash)
    })

    it('should only alert the same event once', async () => {
      const filter = 'TestEvent'
      const spy = new ListenerSpy()

      // Mock out the events that will be returned.
      const event: EventLog = new EventLog({
        transactionHash: '0x123',
        logIndex: 0,
      })
      mockEth.setEvents([event, event])

      // Subscribe for new events.
      watcher.subscribe(filter, spy.listener.bind(spy))

      // Wait for events to be detected.
      await sleep(10)

      spy.args.should.deep.equal([event])
    })

    it('should only alert once an event is final', async () => {
      // Create a new watcher with a finality depth of 12.
      watcher = createWatcher(12)

      const filter = 'TestEvent'
      const spy = new ListenerSpy()

      // Mock out the events that will be returned.
      const event: EventLog = new EventLog({
        transactionHash: '0x123',
        logIndex: 0,
      })
      mockEth.setEvents([event])

      // Subscribe for new events.
      watcher.subscribe(filter, spy.listener.bind(spy))

      // Wait for events to be detected.
      await sleep(10)

      should.not.exist(spy.args)

      // Move time forward.
      mockEth.setBlock(20)

      // Wait for events to be detected.
      await sleep(10)

      spy.args.should.deep.equal([event])
    })
  })
})
