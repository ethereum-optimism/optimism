/* eslint-disable quotes */
/*
Copyright 2019-present OmiseGO Pte Ltd

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License. */

import { parseUnits, parseEther, formatEther } from '@ethersproject/units'

import { Watcher } from '@eth-optimism/watcher'

import { ethers, BigNumber, utils, ContractFactory } from 'ethers'
import store from 'store'

import { orderBy } from 'lodash'
import BN from 'bn.js'

import { getToken } from 'actions/tokenAction'

import {
  addNFT,
  getNFTs,
  addNFTContract,
  getNFTContracts,
} from 'actions/nftAction'

import {
  updateSignatureStatus_exitLP,
  updateSignatureStatus_exitTRAD,
  updateSignatureStatus_depositLP,
  updateSignatureStatus_depositTRAD
} from 'actions/signAction'

import { WebWalletError } from 'services/errorService'

//Base contracts
import L1StandardBridgeJson from '../deployment/artifacts/optimistic-ethereum/OVM/bridge/tokens/OVM_L1StandardBridge.sol/OVM_L1StandardBridge.json'
import L2StandardBridgeJson from '../deployment/artifacts-ovm/optimistic-ethereum/OVM/bridge/tokens/OVM_L2StandardBridge.sol/OVM_L2StandardBridge.json'

//OMGX LP contracts
import L1LPJson from '../deployment/artifacts/contracts/LP/L1LiquidityPool.sol/L1LiquidityPool.json'
import L2LPJson from '../deployment/artifacts-ovm/contracts/LP/L2LiquidityPool.sol/L2LiquidityPool.json'

//Standard ERC20 jsons - should be very similar?
import L1ERC20Json from '../deployment/artifacts/contracts/L1ERC20.sol/L1ERC20.json'
import L2ERC20Json from '../deployment/artifacts-ovm/optimistic-ethereum/libraries/standards/L2StandardERC20.sol/L2StandardERC20.json'

//OMGX L2 Contracts
import L2ERC721Json from '../deployment/artifacts-ovm/contracts/ERC721Genesis.sol/ERC721Genesis.json'
import L2ERC721RegJson from '../deployment/artifacts-ovm/contracts/ERC721Registry.sol/ERC721Registry.json'

import L2TokenPoolJson from '../deployment/artifacts-ovm/contracts/TokenPool.sol/TokenPool.json'
import AtomicSwapJson from '../deployment/artifacts-ovm/contracts/AtomicSwap.sol/AtomicSwap.json'

// DAO 
import Comp from "../deployment/rinkeby/json/Comp.json"
import GovernorBravoDelegate from "../deployment/rinkeby/json/GovernorBravoDelegate.json"
import GovernorBravoDelegator from "../deployment/rinkeby/json/GovernorBravoDelegator.json"
import Timelock from "../deployment/rinkeby/json/Timelock.json"

import { powAmount, logAmount } from 'util/amountConvert'
import { accDiv, accMul } from 'util/calculation'
import {getNftImageUrl} from 'util/nftImage'
import { getAllNetworks } from 'util/masterConfig'

import etherScanInstance from 'api/etherScanAxios'
import omgxWatcherAxiosInstance from 'api/omgxWatcherAxios'
import addressAxiosInstance from 'api/addressAxios'
import addressOMGXAxiosInstance from 'api/addressOMGXAxios'
import coinGeckoAxiosInstance from 'api/coinGeckoAxios'
import ethGasStationAxiosInstance from 'api/ethGasStationAxios'

//All the current addresses for fallback purposes, or live network
const localAddresses = require(`../deployment/local/addresses.json`)
const rinkebyAddresses = require(`../deployment/rinkeby/addresses.json`)
const mainnetAddresses = require(`../deployment/mainnet/addresses.json`)
const rinkebyIntegrationAddresses = require(`../deployment/rinkeby-integration/addresses.json`)

class NetworkService {

  constructor() {

    this.L1Provider = null
    this.L2Provider = null

    this.provider = null
    this.environment = null

    this.ERC721Contract = null

    this.L2TokenPoolContract = null
    this.AtomicSwapContract = null

    // L1 or L2
    this.L1orL2 = null
    this.masterSystemConfig = null

    // Watcher
    this.watcher = null
    this.fastWatcher = null

    // addresses
    this.L1StandardBridgeAddress = null
    this.L2StandardBridgeAddress = '0x4200000000000000000000000000000000000010'

    this.ERC721Address = null
    this.ERC721RegAddress = null

    this.L1_TEST_Address = null
    this.L2_TEST_Address = null

    this.L1_TEST_Contract = null
    this.L2_TEST_Contract = null

    this.L1LPAddress = null
    this.L2LPAddress = null

    this.L1_ETH_Address = '0x0000000000000000000000000000000000000000'
    this.L2_ETH_Address = '0x4200000000000000000000000000000000000006'
    this.L2_ETH_Contract = null

    this.L1MessengerAddress = null
    this.L2MessengerAddress = '0x4200000000000000000000000000000000000007'

    this.L1Message = null
    this.L2Message = null

    this.tokenAddresses = null

    // address
    this.addresses = null

    // chain ID
    this.chainID = null
    this.networkName = null

    // gas
    this.L1GasLimit = 9999999
    this.L2GasLimit = 10000000

    // Dao
    this.comp = null
    this.delegate = null
    this.delegator = null
    this.timelock = null
  }

  async enableBrowserWallet() {
    console.log('NS: enableBrowserWallet()')
    try {
      // connect to the wallet
      await window.ethereum.request({method: 'eth_requestAccounts'})
      this.provider = new ethers.providers.Web3Provider(window.ethereum)
      return true
    } catch (error) {
      return false
    }
  }

  bindProviderListeners() {
    window.ethereum.on('accountsChanged', () => {
      window.location.reload()
    })

    window.ethereum.on('chainChanged', () => {
      console.log('chainChanged')
      localStorage.setItem('changeChain', true)
      window.location.reload()
    })
  }

  async mintAndSendNFT(receiverAddress, contractAddress, tokenURI) {

    try {

      let meta = Date.now().toString() + '#' + tokenURI + '#'

      const contract = new ethers.Contract(
        contractAddress,
        L2ERC721Json.abi,
        this.L2Provider
      )

      let nft = await contract.connect(
        this.provider.getSigner()
      ).mintNFT(
        receiverAddress,
        meta
      )

      await nft.wait()

      const registry = new ethers.Contract(
        this.ERC721RegAddress,
        L2ERC721RegJson.abi,
        this.L2Provider
      )

      //what types of NFTs does this address already own? 
      const addresses = await registry.lookupAddress(
        receiverAddress
      )

      //console.log("the receiver's NFT contract addresses:", addresses)
      
      //the receiverAddress already knows about this contract
      const alreadyHaveAddresss = addresses.find((str) => str.toLowerCase() === contractAddress.toLowerCase())

      if (alreadyHaveAddresss) {
        //we are done - no need to double register addresss
        console.log('Done - no need to double register address')
      } else {
        //register address for the recipiant
        await registry.connect(
          this.provider.getSigner()
        ).registerAddress(
          receiverAddress,
          contractAddress
        )
        //console.log("Reg:",reg)
        console.log(`Contract registered in recipient's wallet`)
      }

      return true
    } catch (error) {
      console.log(error)
      return false
    }
  }

  async deployNFTContract(
      nftSymbol,
      nftName)
  {

    try {

      console.log("Deploying NFT Contract")

      let Factory__L2ERC721 = new ContractFactory(
        L2ERC721Json.abi,
        L2ERC721Json.bytecode,
        this.provider.getSigner()
      )

      let contract = await Factory__L2ERC721.deploy(
        nftSymbol,
        nftName,
        BigNumber.from(String(0)), //starting index for the tokenIDs
        '0x0000000000000000000000000000000000000042',
        'simple',
        'boba_L2'
      )

      await contract.deployTransaction.wait()
      console.log('New NFT ERC721 contract deployed to:', contract.address)

      const registry = new ethers.Contract(
        this.ERC721RegAddress,
        L2ERC721RegJson.abi,
        this.L2Provider
      )

      //register address for the contract owner
      await registry.connect(
        this.provider.getSigner()
      ).registerAddress(
        this.account,
        contract.address
      )
      console.log(`New NFT ERC721 contract registered in Boba NFT registry`)

      //addNFTContract({address: contract.address})
      //this will get picked up automatically from the blockchain

      return true
    } catch (error) {
      console.log(error)
      return false
    }

  }

