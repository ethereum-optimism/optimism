/* Imports: External */
import { ContractFactory } from 'ethers'
import { ethers } from 'hardhat'
import { predeploys } from '@eth-optimism/contracts'

/* Imports: Internal */
import { expect } from './shared/setup'
import { OptimismEnv } from './shared/env'

describe('Whitelist', async () => {
  const initialAmount = 1000
  const tokenName = 'OVM Test'
  const tokenDecimals = 8
  const TokenSymbol = 'OVM'

  let Factory__ERC20: ContractFactory
  let env: OptimismEnv
  before(async () => {
    env = await OptimismEnv.new()
    Factory__ERC20 = await ethers.getContractFactory('ERC20', env.l2Wallet)
  })

  describe('when the whitelist is disabled', () => {
    it('should be able to deploy a contract', async () => {
      await expect(
        env.l2Provider.send('eth_call', [
          Factory__ERC20.getDeployTransaction(
            initialAmount,
            tokenName,
            tokenDecimals,
            TokenSymbol
          ),
          'latest',
          {
            [predeploys.OVM_DeployerWhitelist]: {
              state: {
                ['0x0000000000000000000000000000000000000000000000000000000000000000']:
                  '0x0000000000000000000000000000000000000000000000000000000000000000',
              },
            },
          },
        ])
      ).to.not.be.reverted
    })
  })

  describe('when the whitelist is enabled', () => {
    const sender = '0x' + '22'.repeat(20)

    it('should fail if the user is not whitelisted', async () => {
      await expect(
        env.l2Provider.send('eth_call', [
          {
            ...Factory__ERC20.getDeployTransaction(
              initialAmount,
              tokenName,
              tokenDecimals,
              TokenSymbol
            ),
            from: sender,
          },
          'latest',
          {
            [predeploys.OVM_DeployerWhitelist]: {
              state: {
                // Set an owner but don't allow this user to deploy
                // Owner here is address(1) instead of address(0)
                ['0x0000000000000000000000000000000000000000000000000000000000000000']:
                  '0x0000000000000000000000000000000000000000000000000000000000000001',
              },
            },
          },
        ])
      ).to.be.revertedWith(`deployer address not whitelisted: ${sender}`)
    })

    it('should succeed if the user is whitelisted', async () => {
      await expect(
        env.l2Provider.send('eth_call', [
          {
            ...Factory__ERC20.getDeployTransaction(
              initialAmount,
              tokenName,
              tokenDecimals,
              TokenSymbol
            ),
            from: sender,
          },
          'latest',
          {
            [predeploys.OVM_DeployerWhitelist]: {
              state: {
                // Set an owner
                ['0x0000000000000000000000000000000000000000000000000000000000000000']:
                  '0x0000000000000000000000000000000000000000000000000000000000000001',

                // See https://docs.soliditylang.org/en/v0.8.9/internals/layout_in_storage.html for
                // reference on how the correct storage slot should be set.
                // Whitelist mapping is located at storage slot 1.
                // whitelist[address] will be located at:
                // keccak256(uint256(address) . uint256(1)))
                [ethers.utils.keccak256(
                  '0x' +
                    // uint256(address)
                    '0000000000000000000000002222222222222222222222222222222222222222' +
                    // uint256(1)
                    '0000000000000000000000000000000000000000000000000000000000000001'
                )]:
                  '0x0000000000000000000000000000000000000000000000000000000000000001', // Boolean (1)
              },
            },
          },
        ])
      ).to.not.be.reverted
    })
  })
})
