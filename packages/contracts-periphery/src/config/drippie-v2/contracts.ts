import * as weiroll from '@weiroll/weiroll.js'
import { ethers } from 'ethers'

import { abi as EthereumABI } from '../../../artifacts/contracts/universal/drippie-v2/helpers/Ethereum.sol/Ethereum.json'
import { abi as MathABI } from '../../../artifacts/contracts/universal/drippie-v2/helpers/Math.sol/Math.json'
import { abi as ComparisonABI } from '../../../artifacts/contracts/universal/drippie-v2/helpers/Comparison.sol/Comparison.json'
import { abi as GelatoABI } from '../../../artifacts/contracts/universal/drippie-v2/helpers/Gelato.sol/Gelato.json'
import { abi as StateABI } from '../../../artifacts/contracts/universal/drippie-v2/helpers/State.sol/State.json'
import { abi as CoersionABI } from '../../../artifacts/contracts/universal/drippie-v2/helpers/Coersion.sol/Coersion.json'
import { abi as AssertABI } from '../../../artifacts/contracts/universal/drippie-v2/helpers/Assert.sol/Assert.json'

export const asWeirollContract = (
  address: string,
  abi: any
): weiroll.Contract => {
  return weiroll.Contract.createContract(
    new ethers.Contract(address, abi) as any
  )
}

export const contracts: {
  [key: string]: weiroll.Contract
} = {
  Ethereum: asWeirollContract(
    '0xcC2AEb294F86ca2F1cdF9865c5a040c1E0c6754a',
    EthereumABI
  ),
  Math: asWeirollContract(
    '0x3c8B2cc7C3BE05B4B19f98d5097e96f1704397aB',
    MathABI
  ),
  Comparison: asWeirollContract(
    '0x471bB093F4bCe8c69A6084df9f2bDC1B3F154101',
    ComparisonABI
  ),
  Gelato: asWeirollContract(
    '0x55376fd280748BaDAD6849191F680B8d53b363B4',
    GelatoABI
  ),
  State: asWeirollContract(
    '0x2b74a9Fa15c5C41112c966aB4d9A3Ce2Ff43D988',
    StateABI
  ),
  Coersion: asWeirollContract(
    '0x12a204feFf785Cd6a49b9A22770DaBE9DCD8a98C',
    CoersionABI
  ),
  Assert: asWeirollContract(
    '0x16355a16687d638d4B3136E4f490528442b0F0D3',
    AssertABI
  ),
}
