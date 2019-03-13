/* External Imports */
import { Service } from '@nestd/core'

/* Services */
import { OperatorService } from '../../operator.service'

/* Internal Imports */
import { BaseRpcModule } from './base-rpc-module'
import { EthInfo } from '../../../models/operator'

/**
 * Subdispatcher that handles Operator-related requests.
 */
@Service()
export class OperatorRpcModule extends BaseRpcModule {
  public readonly prefix = 'pg_'

  constructor(private readonly operator: OperatorService) {
    super()
  }

  /**
   * @returns contract information.
   */
  public async getEthInfo(): Promise<EthInfo> {
    return this.operator.getEthInfo()
  }

  /**
   * @returns the next plasma block that will be published.
   */
  public async getNextPlasmaBlock(): Promise<number> {
    return this.operator.getNextBlock()
  }

  /**
   * Attempts to force the operator to submit a block.
   * Will only work if this is enabled on the operator's
   * side of things. Usually only used for testing.
   */
  public async submitBlock(): Promise<void> {
    return this.operator.submitBlock()
  }
}
