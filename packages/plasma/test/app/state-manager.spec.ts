import '../setup'

/* External Imports */
import { BigNumber, ONE } from '@pigi/core-utils'
import * as assert from 'assert'

/* Internal Imports */
import {
  DefaultStateDB,
  DefaultStateManager,
  PluginManager,
  PredicatePlugin,
  StateDB,
  StateManager,
  StateObject,
  StateUpdate,
  Transaction,
  TransactionResult,
  VerifiedStateUpdate,
} from '../../src/'
import { stateUpdatesEqual } from '../../src/app/utils'

/*******************
 * Mocks & Helpers *
 *******************/

class DummyPluginManager implements PluginManager {
  public getPlugin(address: string): Promise<PredicatePlugin | undefined> {
    return undefined
  }

  public loadPlugin(address: string, path: string): Promise<PredicatePlugin> {
    return undefined
  }
}

class DummyPredicatePlugin implements PredicatePlugin {
  public executeStateTransition(
    previousStateUpdate: StateUpdate,
    transaction: Transaction,
    witness: string = 'none'
  ): Promise<StateObject> {
    return undefined
  }
}

function getPluginThatReturns(stateObjects: StateObject[]): PredicatePlugin {
  const predicatePlugin: PredicatePlugin = new DummyPredicatePlugin()
  predicatePlugin.executeStateTransition = async (
    previousStateUpdate: StateUpdate,
    transaction: Transaction,
    witness: string = 'none'
  ): Promise<StateObject> => {
    if (stateObjects.length > 1) {
      return stateObjects.shift()
    }
    return stateObjects[0]
  }
  return predicatePlugin
}

function getPluginManagerThatReturns(
  pluginMap: Map<string, PredicatePlugin>
): PluginManager {
  const pluginManager: PluginManager = new DummyPluginManager()
  pluginManager.getPlugin = async (
    address: string
  ): Promise<PredicatePlugin | undefined> => {
    return pluginMap.get(address)
  }
  return pluginManager
}

function getStateDBThatReturns(
  verifiedStateUpdates: VerifiedStateUpdate[]
): StateDB {
  const stateDB = new DefaultStateDB()
  stateDB.getVerifiedStateUpdates = async (
    start: BigNumber,
    end: BigNumber
  ): Promise<VerifiedStateUpdate[]> => {
    return verifiedStateUpdates
  }
  return stateDB
}

function getStateUpdate(
  start: BigNumber,
  end: BigNumber,
  plasmaBlockNumber: BigNumber,
  stateObject: StateObject,
  depositAddress: string = '0x1234'
): StateUpdate {
  return {
    range: {
      start,
      end,
    },
    stateObject,
    depositAddress,
    plasmaBlockNumber: new BigNumber(plasmaBlockNumber),
  }
}

function getVerifiedStateUpdate(
  start: BigNumber,
  end: BigNumber,
  block: BigNumber,
  depositAddress: string,
  predicateAddress: string,
  data: any = { dummyData: false }
): VerifiedStateUpdate {
  return {
    range: {
      start,
      end,
    },
    verifiedBlockNumber: block,
    stateUpdate: getStateUpdate(
      start,
      end,
      block,
      { predicateAddress, data },
      depositAddress
    ),
  }
}

function getTransaction(
  depositAddress: string,
  start: BigNumber,
  end: BigNumber,
  body: any = { dummyData: false }
) {
  return {
    depositAddress,
    body,
    range: {
      start,
      end,
    },
  }
}

/*********
 * TESTS *
 *********/

