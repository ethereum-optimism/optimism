import fs from 'fs'

import { ethers } from 'ethers'
import { task, types } from 'hardhat/config'
import { Genesis, State } from '@eth-optimism/core-utils'
import '@eth-optimism/hardhat-deploy-config'

task('genesis-l1', 'create a genesis config')
  .addOptionalParam(
    'outfile',
    'The file to write the output JSON to',
    'genesis.json'
  )
  .addOptionalParam(
    'l1GenesisBlockTimestamp',
    'Timestamp to embed in L1 genesis block, current time will be used if the timestamp is zero',
    0,
    types.int
  )
  .setAction(async (args, hre) => {
    const { deployConfig } = hre
    const alloc: State = {}

    const l1GenesisBlockTimestamp =
      args.l1GenesisBlockTimestamp === 0
        ? Math.floor(Date.now() / 1000)
        : args.l1GenesisBlockTimestamp

    // Give each predeploy a single wei
    for (let i = 0; i <= 0xff; i++) {
      const buf = Buffer.alloc(2)
      buf.writeUInt16BE(i, 0)
      const addr = ethers.utils.hexConcat([
        '0x000000000000000000000000000000000000',
        ethers.utils.hexZeroPad(buf, 2),
      ])
      alloc[addr] = {
        balance: '0x1',
      }
    }

    if (deployConfig.fundDevAccounts) {
      const accounts = [
        '0x14dC79964da2C08b23698B3D3cc7Ca32193d9955',
        '0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65',
        '0x1CBd3b2770909D4e10f157cABC84C7264073C9Ec',
        '0x23618e81E3f5cdF7f54C3d65f7FBc0aBf5B21E8f',
        '0x2546BcD3c84621e976D8185a91A922aE77ECEc30',
        '0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC',
        '0x70997970C51812dc3A010C7d01b50e0d17dc79C8',
        '0x71bE63f3384f5fb98995898A86B02Fb2426c5788',
        '0x8626f6940E2eb28930eFb4CeF49B2d1F2C9C1199',
        '0x90F79bf6EB2c4f870365E785982E1f101E93b906',
        '0x976EA74026E726554dB657fA54763abd0C3a0aa9',
        '0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc',
        '0xBcd4042DE499D14e55001CcbB24a551F3b954096',
        '0xFABB0ac9d68B0B445fB7357272Ff202C5651694a',
        '0xa0Ee7A142d267C1f36714E4a8F75612F20a79720',
        '0xbDA5747bFD65F08deb54cb465eB87D40e51B197E',
        '0xcd3B766CCDd6AE721141F452C550Ca635964ce71',
        '0xdD2FD4581271e230360230F9337D5c0430Bf44C0',
        '0xdF3e18d64BC6A983f673Ab319CCaE4f1a57C7097',
        '0xde3829a23df1479438622a08a116e8eb3f620bb5',
        '0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266',
      ]

      for (const account of accounts) {
        alloc[ethers.utils.getAddress(account)] = {
          balance:
            '0x200000000000000000000000000000000000000000000000000000000000000',
        }
      }
    }

    const genesis: Genesis = {
      config: {
        chainId: deployConfig.l1ChainID,
        homesteadBlock: 0,
        eip150Block: 0,
        eip150Hash: ethers.constants.HashZero,
        eip155Block: 0,
        eip158Block: 0,
        byzantiumBlock: 0,
        constantinopleBlock: 0,
        petersburgBlock: 0,
        istanbulBlock: 0,
        muirGlacierBlock: 0,
        berlinBlock: 0,
        londonBlock: 0,
        clique: {
          period: deployConfig.l1BlockTime,
          epoch: 30000,
        },
      },
      nonce: deployConfig.l1GenesisBlockNonce,
      timestamp: ethers.BigNumber.from(l1GenesisBlockTimestamp).toHexString(),
      extraData: ethers.utils.hexConcat([
        ethers.constants.HashZero,
        deployConfig.cliqueSignerAddress,
        ethers.utils.hexZeroPad('0x', 65),
      ]),
      gasLimit: deployConfig.l1GenesisBlockGasLimit,
      difficulty: deployConfig.l1GenesisBlockDifficulty,
      mixHash: deployConfig.l1GenesisBlockMixHash,
      coinbase: deployConfig.l1GenesisBlockCoinbase,
      alloc,
      number: deployConfig.l1GenesisBlockNumber,
      gasUsed: deployConfig.l1GenesisBlockGasUsed,
      parentHash: deployConfig.l1GenesisBlockParentHash,
      baseFeePerGas: deployConfig.l1GenesisBlockBaseFeePerGas,
    }

    fs.writeFileSync(args.outfile, JSON.stringify(genesis, null, 2))
  })
