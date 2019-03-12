import { instance, mock, when } from 'ts-mockito'

import { ChainService } from '../../../src/services'

const mockChainService = mock(ChainService)
when(mockChainService.started).thenReturn(true)

const chain = instance(mockChainService)

export { mockChainService, chain }