describe('DefaultStateManager', () => {
  describe('Construction', () => {
    it('should initialize', async () => {
      new DefaultStateManager(new DefaultStateDB(), new DummyPluginManager())
    })
  })

  describe('executeTransaction', () => {
    const start: BigNumber = new BigNumber(10)
    const end: BigNumber = new BigNumber(20)
    const defaultWitness: string = 'none'
    const previousBlockNumber: BigNumber = new BigNumber(10)
    const nextBlockNumber: BigNumber = new BigNumber(11)
    const depositAddress = '0x1234'
    const predicateAddress = '0x12345678'
    const defaultEndData = { testResult: 'test' }

    const transaction: Transaction = getTransaction(depositAddress, start, end)
    const endStateObject: StateObject = {
      predicateAddress,
      data: defaultEndData,
    }

    const endStateUpdate: StateUpdate = getStateUpdate(
      start,
      end,
      nextBlockNumber,
      endStateObject,
      depositAddress
    )

    it('should process simple transaction for contiguous range', async () => {
      const verifiedStateUpdates: VerifiedStateUpdate[] = [
        getVerifiedStateUpdate(
          start,
          end,
          previousBlockNumber,
          depositAddress,
          predicateAddress
        ),
      ]

      const plugin: PredicatePlugin = getPluginThatReturns([endStateObject])

      const stateDB: StateDB = getStateDBThatReturns(verifiedStateUpdates)
      const pluginManager: PluginManager = getPluginManagerThatReturns(
        new Map<string, PredicatePlugin>([[predicateAddress, plugin]])
      )
      const stateManager: StateManager = new DefaultStateManager(
        stateDB,
        pluginManager
      )

      const result: TransactionResult = await stateManager.executeTransaction(
        transaction,
        nextBlockNumber,
        defaultWitness
      )

      assert(stateUpdatesEqual(result.stateUpdate, endStateUpdate))

      result.validRanges.length.should.equal(1)
      assert(
        result.validRanges[0].start.eq(start),
        `Valid Range start is [${result.validRanges[0].start.toString()}], when [${start.toString()}] was expected.`
      )
      assert(
        result.validRanges[0].end.eq(end),
        `Valid Range end is [${result.validRanges[0].end.toString()}], when [${end.toString()}] was expected.`
      )
    })

    it('should process complex transaction for contiguous range', async () => {
      const midPoint = end
        .sub(start)
        .divRound(new BigNumber(2))
        .add(start)
      const verifiedStateUpdates: VerifiedStateUpdate[] = [
        getVerifiedStateUpdate(
          start,
          midPoint,
          previousBlockNumber,
          depositAddress,
          predicateAddress
        ),
        getVerifiedStateUpdate(
          midPoint,
          end,
          previousBlockNumber,
          depositAddress,
          predicateAddress
        ),
      ]

      const plugin: PredicatePlugin = getPluginThatReturns([endStateObject])

      const stateDB: StateDB = getStateDBThatReturns(verifiedStateUpdates)
      const pluginManager: PluginManager = getPluginManagerThatReturns(
        new Map<string, PredicatePlugin>([[predicateAddress, plugin]])
      )
      const stateManager: StateManager = new DefaultStateManager(
        stateDB,
        pluginManager
      )

      const result: TransactionResult = await stateManager.executeTransaction(
        transaction,
        nextBlockNumber,
        defaultWitness
      )

      assert(stateUpdatesEqual(result.stateUpdate, endStateUpdate))

      result.validRanges.length.should.equal(2)
      assert(
        result.validRanges[0].start.eq(start),
        `Valid Range [0] start is [${result.validRanges[0].start.toString()}], when [${start.toString()}] was expected.`
      )
      assert(
        result.validRanges[0].end.eq(midPoint),
        `Valid Range [0] end is [${result.validRanges[0].end.toString()}], when [${midPoint.toString()}] was expected.`
      )
      assert(
        result.validRanges[1].start.eq(midPoint),
        `Valid Range [1] start is [${result.validRanges[1].start.toString()}], when [${midPoint.toString()}] was expected.`
      )
      assert(
        result.validRanges[1].end.eq(end),
        `Valid Range [1] end is [${result.validRanges[1].end.toString()}], when [${end.toString()}] was expected.`
      )
    })

    it('should process complex transaction for non-subset range', async () => {
      const midPoint = end
        .sub(start)
        .divRound(new BigNumber(2))
        .add(start)
      const verifiedStateUpdates: VerifiedStateUpdate[] = [
        getVerifiedStateUpdate(
          start.sub(new BigNumber(5)),
          midPoint,
          previousBlockNumber,
          depositAddress,
          predicateAddress
        ),
        getVerifiedStateUpdate(
          midPoint,
          end.add(new BigNumber(4)),
          previousBlockNumber,
          depositAddress,
          predicateAddress
        ),
      ]

      const plugin: PredicatePlugin = getPluginThatReturns([endStateObject])

      const stateDB: StateDB = getStateDBThatReturns(verifiedStateUpdates)
      const pluginManager: PluginManager = getPluginManagerThatReturns(
        new Map<string, PredicatePlugin>([[predicateAddress, plugin]])
      )
      const stateManager: StateManager = new DefaultStateManager(
        stateDB,
        pluginManager
      )

      const result: TransactionResult = await stateManager.executeTransaction(
        transaction,
        nextBlockNumber,
        defaultWitness
      )

      assert(stateUpdatesEqual(result.stateUpdate, endStateUpdate))

      result.validRanges.length.should.equal(2)
      assert(
        result.validRanges[0].start.eq(start),
        `Valid Range [0] start is [${result.validRanges[0].start.toString()}], when [${start.toString()}] was expected.`
      )
      assert(
        result.validRanges[0].end.eq(midPoint),
        `Valid Range [0] end is [${result.validRanges[0].end.toString()}], when [${midPoint.toString()}] was expected.`
      )
      assert(
        result.validRanges[1].start.eq(midPoint),
        `Valid Range [1] start is [${result.validRanges[1].start.toString()}], when [${midPoint.toString()}] was expected.`
      )
      assert(
        result.validRanges[1].end.eq(end),
        `Valid Range [1] end is [${result.validRanges[1].end.toString()}], when [${end.toString()}] was expected.`
      )
    })

    it('should process complex transaction for non-contiguous range', async () => {
      const endRange1: BigNumber = start.add(ONE)
      const startRange2: BigNumber = end.sub(ONE)
      const verifiedStateUpdates: VerifiedStateUpdate[] = [
        getVerifiedStateUpdate(
          start,
          endRange1,
          previousBlockNumber,
          depositAddress,
          predicateAddress
        ),
        getVerifiedStateUpdate(
          startRange2,
          end,
          previousBlockNumber,
          depositAddress,
          predicateAddress
        ),
      ]

      const plugin: PredicatePlugin = getPluginThatReturns([endStateObject])

      const stateDB: StateDB = getStateDBThatReturns(verifiedStateUpdates)
      const pluginManager: PluginManager = getPluginManagerThatReturns(
        new Map<string, PredicatePlugin>([[predicateAddress, plugin]])
      )
      const stateManager: StateManager = new DefaultStateManager(
        stateDB,
        pluginManager
      )

      const result: TransactionResult = await stateManager.executeTransaction(
        transaction,
        nextBlockNumber,
        defaultWitness
      )

      assert(stateUpdatesEqual(result.stateUpdate, endStateUpdate))

      result.validRanges.length.should.equal(2)
      assert(
        result.validRanges[0].start.eq(start),
        `Valid Range [0] start is [${result.validRanges[0].start.toString()}], when [${start.toString()}] was expected.`
      )
      assert(
        result.validRanges[0].end.eq(endRange1),
        `Valid Range [0] end is [${result.validRanges[0].end.toString()}], when [${endRange1.toString()}] was expected.`
      )
      assert(
        result.validRanges[1].start.eq(startRange2),
        `Valid Range [1] start is [${result.validRanges[1].start.toString()}], when [${startRange2.toString()}] was expected.`
      )
      assert(
        result.validRanges[1].end.eq(end),
        `Valid Range [1] end is [${result.validRanges[1].end.toString()}], when [${end.toString()}] was expected.`
      )
    })

    it('should return empty range if no VerifiedStateUpdates', async () => {
      const verifiedStateUpdates: VerifiedStateUpdate[] = []

      // This should never be called
      const plugin: PredicatePlugin = undefined

      const stateDB: StateDB = getStateDBThatReturns(verifiedStateUpdates)
      const pluginManager: PluginManager = getPluginManagerThatReturns(
        new Map<string, PredicatePlugin>([[predicateAddress, plugin]])
      )
      const stateManager: StateManager = new DefaultStateManager(
        stateDB,
        pluginManager
      )

      const result: TransactionResult = await stateManager.executeTransaction(
        transaction,
        nextBlockNumber,
        defaultWitness
      )

      assert(result.stateUpdate === undefined)
      result.validRanges.length.should.equal(0)
    })

    it('should return empty range if VerifiedStateUpdates do not overlap', async () => {
      const verifiedStateUpdates: VerifiedStateUpdate[] = [
        getVerifiedStateUpdate(
          end,
          end.add(ONE),
          previousBlockNumber,
          depositAddress,
          predicateAddress
        ),
        getVerifiedStateUpdate(
          start.sub(ONE),
          start,
          previousBlockNumber,
          depositAddress,
          predicateAddress
        ),
      ]

      const plugin: PredicatePlugin = getPluginThatReturns([endStateObject])

      const stateDB: StateDB = getStateDBThatReturns(verifiedStateUpdates)
      const pluginManager: PluginManager = getPluginManagerThatReturns(
        new Map<string, PredicatePlugin>([[predicateAddress, plugin]])
      )
      const stateManager: StateManager = new DefaultStateManager(
        stateDB,
        pluginManager
      )

      try {
        await stateManager.executeTransaction(
          transaction,
          nextBlockNumber,
          defaultWitness
        )
        assert(false, 'this call should have thrown an error.')
      } catch (e) {
        assert(true, 'this call threw an error as expected.')
      }
    })

    it('should throw if VerifiedStateUpdates have different predicates', async () => {
      const secondPredicateAddress = '0x87654321'
      const midPoint = end
        .sub(start)
        .divRound(new BigNumber(2))
        .add(start)
      const verifiedStateUpdates: VerifiedStateUpdate[] = [
        getVerifiedStateUpdate(
          start,
          midPoint,
          previousBlockNumber,
          depositAddress,
          predicateAddress
        ),
        getVerifiedStateUpdate(
          midPoint,
          end,
          previousBlockNumber,
          depositAddress,
          predicateAddress
        ),
      ]

      const firstStateObject: StateObject = {
        predicateAddress,
        data: { testResult: 'test' },
      }

      const plugin: PredicatePlugin = getPluginThatReturns([firstStateObject])

      const secondStateObject: StateObject = {
        predicateAddress: secondPredicateAddress,
        data: { testResult: 'test' },
      }
      const secondPlugin: PredicatePlugin = getPluginThatReturns([
        secondStateObject,
      ])

      const stateDB: StateDB = getStateDBThatReturns(verifiedStateUpdates)
      const pluginManager: PluginManager = getPluginManagerThatReturns(
        new Map<string, PredicatePlugin>([
          [predicateAddress, plugin],
          [secondPredicateAddress, secondPlugin],
        ])
      )
      const stateManager: StateManager = new DefaultStateManager(
        stateDB,
        pluginManager
      )

      try {
        await stateManager.executeTransaction(
          transaction,
          nextBlockNumber,
          defaultWitness
        )
        assert(false, 'this call should have thrown an error.')
      } catch (e) {
        assert(true, 'this call threw an error as expected.')
      }
    })

    it('should fail if same predicate but StateObjects do not match', async () => {
      const midPoint = end
        .sub(start)
        .divRound(new BigNumber(2))
        .add(start)
      const verifiedStateUpdates: VerifiedStateUpdate[] = [
        getVerifiedStateUpdate(
          start,
          midPoint,
          previousBlockNumber,
          depositAddress,
          predicateAddress
        ),
        getVerifiedStateUpdate(
          midPoint,
          end,
          previousBlockNumber,
          depositAddress,
          predicateAddress
        ),
      ]

      const firstStateObject: StateObject = {
        predicateAddress,
        data: { testResult: 'test' },
      }

      const secondStateObject: StateObject = {
        predicateAddress,
        data: { testResult: 'test 2' },
      }
      const plugin: PredicatePlugin = getPluginThatReturns([
        firstStateObject,
        secondStateObject,
      ])

      const stateDB: StateDB = getStateDBThatReturns(verifiedStateUpdates)
      const pluginManager: PluginManager = getPluginManagerThatReturns(
        new Map<string, PredicatePlugin>([[predicateAddress, plugin]])
      )
      const stateManager: StateManager = new DefaultStateManager(
        stateDB,
        pluginManager
      )

      try {
        await stateManager.executeTransaction(
          transaction,
          nextBlockNumber,
          defaultWitness
        )
        assert(false, 'this call should have thrown an error.')
      } catch (e) {
        assert(true, 'this call threw an error as expected.')
      }
    })

    it('should throw if block number is incorrect', async () => {
      const verifiedStateUpdates: VerifiedStateUpdate[] = [
        getVerifiedStateUpdate(
          start,
          end,
          previousBlockNumber,
          depositAddress,
          predicateAddress
        ),
      ]

      const plugin: PredicatePlugin = getPluginThatReturns([endStateObject])

      const stateDB: StateDB = getStateDBThatReturns(verifiedStateUpdates)
      const pluginManager: PluginManager = getPluginManagerThatReturns(
        new Map<string, PredicatePlugin>([[predicateAddress, plugin]])
      )
      const stateManager: StateManager = new DefaultStateManager(
        stateDB,
        pluginManager
      )

      try {
        await stateManager.executeTransaction(
          transaction,
          nextBlockNumber.add(ONE),
          defaultWitness
        )
        assert(false, 'this call should have thrown an error.')
      } catch (e) {
        assert(true, 'this call threw an error as expected.')
      }
    })
  })
})
