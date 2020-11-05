import BigNum = require('bn.js')
import assert from 'assert'

export type Endianness = 'B' | 'L'
export const BIG_ENDIAN: Endianness = 'B'
export const LITTLE_ENDIAN: Endianness = 'L'

/**
 * Class to represent big numbers in the codebase.
 * This is a façade, wrapping the underlying Big Number implementation so that we can
 * swap out implementations if necessary without breaking usages.
 */
export class BigNumber {
  private readonly num: BigNum

  constructor(
    number:
      | number
      | string
      | number[]
      | Uint8Array
      | Buffer
      | BigNumber
      | BigNum,
    base?: number | 'hex',
    endian?: Endianness
  ) {
    const parsedNumber = number instanceof BigNumber ? number.num : number
    const endianness = !!endian ? this.getBigNumEndianness(endian) : undefined

    if (!!base) {
      this.num = new BigNum(parsedNumber, base, endianness)
    } else if (!!endian) {
      this.num = new BigNum(parsedNumber, endianness)
    } else {
      this.num = new BigNum(parsedNumber)
    }
  }

  /**
   * Returns the min of the two provided BigNumbers.
   *
   * @param left the first BigNumber
   * @param right the second BigNumber
   */
  public static min(left: BigNumber, right: BigNumber): BigNumber {
    return new BigNumber(BigNum.min(left.num, right.num))
  }

  /**
   * Returns the max of the two provided BigNumbers.
   *
   * @param left the first BigNumber
   * @param right the second BigNumber
   */
  public static max(left: BigNumber, right: BigNumber): BigNumber {
    return new BigNumber(BigNum.max(left.num, right.num))
  }

  /**
   * Determines whether or not the provided input is a BigNumber
   *
   * @param num the number to inspect
   * @returns true if so, false otherwise
   */
  public static isBigNumber(num: any): boolean {
    if (num instanceof BigNumber) {
      return BigNum.isBN(num.num)
    }

    return false
  }

  /**
   * Creates a copy of `this` without the same memory reference.
   */
  public clone(): BigNumber {
    return new BigNumber(this)
  }

  /**
   * returns a string representation of this number with the provided base.
   *
   * @param base the base of the string number to output
   * @param length the desired length of the resulting string (will pad with 0s if necessary)
   * @returns the string representation
   */
  public toString(base: number | 'hex' = 'hex', length?: number): string {
    return length === undefined
      ? this.num.toString(base)
      : this.num.toString(base, length)
  }

  /**
   * Serializes this object to JSON by simply returning the represented number as a string.
   * @returns the JSON representing the number in question
   */
  public toJSON(): string {
    return this.toString('hex')
  }

  /**
   * Creates and returns a regular number from this number.
   * Note: Precision may be lost.
   *
   * @returns the number representing this
   */
  public toNumber(): number {
    return this.num.toNumber()
  }

  /**
   * Gets the Node.js Buffer representation of this BigNumber.
   *
   * @param endian the Endianness to use
   * @param length the length of the buffer
   */
  public toBuffer(endian?: Endianness, length?: number): Buffer {
    if (endian) {
      const endianness = this.getBigNumEndianness(endian)
      if (length) {
        return this.num.toBuffer(endianness, length)
      }
      return this.num.toBuffer(endianness)
    }
    return this.num.toBuffer()
  }

  /**************
   * Operations *
   **************/

  /**
   * Adds this BigNumber to the provided BigNumber and returns the result.
   *
   * @param other the BigNumber to add
   * @returns a *new* BigNumber with the result
   */
  public add(other: BigNumber): BigNumber {
    return new BigNumber(this.num.add(other.num))
  }

  /**
   * Subtracts the provided BigNumber from this BigNumber and returns the result.
   *
   * @param other the BigNumber to subtract
   * @returns a *new* BigNumber with the result
   */
  public sub(other: BigNumber): BigNumber {
    return new BigNumber(this.num.sub(other.num))
  }

  /**
   * Multiplies this BigNumber by the provided BigNumber and returns the result.
   *
   * @param other the BigNumber to multiply
   * @returns a *new* BigNumber with the result
   */
  public mul(other: BigNumber): BigNumber {
    return new BigNumber(this.num.mul(other.num))
  }

  /**
   * Divides this BigNumber by the provided BigNumber and returns the result.
   *
   * @param other the BigNumber to divide by
   * @returns a *new* BigNumber with the result
   */
  public div(other: BigNumber): BigNumber {
    return new BigNumber(this.num.div(other.num))
  }

  /**
   * Divides this BigNumber by the provided BigNumber and returns the *rounded* result.
   *
   * @param other the BigNumber to divide by
   * @returns a *new* BigNumber with the result
   */
  public divRound(other: BigNumber): BigNumber {
    // // TODO: This is only overridden because bn.js divRound rounds -3.3 to -4 instead of -3
    const thisAbs: BigNumber = this.abs()
    const otherAbs: BigNumber = other.abs()

    const remainderAbs: BigNumber = thisAbs.mod(otherAbs)
    const div: BigNumber = this.div(other)

    // if there's no remainder, it's the same as regular division
    if (remainderAbs.eq(ZERO)) {
      return div
    }

    const decimalAbs = remainderAbs.div(otherAbs)
    // if the decimal portion is GTE .5, round up, else round down (absolute)
    if (decimalAbs.gte(ONE_HALF)) {
      if (div.num.isNeg()) {
        return div.add(remainderAbs).sub(ONE)
      } else {
        return div.sub(remainderAbs).add(ONE)
      }
    } else {
      // rounding down (absolute)
      if (div.num.isNeg()) {
        return div.add(remainderAbs)
      } else {
        return div.sub(remainderAbs)
      }
    }
  }