  async initializeAccounts( masterSystemConfig ) {

    console.log('NS: initializeAccounts() for', masterSystemConfig)

    let resOMGX = null
    let resBase = null
    let addresses = null

    try {

      console.log('Loading OMGX contract addresses')

      if (masterSystemConfig === 'local') {

        try {
          resOMGX = await addressOMGXAxiosInstance('local').get()
        } catch (error) {
          console.log(error)
        }

        try {
          resBase = await addressAxiosInstance('local').get()
        } catch (error) {
          console.log(error)
        }

        if (resOMGX !== null && resBase !== null) {
          addresses = { ...resBase.data, ...resOMGX.data }
        } else {
          addresses = localAddresses //emergency fallback
        }

        console.log('Final Local Addresses:', addresses)
      } else if (masterSystemConfig === 'rinkeby') {
        addresses = rinkebyAddresses
        console.log('Rinkeby Addresses:', addresses)
      } else if (masterSystemConfig === 'mainnet') {
        addresses = mainnetAddresses
        console.log('Mainnet Addresses:', addresses)
      } else if (masterSystemConfig === 'rinkeby-integration') {
        addresses = rinkebyIntegrationAddresses
        console.log('Rinkeby Integration Addresses:', addresses)
      }

      //at this point, the wallet should be connected
      this.account = await this.provider.getSigner().getAddress()
      console.log('this.account', this.account)

      const network = await this.provider.getNetwork()

      this.chainID = network.chainId
      this.networkName = network.name

      this.masterSystemConfig = masterSystemConfig

      this.tokenAddresses = addresses.TOKENS

      console.log('NS: network:', network)
      console.log('NS: masterConfig:', this.masterSystemConfig)
      console.log('NS: this.chainID:', this.chainID)
      console.log('NS: this.networkName:', this.networkName)

      //there are numerous possible chains we could be on
      //either local, rinkeby etc
      //and then, also, either L1 or L2

      //at this point, we only know whether we want to be on local or rinkeby etc
      if (masterSystemConfig === 'local' && network.chainId === 31338) {
        //ok, that's reasonable
        //local deployment, L2
        this.L1orL2 = 'L2'
      } else if (masterSystemConfig === 'local' && network.chainId === 31337) {
        //ok, that's reasonable
        //local deployment, L1
        this.L1orL2 = 'L1'
      } else if (masterSystemConfig === 'rinkeby' && network.chainId === 4) {
        //ok, that's reasonable
        //rinkeby, L1
        this.L1orL2 = 'L1'
      } else if (masterSystemConfig === 'rinkeby' && network.chainId === 28) {
        //ok, that's reasonable
        //rinkeby, L2
        this.L1orL2 = 'L2'
      } else if (masterSystemConfig === 'rinkeby_integration' && network.chainId === 4) {
        //ok, that's reasonable
        //rinkeby, L1
        this.L1orL2 = 'L1'
      } else if (masterSystemConfig === 'rinkeby_integration' && network.chainId === 29) {
        //ok, that's reasonable
        //rinkeby, L2
        this.L1orL2 = 'L2'
      } else if (masterSystemConfig === 'mainnet' && network.chainId === 1) {
        //ok, that's reasonable
        //rinkeby, L2
        this.L1orL2 = 'L1'
      } else if (masterSystemConfig === 'mainnet' && network.chainId === 288) {
        //ok, that's reasonable
        //rinkeby, L2
        this.L1orL2 = 'L2'
      } else {
        console.log("ERROR: masterSystemConfig does not match actual network.chainId")
        this.bindProviderListeners()
        return 'wrongnetwork'
      }

      //dispatch(setLayer(this.L1orL2))
      //const dispatch = useDispatch();

      // defines the set of possible networks
      const nw = getAllNetworks()

      this.L1Provider = new ethers.providers.JsonRpcProvider(
        nw[masterSystemConfig]['L1']['rpcUrl']
      )
      this.L2Provider = new ethers.providers.JsonRpcProvider(
        nw[masterSystemConfig]['L2']['rpcUrl']
      )

      //this.L1MessengerAddress = addresses.Proxy__OVM_L1CrossDomainMessenger
      //backwards compat
      if (addresses.hasOwnProperty('Proxy__OVM_L1CrossDomainMessenger')) {
        this.L1MessengerAddress = addresses.Proxy__OVM_L1CrossDomainMessenger
        console.log('L1MessengerAddress set to:', this.L1MessengerAddress)
      }

      if(addresses.hasOwnProperty('Proxy__OVM_L1CrossDomainMessengerFast')) {
        this.L1FastMessengerAddress = addresses.Proxy__OVM_L1CrossDomainMessengerFast
      }
      else if (addresses.hasOwnProperty('OVM_L1CrossDomainMessengerFast')) {
        this.L1FastMessengerAddress = addresses.OVM_L1CrossDomainMessengerFast
      }
      else {
        this.L1FastMessengerAddress = addresses.L1FastMessengerAddress
      }
      console.log('L1FastMessengerAddress set to:',this.L1FastMessengerAddress)

      //backwards compat
      if (addresses.hasOwnProperty('Proxy__OVM_L1StandardBridge'))
        this.L1StandardBridgeAddress = addresses.Proxy__OVM_L1StandardBridge
      else
        this.L1StandardBridgeAddress = addresses.L1StandardBridge

      Object.keys(addresses.TOKENS).forEach((key) => {
        this["L1_" + key + "_Address"] = addresses.TOKENS[key].L1;
        this["L2_" + key + "_Address"] = addresses.TOKENS[key].L2;
      })

      this.L1LPAddress = addresses.Proxy__L1LiquidityPool
      this.L2LPAddress = addresses.Proxy__L2LiquidityPool

      this.L1Message = addresses.L1Message
      this.L2Message = addresses.L2Message

      //backwards compat
      if (addresses.hasOwnProperty('L2ERC721'))
        this.ERC721Address = addresses.L2ERC721
      else
        this.ERC721Address = addresses.ERC721

      //backwards compat
      if (addresses.hasOwnProperty('L2ERC721Reg'))
        this.ERC721RegAddress = addresses.L2ERC721Reg
      else
        this.ERC721RegAddress = addresses.ERC721Reg

      this.L2TokenPoolAddress = addresses.L2TokenPool
      this.AtomicSwapAddress = addresses.AtomicSwap

      this.addresses = addresses

      this.L1StandardBridgeContract = new ethers.Contract(
        this.L1StandardBridgeAddress,
        L1StandardBridgeJson.abi,
        this.provider.getSigner()
      )
      //console.log("L1StandardBridgeContract:", this.L1StandardBridgeContract.address)

      this.L2StandardBridgeContract = new ethers.Contract(
        this.L2StandardBridgeAddress,
        L2StandardBridgeJson.abi,
        this.provider.getSigner()
      )
      //console.log("L2StandardBridgeContract:", this.L2StandardBridgeContract.address)

      this.L2_ETH_Contract = new ethers.Contract(
        this.L2_ETH_Address,
        L2ERC20Json.abi,
        this.provider.getSigner()
      )
      //console.log("L2_ETH_Contract:", this.L2_ETH_Contract.address)

      /*The test token*/
      this.L1_TEST_Contract = new ethers.Contract(
        addresses.TOKENS.TEST.L1,
        L1ERC20Json.abi,
        this.provider.getSigner()
      )
      console.log('L1_TEST_Contract:', this.L1_TEST_Contract.address)

      this.L2_TEST_Contract = new ethers.Contract(
        addresses.TOKENS.TEST.L2,
        L2ERC20Json.abi,
        this.provider.getSigner()
      )
      console.log('L2_TEST_Contract:', this.L2_TEST_Contract.address)

      // Liquidity pools
      this.L1LPContract = new ethers.Contract(
        this.L1LPAddress,
        L1LPJson.abi,
        this.provider.getSigner()
      )

      this.L2LPContract = new ethers.Contract(
        this.L2LPAddress,
        L2LPJson.abi,
        this.provider.getSigner()
      )

      this.ERC721Contract = new ethers.Contract(
        this.ERC721Address,
        L2ERC721Json.abi,
        this.L2Provider
      )

      this.L2TokenPoolContract = new ethers.Contract(
        this.L2TokenPoolAddress,
        L2TokenPoolJson.abi,
        this.provider.getSigner()
      )

      this.AtomicSwapContract = new ethers.Contract(
        this.AtomicSwapAddress,
        AtomicSwapJson.abi,
        this.provider.getSigner()
      )

      //this one is always there...
      //await addNFTContract(this.ERC721Contract.address)

      //yes, this looks weird, but think before you change it...
      //there may be some in the cache, and this makes sure we get them all, and if not,
      //we at least have the basic one
      //const NFTcontracts = Object.values(await getNFTContracts())

      //Add factories based on cached contract addresses
      //this information is also used for the balance lookup
      //for(var i = 0; i < NFTcontracts.length; i++) {
      // const address = NFTcontracts[i]
      //  console.log("Adding NFT contract:",address)
      //  this.addNFTFactoryNS( address )
      //}

      this.watcher = new Watcher({
        l1: {
          provider: this.L1Provider,
          messengerAddress: this.L1MessengerAddress,
        },
        l2: {
          provider: this.L2Provider,
          messengerAddress: this.L2MessengerAddress,
        },
      })

      this.fastWatcher = new Watcher({
        l1: {
          provider: this.L1Provider,
          messengerAddress: this.L1FastMessengerAddress,
        },
        l2: {
          provider: this.L2Provider,
          messengerAddress: this.L2MessengerAddress,
        },
      })

      if(masterSystemConfig === 'rinkeby') {

        this.comp = new ethers.Contract(
          addresses.DAO_Comp,
          Comp.abi,
          this.provider.getSigner()
        )

        this.delegate = new ethers.Contract(
          addresses.DAO_GovernorBravoDelegate,
          GovernorBravoDelegate.abi,
          this.provider.getSigner()
        )

        this.delegator = new ethers.Contract(
          addresses.DAO_GovernorBravoDelegator,
          GovernorBravoDelegator.abi,
          this.provider.getSigner()
        )

        this.timelock = new ethers.Contract(
          addresses.DAO_Timelock,
          Timelock.abi,
          this.provider.getSigner()
        )

      }

/*
  this.comp = null
  this.delegate = null
  this.delegator = null
  this.timelock = null
*/

      this.bindProviderListeners()

      return 'enabled'
    } catch (error) {
      console.log(error)
      return false
    }
  }

