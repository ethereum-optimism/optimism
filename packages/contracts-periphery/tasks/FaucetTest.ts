import hre from 'hardhat'
import '@nomiclabs/hardhat-ethers'

import type { DeployConfig } from '../src'

const main = async () => {
  const deployConfig = hre.deployConfig as DeployConfig

  const [owner, nonAdmin] = await hre.ethers.getSigners()

  console.log('owner', owner.address)
  const optimistFamAddress = (
    await hre.deployments.get('OptimistAdminFaucetAuthModule')
  ).address
  const faucetDeployment = await hre.deployments.get('Faucet')
  const Faucet = await hre.ethers.getContractAt(
    'Faucet',
    faucetDeployment.address,
    // owner
    nonAdmin
  )

  // Send eth to faucet
  const transactionHash = await owner.sendTransaction({
    to: faucetDeployment.address,
    value: hre.ethers.utils.parseEther('10.0'),
  })
  await transactionHash.wait()

  const encodedNonce = hre.ethers.utils.keccak256(
    hre.ethers.utils.defaultAbiCoder.encode(
      ['uint256'],
      [await hre.ethers.provider.getTransactionCount(owner.address)]
    )
  )
  const proof = {
    recipient: owner.address,
    // recipient: nonAdmin.address,
    nonce: encodedNonce,
    id: hre.ethers.utils.keccak256(owner.address),
    // id: hre.ethers.utils.keccak256(nonAdmin.address),
  }
  const domain = {
    name: deployConfig.optimistFamName,
    version: deployConfig.optimistFamVersion,
    chainId: hre.network.config.chainId ?? 31337,
    verifyingContract: optimistFamAddress,
  }
  const types = {
    Proof: [
      { name: 'recipient', type: 'address' },
      { name: 'nonce', type: 'bytes32' },
      { name: 'id', type: 'bytes32' },
    ],
  }
  const signature = await owner._signTypedData(domain, types, proof)
  const dripParams = {
    recipient: owner.address,
    // recipient: nonAdmin.address,
    nonce: encodedNonce,
  }
  const authParams = {
    module: optimistFamAddress,
    id: proof.id,
    proof: signature,
  }
  // const ownerBalanceBefore = await owner.getBalance()
  const ownerBalanceBefore = hre.ethers.utils.formatEther(
    await hre.ethers.provider.getBalance(owner.address)
  )
  // const nonAdminBalanceBefore = await owner.getBalance()
  const nonAdminBalanceBefore = hre.ethers.utils.formatEther(
    await hre.ethers.provider.getBalance(nonAdmin.address)
  )
  console.log(`owner balance before ${ownerBalanceBefore}`)
  console.log(`non admin balance before ${nonAdminBalanceBefore}`)
  const dripTx = await Faucet.drip(dripParams, authParams)
  await dripTx.wait()

  // const ownerBalanceBefore = await owner.getBalance()
  const ownerBalanceAfter = hre.ethers.utils.formatEther(
    await hre.ethers.provider.getBalance(owner.address)
  )
  // const nonAdminBalanceBefore = await owner.getBalance()
  const nonAdminBalanceAfter = hre.ethers.utils.formatEther(
    await hre.ethers.provider.getBalance(nonAdmin.address)
  )
  console.log(`owner balance after: ${ownerBalanceAfter}`)
  console.log(`non admin balance after: ${nonAdminBalanceAfter}`)
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error)
    process.exit(1)
  })
