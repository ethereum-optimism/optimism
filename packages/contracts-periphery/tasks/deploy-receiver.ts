import { task } from 'hardhat/config'
import * as types from 'hardhat/internal/core/params/argumentTypes'
import { LedgerSigner } from '@ethersproject/hardware-wallets'

task('deploy-receiver')
  .addParam('creator', 'Creator address', undefined, types.string)
  .addParam('owner', 'Owner address', undefined, types.string)
  .setAction(async (args, hre) => {
    console.log(`connecting to ledger...`)
    const signer = new LedgerSigner(
      hre.ethers.provider,
      'default',
      hre.ethers.utils.defaultPath
    )

    const addr = await signer.getAddress()
    if (args.creator !== addr) {
      throw new Error(`Incorrect key. Creator ${args.creator}, Signer ${addr}`)
    }

    const singleton = new hre.ethers.Contract(
      '0xce0042B868300000d44A59004Da54A005ffdcf9f',
      [
        {
          constant: false,
          inputs: [
            {
              internalType: 'bytes',
              name: '_initCode',
              type: 'bytes',
            },
            {
              internalType: 'bytes32',
              name: '_salt',
              type: 'bytes32',
            },
          ],
          name: 'deploy',
          outputs: [
            {
              internalType: 'address payable',
              name: 'createdContract',
              type: 'address',
            },
          ],
          payable: false,
          stateMutability: 'nonpayable',
          type: 'function',
        },
      ],
      signer
    )

    const salt =
      '0x0000000000000000000000000000000000000000000000000000000000000001'
    const code = hre.ethers.utils.hexConcat([
      hre.artifacts.readArtifactSync('RetroReceiver').bytecode,
      hre.ethers.utils.defaultAbiCoder.encode(['address'], [addr]),
    ])

    // Predict and connect to the contract address
    const receiver = await hre.ethers.getContractAt(
      'RetroReceiver',
      await singleton.callStatic.deploy(code, salt, {
        gasLimit: 2_000_000,
      }),
      signer
    )

    console.log(`creating contract: ${receiver.address}...`)
    const tx1 = await singleton.deploy(code, salt, {
      gasLimit: 2_000_000,
    })

    console.log(`waiting for tx: ${tx1.hash}...`)
    await tx1.wait()

    console.log(`transferring ownership to: ${args.owner}...`)
    const tx2 = await receiver.setOwner(args.owner)

    console.log(`waiting for tx: ${tx2.hash}...`)
    await tx2.wait()

    console.log(`verifying contract: ${receiver.address}...`)
    await hre.run('verify:verify', {
      address: receiver.address,
      constructorArguments: [addr],
    })

    console.log(`all done`)
  })
