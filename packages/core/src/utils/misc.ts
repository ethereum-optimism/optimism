/* External Imports */
import BigNum from 'bn.js'

export const stringify = (value: any): string => {
  if (!(typeof value === 'string')) {
    value = JSON.stringify(value)
  }
  return value as string
}

export const jsonify = (value: string): {} => {
  return this.isJson(value) ? JSON.parse(value) : value
}

export const isJson = (value: string): boolean => {
  try {
    JSON.parse(value)
  } catch (err) {
    return false
  }
  return true
}

export const bnMin = (a: BigNum, b: BigNum): BigNum => {
  return a.lt(b) ? a : b
}

export const bnMax = (a: BigNum, b: BigNum): BigNum => {
  return a.gt(b) ? a : b
}
