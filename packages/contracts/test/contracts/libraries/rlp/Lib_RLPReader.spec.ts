/* External Imports */
import * as rlp from 'rlp'

/* Internal Imports */
import { Lib_RLPReader_TEST_JSON } from '../../../data'
import { runJsonTest, toHexString } from '../../../helpers'

describe('Lib_RLPReader', () => {
  //console.log(JSON.stringify(Lib_RLPReader_TEST_JSON2, null, 4))
  describe('JSON tests', () => {
    runJsonTest('TestLib_RLPReader', Lib_RLPReader_TEST_JSON)
  })
})
