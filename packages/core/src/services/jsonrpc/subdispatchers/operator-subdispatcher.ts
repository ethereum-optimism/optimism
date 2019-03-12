/* External Imports */
import { Service } from '@nestd/core'

/* Services */
import { OperatorProvider } from '../../operator/operator-provider'

/* Internal Imports */
import { BaseSubdispatcher } from './base-subdispatcher'

/**
 * Subdispatcher that handles Operator-related requests.
 */
@Service()
export class OperatorSubdispatcher extends BaseSubdispatcher {
  public readonly prefix = 'pg_'

  constructor(private readonly operator: OperatorProvider) {
    super()
  }

  get methods(): { [key: string]: (...args: any) => any } {
    const operator = this.operator

    return {
      /* Operator */
      getEthInfo: operator.getEthInfo.bind(operator),
      getNextBlock: operator.getNextBlock.bind(operator),
      submitBlock: operator.submitBlock.bind(operator),
    }
  }
}
