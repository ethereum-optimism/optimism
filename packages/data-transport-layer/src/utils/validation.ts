import * as url from 'url'

import { fromHexString } from '@eth-optimism/core-utils'

export const validators = {
  isBoolean: (val: any): boolean => {
    return typeof val === 'boolean'
  },
  isString: (val: any): boolean => {
    return typeof val === 'string'
  },
  isHexString: (val: any): boolean => {
    return (
      validators.isString(val) &&
      val.startsWith('0x') &&
      fromHexString(val).length === (val.length - 2) / 2
    )
  },
  isAddress: (val: any): boolean => {
    return validators.isHexString(val) && val.length === 42
  },
  isInteger: (val: any): boolean => {
    return Number.isInteger(val)
  },
  isUrl: (val: any): boolean => {
    try {
      const parsed = new url.URL(val)
      return (
        parsed.protocol === 'ws:' ||
        parsed.protocol === 'http:' ||
        parsed.protocol === 'https:'
      )
    } catch (err) {
      return false
    }
  },
  isJsonRpcProvider: (val: any): boolean => {
    return val && val.ready !== undefined
  },
  isLevelUP: (val: any): boolean => {
    // TODO: Fix?
    return val && val.db
  },
}
