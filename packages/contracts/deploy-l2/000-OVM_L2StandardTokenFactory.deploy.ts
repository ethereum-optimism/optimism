import { ethers } from 'hardhat'

// eslint-disable-next-line prefer-arrow/prefer-arrow-functions
async function main() {
  const l2TokenFactory = await ethers.getContractFactory(
    'OVM_L2StandardTokenFactory'
  )
  const l2StandardTokenFactory = await l2TokenFactory.deploy()

  console.log(
    'L2 Standard Token Factory deployed to:',
    l2StandardTokenFactory.address
  )
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error)
    process.exit(1)
  })
