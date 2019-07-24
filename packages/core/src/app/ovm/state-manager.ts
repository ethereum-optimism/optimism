import {
  HistoryProof,
  PluginManager,
  PredicatePlugin,
  Range,
  StateDB,
  StateManager,
  StateQuery,
  StateQueryResult,
  StateUpdate,
  Transaction,
  VerifiedStateUpdate,
  isValidTransaction,
  isValidVerifiedStateUpdate,
  StateObject,
} from '../../types'
import { BigNumber, getOverlappingRange, ONE, rangesIntersect } from '../../app'

/**
 * StateManager that validates transactions and wraps and modifies StateDB as necessary.
 *
 * See: http://spec.plasma.group/en/latest/src/05-client-architecture/state-manager.html for more details.
 */
export class DefaultStateManager implements StateManager {
  public constructor(
    private stateDB: StateDB,
    private pluginManager: PluginManager
  ) {}

  public async executeTransaction(
    transaction: Transaction,
    inBlock: BigNumber,
    witness: string
  ): Promise<{ stateUpdate: StateUpdate; validRanges: Range[] }> {
    const result = {
      stateUpdate: undefined,
      validRanges: [],
    }

    if (!isValidTransaction(transaction)) {
      throw new Error(
        `Cannot execute invalid Transaction: ${JSON.stringify(transaction)}`
      )
    }

    // Get verified updates for range
    const { start, end }: Range = transaction.range
    const verifiedUpdates: VerifiedStateUpdate[] = await this.stateDB.getVerifiedStateUpdates(
      start,
      end
    )

    // Iterate over the verified updates, transition their state, and add their ranges to the return object
    for (const verifiedUpdate of verifiedUpdates) {
      this.validateVerifiedStateUpdateForTransaction(
        verifiedUpdate,
        transaction,
        inBlock
      )

      const predicatePlugin: PredicatePlugin = await this.pluginManager.getPlugin(
        verifiedUpdate.stateUpdate.stateObject.predicateAddress
      )

      const computedState: StateObject = await predicatePlugin.executeStateTransition(
        verifiedUpdate.stateUpdate,
        transaction,
        witness
      )

      result.validRanges.push(
        getOverlappingRange(transaction.range, verifiedUpdate.range)
      )

      if (result.stateUpdate === undefined) {
        result.stateUpdate = {
          range: transaction.range,
          stateObject: computedState,
          depositAddress: transaction.depositAddress,
          plasmaBlockNumber: inBlock,
        }
      } else if (result.stateUpdate.stateObject !== computedState) {
        throw new Error(`State transition resulted in two different states: ${JSON.stringify(
          result.stateUpdate.stateObject
        )} and 
          ${computedState}. Latter differed from former at range ${JSON.stringify(
          result.validRanges.pop()
        )}.`)
      }
    }

    return result
  }

  public ingestHistoryProof(historyProof: HistoryProof): Promise<void> {
    throw Error('DefaultStateManager.ingestHistoryProof is not implemented.')
  }

  public queryState(query: StateQuery): Promise<StateQueryResult[]> {
    throw Error('DefaultStateManager.queryState is not implemented.')
  }

  /**
   * Validates that the provided VerifiedStateUpdate is valid for the provided Transaction and block.
   *
   * @param verifiedUpdate the VerifiedStateUpdate in question
   * @param transaction the Transaction in question
   * @param inBlock the Block number
   *
   * @throws if invalid
   */
  private validateVerifiedStateUpdateForTransaction(
    verifiedUpdate: VerifiedStateUpdate,
    transaction: Transaction,
    inBlock: BigNumber
  ): void {
    if (!isValidVerifiedStateUpdate(verifiedUpdate)) {
      throw new Error(
        `Cannot process transaction for invalid VerifiedStateUpdate: ${JSON.stringify(
          verifiedUpdate
        )}`
      )
    }

    // If the ranges don't overlap, eagerly exit
    if (!rangesIntersect(verifiedUpdate.range, transaction.range)) {
      throw Error(`VerifiedStateUpdate for range [${transaction.range.start}, ${
        transaction.range.end
      }) is outside of range: 
        ${JSON.stringify(
          verifiedUpdate.range
        )}. VerifiedStateUpdate: ${JSON.stringify(verifiedUpdate)}.`)
    }

    if (!verifiedUpdate.verifiedBlockNumber.add(ONE).eq(inBlock)) {
      throw Error(`VerifiedStateUpdate has block ${
        verifiedUpdate.verifiedBlockNumber
      } and ${inBlock.sub(ONE)} was expected. 
          VerifiedStateUpdate: ${JSON.stringify(verifiedUpdate)}`)
    }
  }
}