  async checkStatus() {
    return {
      connection: true,
      byzantine: false,
      watcherSynced: true,
      lastSeenBlock: 0,
    }
  }

  async addL2Network() {
    
    const nw = getAllNetworks()
    const masterConfig = store.getState().setup.masterConfig;
    let chainParam = {}
    
    if (masterConfig === 'mainnet') {
      chainParam = {
        chainId: '0x' + nw.mainnet.L2.chainId.toString(16),
        chainName: nw.mainnet.L2.name,
        rpcUrls: [nw.mainnet.L2.rpcUrl],
      }
    } else if (masterConfig === 'rinkeby') {
      chainParam = {
        chainId: '0x' + nw.rinkeby.L2.chainId.toString(16),
        chainName: nw.rinkeby.L2.name,
        rpcUrls: [nw.rinkeby.L2.rpcUrl],
      }
    } else if (masterConfig === 'rinkeby_integration') {
      chainParam = {
        chainId: '0x' + nw.rinkeby_integration.L2.chainId.toString(16),
        chainName: nw.rinkeby_integration.L2.name,
        rpcUrls: [nw.rinkeby_integration.L2.rpcUrl],
      }
    } else if (masterConfig === 'local') {
      chainParam = {
        chainId: '0x' + nw.local.L2.chainId.toString(16),
        chainName: nw.local.L2.name,
        rpcUrls: [nw.local.L2.rpcUrl],
      }
    }
    
    console.log("MetaMask: Trying to add ", chainParam)
    
    // connect to the wallet
    this.provider = new ethers.providers.Web3Provider(window.ethereum)
    let res = await this.provider.send('wallet_addEthereumChain', [chainParam, this.account])

    if( res === null ){
      console.log("MetaMask - Added new RPC")
    } else {
      console.log("MetaMask - Error adding new RPC: ", res)
    }
    
  }

  async switchChain( layer ) {

    if(this.L1orL2 === layer) {
      console.log("Nothing to do - You are already on ",layer)
      return
    }

    const nw = getAllNetworks()
    const masterConfig = store.getState().setup.masterConfig
    
    const chainParam = {
      chainId: '0x' + nw[masterConfig].L2.chainId.toString(16),
      chainName: nw[masterConfig].L2.name,
      rpcUrls: [nw[masterConfig].L2.rpcUrl],
    }

    // connect to the wallet
    this.provider = new ethers.providers.Web3Provider(window.ethereum)
    
    /********************* Switch to Mainnet L2 ****************/
    if (masterConfig === 'mainnet' && this.L1orL2 === 'L1') {
      //ok, so then, we want to switch to 'mainnet' && 'L2'
      try {
        await this.provider.send('wallet_switchEthereumChain', [{ chainId: '0x120' }]) //ChainID 288
      } catch (error) {
        // This error code indicates that the chain has not been added to MetaMask.
        if (error.code === 4902) {
          try {
            await this.provider.send('wallet_addEthereumChain', [chainParam, this.account])
          } catch (addError) {
            console.log("MetaMask - Error adding new RPC: ", addError)
            // handle "add" error via alert message
          }
        } else { //some other error code
          console.log("MetaMask - Switch Error: ", error.code)
        }
      }
    } else if ( masterConfig === 'mainnet' && this.L1orL2 === 'L2') {
      //ok, so then, we want to switch to 'mainnet' && 'L1' - no need to add 
      //if(fail) since mainnet L1 is always there unless the planet 
      //has been vaporized by space aliens with a blaster ray
      try {
        await this.provider.send('wallet_switchEthereumChain',[{ chainId: '0x1' }]) //ChainID 1
      } catch (switchError) {
        console.log("MetaMask - could not switch to Ethereum Mainchain. Needless to say, this should never happen.")
      }
    } else if (masterConfig === 'rinkeby' && this.L1orL2 === 'L1') {
      //ok, so then, we want to switch to 'rinkeby' && 'L2'
      try {
        await this.provider.send('wallet_switchEthereumChain', [{ chainId: '0x1C' }]) //ChainID 28
      } catch (error) {
        if (error.code === 4902) {
          try {
            await this.provider.send('wallet_addEthereumChain', [chainParam, this.account])
          } catch (addError) {
            console.log("MetaMask - Error adding new RPC: ", addError)
            // handle "add" error via alert message
          }
        } else { //some other error code
          console.log("MetaMask - Switch Error: ", error.code)
        }
      }
    } else if (masterConfig === 'rinkeby_integration' && this.L1orL2 === 'L1') {
      //ok, so then, we want to switch to 'rinkeby_integration' && 'L2'
      try {
        await this.provider.send('wallet_switchEthereumChain', [{ chainId: '0x1D' }]) //ChainID 29
      } catch (error) {
        if (error.code === 4902) {
          try {
            await this.provider.send('wallet_addEthereumChain', [chainParam, this.account])
          } catch (addError) {
            console.log("MetaMask - Error adding new RPC: ", addError)
            // handle "add" error via alert message
          }
        } else { //some other error code
          console.log("MetaMask - Switch Error: ", error.code)
        }
      }
    } else if ((masterConfig === 'rinkeby' || masterConfig === 'rinkeby_integration') && this.L1orL2 === 'L2') {
      try {
        await this.provider.send('wallet_switchEthereumChain',[{ chainId: '0x4' }]) //ChainID 4
      } catch (switchError) {
        console.log("MetaMask - could not switch to Rinkeby. Needless to say, this should never happen.")
      }
    } else if (masterConfig === 'local' && this.L1orL2 === 'L1') {
      //ok, so then, we want to switch to 'local' && 'L2'
      try {
        await this.provider.send('wallet_switchEthereumChain', [{ chainId: '0x7A6A' }]) //ChainID 31338
      } catch (error) {
        if (error.code === 4902) {
          try {
            await this.provider.send('wallet_addEthereumChain', [chainParam, this.account])
          } catch (addError) {
            console.log("MetaMask - Error adding new RPC: ", addError)
            // handle "add" error via alert message
          }
        } else { //some other error code
          console.log("MetaMask - Switch Error: ", error.code)
        }
      }
    } else if (masterConfig === 'local' && this.L1orL2 === 'L2') {
      try {
        await this.provider.send('wallet_switchEthereumChain',[{ chainId: '0x7A69' }]) //ChainID 31337
      } catch (switchError) {
        console.log("MetaMask - could not switch to Local L1")
      }
    } 
  }

