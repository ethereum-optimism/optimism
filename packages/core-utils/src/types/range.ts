import { BigNumber } from '../app'

export interface Range {
  start: BigNumber
  end: BigNumber
}

export interface BlockRange extends Range {
  block: BigNumber
}