  /**
   * Raises this BigNumber to exponent of the provided BigNumber and returns the result.
   *
   * @param other the BigNumber to divide by
   * @returns a *new* BigNumber with the result
   */
  public pow(exp: BigNumber): BigNumber {
    assert(
      exp.mod(ONE).eq(ZERO),
      'BigNumber.pow(...) does not support fractions at this time.'
    )
    assert(
      !exp.num.isNeg(),
      'BigNumber.pow(...) does not support negative exponents at this time.'
    )

    return new BigNumber(this.num.pow(exp.num))
  }

  /**
   * Mods this BigNumber by the provided BigNumber and returns the result.
   *
   * @param other the BigNumber to mod by
   * @returns a *new* BigNumber with the result
   */
  public mod(mod: BigNumber): BigNumber {
    assert(
      !this.num.isNeg() || !mod.num.isNeg(),
      'Big number does not support negative mod negative.'
    )
    return new BigNumber(this.num.mod(mod.num))
  }

  /**
   * Mods this BigNumber by the provided BigNumber and returns the result.
   *
   * @param other the BigNumber to mod by
   * @returns a *new* BigNumber with the result
   */
  public modNum(mod: number): BigNumber {
    assert(
      !this.num.isNeg() || mod >= 0,
      'Big number does not support negative mod negative.'
    )
    return new BigNumber(this.num.modn(mod))
  }

  /**
   * Returns the absolute value of this BigNumber as a *new* BigNumber.
   */
  public abs(): BigNumber {
    return new BigNumber(this.num.abs())
  }

  /**
   * Bitwise XORs the BigNumber with the provided BigNumber
   *
   * @param num The BigNumber to XOR
   * @returns The resulting BigNumber
   */
  public xor(num: BigNumber): BigNumber {
    return new BigNumber(this.num.xor(num.num))
  }

  /**
   * Bitwise ANDs the BigNumber with the provided BigNumber
   *
   * @param num The BigNumber to AND
   * @returns The resulting BigNumber
   */
  public and(num: BigNumber): BigNumber {
    return new BigNumber(this.num.and(num.num))
  }

  /**
   * Bitwise left-shifts the BigNumber the provided number of places
   * returning a new BigNumber as the result.
   *
   * @param num the number of places to shift
   */
  public shiftLeft(num: number): BigNumber {
    return new BigNumber(this.num.shln(num))
  }

  /**
   * Bitwise right-shifts the BigNumber the provided number of places
   * returning a new BigNumber as the result.
   *
   * @param num the number of places to shift
   */
  public shiftRight(num: number): BigNumber {
    return new BigNumber(this.num.shrn(num))
  }

  /**
   * Bitwise left-shifts the BigNumber the provided number of places
   *
   * @param num the number of places to shift
   */
  public shiftLeftInPlace(num: number): BigNumber {
    this.num.ishln(num)
    return this
  }

  /**
   * Bitwise right-shifts the BigNumber the provided number of places
   *
   * @param num the number of places to shift
   */
  public shiftRightInPlace(num: number): BigNumber {
    this.num.ishrn(num)
    return this
  }

  /***************
   * Comparisons *
   ***************/

  /**
   * Returns whether or not this BigNumber is greater than the provided BigNumber.
   *
   * @param other the BigNumber to compare to
   * @returns true if so, false otherwise
   */
  public gt(other: BigNumber): boolean {
    return this.num.gt(other.num)
  }

  /**
   * Returns whether or not this BigNumber is greater than or equal to the provided BigNumber.
   *
   * @param other the BigNumber to compare to
   * @returns true if so, false otherwise
   */
  public gte(other: BigNumber): boolean {
    return this.num.gte(other.num)
  }

  /**
   * Returns whether or not this BigNumber is less than the provided BigNumber.
   *
   * @param other the BigNumber to compare to
   * @returns true if so, false otherwise
   */
  public lt(other: BigNumber): boolean {
    return this.num.lt(other.num)
  }

  /**
   * Returns whether or not this BigNumber is less than or equal to the provided BigNumber.
   *
   * @param other the BigNumber to compare to
   * @returns true if so, false otherwise
   */
  public lte(other: BigNumber): boolean {
    return this.num.lte(other.num)
  }

  /**
   * Returns whether or not this BigNumber is equal to the provided BigNumber.
   *
   * @param other the BigNumber to compare to
   * @returns true if so, false otherwise
   */
  public eq(other: BigNumber): boolean {
    return this.num.eq(other.num)
  }

  /**
   * eq alias to comply with standard
   */
  public equals(other: BigNumber): boolean {
    return this.eq(other)
  }

  /**
   * Compares this BigNumber to the provided BigNumber.
   *
   * @param other the BigNumber to compare to
   * @returns -1 if this is smaller, 0 if they're equal, 1 if other is less than this
   */
  public compare(other: BigNumber): -1 | 0 | 1 {
    return this.lt(other) ? -1 : this.eq(other) ? 0 : 1
  }

  /**
   * Gets the bn.js endianness from the provided Endianness
   *
   * @param endianness the Endianness in question
   * @returns the bn.js endianness
   */
  private getBigNumEndianness(endianness: Endianness): 'be' | 'le' {
    if (endianness === BIG_ENDIAN) {
      return 'be'
    } else if (endianness === LITTLE_ENDIAN) {
      return 'le'
    }
    throw Error(`Cannot get Endianness from ${JSON.stringify(endianness)}`)
  }
}

export const ZERO = new BigNumber(0)
export const ONE = new BigNumber(1)
export const TWO = new BigNumber(2)
export const THREE = new BigNumber(3)
export const ONE_HALF = new BigNumber(0.5)
export const MAX_BIG_NUM = new BigNumber('0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF')
