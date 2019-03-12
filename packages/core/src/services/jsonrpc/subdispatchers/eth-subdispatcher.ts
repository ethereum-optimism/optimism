/* External Imports */
import { Service } from '@nestd/core'

/* Services */
import { ETHProvider } from '../../eth/eth-provider'
import { ContractProvider } from '../../eth/contract-provider'

/* Internal Imports */
import { BaseSubdispatcher } from './base-subdispatcher'

/**
 * Subdispatcher that handles Ethereum-related requests.
 */
@Service()
export class ETHSubdispatcher extends BaseSubdispatcher {
  public readonly prefix = 'pg_'

  constructor(
    private readonly eth: ETHProvider,
    private readonly contract: ContractProvider
  ) {
    super()
  }

  get methods(): { [key: string]: (...args: any) => any } {
    const eth = this.eth
    const contract = this.contract

    return {
      /* Contract */
      deposit: contract.deposit.bind(contract),
      getCurrentBlock: contract.getCurrentBlock.bind(contract),
      getTokenId: contract.getTokenId.bind(contract),
      listToken: contract.listToken.bind(contract),

      /* ETH */
      getCurrentEthBlock: eth.getCurrentBlock.bind(eth),
      getEthBalance: eth.getBalance.bind(eth),
    }
  }
}
