import { Lib_RLPReader_TEST_JSON } from '../../../data'
import { runJsonTest } from '../../../helpers'

describe('Lib_RLPReader', () => {
  describe('JSON tests', () => {
    runJsonTest('TestLib_RLPReader', Lib_RLPReader_TEST_JSON)
  })
})