  async getTransactions() {

    // NOT SUPPORTED on LOCAL
    if (this.masterSystemConfig === 'local') return

    console.log("Getting transactions...")
    
    let txL1
    let txL1pending
    let txL2

    //console.log("trying")

    const responseL1 = await etherScanInstance(
      this.masterSystemConfig,
      'L1'
    ).get(`&address=${this.account}`)

    //console.log("responseL1",responseL1)

    if (responseL1.status === 200) {
      const transactionsL1 = await responseL1.data
      if (transactionsL1.status === '1') {
        //thread in ChainID
        txL1 = transactionsL1.result.map(v => ({
          ...v,
          blockNumber: parseInt(v.blockNumber), //fix bug - sometimes this is string, sometimes an integer
          timeStamp: parseInt(v.timeStamp),     //fix bug - sometimes this is string, sometimes an integer 
          chain: 'L1'
        }))
        //console.log("txL1",txL1)
        //return transactions.result
      }
    }

    const responseL2 = await omgxWatcherAxiosInstance(
      this.masterSystemConfig
    ).post('get.l2.transactions', {
      address: this.account,
      fromRange:  0,
      toRange: 1000,
    })

    if (responseL2.status === 201) {
      //add the chain: 'L2' field 
      txL2 = responseL2.data.map(v => ({...v, chain: 'L2'}))
      //console.log("txL2",txL2)
      //const annotated = await this.parseTransaction( [...txL1, ...txL2] )
      //return annotated
    }

    const responseL1pending = await omgxWatcherAxiosInstance(
      this.masterSystemConfig
    ).post('get.l1.transactions', {
      address: this.account,
      fromRange:  0,
      toRange: 1000,
    })

    if (responseL1pending.status === 201) {
      //add the chain: 'L1pending' field 
      txL1pending = responseL1pending.data.map(v => ({...v, chain: 'L1pending'}))
      //console.log("txL1pending",txL1pending)
      const annotated = await this.parseTransaction( 
        [
          ...txL1, 
          ...txL2,
          ...txL1pending //the new data product
        ]
      )
      //console.log("annotated:",annotated)
      return annotated
    }

  }

  /* Where possible, annotate the transactions
  based on contract addresses */
  async parseTransaction( transactions ) {

    // NOT SUPPORTED on LOCAL
    if (this.masterSystemConfig === 'local') return

    var annotatedTX = transactions.map(item => {

      let to = item.to

      if ( to === null || to === '') {
        return item
      }

      to = to.toLowerCase()

      if (to === this.L2LPAddress.toLowerCase()) {
        //console.log("L2->L1 Swap Off")
        return Object.assign({}, item, { typeTX: 'Fast Offramp' })
      }

      if (to === this.L1LPAddress.toLowerCase()) {
        //console.log("L1->L2 Swap On")
        return Object.assign({}, item, { typeTX: 'Fast Onramp' })
      }

      if (to === this.L1StandardBridgeAddress.toLowerCase()) {
        //console.log("L1->L2 Traditional Deposit")
        return Object.assign({}, item, { typeTX: 'Traditional' })
      }

      if (to === this.L1_TEST_Address.toLowerCase()) {
        //console.log("L1 ERC20 Amount Approval")
        return Object.assign({}, item, { typeTX: 'L1 ERC20 Amount Approval' })
      }

      if (to === this.L2StandardBridgeAddress.toLowerCase()) {
        //console.log("L2 Standard Bridge")
        return Object.assign({}, item, { typeTX: 'L2 Standard Bridge' })
      }

      if (to === this.L1Message.toLowerCase()) {
        //console.log("L1 Message")
        return Object.assign({}, item, { typeTX: 'L1 Message' })
      }

      if (to === this.L2Message.toLowerCase()) {
        //console.log("L2 Message")
        return Object.assign({}, item, { typeTX: 'L2 Message' })
      }

      if (to === this.L2_TEST_Address.toLowerCase()) {
        //console.log("L2 TEST Message")
        return Object.assign({}, item, { typeTX: 'L2 TEST Token' })
      }

      if (to === this.L2_ETH_Address.toLowerCase()) {
        //console.log("L2 ETH Message")
        return Object.assign({}, item, { typeTX: 'L2 ETH Ops (such as a L2->L2 Transfer)' })
      }

      // if (to === this.L2_ETH_Address.toLowerCase()) {
      //   //console.log("L2 ETH Message")
      //   return Object.assign({}, item, { typeTX: 'L2 ETH Token' })
      // }

      if (item.crossDomainMessage) {
        if(to === this.L2LPAddress.toLowerCase()) {
          //console.log("Found EXIT: L2LPAddress")
          return Object.assign({}, item, { typeTX: 'FAST EXIT via L2LP' })
        }
        else if (to === this.L2_TEST_Address.toLowerCase()) {
          //console.log("Found EXIT: L2_TEST_Address")
          return Object.assign({}, item, { typeTX: 'EXIT (TEST Token)' })
        }
        else if (to === this.L2_ETH_Address.toLowerCase()) {
          //console.log("Found EXIT: L2_ETH_Address")
          return Object.assign({}, item, { typeTX: 'EXIT ETH' })
        }
      }

      return Object.assign({}, item, { typeTX: to })

    }) //map

    return annotatedTX

  }

  async getExits() {

    // NOT SUPPORTED on LOCAL
    if (this.masterSystemConfig === 'local') return

    const response = await omgxWatcherAxiosInstance(
      this.masterSystemConfig
    ).post('get.l2.transactions', {
      address: this.account,
      fromRange:  0,
      toRange: 1000,
    })
    if (response.status === 201) {
      const transactions = response.data
      const filteredTransactions = transactions.filter(
        (i) =>
          [
            this.L2LPAddress.toLowerCase(),
            this.L2_TEST_Address.toLowerCase(),
            this.L2_ETH_Address.toLowerCase(),
          ].includes(i.to ? i.to.toLowerCase() : null) && i.crossDomainMessage
      )
      return { exited: filteredTransactions }
    }

  }

  //goal is to find your NFTs and NFT contracts based on local cache and registry data 
  async fetchNFTs() {

    //the current list of contracts we know about
    //based in part on the cache and anything we recently generated in this session
    //console.log("NFTContracts 1:",await getNFTContracts())

    let NFTContracts = Object.entries(await getNFTContracts())
    //console.log("Step 1 - NFTContracts:",NFTContracts)

    //list of NFT contract addresses we know about, locally
    const localCache = NFTContracts.map(item => {
      return item[0].toLowerCase()
    })

    //console.log("Step 2 - localCache addresses:",localCache)

    //the Boba NFT registry
    const registry = new ethers.Contract(
      this.ERC721RegAddress,
      L2ERC721RegJson.abi,
      this.L2Provider
    )
    
    //This account's NFT contract addresses in that registry
    const addresses = await registry.lookupAddress(this.account)
    //console.log("Step 3 - Blockchain NFT wallet addresses:", addresses)

    //make sure we have all the contracts relevant to this user
    for(let i = 0; i < addresses.length; i++) {
      const address = addresses[i]
      var inCache = (localCache.indexOf(address.toLowerCase()) > -1)
      if(!inCache) {
        console.log("Found a new NFT contract - adding:",address)
        //Add to local NFT contracts structure
        const contract = new ethers.Contract(
          address,
          L2ERC721Json.abi,
          this.L2Provider
        )

        //always the same, no need to have in the loop
        let nftName = await contract.name()
        let nftSymbol = await contract.symbol()
        let owner = await contract.owner()

        const newContract = {
          name: nftName,
          symbol: nftSymbol,
          owner: owner.toLowerCase(), 
          address,
        }

        console.log("newContract just added:",newContract)

        await addNFTContract( newContract )
      }
    }

    //How many NFTs do you have right now?
    let numberOfNFTS = 0

    NFTContracts = Object.entries(await getNFTContracts())

    for(let i = 0; i < NFTContracts.length; i++) {

      let contract = new ethers.Contract(
        NFTContracts[i][1].address,
        L2ERC721Json.abi,
        this.L2Provider
      )

      //how many NFTs of this flavor do I own?
      const balance = await contract.connect(
        this.L2Provider
      ).balanceOf(this.account)

      numberOfNFTS = numberOfNFTS + Number(balance.toString())

    }

    //let's see if we already know about them
    const myNFTS = getNFTs()
    const numberOfStoredNFTS = Object.keys(myNFTS).length

    if (numberOfNFTS !== numberOfStoredNFTS) {

      console.log('NFT change - need to add one or more NFTs')

      for(let i = 0; i < NFTContracts.length; i++) {

        const address = NFTContracts[i][1].address

        const contract = new ethers.Contract(
          address,
          L2ERC721Json.abi,
          this.L2Provider
        )

        const balance = await contract.connect(
          this.L2Provider
        ).balanceOf(this.account)

        //always the same, no need to have in the loop
        let nftName = await contract.name()
        let nftSymbol = await contract.symbol()

        //can have more than 1 per contract
        for (let i = 0; i < Number(balance.toString()); i++) {

          //Goal here is to get all the tokenIDs, e.g. 3, 7, and 98,
          //based on knowing the user's balance - e.g. three NFTs
          const tokenIndex = BigNumber.from(i)

          const tokenID = await contract.tokenOfOwnerByIndex(
            this.account,
            tokenIndex
          )

          const nftMeta = await contract.getTokenURI(tokenID)
          
          const meta = nftMeta.split('#')
          
          const time = new Date(parseInt(meta[0]))

          const mintedTime = String(
              time.toLocaleString('en-US', {
                day: '2-digit',
                month: '2-digit',
                year: 'numeric',
                hour: 'numeric',
                minute: 'numeric',
                hour12: true,
              })
            )

          const UUID = address.substring(1, 6) + '_' + tokenID.toString() + '_' + this.account.substring(1, 6)

          const { url , attributes = []} = await getNftImageUrl(meta[1]);
          // Uncomment Just to test locally
          // const { url , attributes = []} = await getNftImageUrl('https://boredapeyachtclub.com/api/mutants/111');

          // const { url , attributes = []} = await getNftImageUrl('ipfs://QmeSjSinHpPnmXmspMjwiXyN6zS4E9zccariGR3jxcaWtq/6190');
          
          const NFT = {
            UUID,
            mintedTime,
            url,
            tokenID,
            name: nftName,
            symbol: nftSymbol,
            address,
            attributes,
          }

          await addNFT( NFT )

        }
      }
    }
  }

