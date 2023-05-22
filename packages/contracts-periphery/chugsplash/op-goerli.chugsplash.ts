import { ethers } from 'ethers'

import { Time } from '../src'

const config = {
  options: {
    projectName: 'op-goerli-periphery-1',
    organizationID:
      '0x0113d7fbe90f8258a78acf27939f999969bc54e39d74cf56fcca26f7ead85fe0',
  },
  contracts: {
    Faucet: {
      contract: 'Faucet',
      kind: 'proxy',
      constructorArgs: {
        _admin: '0x450beB92D9a472A76f5Ff6aEC5aF793A084Eac9E',
      },
      variables: {
        modules: {
          '{{ GitHubFAM }}': {
            ttl: Time.DAY,
            amount: ethers.utils.parseEther('0.05'),
            name: 'GITHUB_ADMIN_FAM',
            enabled: true,
          },
          '{{ OptimistFAM }}': {
            ttl: Time.DAY,
            amount: ethers.utils.parseEther('1.00'),
            name: 'OPTIMIST_ADMIN_FAM',
            enabled: true,
          },
        },
        timeouts: {},
        nonces: {},
      },
    },
    GitHubFAM: {
      contract: 'AdminFaucetAuthModule',
      kind: 'immutable',
      salt: 'GitHubFAM@1',
      unsafeAllowFlexibleConstructor: true,
      constructorArgs: {
        _admin: '0xD6015C64561D2296E3Af872fce263B59F4A4d9De',
        _name: 'GithubFAM',
        _version: '1',
      },
    },
    OptimistFAM: {
      contract: 'AdminFaucetAuthModule',
      kind: 'immutable',
      salt: 'OptimistFAM@1',
      unsafeAllowFlexibleConstructor: true,
      constructorArgs: {
        _admin: '0xD6015C64561D2296E3Af872fce263B59F4A4d9De',
        _name: 'OptimistFAM',
        _version: '1',
      },
    },
  },
}

export default config
