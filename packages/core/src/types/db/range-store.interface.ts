import { BigNumber } from '../../app/utils'

export interface Range {
  start: BigNumber
  end: BigNumber
}

export interface BlockRange extends Range {
  block: BigNumber
}