  async addTokenList() {
    // Add the token to our master list, if we do not have it yet
    // if the token is already in the list, then this function does nothing
    // but if a new token shows up, then it will get added
    Object.keys(this.tokenAddresses).forEach((token, i) => {
      getToken(this.tokenAddresses[token].L1)
    })
  }

  async getBalances() {
    try {
      // Always check ETH and oETH
      const layer1Balance = await this.L1Provider.getBalance(this.account)
      const layer2Balance = await this.L2Provider.getBalance(this.account)

      const layer1Balances = [
        {
          address: this.L1_ETH_Address,
          addressL2: this.L2_ETH_Address,
          currency: this.L1_ETH_Address,
          symbol: 'ETH',
          decimals: 18,
          balance: new BN(layer1Balance.toString()),
        },
      ]

      const layer2Balances = [
        {
          address: this.L2_ETH_Address,
          addressL1: this.L1_ETH_Address,
          currency: this.L1_ETH_Address,
          symbol: 'oETH',
          decimals: 18,
          balance: new BN(layer2Balance.toString()),
        },
      ]

      const state = store.getState()
      const tA = Object.values(state.tokenList)

      const tokenC = new ethers.Contract(
        this.L1_ETH_Address,
        L1ERC20Json.abi,
        this.L1Provider
      )

      const getERC20Balance = async(token, tokenAddress, layer, provider=this.L1Provider) => {
        const balance = await tokenC.attach(tokenAddress).connect(provider).balanceOf(this.account);
        return {
          ...token, balance: new BN(balance.toString()),
          layer, address: layer === 'L1' ? token.addressL1: token.addressL2,
          symbol: token.symbolL1
        };
      }

      const getBalancePromise = [];

      tA.forEach((token) => {
        if (token.addressL1 === this.L1_ETH_Address) return;
        getBalancePromise.push(getERC20Balance(token, token.addressL1, "L1"))
        getBalancePromise.push(getERC20Balance(token, token.addressL2, "L2", this.L2Provider))
      })

      const tokenBalances = await Promise.all(getBalancePromise)
      tokenBalances.forEach((token) => {
        if (token.layer === 'L1' && token.balance.gt(new BN(0))) {
          layer1Balances.push(token)
        } else if (token.layer === 'L2' && token.balance.gt(new BN(0))){
          layer2Balances.push(token)
        }
      })

      return {
        layer1: orderBy(layer1Balances, (i) => i.currency),
        layer2: orderBy(layer2Balances, (i) => i.currency),
      }
    } catch (error) {
      throw new WebWalletError({
        originalError: error,
        reportToSentry: false,
        reportToUi: false,
      })
    }
  }

  //Move ETH from L1 to L2 using the standard deposit system
  depositETHL2 = async (value = '1', gasPrice) => {

    updateSignatureStatus_depositTRAD(false)

    try {
      const depositTxStatus = await this.L1StandardBridgeContract.depositETH(
        this.L2GasLimit,
        utils.formatBytes32String(new Date().getTime().toString()),
        {
          value: parseEther(value),
          gasPrice: ethers.utils.parseUnits(`${gasPrice}`, 'wei'),
        }
      )
      //closes the Deposit modal
      updateSignatureStatus_depositTRAD(true)
      
      //at this point the tx has been submitted, and we are waiting...
      await depositTxStatus.wait()

      const [l1ToL2msgHash] = await this.watcher.getMessageHashesFromL1Tx(
        depositTxStatus.hash
      )
      console.log(' got L1->L2 message hash', l1ToL2msgHash)

      const l2Receipt = await this.watcher.getL2TransactionReceipt(
        l1ToL2msgHash
      )
      console.log(' completed Deposit! L2 tx hash:', l2Receipt.transactionHash)

      return l2Receipt
    } catch(error) {
      console.log(error)
      return false
    }
  }

  //Transfer funds from one account to another, on the L2
  async transfer(address, value, currency) {
    try {
      //any old ERC20 json will do....
      const tx = await this.L2_TEST_Contract.attach(currency).transfer(
        address,
        parseEther(value.toString())
      )
      await tx.wait()
      return tx
    } catch (error) {
      console.log(error)
    }
  }

  //figure out which layer we are on right now
  confirmLayer = (layerToConfirm) => async (dispatch) => {
    if (layerToConfirm === this.L1orL2) {
      return true
    } else {
      return false
    }
  }

  async checkAllowance(
    currencyAddress,
    targetContract
  ) {
    try {
      const ERC20Contract = new ethers.Contract(
        currencyAddress,
        L1ERC20Json.abi, //could use any abi - just something with .allowance
        this.provider.getSigner()
      )
      const allowance = await ERC20Contract.allowance(
        this.account,
        targetContract
      )
      return allowance.toString()
    } catch (error) {
      throw new WebWalletError({
        originalError: error,
        customErrorMessage: 'Could not check ERC20 allowance.',
        reportToSentry: false,
        reportToUi: true,
      })
    }
  }

  /*Used when people want to fast exit - they have to deposit funds into the L2LP*/
  async approveERC20_L2LP(
    depositAmount_string,
    currencyAddress
  ) {

    try {

      console.log("approveERC20_L2LP")

      //we could use any L2 ERC contract here - just getting generic parts of the abi
      //but we know we alaways have the TEST contract, so will use that
      const L2ERC20Contract = this.L2_TEST_Contract.attach(currencyAddress)

      let allowance_BN = await L2ERC20Contract.allowance(
        this.account,
        this.L2LPAddress
      )

      let depositAmount_BN = new BN(depositAmount_string)

      if (depositAmount_BN.gt(allowance_BN)) {
        const approveStatus = await L2ERC20Contract.approve(
          this.L2LPAddress,
          depositAmount_string
        )
        await approveStatus.wait()
      }

      return true
    } catch (error) {
      return false
    }
  }

  async approveERC20_L1LP(
    depositAmount_string,
    currencyAddress
  ) {

    try {

      console.log("approveERC20_L1LP")

      const ERC20Contract = new ethers.Contract(
        currencyAddress,
        L1ERC20Json.abi,
        this.provider.getSigner()
      )

      const approveStatus = await ERC20Contract.approve(
        this.L1LPAddress,
        depositAmount_string
      )
      await approveStatus.wait()

      return true
    } catch (error) {
      return false
    }
  }

