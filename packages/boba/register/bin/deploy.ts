import { Wallet, providers } from 'ethers'
import { getContractFactory } from '@bobanetwork/core_contracts'
import { DeployFunction, DeploymentSubmission } from 'hardhat-deploy/dist/types'
import { Contract, ContractFactory } from 'ethers'
import { Provider } from '@ethersproject/abstract-provider'
import { Signer } from '@ethersproject/abstract-signer'
import { sleep, hexStringEquals } from '@eth-optimism/core-utils'

/* eslint-disable */
require('dotenv').config()

import hre from 'hardhat'
import toRegister from '../addresses/addressesGoerli_0x6FF9c8FF8F0B6a0763a3030540c21aFC721A9148.json'

const waitUntilTrue = async (
  check: () => Promise<boolean>,
  opts: {
    retries?: number
    delay?: number
  } = {}
) => {
  opts.retries = opts.retries || 100
  opts.delay = opts.delay || 5000

  let retries = 0
  while (!(await check())) {
    if (retries > opts.retries) {
      throw new Error(`check failed after ${opts.retries} attempts`)
    }
    retries++
    await sleep(opts.delay)
  }
}

const registerBobaAddress = async (
  addressManager: any,
  name: string,
  address: string
): Promise<void> => {

  const currentAddress = await addressManager.getAddress(name)

  console.log(`\nCurrent Address of ${name} is ${currentAddress}`)

  if (address.toLowerCase() === currentAddress.toLowerCase()) {
    console.log(
      `✓ Not registering address for ${name} because it's already been correctly registered`
    )
    return
  }

  console.log(`Registering address for ${name} to ${address}...`)
  await addressManager.setAddress(name, address)

  console.log(`Waiting for registration to reflect on-chain...`)
  await waitUntilTrue(async () => {
    return hexStringEquals(await addressManager.getAddress(name), address)
  })

  console.log(`✓ Registered address for ${name}`)
}

const main = async () => {

  console.log('Starting BOBA manual registration...')

  const l1Provider = new providers.JsonRpcProvider(process.env.L1_NODE_WEB3_URL)

  const deployer_l1 = new Wallet(process.env.DEPLOYER_PRIVATE_KEY, l1Provider)

  const getAddressManager = (provider: any, addressManagerAddress: any) => {
    return getContractFactory('Lib_AddressManager')
      .connect(provider)
      .attach(addressManagerAddress) as any
  }

  console.log(
    `ADDRESS_MANAGER_ADDRESS was set to ${process.env.ADDRESS_MANAGER_ADDRESS}`
  )

  const addressManager = getAddressManager(
    deployer_l1,
    process.env.ADDRESS_MANAGER_ADDRESS
  )

  const entries = Object.keys(toRegister)

  for (const entry of entries) {
    await registerBobaAddress(
      addressManager,
      entry,
      toRegister[entry]
    )
  }

}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.log(
      JSON.stringify({ error: error.message, stack: error.stack }, null, 2)
    )
    process.exit(1)
  })
