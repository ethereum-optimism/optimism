/* External Imports */
import {
  AndDecider,
  BigNumber,
  Decider,
  Decision,
  getLogger,
  ImplicationProofItem,
  MerkleInclusionProofDecider,
  SignedByDBInterface,
  hexStrToBuf,
} from '@pigi/core'

/* Internal Imports */
import {
  abiEncodeState,
  abiEncodeStateReceipt,
  Address,
  RollupStateSolver,
  SignedStateReceipt,
  StateReceipt,
} from './index'

const log = getLogger('rollup-ovm')

/**
 * The State Solver. Stores state, evaluates state validity, and sends fraud proofs to L1 as necessary.
 */
export class DefaultRollupStateSolver implements RollupStateSolver {
  constructor(
    private readonly signedByDB: SignedByDBInterface,
    private readonly signedByDecider: Decider,
    private readonly merkleInclusionDecider: Decider = new MerkleInclusionProofDecider()
  ) {}

  /**
   * Stores the SignedStateReceipt
   * @param signedReceipt The signed receipt
   */
  public async storeSignedStateReceipt(
    signedReceipt: SignedStateReceipt
  ): Promise<void> {
    await this.signedByDB.storeSignedMessage(
      abiEncodeStateReceipt(signedReceipt.stateReceipt),
      signedReceipt.signature
    )
  }

  /**
   * Determines whether or not the provided StateReceipt is valid, checking that
   * there is a signature for it, and it has a valid inclusion proof.
   * @param stateReceipt
   * @param signer
   */
  public async isStateReceiptProvablyValid(
    stateReceipt: StateReceipt,
    signer: Address
  ): Promise<boolean> {
    return (await this.decideIfStateReceiptIsValid(stateReceipt, signer))
      .outcome
  }

  /**
   * Gets the proof that the provided state receipt is valid.
   * @param stateReceipt The State Receipt in question
   * @param signer The Signer of the StateReceipt
   * @returns The implication proof items of state receipt being valid, else undefined
   */
  public async getFraudProof(
    stateReceipt: StateReceipt,
    signer: Address
  ): Promise<ImplicationProofItem[]> {
    const decision = await this.decideIfStateReceiptIsValid(
      stateReceipt,
      signer
    )

    return decision.outcome ? decision.justification : undefined
  }

  private async decideIfStateReceiptIsValid(
    stateReceipt: StateReceipt,
    aggregatorAddress: Address
  ): Promise<Decision> {
    const input = {
      properties: [
        {
          decider: this.signedByDecider,
          input: {
            publicKey: aggregatorAddress,
            serializedMessage: abiEncodeStateReceipt(stateReceipt),
          },
        },
        {
          decider: this.merkleInclusionDecider,
          input: {
            merkleProof: {
              rootHash: hexStrToBuf(stateReceipt.stateRoot),
              key: new BigNumber(stateReceipt.slotIndex),
              value: Buffer.from(abiEncodeState(stateReceipt.state)),
              siblings: stateReceipt.inclusionProof.map((x) =>
                Buffer.from(x, 'hex')
              ),
            },
          },
        },
      ],
    }

    log.debug(
      `Deciding if state is valid ${JSON.stringify(
        stateReceipt
      )}, input: ${JSON.stringify(input)}`
    )

    const decision: Decision = await AndDecider.instance().decide(input)

    log.debug(
      `State is ${decision.outcome ? 'valid' : 'invalid'}: ${JSON.stringify(
        stateReceipt
      )}`
    )

    return decision
  }
}
