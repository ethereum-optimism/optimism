import { instance, mock, when } from 'ts-mockito'

import { ChainDB } from '../../../src/services'

const mockChainDB = mock(ChainDB)
when(mockChainDB.started).thenReturn(true)

const chaindb = instance(mockChainDB)

export { mockChainDB, chaindb }
