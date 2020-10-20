/* External Imports */
import * as rlp from 'rlp'

/* Internal Imports */
import { Lib_RLPReader_TEST_JSON } from '../../../data'
import { runJsonTest, toHexString } from '../../../helpers'

describe('Lib_RLPReader', () => {
  describe('JSON tests', () => {
    runJsonTest('TestLib_RLPReader', Lib_RLPReader_TEST_JSON)
  })
})
