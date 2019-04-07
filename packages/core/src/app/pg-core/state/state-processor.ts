/* External Imports */
import { StateUpdate } from '@pigi/utils'

/* Internal Imports */
import { RangeStore } from '../../common'

/**
 * Used to process state updates.
 */
export class StateProcessor {
  private rangeStore: RangeStore<StateUpdate>

  /**
   * Creates the processor.
   * @param state Initial state to load.
   */
  constructor(state: StateUpdate[] = []) {
    this.rangeStore = new RangeStore<StateUpdate>(state)
  }

  /**
   * @returns the current state.
   */
  get state(): StateUpdate[] {
    return this.rangeStore.ranges
  }

  /**
   * Merges another StateManager into this one.
   * @param other Other state manager to merge.
   */
  public merge(other: StateProcessor): void {
    for (const stateUpdate of other.state) {
      this.addStateUpdate(stateUpdate)
    }
  }

  /**
   * Returns any states that overlap with the given state.
   * Only returns the portion of those states that actually overlap.
   * @param stateUpdate Object to overlap with.
   * @returns any overlapping state portions.
   */
  public getOldStates(stateUpdate: StateUpdate): StateUpdate[] {
    return this.rangeStore.getOverlapping(stateUpdate)
  }

  /**
   * Checks if the manager has a certain object.
   * @param stateUpdate Object to query.
   * @returns `true` if it has the object, `false` otherwise.
   */
  public hasStateUpdate(stateUpdate: StateUpdate): boolean {
    const overlap = this.getOldStates(stateUpdate)
    return overlap.some((existing) => {
      return existing.equals(stateUpdate)
    })
  }

  /**
   * Forcibly adds a state object to the local state.
   * Should *only* be used for adding deposits and exits.
   * @param stateUpdate Object to add.
   */
  public addStateUpdate(stateUpdate: StateUpdate): void {
    this.rangeStore.addRange(stateUpdate)
  }

  /**
   * Applies a state object against the current state.
   * @param stateUpdate Object to apply.
   */
  public applyStateUpdate(stateUpdate: StateUpdate): void {
    const components = stateUpdate.components()

    for (const component of components) {
      if (component.implicit) {
        this.rangeStore.incrementBlocks(component)
      } else {
        this.rangeStore.addRange(component)
      }
    }
  }
}
