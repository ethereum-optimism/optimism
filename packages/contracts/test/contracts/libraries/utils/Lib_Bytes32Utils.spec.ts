import { Lib_Bytes32Utils_TEST_JSON } from '../../../data'
import { runJsonTest } from '../../../helpers'

describe('Lib_Bytes32Utils', () => {
  describe('JSON tests', () => {
    runJsonTest('TestLib_Bytes32Utils', Lib_Bytes32Utils_TEST_JSON)
  })
})
