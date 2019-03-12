/* External Imports */
import { StateObject } from '@pigi/utils'

/* Internal Imports */
import { RangeStore } from './range-store'

export class StateManager {
  private rangeStore: RangeStore<StateObject>

  constructor(state: StateObject[] = []) {
    this.rangeStore = new RangeStore<StateObject>(state)
  }

  /**
   * @returns the current state.
   */
  get state(): StateObject[] {
    return this.rangeStore.ranges
  }

  /**
   * Merges another StateManager into this one.
   * @param other Other state manager to merge.
   */
  public merge(other: StateManager): void {
    for (const stateObject of other.state) {
      this.addStateObject(stateObject)
    }
  }

  /**
   * Returns any states that overlap with the given state.
   * Only returns the portion of those states that actually overlap.
   * @param stateObject Object to overlap with.
   * @returns any overlapping state portions.
   */
  public getOldStates(stateObject: StateObject): StateObject[] {
    return this.rangeStore.getOverlapping(stateObject)
  }

  /**
   * Checks if the manager has a certain object.
   * @param stateObject Object to query.
   * @returns `true` if it has the object, `false` otherwise.
   */
  public hasStateObject(stateObject: StateObject): boolean {
    const overlap = this.getOldStates(stateObject)
    return overlap.some((existing) => {
      return existing.equals(stateObject)
    })
  }

  /**
   * Forcibly adds a state object to the local state.
   * Should *only* be used for adding deposits and exits.
   * @param stateObject Object to add.
   */
  public addStateObject(stateObject: StateObject): void {
    this.rangeStore.addRange(stateObject)
  }

  /**
   * Applies a state object against the current state.
   * @param stateObject Object to apply.
   */
  public applyStateObject(stateObject: StateObject): void {
    const components = stateObject.components()

    for (const component of components) {
      if (component.implicit) {
        this.rangeStore.incrementBlocks(component)
      } else {
        this.rangeStore.addRange(component)
      }
    }
  }
}
