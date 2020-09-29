/* Internal Imports */
import { Lib_SecureMerkleTrie_TEST_JSON } from '../../../data'
import { runJsonTest } from '../../../helpers'

describe('Lib_SecureMerkleTrie', () => {
  describe('JSON tests', () => {
    runJsonTest('TestLib_SecureMerkleTrie', Lib_SecureMerkleTrie_TEST_JSON)
  })
})
