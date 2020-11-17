import { expect } from '../setup'

import ganache from 'ganache-core'
import { getContractFactory } from '@eth-optimism/contracts'

import { main } from '../../src'

describe('Message Relayer: basic tests', () => {
  before(async () => {
    const l1Server = ganache.server()
    const l2Server = ganache.server()

    await new Promise<void>((resolve) => {
      l1Server.listen(8545, null, null, () => {
        resolve()
      })
    })

    await new Promise<void>((resolve) => {
      l2Server.listen(8545, null, null, () => {
        resolve()
      })
    })
  })

  before(async () => {
    main({
      l1RpcEndpoint: 'http://localhost:8545',
      l2RpcEndpoint: 'http://localhost:8546',
      stateCommitmentChainAddress: '',
      l1CrossDomainMessengerAddress: '',
      l2CrossDomainMessengerAddress: '',
      l2ChainStartingHeight: 0,
      pollingInterval: 15,
      relayerPrivateKey: '',
    })
  })
})
