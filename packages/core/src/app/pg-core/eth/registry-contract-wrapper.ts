import { compiledPlasmaRegistry } from '@pigi/contracts'
import Web3 from 'web3/types'
import { Contract } from 'web3-eth-contract/types'

/**
 * Basic registry contract wrapper.
 */
export class RegistryContractWrapper {
  private contract: Contract

  /**
   * Creates the wrapper.
   * @param web3 Web3 instance used to make calls.
   * @param address Address to interact with.
   */
  constructor(web3: Web3, address: string) {
    this.contract = new web3.eth.Contract(
      compiledPlasmaRegistry.abi as any,
      address
    )
  }

  /**
   * Queries the address of a plasma chain.
   * @param plasmaChainName Name of the chain.
   * @returns the address of the chain.
   */
  public async getPlasmaChainAddress(plasmaChainName: string): Promise<string> {
    return this.contract.methods.getPlasmaChainAddress(plasmaChainName)
  }
}