  async approveERC20(
    value,
    currency,
    approveContractAddress = this.L1StandardBridgeAddress,
    contractABI = L1ERC20Json.abi
  ) {

    try {

      console.log("approveERC20")

      const ERC20Contract = new ethers.Contract(
        currency,
        contractABI,
        this.provider.getSigner()
      )

      const approveStatus = await ERC20Contract.approve(
        approveContractAddress,
        value
      )
      await approveStatus.wait()

      return true
    } catch (error) {
      return false
    }
  }

  async resetApprove(
    value,
    currency,
    approveContractAddress = this.L1StandardBridgeAddress,
    contractABI = L1ERC20Json.abi
  ) {
    try {
      const ERC20Contract = new ethers.Contract(
        currency,
        contractABI,
        this.provider.getSigner()
      )

      const resetApproveStatus = await ERC20Contract.approve(
        approveContractAddress,
        0
      )
      await resetApproveStatus.wait()

      const approveStatus = await ERC20Contract.approve(
        approveContractAddress,
        value
      )
      await approveStatus.wait()
      return true
    } catch (error) {
      throw new WebWalletError({
        originalError: error,
        customErrorMessage: 'Could not reset allowance for ERC20.',
        reportToSentry: false,
        reportToUi: true,
      })
    }
  }

  //Used to move ERC20 Tokens from L1 to L2
  async depositErc20(value, currency, gasPrice, currencyL2) {

    updateSignatureStatus_depositTRAD(false)

    try {
      //could use any ERC20 here...
      const L1_TEST_Contract = this.L1_TEST_Contract.attach(currency)

      const approveStatus = await L1_TEST_Contract.approve(
        this.L1StandardBridgeAddress, //this is the spender
        value
      )
      await approveStatus.wait()

      const depositTxStatus = await this.L1StandardBridgeContract.depositERC20(
        currency,
        currencyL2,
        value,
        this.L2GasLimit,
        utils.formatBytes32String(new Date().getTime().toString())
      )
      
      //closes the Deposit modal
      updateSignatureStatus_depositTRAD(true)
      
      //at this point the tx has been submitted, and we are waiting...
      await depositTxStatus.wait()

      const [l1ToL2msgHash] = await this.watcher.getMessageHashesFromL1Tx(
        depositTxStatus.hash
      )
      console.log(' got L1->L2 message hash', l1ToL2msgHash)

      const l2Receipt = await this.watcher.getL2TransactionReceipt(
        l1ToL2msgHash
      )
      console.log(' completed Deposit! L2 tx hash:', l2Receipt.transactionHash)

      this.getBalances()

      return l2Receipt
    } catch (error) {
      throw new WebWalletError({
        originalError: error,
        customErrorMessage:
          'Could not deposit ERC20. Please check to make sure you have enough in your wallet to cover both the amount you want to deposit and the associated gas fees.',
        reportToSentry: false,
        reportToUi: true,
      })
    }
  }

  //Standard 7 day exit from BOBA
  async exitBOBA(currencyAddress, value) {

    updateSignatureStatus_exitTRAD(false)

    const allowance = await this.checkAllowance(
      currencyAddress,
      this.L2StandardBridgeAddress
    )
    const L2_ERC20_Contract = new ethers.Contract(
      currencyAddress,
      L2ERC20Json.abi,
      this.provider.getSigner()
    )
    const decimals = await L2_ERC20_Contract.decimals()
    // need the frontend updates
    if (BigNumber.from(allowance).lt(parseUnits(value, decimals))) {
      const res = await this.approveERC20(
        // take caution while using parseEther() with erc20
        // l2 erc20 can be customised to have non 18 decimals
        parseUnits(value, decimals),
        currencyAddress,
        this.L2StandardBridgeAddress
      )
      if (!res) return false
    }

    const tx = await this.L2StandardBridgeContract.withdraw(
      currencyAddress,
      parseUnits(value, decimals),
      this.L1GasLimit,
      utils.formatBytes32String(new Date().getTime().toString())
    )
    //can close window now
    updateSignatureStatus_exitTRAD(true)    
    await tx.wait()

    const [L2ToL1msgHash] = await this.watcher.getMessageHashesFromL2Tx(tx.hash)
    console.log(' got L2->L1 message hash', L2ToL1msgHash)

    return tx
  }

  /***********************************************/
  /*****                  Fee                *****/
  /***********************************************/

  // Total exit fee
  async getTotalFeeRate() {
    const L2LPContract = new ethers.Contract(
      this.L2LPAddress,
      L2LPJson.abi,
      this.L2Provider
    )
    const [userRewardFeeRate, ownerRewardFeeRate] = await Promise.all([
      L2LPContract.userRewardFeeRate(),
      L2LPContract.ownerRewardFeeRate()
    ])
    const feeRate = Number(userRewardFeeRate) + Number(ownerRewardFeeRate);
    return ((feeRate / 1000) * 100).toFixed(0)
  }

  async getUserRewardFeeRate() {
    const L2LPContract = new ethers.Contract(
      this.L2LPAddress,
      L2LPJson.abi,
      this.L2Provider
    )
    const feeRate = await L2LPContract.userRewardFeeRate()
    return ((feeRate / 1000) * 100).toFixed(1)
  }

  /*****************************************************/
  /***** Pool, User Info, to populate the Farm tab *****/
  /*****************************************************/
  async getL1LPInfo() {

    const tokenAddressList = Object.keys(this.addresses.TOKENS).reduce((acc, cur) => {
      acc.push(this["L1_" + cur + "_Address"]);
      return acc;
    }, [this.L1_ETH_Address]);

    const L1LPContract = new ethers.Contract(
      this.L1LPAddress,
      L1LPJson.abi,
      this.L1Provider
    )

    const poolInfo = {}
    const userInfo = {}

    const L1LPInfoPromise = [];

    const getL1LPInfoPromise = async(tokenAddress) => {
      let tokenBalance
      let tokenSymbol
      let tokenName
      let decimals

      if (tokenAddress === this.L1_ETH_Address) {
        tokenBalance = await this.L1Provider.getBalance(this.L1LPAddress)
        tokenSymbol = 'ETH'
        tokenName = 'Ethereum'
        decimals = 18
      } else {
        tokenBalance = await this.L1_TEST_Contract.attach(tokenAddress).connect(this.L1Provider).balanceOf(this.L1LPAddress)
        tokenSymbol = await this.L1_TEST_Contract.attach(tokenAddress).connect(this.L1Provider).symbol()
        tokenName = await this.L1_TEST_Contract.attach(tokenAddress).connect(this.L1Provider).name()
        decimals = await this.L1_TEST_Contract.attach(tokenAddress).connect(this.L1Provider).decimals()
      }
      const poolTokenInfo = await L1LPContract.poolInfo(tokenAddress)
      const userTokenInfo = await L1LPContract.userInfo(tokenAddress, this.account)
      return { tokenAddress, tokenBalance, tokenSymbol, tokenName, poolTokenInfo, userTokenInfo,decimals }
    }

    tokenAddressList.forEach((tokenAddress) => L1LPInfoPromise.push(getL1LPInfoPromise(tokenAddress)))
    const L1LPInfo = await Promise.all(L1LPInfoPromise);
    
    L1LPInfo.forEach((token) => {
      poolInfo[token.tokenAddress] = {
        symbol: token.tokenSymbol,
        name: token.tokenName,
        decimals: token.decimals,
        l1TokenAddress: token.poolTokenInfo.l1TokenAddress,
        l2TokenAddress: token.poolTokenInfo.l2TokenAddress,
        accUserReward: token.poolTokenInfo.accUserReward.toString(),
        accUserRewardPerShare: token.poolTokenInfo.accUserRewardPerShare.toString(),
        userDepositAmount: token.poolTokenInfo.userDepositAmount.toString(),
        startTime: token.poolTokenInfo.startTime.toString(),
        APR:
          Number(token.poolTokenInfo.userDepositAmount.toString()) === 0
            ? 0
            : accMul(
                accDiv(
                  accDiv(
                    token.poolTokenInfo.accUserReward,
                    token.poolTokenInfo.userDepositAmount
                  ),
                  accDiv(
                    new Date().getTime() -
                      Number(token.poolTokenInfo.startTime) * 1000,
                    365 * 24 * 60 * 60 * 1000
                  )
                ),
                100
              ), // ( accUserReward - userDepositAmount ) / timeDuration
        tokenBalance: token.tokenBalance.toString()
      }
      userInfo[token.tokenAddress] = {
        l1TokenAddress: token.tokenAddress,
        amount: token.userTokenInfo.amount.toString(),
        pendingReward: token.userTokenInfo.pendingReward.toString(),
        rewardDebt: token.userTokenInfo.rewardDebt.toString()
      }
    })
    return { poolInfo, userInfo }
  }

