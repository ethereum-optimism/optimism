import { expect } from 'chai'
import sinon from 'sinon'

import {
  addOptimismNetworkToProvider,
  Optimism,
  toggleLayer,
} from '../../src/utils/chain'

describe('chain utils', () => {
  describe('addOptimismNetworkToMetamask', () => {
    it('should issue an rpc request for wallet_addEthereumChain', async () => {
      const provider = { request: sinon.spy() }
      await addOptimismNetworkToProvider(provider)
      expect(provider.request.getCalls()[0].firstArg.params[0]).to.eq(Optimism)
      expect(provider.request.getCalls()[0].firstArg.method).to.eql(
        'wallet_addEthereumChain'
      )
    })
  })
  describe('toggleLayer', () => {
    it('should issue an rpc request for wallet_switchEthereumChain', async () => {
      const provider = {
        request: sinon.spy(({ method }: { method: string; params?: any[] }) => {
          if (method === 'eth_chainId') {
            return Promise.resolve(1)
          }
          return Promise.resolve()
        }),
      }
      await toggleLayer(provider)
      expect(
        provider.request.calledWith({
          method: 'eth_chainId',
        })
      ).to.eql(true)
      expect(provider.request.getCalls()[1].firstArg.params[0].chainId).to.eq(
        '0xa'
      )
      expect(provider.request.getCalls()[1].firstArg.method).to.eql(
        'wallet_switchEthereumChain'
      )
    })
  })
})
