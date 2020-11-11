/* tslint:disable:no-empty */
import { expect } from '../../../setup'

/* Internal Imports */
import { Lib_OVMCodec_TEST_JSON } from '../../../data'
import { runJsonTest, toHexString } from '../../../helpers'

describe('Lib_OVMCodec', () => {
  describe('JSON tests', () => {
    runJsonTest('TestLib_OVMCodec', Lib_OVMCodec_TEST_JSON)
  })
})
