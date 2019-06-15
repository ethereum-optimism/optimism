import BigNum = require('bn.js')

export interface Range {
  start: BigNum
  end: BigNum
}

export interface BlockRange extends Range {
  block: BigNum
}