  async getL2LPInfo() {

    const tokenAddressList = Object.keys(this.addresses.TOKENS).reduce((acc, cur) => {
      acc.push({
        L1: this["L1_" + cur + "_Address"],
        L2: this["L2_" + cur + "_Address"]
      });
      return acc;
    }, [{
      L1: this.L1_ETH_Address, 
      L2: this.L2_ETH_Address
    }]);

    const L2LPContract = new ethers.Contract(
      this.L2LPAddress,
      L2LPJson.abi,
      this.L2Provider
    )

    const poolInfo = {}
    const userInfo = {}

    const L2LPInfoPromise = [];

    const getL2LPInfoPromise = async(tokenAddress, tokenAddressL1 ) => {
      let tokenBalance
      let tokenSymbol
      let tokenName
      let decimals

      if (tokenAddress === this.L2_ETH_Address) {
        tokenBalance = await this.L2Provider.getBalance(this.L2LPAddress)
        tokenSymbol = 'oETH'
        tokenName = 'Ethereum'
        decimals = 18;
      } else {
        tokenBalance = await this.L2_TEST_Contract.attach(tokenAddress).connect(this.L2Provider).balanceOf(this.L2LPAddress)
        tokenSymbol = await this.L2_TEST_Contract.attach(tokenAddress).connect(this.L2Provider).symbol()
        tokenName = await this.L2_TEST_Contract.attach(tokenAddress).connect(this.L2Provider).name()
        decimals = await this.L1_TEST_Contract.attach(tokenAddressL1).connect(this.L1Provider).decimals()
      }
      const poolTokenInfo = await L2LPContract.poolInfo(tokenAddress)
      const userTokenInfo = await L2LPContract.userInfo(tokenAddress, this.account)
      return { tokenAddress, tokenBalance, tokenSymbol, tokenName, poolTokenInfo, userTokenInfo, decimals}
    }

    tokenAddressList.forEach(({L1, L2}) => L2LPInfoPromise.push(getL2LPInfoPromise(L2, L1)))

    const L2LPInfo = await Promise.all(L2LPInfoPromise);
    L2LPInfo.forEach((token) => {
      poolInfo[token.tokenAddress] = {
        symbol: token.tokenSymbol,
        name: token.tokenName,
        decimals: token.decimals,
        l1TokenAddress: token.poolTokenInfo.l1TokenAddress,
        l2TokenAddress: token.poolTokenInfo.l2TokenAddress,
        accUserReward: token.poolTokenInfo.accUserReward.toString(),
        accUserRewardPerShare: token.poolTokenInfo.accUserRewardPerShare.toString(),
        userDepositAmount: token.poolTokenInfo.userDepositAmount.toString(),
        startTime: token.poolTokenInfo.startTime.toString(),
        APR:
          Number(token.poolTokenInfo.userDepositAmount.toString()) === 0
            ? 0
            : accMul(
                accDiv(
                  accDiv(
                    token.poolTokenInfo.accUserReward,
                    token.poolTokenInfo.userDepositAmount
                  ),
                  accDiv(
                    new Date().getTime() -
                      Number(token.poolTokenInfo.startTime) * 1000,
                    365 * 24 * 60 * 60 * 1000
                  )
                ),
                100
              ), // ( accUserReward - userDepositAmount ) / timeDuration
        tokenBalance: token.tokenBalance.toString()
      }
      userInfo[token.tokenAddress] = {
        l2TokenAddress: token.tokenAddress,
        amount: token.userTokenInfo.amount.toString(),
        pendingReward: token.userTokenInfo.pendingReward.toString(),
        rewardDebt: token.userTokenInfo.rewardDebt.toString()
      }
    })

    return { poolInfo, userInfo }
  }

  /***********************************************/
  /*****            Add Liquidity            *****/
  /***********************************************/
  async addLiquidity(currency, value, L1orL2Pool, decimals) {

    let depositAmount = powAmount(value, decimals)

    try {
      // Deposit
      const addLiquidityTX = await (L1orL2Pool === 'L1LP'
        ? this.L1LPContract
        : this.L2LPContract
      ).addLiquidity(
        depositAmount,
        currency,
        currency === this.L1_ETH_Address ? { value: depositAmount } : {}
      )
      await addLiquidityTX.wait()
      return true
    } catch (err) {
      console.log(err)
      return false
    }
  }

  /***********************************************/
  /*****           Get Reward L1             *****/
  /***********************************************/
  async getRewardL1(currencyL1Address, userReward_BN) {

    try {
      const withdrawRewardTX = await this.L1LPContract.withdrawReward(
        userReward_BN,
        currencyL1Address,
        this.account
      )
      await withdrawRewardTX.wait()
      return true
    } catch (err) {
      return false
    }
  }

  /***********************************************/
  /*****           Get Reward L2             *****/
  /***********************************************/
  async getRewardL2(currencyL2Address, userReward_BN) {

    try {
      const withdrawRewardTX = await this.L2LPContract.withdrawReward(
        userReward_BN,
        currencyL2Address,
        this.account
      )
      await withdrawRewardTX.wait()
      return true
    } catch (err) {
      return false
    }
  }

  /***********************************************/
  /*****          Withdraw Liquidity         *****/
  /***********************************************/
  async withdrawLiquidity(currency, value, L1orL2Pool, decimals) {
    
    let withdrawAmount = powAmount(value, decimals)
    try {
      // Deposit
      const withdrawLiquidityTX = await (L1orL2Pool === 'L1LP'
        ? this.L1LPContract
        : this.L2LPContract
      ).withdrawLiquidity(
        withdrawAmount,
        currency,
        this.account
      )
      await withdrawLiquidityTX.wait()
      return true
    } catch (err) {
      return false
    }
  }

  /***********************************************************/
  /***** SWAP ON to BOBA by depositing funds to the L1LP *****/
  /***********************************************************/
  async depositL1LP(currency, value, decimals) {

    updateSignatureStatus_depositLP(false)
    let depositAmount = powAmount(value, decimals)

    const depositTX = await this.L1LPContract.clientDepositL1(
      depositAmount.toString(),
      currency,
      currency === this.L1_ETH_Address ? { value: depositAmount } : {}
    )

    updateSignatureStatus_depositLP(true)
    //at this point the tx has been submitted, and we are waiting...
    await depositTX.wait()

    // Waiting the response from L2
    const [l1ToL2msgHash] = await this.watcher.getMessageHashesFromL1Tx(
      depositTX.hash
    )
    console.log(' got L1->L2 message hash', l1ToL2msgHash)
    const l2Receipt = await this.watcher.getL2TransactionReceipt(l1ToL2msgHash)
    console.log(' completed swap-on ! L2 tx hash:', l2Receipt.transactionHash)

    return l2Receipt
  }

  /***************************************/
  /************ L1LP Pool size ***********/
  /***************************************/
  async L1LPBalance(tokenAddress, decimals) {
    let balance
    let tokenAddressLC = tokenAddress.toLowerCase()
    if (
      tokenAddressLC === this.L2_ETH_Address ||
      tokenAddressLC === this.L1_ETH_Address
    ) {
      balance = await this.L1Provider.getBalance(this.L1LPAddress)
    } else {
      balance = await this.L1_TEST_Contract.attach(tokenAddress).connect(this.L1Provider).balanceOf(
        this.L1LPAddress
      )
    }

    if(typeof(balance) === 'undefined') {
      return logAmount('0', decimals)
    } else {
      return logAmount(balance.toString(), decimals)
    }

  }

