import { getContractInterface } from '../../../src/contract-defs'

export const encodeXDomainCalldata = (
  target: string,
  sender: string,
  message: string,
  messageNonce: number
): string => {
  return getContractInterface(
    'OVM_L2CrossDomainMessenger'
  ).encodeFunctionData('relayMessage', [target, sender, message, messageNonce])
}
