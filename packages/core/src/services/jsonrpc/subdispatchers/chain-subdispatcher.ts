/* External Imports */
import { Service } from '@nestd/core'

/* Services */
import { ChainService } from '../../chain.service'
import { ChainDB } from '../../db/interfaces/chain-db'

/* Internal Imports */
import { BaseSubdispatcher } from './base-subdispatcher'

/**
 * Subdispatcher that handles chain-related requests.
 */
@Service()
export class ChainSubdispatcher extends BaseSubdispatcher {
  public readonly prefix = '_pg'

  constructor(
    private readonly chain: ChainService,
    private readonly chaindb: ChainDB
  ) {
    super()
  }

  get methods(): { [key: string]: (...args: any) => any } {
    const chain = this.chain
    const chaindb = this.chaindb

    return {
      /* ChainDB */
      getBlockHeader: chaindb.getBlockHeader.bind(chaindb),
      getLastSyncedBlock: chaindb.getLatestBlock.bind(chaindb),
      getTransaction: chaindb.getTransaction.bind(chaindb),

      /* Chain */
      finalizeExits: chain.finalizeExits.bind(chain),
      getExits: chain.getExitsWithStatus.bind(chain),
      sendTransaction: chain.sendTransaction.bind(chain),
      startExit: chain.startExit.bind(chain),
    }
  }
}
