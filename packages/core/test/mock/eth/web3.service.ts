import { Web3Service } from '../../../src/services/eth/web3.service'
import { config } from '../config.service'

export const web3Service = new Web3Service(config)
