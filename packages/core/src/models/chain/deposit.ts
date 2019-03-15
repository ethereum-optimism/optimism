import { StateObject, StateObjectData } from '@pigi/utils'

export interface DepositArgs extends StateObjectData {
  owner: string
}

export class Deposit extends StateObject {
  public readonly owner: string

  constructor(args: DepositArgs) {
    super(args)

    this.owner = args.owner
  }

  /**
   * Checks if this deposit equals some other deposit.
   * @param other Other deposit to check against.
   * @returns `true` if this deposit equals the other, `false` otherwise.
   */
  public equals(other: Deposit): boolean {
    return (
      this.owner === other.owner &&
      this.state === other.state &&
      this.predicate === other.predicate &&
      this.start.eq(other.start) &&
      this.end.eq(other.end) &&
      this.block.eq(other.block)
    )
  }
}
