/* External Imports */
import { getContractDefinition } from '@eth-optimism/contracts'

/**
 * Generates an ethers contract from a definition pulled from the optimism
 * contracts package.
 * @param ethers Ethers instance.
 * @param name Name of the contract to generate
 * @param args Constructor arguments to the contract.
 * @returns Ethers contract object.
 */
const getContractFromDefinition = (
  ethers: any,
  signer: any,
  name: string,
  args: any[] = []
): any => {
  const contractDefinition = getContractDefinition(name)
  const contractFactory = new ethers.ContractFactory(
    contractDefinition.abi,
    contractDefinition.bytecode,
    signer
  )

  return contractFactory.deploy(...args)
}

/**
 * Initializes the cross domain messengers.
 * @param ethers Ethers instance to use.
 * @param provider Provider to attach messengers to.
 * @returns Both cross domain messenger objects.
 */
export const initCrossDomainMessengers = async (
  l1ToL2MessageDelay: number,
  l2ToL1MessageDelay: number,
  ethers: any,
  signer: any
): Promise<{
  l1CrossDomainMessenger: any
  l2CrossDomainMessenger: any
}> => {
  const l1CrossDomainMessenger = await getContractFromDefinition(
    ethers,
    signer,
    'mockOVM_CrossDomainMessenger',
    [l2ToL1MessageDelay]
  )

  const l2CrossDomainMessenger = await getContractFromDefinition(
    ethers,
    signer,
    'mockOVM_CrossDomainMessenger',
    [l1ToL2MessageDelay]
  )

  await l1CrossDomainMessenger.setTargetMessengerAddress(
    l2CrossDomainMessenger.address
  )
  await l2CrossDomainMessenger.setTargetMessengerAddress(
    l1CrossDomainMessenger.address
  )

  signer.provider.__l1CrossDomainMessenger = l1CrossDomainMessenger
  signer.provider.__l2CrossDomainMessenger = l2CrossDomainMessenger

  return {
    l1CrossDomainMessenger,
    l2CrossDomainMessenger,
  }
}

/**
 * Relays all L2 to L1 messages to their respective L1 targets.
 * @param provider Ethers provider with attached messengers.
 */
export const relayL2ToL1Messages = async (signer: any): Promise<void> => {
  return relayXDomainMessages(true, signer)
}

/**
 * Relays all L1 to L2 messages to their respective L2 targets.
 * @param provider Ethers provider with attached messengers.
 */
export const relayL1ToL2Messages = async (signer: any): Promise<void> => {
  return relayXDomainMessages(false, signer)
}

const relayXDomainMessages = async (
  isL1: boolean,
  signer: any
): Promise<void> => {
  const messenger = isL1
    ? signer.provider.__l1CrossDomainMessenger
    : signer.provider.__l2CrossDomainMessenger
  if (!messenger) {
    throw new Error(
      'Messengers are not initialized. Please make sure to call initCrossDomainMessengers!'
    )
  }

  do {
    await messenger.relayNextMessage()
  } while (await messenger.hasNextMessage())
}
