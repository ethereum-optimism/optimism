/* Internal Imports */
import { Lib_MerkleTrie_TEST_JSON } from '../../../data'
import { runJsonTest } from '../../../helpers'

describe('Lib_MerkleTrie', () => {
  describe('JSON tests', () => {
    runJsonTest('TestLib_MerkleTrie', Lib_MerkleTrie_TEST_JSON)
  })
})
