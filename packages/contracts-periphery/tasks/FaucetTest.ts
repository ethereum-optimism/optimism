import hre from 'hardhat'
import '@nomiclabs/hardhat-ethers'

import type { DeployConfig } from '../src'

const main = async () => {
  const deployConfig = hre.deployConfig as DeployConfig

  const [owner] = await hre.ethers.getSigners()

  console.log('owner', owner.address)
  const optimistFamAddress = (
    await hre.deployments.get('OptimistAdminFaucetAuthModule')
  ).address
  const OptimistFam = await hre.ethers.getContractAt(
    'AdminFaucetAuthModule',
    optimistFamAddress,
    owner
  )
  const faucetDeployment = await hre.deployments.get('Faucet')
  const Faucet = await hre.ethers.getContractAt(
    'Faucet',
    faucetDeployment.address,
    owner
  )
  // Send eth to faucet
  const transactionHash = await owner.sendTransaction({
    to: faucetDeployment.address,
    value: hre.ethers.utils.parseEther('1.0'),
  })
  await transactionHash.wait()
  const optimistModuleConfig = {
    name: 'OPTIMIST_ADMIN_AUTH',
    enabled: true,
    ttl: 3600,
    amount: hre.ethers.utils.parseEther('0.02'),
  }
  const optimistConfigureTx = await Faucet.configure(
    optimistFamAddress,
    optimistModuleConfig
  )
  await optimistConfigureTx.wait()

  const encodedNonce = hre.ethers.utils.keccak256(
    hre.ethers.utils.defaultAbiCoder.encode(
      ['uint256'],
      [await hre.ethers.provider.getTransactionCount(owner.address)]
    )
  )
  const proof = {
    recipient: owner.address,
    nonce: encodedNonce,
    id: hre.ethers.utils.keccak256(
      hre.ethers.utils.defaultAbiCoder.encode(['address'], [owner.address])
    ),
  }
  const domain = {
    name: deployConfig.optimistFamName,
    version: deployConfig.optimistFamVersion,
    chainId: hre.network.config.chainId ?? 31337,
    verifyingContract: OptimistFam.address,
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
    nonce: encodedNonce,
  }
  const authParams = {
    module: optimistFamAddress,
    id: proof.id,
    proof: signature,
  }
  const dripTx = await Faucet.drip(dripParams, authParams)
  await dripTx.wait()
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error)
    process.exit(1)
  })
