import { StateObject } from '@pigi/utils'

export interface ExitArgs extends StateObject {
  owner: string
  id: string
}

export class Exit extends StateObject {
  public readonly owner: string
  public readonly id: string

  constructor(args: ExitArgs) {
    super({
      ...args,
      predicate: null,
      state: null,
    })

    this.owner = args.owner
    this.id = args.id
  }
}
