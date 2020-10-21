/* Internal Imports */
import { NON_ZERO_ADDRESS } from '../constants'

export const DUMMY_CONTEXT = {
  GLOBAL: {
    ovmCHAINID: 11,
  },
  TRANSACTION: {
    ovmORIGIN: NON_ZERO_ADDRESS,
    ovmTIMESTAMP: 22,
    ovmGASLIMIT: 33,
    ovmTXGASLIMIT: 44,
    ovmQUEUEORIGIN: 55,
  },
  MESSAGE: {
    ovmCALLER: NON_ZERO_ADDRESS,
    ovmADDRESS: NON_ZERO_ADDRESS,
    ovmSTATICCTX: true,
  },
}
