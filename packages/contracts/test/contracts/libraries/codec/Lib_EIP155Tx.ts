/* tslint:disable:no-empty */
import '../../../setup'

/* Internal Imports */
import { Lib_EIP155Tx_TEST_JSON } from '../../../data'
import { runJsonTest } from '../../../helpers'

// Currently running tests from here:
// https://github.com/ethereumjs/ethereumjs-tx/blob/master/test/ttTransactionTestEip155VitaliksTests.json

describe('Lib_EIP155Tx', () => {
  describe('JSON tests', () => {
    runJsonTest('TestLib_EIP155Tx', Lib_EIP155Tx_TEST_JSON)
  })
})