  /***************************************/
  /************ L2LP Pool size ***********/
  /***************************************/
  async L2LPBalance(tokenAddress, decimals) {
    let balance
    let tokenAddressLC = tokenAddress.toLowerCase()
    if (
      tokenAddressLC === this.L2_ETH_Address ||
      tokenAddressLC === this.L1_ETH_Address
    ) {
      //We are dealing with ETH
      balance = await this.L2_ETH_Contract.connect(this.L2Provider).balanceOf(
        this.L2LPAddress
      )
    } else {
      balance = await this.L2_TEST_Contract.attach(tokenAddress).connect(this.L2Provider).balanceOf(
        this.L2LPAddress
      )
    }

    if(typeof(balance) === 'undefined') {
      return logAmount('0', decimals)
    } else {
      return logAmount(balance.toString(), decimals)
    }

  }

  /**************************************************************/
  /***** SWAP OFF from BOBA by depositing funds to the L2LP *****/
  /**************************************************************/
  async depositL2LP(currencyAddress, depositAmount_string) {

    updateSignatureStatus_exitLP(false)

    const L2ERC20Contract = new ethers.Contract(
      currencyAddress,
      L2ERC20Json.abi,
      this.provider.getSigner()
    )

    let allowance_BN = await L2ERC20Contract.allowance(
      this.account,
      this.L2LPAddress
    )

    let depositAmount_BN = new BN(depositAmount_string)

    if (depositAmount_BN.gt(allowance_BN)) {
      const approveStatus = await L2ERC20Contract.approve(
        this.L2LPAddress,
        depositAmount_string
      )
      await approveStatus.wait()
      if (!approveStatus) return false
    }

    const depositTX = await this.L2LPContract.clientDepositL2(
      depositAmount_string,
      currencyAddress
    )

    updateSignatureStatus_exitLP(true)
    //at this point the tx has been submitted, and we are waiting...
    await depositTX.wait()

    // Waiting for the response from L1
    const [L2ToL1msgHash] = await this.fastWatcher.getMessageHashesFromL2Tx(
      depositTX.hash
    )
    console.log(' got L2->L1 message hash', L2ToL1msgHash)

    const L1Receipt = await this.fastWatcher.getL1TransactionReceipt(
      L2ToL1msgHash
    )
    console.log(' completed Deposit! L1 tx hash:', L1Receipt.transactionHash)

    return L1Receipt
  }

  async fetchLookUpPrice(params) {
    try {
       // fetching only the prices compare to usd.
       const res = await coinGeckoAxiosInstance.get(
         `simple/price?ids=${params.join()}&vs_currencies=usd`
       )
       return res.data;
    } catch(error) {
      return error
    }
  }

  async getGasPrice({
    network, networkLayer
  }) {
    if(network === 'mainnet' && networkLayer === 'L1') {
      try {
        const { data: { safeLow, average, fast } } = await ethGasStationAxiosInstance.get('json/ethgasAPI.json');
        return {
          slow: safeLow * 100000000,
          normal: average * 100000000,
          fast: fast * 100000000
        };
      } catch (error) {
        //
      }
      // if not web3 oracle
      try {
        const _medianEstimate = await this.web3.eth.getGasPrice();
        const medianEstimate = Number(_medianEstimate);
        return {
          slow: Math.max(medianEstimate / 2, 1000000000),
          normal: medianEstimate,
          fast: medianEstimate * 5
        };
      } catch (error) {
        //
      }
    }

    return {
      slow: 1000000000,
      normal: 2000000000,
      fast: 10000000000
    }
  }

  /***********************************************/
  /*****         DAO Functions               *****/
  /***********************************************/

  // get DAO Balance
  async getDaoBalance() {
    if(this.masterSystemConfig !== 'rinkeby' || this.L1orL2 !== 'L2') return

    try {
      let balance = await this.comp.balanceOf(this.account)
      return { balance: formatEther(balance) }
    } catch (error) {
      console.log('Error: DAO Balance', error)
      throw new Error(error.message)
    }
  }

  // get DAO Votes
  async getDaoVotes() {
    if(this.masterSystemConfig !== 'rinkeby' || this.L1orL2 !== 'L2') return
    try {
      let votes = await this.comp.getCurrentVotes(this.account)
      return { votes: formatEther(votes) }
    } catch (error) {
      console.log('Error: DAO Votes', error)
      throw new Error(error.message);
    }
  }

  //Transfer DAO Funds
  async transferDao({ recipient, amount }) {
    try {
      const tx = await this.comp.transfer(recipient, parseEther(amount.toString()))
      await tx.wait()
      return tx
    } catch (error) {
      console.log('Error: DAO Transfer', error)
      throw new Error(error.message);
    }
  }

  //Delegate DAO Authority
  async delegateVotes({ recipient }) {
    try {
      const tx = await this.comp.delegate(recipient)
      await tx.wait()
      return tx
    } catch (error) {
      console.log('Error: DAO Delegate', error)
      throw new Error(error.message)
    }
  }

  // Proposal Create Threshold

  async getProposalThreshold() {
    try {
      // get the threshold proposal only in case of L2
      if(this.L1orL2 === 'L2') { 
        const delegateCheck = await this.delegate.attach(this.delegator.address);
        let rawThreshold = await delegateCheck.proposalThreshold();
        return { threshold: formatEther(rawThreshold) };
      }
      else {
        return { threshold: 0 };
      }
    } catch (error) {
      console.log(error);
      throw new Error(error.message);
    }
  }

  //Create Proposal
  async createProposal({ votingThreshold = null, text = null }) {
    try {
      const delegateCheck = await this.delegate.attach(this.delegator.address)

      //console.log(":",delegateCheck)
      let address = [delegateCheck.address];
      let values = [0];
      let signatures = !text ? ['_setProposalThreshold(uint256)'] : [''] // the function that will carry out the proposal
      let voting = !text ? ethers.utils.parseEther(votingThreshold) : 0;
      let calldatas = [ethers.utils.defaultAbiCoder.encode( // the parameter for the above function
        ['uint256'],
        [voting]
      )]
      let description = !text ? `# Changing Proposal Threshold to ${votingThreshold} Comp` : text;
      
      let setGas = {
        gasPrice: 15000000,
        gasLimit: 8000000
      };

      let res = await delegateCheck.propose(
        address,
        values,
        signatures,
        calldatas,
        description,
        setGas
      )
      return res;
    } catch (error) {
      console.log(error);
      throw new Error(error.message);
    }
  }

  //Fetch Proposals
  async fetchProposals() {
    
    if(this.masterSystemConfig !== 'rinkeby' || this.L1orL2 !== 'L2') return
      
    const delegateCheck = await this.delegate.attach(this.delegator.address)
    
    try {
      let proposalList = [];
      const proposalCounts = await delegateCheck.proposalCount()
      const totalProposals = await proposalCounts.toNumber() - 1 //it's always off by one??
      
      const filter = delegateCheck.filters.ProposalCreated(
        null,
        null,
        null,
        null,
        null,
        null,
        null,
        null,
        null
      )
      
      const descriptionList = await delegateCheck.queryFilter(filter);
      for (let i = 0; i < totalProposals; i++) {
        
        let proposalID = descriptionList[i].args[0]
        //this is a number such as 2
        let proposalData = await delegateCheck.proposals(proposalID)
        const proposalStates = [
          'Pending',
          'Active',
          'Canceled',
          'Defeated',
          'Succeeded',
          'Queued',
          'Expired',
          'Executed',
        ]

        let state = await delegateCheck.state(proposalID)
        let againstVotes = parseInt(formatEther(proposalData.againstVotes))
        let forVotes = parseInt(formatEther(proposalData.forVotes))
        let abstainVotes = parseInt(formatEther(proposalData.abstainVotes))

        let startBlock = proposalData.startBlock.toString()
        let endBlock = proposalData.endBlock.toString()

        let proposal = await delegateCheck.getActions(i+2)
        
        let description = descriptionList[i].args[8].toString()
        
        proposalList.push({
           id: proposalID.toString(),
           proposal,
           description,
           totalVotes: forVotes + againstVotes,
           forVotes,
           againstVotes,
           abstainVotes,
           state: proposalStates[state],
           startBlock,
           endBlock

        })

      }
      return { proposalList }
    } catch (error) {
      console.log(error)
      throw new Error(error.message)
    }
  }

  //Cast vote for proposal 
  async castProposalVote({id, userVote}) {
    try {
      
      const delegateCheck = await this.delegate.attach(this.delegator.address);
      let res = delegateCheck.castVote(id, userVote, {
        gasPrice: 15000000,
        gasLimit: 8000000
      });
      return res;
    } catch(error) {
      console.log('Error: cast vote', error);
      throw new Error(error.message);
    }
  }

}

const networkService = new NetworkService()
export default networkService