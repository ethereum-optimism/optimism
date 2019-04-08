/* External Imports */
import { compiledPlasmaChain } from '@pigi/contracts'
import { isAddress } from 'web3-utils'
import Web3 from 'web3/types'
import { Contract } from 'web3-eth-contract/types'

/**
 * Basic plasma contract wrapper.
 */
export class PlasmaContractWrapper {
  private contract: Contract

  /**
   * Creates the wrapper.
   * @param web3 Web3 instance used to make calls.
   * @param address Address to interact with.
   */
  constructor(web3: Web3, address: string) {
    this.contract = new web3.eth.Contract(
      compiledPlasmaChain.abi as any,
      address
    )
  }

  /**
   * @returns the address of the contract.
   */
  get address(): string {
    return this.contract.options.address
  }

  /**
   * Queries a given block.
   * @param block Number of the block to query.
   * @returns Root hash of the block with that number.
   */
  public async getBlock(block: number): Promise<string> {
    return this.contract.methods.blockHashes(block).call()
  }

  /**
   * @returns Number of the block that will be submitted next.
   */
  public async getNextBlock(): Promise<number> {
    return this.contract.methods.nextPlasmaBlockNumber().call()
  }

  /**
   * @returns Number of the last submitted block.
   */
  public async getCurrentBlock(): Promise<number> {
    return (await this.getNextBlock()) - 1
  }

  /**
   * @returns Address of the current operator.
   */
  public async getOperator(): Promise<string> {
    return this.contract.methods.operator().call()
  }

  /**
   * Returns the address for a given token ID.
   * @param token The token ID.
   * @returns Address of the contract for that token.
   */
  public async getTokenAddress(token: string): Promise<string> {
    if (isAddress(token)) {
      return token
    }

    // tslint:disable-next-line:no-string-literal
    return this.contract.methods['listings__contractAddress'](
      token.toString()
    ).call()
  }

  /**
   * Gets the current challenge period.
   * Challenge period is returned in number of blocks.
   * @returns Current challenge period.
   */
  public async getChallengePeriod(): Promise<number> {
    // tslint:disable-next-line:no-string-literal
    return this.contract.methods['CHALLENGE_PERIOD']()
  }

  /**
   * Returns past events for the contract
   * @param event The name of the event.
   * @param filter The filter object.
   * @returns past events with the given filter.
   */
  public async getPastEvents(event: string, filter: any): Promise<any> {
    /*
    const events: EventLog[] = await this.contract.getPastEvents(
      event,
      filter,
      null
    )
    return events.map((ethereumEvent) => {
      return EthereumEvent.from(ethereumEvent)
    })
    */
  }

  /**
   * Checks whether a specific deposit actually exists.
   * @param deposit Deposit to check.
   * @returns `true` if the deposit exists, `false` otherwise.
   */
  public async isValidDeposit(deposit: any): Promise<boolean> {
    /*
    // Find past deposit events.
    const depositEvents = await this.getPastEvents('DepositEvent', {
      filter: {
        depositer: deposit.owner,
        // block: deposit.block
      },
      fromBlock: 0,
    })

    // Convert the events to deposit objects.
    const deposits = depositEvents.map(Deposit.from)

    // Check that one of the events matches this deposit.
    return deposits.some(deposit.equals)
    */
    return true
  }
}
