import { toHexString } from '@eth-optimism/core-utils'

/**
 * Basic timeout-based async sleep function.
 * @param ms Number of milliseconds to sleep.
 */
export const sleep = async (ms: number): Promise<void> => {
  return new Promise<void>((resolve) => {
    setTimeout(resolve, ms)
  })
}

export const assert = (condition: () => boolean, reason?: string) => {
  try {
    if (condition() === false) {
      throw new Error(`Assertion failed: ${reason}`)
    }
  } catch (err) {
    throw new Error(`Assertion failed: ${reason}\n${err}`)
  }
}

export const toRpcHexString = (n: number): string => {
  if (n === 0) {
    return '0x0'
  } else {
    // prettier-ignore
    return '0x' + toHexString(n).slice(2).replace(/^0+/, '')
  }
}

export const padHexString = (str: string, length: number): string => {
  if (str.length === 2 + length * 2) {
    return str
  } else {
    return '0x' + str.slice(2).padStart(length * 2, '0')
  }
}
