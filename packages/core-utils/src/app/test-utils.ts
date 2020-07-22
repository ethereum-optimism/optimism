/* External Imports */
import * as assert from 'assert'

export class TestUtils {

  public static nullSafeEquals(first: any, second: any, msg?: string) {
    const firstExists: boolean = first !== undefined && first !== null
    const secondExists: boolean = second !== undefined && first !== null

    firstExists.should.equal(secondExists, msg || `undefined / not undefined mismatch!`)
    if (firstExists) {
      first.should.equal(second, msg || `values don't match!`)
    }
  }

  public static assertThrows(func: () => any, errorType?: any): void {
    let succeeded = true
    try {
      func()
      succeeded = false
    } catch (e) {
      if (!!errorType && !(e instanceof errorType)) {
        succeeded = false
      }
    }

    assert(
      succeeded,
      "Function didn't throw as expected or threw the wrong error."
    )
  }

  public static async assertThrowsAsync(
    func: () => Promise<any>,
    errorType?: any
  ): Promise<Error> {
    let succeeded = true
    let error: Error
    try {
      await func()
      succeeded = false
    } catch (e) {
      if (!!errorType && !(e instanceof errorType)) {
        succeeded = false
      }
      error = e
    }

    assert(
      succeeded,
      "Function didn't throw as expected or threw the wrong error."
    )
    return error
  }

  public static async assertRevertsAsync(
    revertMessage: string,
    func: () => Promise<any>
  ): Promise<void> {
    let succeeded = true
    try {
      await func()
      succeeded = false
    } catch (e) {
      if (e instanceof Error) {
        assert.equal(
          e.message,
          `VM Exception while processing transaction: revert ${revertMessage}`
        )
      } else {
        succeeded = false
      }
    }

    assert(
      succeeded,
      "Function didn't throw as expected or threw the wrong error."
    )
  }
}
