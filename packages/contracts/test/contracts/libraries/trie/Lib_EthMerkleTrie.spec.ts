/* Internal Imports */
import { Lib_EthMerkleTrie_TEST_JSON } from '../../../data'
import { runJsonTest } from '../../../helpers'

describe('Lib_EthMerkleTrie', () => {
  describe('JSON tests', () => {
    runJsonTest('TestLib_EthMerkleTrie', Lib_EthMerkleTrie_TEST_JSON)
  })
})
