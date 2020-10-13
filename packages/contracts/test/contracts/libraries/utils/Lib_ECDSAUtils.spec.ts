/* Internal Imports */
import { Lib_ECDSAUtils_TEST_JSON } from '../../../data'
import { runJsonTest } from '../../../helpers'

describe('Lib_ECDSAUtils', () => {
  describe('JSON tests', () => {
    runJsonTest('TestLib_ECDSAUtils', Lib_ECDSAUtils_TEST_JSON)
  })
})
