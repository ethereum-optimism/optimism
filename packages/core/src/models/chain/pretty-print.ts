import BigNum from 'bn.js'

type PrettyPrintable = string | number | BigNum | boolean | any

/**
 * Base class that allows for printing objects in a prettified manner.
 */
export class PrettyPrint {
  [key: string]: PrettyPrintable

  /**
   * Returns the object as a pretty JSON string.
   * Converts BigNums to decimal strings.
   * @returns the JSON string.
   */
  public prettify(): string {
    const parsed: { [key: string]: PrettyPrintable } = {}
    Object.keys(this).forEach((key) => {
      const value = this[key]
      parsed[key] = BigNum.isBN(value) ? value.toString(10) : value
    })
    return JSON.stringify(parsed, null, 2)
  }
}
