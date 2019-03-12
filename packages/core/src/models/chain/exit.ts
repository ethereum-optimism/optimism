import BigNum from 'bn.js'
import { StateObject } from '@pigi/utils'

export interface ExitArgs {
  owner: string
  id: BigNum
  token: BigNum
  start: BigNum
  end: BigNum
  block: BigNum
}

export class Exit extends StateObject {
  public owner: string
  public id: BigNum
  public token: BigNum
  public start: BigNum
  public end: BigNum
  public block: BigNum
  public completed?: boolean
  public finalized?: boolean

  constructor(args: ExitArgs) {
    super(args)

    this.owner = args.owner
    this.id = args.id
    this.token = args.token
    this.start = args.start
    this.end = args.end
    this.block = args.block
  }
}
