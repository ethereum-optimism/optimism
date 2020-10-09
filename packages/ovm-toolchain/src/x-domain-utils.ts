/* External Imports */
import { getContractDefinition } from '@eth-optimism/rollup-contracts'

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
    'MockL1CrossDomainMessenger',
    [l2ToL1MessageDelay]
  )

  const l2CrossDomainMessenger = await getContractFromDefinition(
    ethers,
    signer,
    'MockL2CrossDomainMessenger',
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
 * Relays all messages to their respective targets.
 * @param provider Ethers provider with attached messengers.
 */
export const waitForCrossDomainMessages = async (
  signer: any
): Promise<void> => {
  const l1CrossDomainMessenger = signer.provider.__l1CrossDomainMessenger
  const l2CrossDomainMessenger = signer.provider.__l2CrossDomainMessenger

  if (!l1CrossDomainMessenger || !l2CrossDomainMessenger) {
    throw new Error(
      'Messengers are not initialized. Please make sure to call initCrossDomainMessengers!'
    )
  }

  while (await l1CrossDomainMessenger.hasNextMessage()) {
    await l1CrossDomainMessenger.relayNextMessage()
  }

  while (await l2CrossDomainMessenger.hasNextMessage()) {
    await l2CrossDomainMessenger.relayNextMessage()
  }
}
