/* External Imports */
import * as assert from 'assert'

export class TestUtils {
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
  ): Promise<void> {
    let succeeded = true
    try {
      await func()
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
}
