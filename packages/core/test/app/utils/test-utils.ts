import {
  AbiRange,
  AbiStateObject,
  AbiStateUpdate,
} from '../../../src/app/serialization'
import { TransactionResult } from '../../../src/types/serialization'
import { BigNumber, ONE } from '../../../src/app/utils'
import * as assert from 'assert'

export class TestUtils {
  public static generateNSequentialStateUpdates(
    numberOfUpdates: number
  ): AbiStateUpdate[] {
    const stateUpdates: AbiStateUpdate[] = []
    for (let i = 0; i < numberOfUpdates; i++) {
      const stateObject = new AbiStateObject(
        '0xbdAd2846585129Fc98538ce21cfcED21dDDE0a63',
        '0x123456'
      )
      const range = new AbiRange(
        new BigNumber(i * 100),
        new BigNumber((i + 0.5) * 100)
      )
      const stateUpdate = new AbiStateUpdate(
        stateObject,
        range,
        ONE,
        '0xbdAd2846585129Fc98538ce21cfcED21dDDE0a63'
      )
      stateUpdates.push(stateUpdate)
    }
    return stateUpdates
  }

  public static generateNSequentialTransactionResults(
    numberofUpdates: number
  ): TransactionResult[] {
    return this.generateNSequentialStateUpdates(numberofUpdates).map(
      (abiStateUpdate: AbiStateUpdate): TransactionResult => {
        return {
          stateUpdate: abiStateUpdate,
          validRanges: [abiStateUpdate.range],
        }
      }
    )
  }

  public static assertThrows(func: () => any, errorType: any): void {
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
}
