import '../../setup'

/* External Imports */
import debug from 'debug'
const log = debug('test:info:state-manager')
import BigNum = require('bn.js')
import {DefaultStateDB, DefaultStateManager} from "../../../src/app/client";
import {PluginManager, PredicatePlugin, StateDB, StateManager} from "../../../src/interfaces/client";
import {Range, StateUpdate, Transaction, VerifiedStateUpdate} from "../../../src/interfaces/common/utils";


/*******************
 * Mocks & Helpers *
 *******************/

class DummyPluginManager implements PluginManager{
  public getPlugin(address: string): Promise<PredicatePlugin | undefined> {
    return undefined;
  }

  public loadPlugin(address: string, path: string): Promise<PredicatePlugin> {
    return undefined;
  }
}

class DummyPredicatePlugin implements PredicatePlugin {
  public executeStateTransition(previousStateUpdate: StateUpdate, transaction: Transaction): Promise<StateUpdate> {
    return undefined;
  }
}

function getPluginThatReturns(stateUpdate: StateUpdate) : PredicatePlugin {
  const predicatePlugin: PredicatePlugin = new DummyPredicatePlugin();
  predicatePlugin.executeStateTransition = async (previousStateUpdate: StateUpdate, transaction: Transaction): Promise<StateUpdate> => {
    return stateUpdate
  }
  return predicatePlugin
}

function getPluginManagerThatReturns(pluginMap: Map<string, PredicatePlugin>): PluginManager {
  const pluginManager: PluginManager = new DummyPluginManager();
  pluginManager.getPlugin = async (address: string): Promise<PredicatePlugin | undefined> => {
    return pluginMap.get(address)
  }
  return pluginManager
}

function getStateDBThatReturns(verifiedStateUpdates: VerifiedStateUpdate[]) : StateDB {
  const stateDB = new DefaultStateDB();
  stateDB.getVerifiedStateUpdates = async (start: BigNum, end: BigNum): Promise<VerifiedStateUpdate[]> => {
    return verifiedStateUpdates
  }
  return stateDB
}

function getStateUpdate(start: BigNum, end: BigNum, predicateAddress: string, parameters: any = {}) {
  return {
    id: {
      start: start,
      end: end
    },
    newState: {
      predicate: predicateAddress,
      parameters: parameters
    }
  }
}

function getVerifiedStateUpdate(start: BigNum, end: BigNum, block: number, predicateAddress: string, parameters: any = {}) {
  return {
    start: start,
    end: end,
    verifiedBlockNumber: block,
    stateUpdate: getStateUpdate(start, end, predicateAddress, parameters)
  }
}

function getTransaction(start: BigNum, end: BigNum, predicateAddress: string, block: number, witness: any = undefined, parameters: any = {}) {
  return {
    stateUpdate: getStateUpdate(start, end, predicateAddress, parameters),
    witness: witness,
    block: block
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
    const start: BigNum = new BigNum(10)
    const end: BigNum = new BigNum(20)
    const previousBlockNumber: number = 10
    const nextBlockNumber: number = 11


    it('should process transaction for contiguous range', async () => {
      const predicateAddress = '0x12345678'
      const verifiedStateUpdates: VerifiedStateUpdate[] = [getVerifiedStateUpdate(start, end, previousBlockNumber, predicateAddress)]
      const transaction: Transaction = getTransaction(start, end, predicateAddress, nextBlockNumber, '0x12345678')

      const endStateUpdate: StateUpdate = getStateUpdate(start, end, predicateAddress, {testResult: "test"})
      const plugin: PredicatePlugin = getPluginThatReturns(endStateUpdate)

      const stateDB: StateDB = getStateDBThatReturns(verifiedStateUpdates);
      const pluginManager: PluginManager = getPluginManagerThatReturns(new Map<string, PredicatePlugin>([[predicateAddress, plugin]]))
      const stateManager: StateManager = new DefaultStateManager(stateDB, pluginManager)

      const result: {stateUpdate: StateUpdate, validRanges: Range[]} = await stateManager.executeTransaction(transaction)

      result.stateUpdate.should.equal(endStateUpdate)

      result.validRanges.length.should.equal(1)
      result.validRanges[0].start.should.equal(start)
      result.validRanges[0].end.should.equal(end)
    })
  })
})
