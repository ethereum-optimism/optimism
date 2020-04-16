import { hexlify, RLP } from "ethers/utils"
export const AGGREGATOR_ADDRESS = '0xAc001762c6424F4959852A516368DBf970C835a7'
export const ALICE_ADDRESS = '0xaaaf2795C3013711c240244aFF600aD9e8D9727D'
export const BOB_ADDRESS = '0xbbbCAAe85dfE709a25545E610Dba4082f6D02D73'
/**
 * RLP encodes a transaction
 * @param {ethers.Trasaction} transaction
 */
export const rlpEncodeTransaction = async (
  transaction: object
): Promise<string> => {
  return RLP.encode([
    hexlify(transaction["nonce"]),
    hexlify(transaction["gasPrice"]),
    hexlify(transaction["gasLimit"]),
    hexlify(transaction["to"]),
    hexlify(transaction["value"]),
    transaction["data"],
  ])
}

export const AGGREGATOR_MNEMONIC: string =
  'rebel talent argue catalog maple duty file taxi dust hire funny steak'
