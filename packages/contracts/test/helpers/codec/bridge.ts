import { getContractInterface } from '../../../src/contract-defs'

export const getXDomainCalldata = (
  sender: string,
  target: string,
  message: string,
  messageNonce: number
): string => {
  return getContractInterface(
    'OVM_L2CrossDomainMessenger'
  ).encodeFunctionData('relayMessage', [target, sender, message, messageNonce])
}
