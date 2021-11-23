/* eslint-disable @typescript-eslint/no-var-requires */
import { ethers } from 'ethers'

export const ChugSplashProxyABI = require('../artifacts/contracts/ChugSplashProxy.sol/ChugSplashProxy.json')
export const IChugSplashDeployerABI = require('../artifacts/contracts/IChugSplashDeployer.sol/IChugSplashDeployer.json')

export const ChugSplashProxy = new ethers.utils.Interface(ChugSplashProxyABI)
export const ChugSplashDeployer = new ethers.utils.Interface(
  IChugSplashDeployerABI
)
