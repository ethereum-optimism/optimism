/* Internal Imports */
import { Lib_BytesUtils_TEST_JSON } from '../../../data'
import { runJsonTest } from '../../../helpers'

describe('Lib_BytesUtils', () => {
  describe('JSON tests', () => {
    runJsonTest('TestLib_BytesUtils', Lib_BytesUtils_TEST_JSON)
  })
})
