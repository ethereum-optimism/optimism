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

import { parseUnits, parseEther } from '@ethersproject/units'
import { Watcher } from '@eth-optimism/watcher'

import { ethers, BigNumber, utils, ContractFactory } from 'ethers'
import store from 'store'

import { orderBy } from 'lodash'
import BN from 'bn.js'

import { getToken } from 'actions/tokenAction'

import { 
  addNFT, 
  getNFTs, 
  addNFTFactory, 
  getNFTFactories,
  addNFTContract,
  getNFTContracts, 
} from 'actions/nftAction'

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

import { powAmount, logAmount } from 'util/amountConvert'
import { accDiv, accMul } from 'util/calculation'

import { getAllNetworks } from 'util/masterConfig'

import etherScanInstance from 'api/etherScanAxios'
import omgxWatcherAxiosInstance from 'api/omgxWatcherAxios'
import addressAxiosInstance from 'api/addressAxios'
import addressOMGXAxiosInstance from 'api/addressOMGXAxios'
import coinGeckoAxiosInstance from 'api/coinGeckoAxios'

//All the current addresses for fallback purposes only
//These may or may not be present
//Generally, the wallet will get these from the two HTTP deployment servers
const localAddresses = require(`../deployment/local/addresses.json`)
const rinkebyAddresses = require(`../deployment/rinkeby/addresses.json`)

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

    // chain ID
    this.chainID = null

    // gas
    this.L1GasLimit = 9999999
    this.L2GasLimit = 10000000
  }

  async enableBrowserWallet() {
    console.log('NS: enableBrowserWallet()')
    try {
      // connect to the wallet
      await window.ethereum.enable()
      this.provider = new ethers.providers.Web3Provider(window.ethereum)
      await window.ethereum.request({
        method: 'eth_requestAccounts',
      })
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
      // window.location.href = `?change_chain`
    })
  }

  async mintAndSendNFT(receiverAddress, contractAddress, ownerName, tokenURI, type) {
    
    try {

      let meta = ownerName + '#' + Date.now().toString() + '#' + tokenURI + '#' + type 

      console.log('meta:', meta)
      console.log('receiverAddress:', receiverAddress)

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

      const addresses = await registry.lookupAddress(
        receiverAddress
      )

      console.log("the receiver's NFT contract addresses:", addresses)

      const alreadyHaveAddresss = addresses.find((str) => str.toLowerCase() === contractAddress.toLowerCase())

      if (alreadyHaveAddresss) {
        //we are done - no need to double register addresss
        console.log('done - no need to double register address')
      } else {
        //register address for the recipiant
        let reg = await registry.connect(
          this.provider.getSigner()
        ).registerAddress(
          receiverAddress, 
          contractAddress
        )
        console.log("Reg:",reg)
        console.log(`Contract registered in recipient's wallet`)
      }

      return true
    } catch (error) {
      console.log(error)
      return false
    }
  }

  async addNFTFactoryNS( address ) {

    let contract = new ethers.Contract(
      address,
      L2ERC721Json.abi,
      this.L2Provider
    )

    let haveRights = false

    try {

      let owner = await contract.owner()
      owner = owner.toLowerCase()

      if ( this.account.toLowerCase() === owner ) 
        haveRights = true
      
      let nftName = await contract.name()
      let nftSymbol = await contract.symbol()
      let genesis = await contract.getGenesis()

      let genesisContractAddress = genesis[0]

      if( genesisContractAddress === '0x0000000000000000000000000000000000000000') {
        //special case - this is the default NFT factory....
        genesisContractAddress = this.ERC721Address
      }
      
      //wallet address of whomever owns the parent
      const genesisContract = new ethers.Contract(
        genesisContractAddress,
        L2ERC721Json.abi,
        this.L2Provider
      )
      
      const feeRecipient = await genesisContract.owner()

      addNFTFactory({
        name: nftName,
        symbol: nftSymbol,
        owner,
        layer: 'L2',
        address,
        originAddress: genesis[0],
        originID: genesis[1],
        originChain: genesis[2],
        originFeeRecipient: feeRecipient, 
        haveRights
      })

    } catch (error) {
      console.log("addNFTFactoryNS cache is stale:",error)
    }

  }

  async deployNewNFTContract(
      nftSymbol,
      nftName,
      oriAddress,
      oriID,
      oriChain) 
  {
    
    try {
      
      console.log("Deploying new NFT factory")

      let Factory__L2ERC721 = new ContractFactory(
        L2ERC721Json.abi,
        L2ERC721Json.bytecode,
        this.provider.getSigner()
      )

      let contract = await Factory__L2ERC721.deploy(
        nftSymbol,
        nftName,
        BigNumber.from(String(0)), //starting index for the tokenIDs
        oriAddress,
        oriID,
        oriChain
      )

      await contract.deployTransaction.wait()

      this.addNFTFactoryNS( contract.address )
      
      console.log('New NFT ERC721 deployed to:', contract.address)

      return true
    } catch (error) {
      console.log(error)
      return false
    }

  }

  async initializeAccounts(masterSystemConfig) {

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
        /*these endpoints do not exist yet*/
        // try {
        //   resOMGX = await addressOMGXAxiosInstance('rinkeby').get()
        // }
        // catch (error) {
        //   console.log(error)
        // }

        // try {
        //   resBase = await addressAxiosInstance('rinkeby').get()
        // }
        // catch (error) {
        //   console.log(error)
        // }

        // if ( resOMGX !== null && resBase !== null ) {
        //   addresses = {...resBase.data, ...resOMGX.data }
        // } else {
        addresses = rinkebyAddresses //emergency fallback
        // }

        console.log('Final Rinkeby Addresses:', addresses)
      }

      //at this point, the wallet should be connected
      this.account = await this.provider.getSigner().getAddress()
      console.log('this.account', this.account)
      const network = await this.provider.getNetwork()

      this.chainID = network.chainId
      this.masterSystemConfig = masterSystemConfig

      this.tokenAddresses = addresses.TOKENS

      console.log('NS: masterConfig:', this.masterSystemConfig)
      console.log('NS: this.chainID:', this.chainID)
      console.log('NS: network:', network)

      //there are numerous possible chains we could be on
      //either local, rinkeby etc
      //and then, also, either L1 or L2

      //at this point, we only know whether we want to be on local or rinkeby etc
      if (masterSystemConfig === 'local' && network.chainId === 28) {
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
      } else {
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
      } else {
        this.L1MessengerAddress = addresses.L1MessengerAddress
        console.log(
          'LEGACY: L1MessengerAddress set to:',
          this.L1MessengerAddress
        )
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

      this.L1_TEST_Address = addresses.TOKENS.TEST.L1
      this.L2_TEST_Address = addresses.TOKENS.TEST.L2

      this.L1LPAddress = addresses.L1LiquidityPool
      this.L2LPAddress = addresses.L2LiquidityPool

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
      await addNFTContract(this.ERC721Contract.address)

      //yes, this looks weird, but think before you change it...
      //there may be some in the cache, and this makes sure we get them all, and if not, 
      //we at least have the basic one
      const NFTcontracts = Object.values(await getNFTContracts())
      
      //Add factories based on cached contract addresses 
      //this information is also used for the balance lookup
      for(var i = 0; i < NFTcontracts.length; i++) {
        const address = NFTcontracts[i]
        console.log("Adding NFT contract:",address)
        this.addNFTFactoryNS( address )
      } 

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
    const chainParam = {
      chainId: '0x' + nw.rinkeby.L2.chainId.toString(16),
      chainName: 'OMGX L2',
      rpcUrls: [nw.rinkeby.L2.rpcUrl],
    }

    // connect to the wallet
    this.provider = new ethers.providers.Web3Provider(window.ethereum)
    this.provider.send('wallet_addEthereumChain', [chainParam, this.account])
  }

  async getTransactions() {

    if (this.masterSystemConfig === 'rinkeby') {
      
      let txL1
      let txL2

      const responseL1 = await etherScanInstance(
        this.masterSystemConfig,
        /*this.L1orL2*/ 'L1'
      ).get(`&address=${this.account}`)
      if (responseL1.status === 200) {
        const transactionsL1 = await responseL1.data
        if (transactionsL1.status === '1') {
          //thread in ChainID
          txL1 = transactionsL1.result.map(v => ({...v, chain: 'L1'}))
          //return transactions.result
        }
      }

      const responseL2 = await omgxWatcherAxiosInstance(
        this.masterSystemConfig
      ).post('get.transaction', {
        address: this.account,
        fromRange: 0,
        toRange: 1000,
      })
      if (responseL2.status === 201) {
        txL2 = responseL2.data.map(v => ({...v, chain: 'L2'}))
        const annotated = await this.parseTransaction( [...txL1, ...txL2] )
        return annotated
      }
    }

  }

  /* Where possible, annotate the transactions 
  based on contract addresses */
  async parseTransaction( transactions ) {

    var annotatedTX

    if (this.masterSystemConfig === 'rinkeby') {
      
      annotatedTX = transactions.map(item => {
        
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
          //0x4200000000000000000000000000000000000010
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
    
    } //if rinkeby

    return annotatedTX

  }

  async getExits() {
    //this is NOT SUPPORTED on LOCAL
    if (this.masterSystemConfig === 'rinkeby') {
      const response = await omgxWatcherAxiosInstance(
        this.masterSystemConfig
      ).post('get.transaction', {
        address: this.account,
        fromRange: 0,
        toRange: 100,
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
  }

  async fetchNFTs() {

    /*
      Metacomment on how this is coded: 
      Is it messy? Yes.
      Does it use arrow functions well? No.
      Is it elegant? No.
      Is it hard to maintain and understand? Yes.
      Does it work? Yes.
    */

    //console.log('fetchNFTs')
    
    let NFTfactories = Object.entries(await getNFTFactories())

    const localCache = NFTfactories.map(item => {return item[0].toLowerCase()})

    //check user's blockchain NFT registry
    const registry = new ethers.Contract(this.ERC721RegAddress,L2ERC721RegJson.abi,this.L2Provider)
    const addresses = await registry.lookupAddress(this.account)
    //console.log("Blockchain NFT wallet:", addresses)
    
    //make sure we have all the factories relevant to this user
    for(let i = 0; i < addresses.length; i++) {
      const newAddress = addresses[i]
      var inCache = (localCache.indexOf(newAddress.toLowerCase()) > -1)
      if(!inCache) {
        console.log("Found a new NFT contract:",newAddress)
        await addNFTContract( newAddress )
        this.addNFTFactoryNS( newAddress )
      }
    }
    
    //How many NFTs do you have right now?
    let numberOfNFTS = 0

    //need to call this again because it might have changed
    NFTfactories = Object.entries(await getNFTFactories())
    
    for(let i = 0; i < NFTfactories.length; i++) {
      
      let contract = new ethers.Contract(
        NFTfactories[i][1].address,
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

      for(let i = 0; i < NFTfactories.length; i++) {

        const address = NFTfactories[i][1].address
        
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
        let genesis = await contract.getGenesis()

        let genesisContractAddress = genesis[0]

        if( genesisContractAddress === '0x0000000000000000000000000000000000000000') {
          //special case - this is just the default NFT factory....
          genesisContractAddress = this.ERC721Address
        }
        
        //console.log("NFT genesis contract:", genesis)
        //wallet address of whomever owns the parent
        const genesisContract = new ethers.Contract(
          genesisContractAddress,
          L2ERC721Json.abi,
          this.L2Provider
        )
        //console.log("genesisContract:", genesisContract)
        
        const feeRecipient = await genesisContract.owner()
        //console.log("NFT feeRecipient:", feeRecipient)

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
          const time = new Date(parseInt(meta[1]))

          let type = 0          
          //new flavor of NFT has type field
          //default to zero for old NFTs
          if(meta.length === 4) {
            type = parseInt(meta[3])
          }

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

          const NFT = {
            UUID,
            owner: meta[0],
            mintedTime,
            url: meta[2],
            tokenID,
            name: nftName,
            symbol: nftSymbol,
            address,
            originAddress: genesis[0],
            originID: genesis[1],
            originChain: genesis[2],
            originFeeRecipient: feeRecipient,
            type
          }

          await addNFT( NFT)

        }

      }

    }
    
  }

  async getBalances() {

    try {
      
      // Always check ETH and oETH
      const layer1Balance = await this.L1Provider.getBalance(this.account)
      //console.log('ETH balance on L1:', layer1Balance.toString())

      const layer2Balance = await this.L2Provider.getBalance(this.account)
      //console.log("oETH balance on L2:", layer2Balance.toString())

      // Add the token to our master list, if we do not have it yet
      // if the token is already in the list, then this function does nothing
      // but if a new token shows up, then it will get added
      Object.keys(this.tokenAddresses).forEach((token, i) => {
        getToken(this.tokenAddresses[token].L1)
      })

      //const ethToken = await getToken(this.L1_ETH_Address)
      //console.log('Checking ethToken:', ethToken)

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

      for (let i = 0; i < tA.length; i++) {
        
        let token = tA[i]

        //ETH is special - will break things,
        //so, continue
        if (token.addressL1 === this.L1_ETH_Address) continue

        const tokenC = new ethers.Contract(
          token.addressL1,
          L1ERC20Json.abi,
          this.provider.getSigner()
        )
        const balance = await tokenC
          .connect(this.L1Provider)
          .balanceOf(this.account)

        //Value is too small to show
        if (Number(balance.toString()) < 0.1) continue

        layer1Balances.push({
          currency: token.currency,
          address: token.addressL1,
          addressL2: token.addressL2,
          symbol: token.symbolL1,
          decimals: token.decimals,
          balance: new BN(balance.toString())
        })
      }

      for (let i = 0; i < tA.length; i++) {

        let token = tA[i]

        //ETH is special - will break things,
        //so, continue
        if (token.addressL2 === this.L2_ETH_Address) continue

        const tokenC = new ethers.Contract(
          token.addressL2,
          L2ERC20Json.abi,
          this.provider.getSigner()
        )
        const balance = await tokenC
          .connect(this.L2Provider)
          .balanceOf(this.account)

        //Value is too small to show
        if (Number(balance.toString()) < 0.1) continue

        layer2Balances.push({
          currency: token.currency,
          address: token.addressL2,
          addressL1: token.addressL1,
          symbol: token.symbolL2,
          decimals: token.decimals,
          balance: new BN(balance.toString())
        })
      }

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
  depositETHL2 = async (value = '1') => {
    try {
      const depositTxStatus = await this.L1StandardBridgeContract.depositETH(
        this.L2GasLimit,
        utils.formatBytes32String(new Date().getTime().toString()),
        { value: parseEther(value) }
      )
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
    } catch {
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

      // let allowance_BN = await ERC20Contract.allowance(
      //   this.account,
      //   this.L1LPAddress
      // )

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

  //Standard 7 day exit from OMGX
  //updated
  async exitOMGX(currencyAddress, value) {
    
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
    const feeRate = await L2LPContract.totalFeeRate()
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
    
    const tokenAddressList = [
      this.L1_ETH_Address, 
      this.L1_TEST_Address
    ]

    const L1LPContract = new ethers.Contract(
      this.L1LPAddress,
      L1LPJson.abi,
      this.L1Provider
    )

    const poolInfo = {}
    const userInfo = {}

    for (let tokenAddress of tokenAddressList) {
      
      let tokenBalance
      let tokenSymbol
      let tokenName
      
      console.log(tokenAddress)
      console.log(this.L1_ETH_Address)

      if (tokenAddress === this.L1_ETH_Address) {
        tokenBalance = await this.L1Provider.getBalance(this.L1LPAddress)
        console.log('Match')
        tokenSymbol = 'ETH'
        tokenName = 'Ethereum'
      } else if (tokenAddress === this.L1_TEST_Address) {
        tokenBalance = await this.L1_TEST_Contract.connect(
          this.L1Provider
        ).balanceOf(this.L1LPAddress)
        tokenSymbol = await this.L2_TEST_Contract.connect(
          this.L2Provider
        ).symbol()
        tokenName = await this.L2_TEST_Contract.connect(
          this.L2Provider
        ).name()
      }
      // Add new LPs here
      // else if (tokenAddress === __________) {
      //   tokenBalance = await this.___________.connect(this.L1Provider).balanceOf(this.L1LPAddress)
      // }
      else {
        console.log('No liquidity pool for this token:', tokenAddress)
      }

      const [poolTokenInfo, userTokenInfo] = await Promise.all([
        L1LPContract.poolInfo(tokenAddress),
        L1LPContract.userInfo(tokenAddress, this.account),
      ])

      poolInfo[tokenAddress] = {
        symbol: tokenSymbol,
        name: tokenName,
        l1TokenAddress: poolTokenInfo.l1TokenAddress,
        l2TokenAddress: poolTokenInfo.l2TokenAddress,
        accUserReward: poolTokenInfo.accUserReward.toString(),
        accUserRewardPerShare: poolTokenInfo.accUserRewardPerShare.toString(),
        userDepositAmount: poolTokenInfo.userDepositAmount.toString(),
        startTime: poolTokenInfo.startTime.toString(),
        APR:
          Number(poolTokenInfo.userDepositAmount.toString()) === 0
            ? 0
            : accMul(
                accDiv(
                  accDiv(
                    poolTokenInfo.accUserReward,
                    poolTokenInfo.userDepositAmount
                  ),
                  accDiv(
                    new Date().getTime() -
                      Number(poolTokenInfo.startTime) * 1000,
                    365 * 24 * 60 * 60 * 1000
                  )
                ),
                100
              ), // ( accUserReward - userDepositAmount ) / timeDuration
        tokenBalance: tokenBalance.toString()
      }
      userInfo[tokenAddress] = {
        l1TokenAddress: tokenAddress,
        amount: userTokenInfo.amount.toString(),
        pendingReward: userTokenInfo.pendingReward.toString(),
        rewardDebt: userTokenInfo.rewardDebt.toString()
      }
    }
    return { poolInfo, userInfo }
  }

  async getL2LPInfo() {
    
    const tokenAddressList = [
      this.L2_ETH_Address, 
      this.L2_TEST_Address
    ]

    const L2LPContract = new ethers.Contract(
      this.L2LPAddress,
      L2LPJson.abi,
      this.L2Provider
    )

    const poolInfo = {}
    const userInfo = {}

    for (let tokenAddress of tokenAddressList) {
      
      let tokenBalance
      let tokenSymbol
      let tokenName

      if (tokenAddress === this.L2_ETH_Address) {
        tokenBalance = await this.L2Provider.getBalance(this.L2LPAddress)
        tokenSymbol = 'oETH'
        tokenName = 'Ethereum'
      } else if (tokenAddress === this.L2_TEST_Address) {
        tokenBalance = await this.L2_TEST_Contract.connect(
          this.L2Provider
        ).balanceOf(this.L2LPAddress)
        tokenSymbol = await this.L2_TEST_Contract.connect(
          this.L2Provider
        ).symbol()
         tokenName = await this.L2_TEST_Contract.connect(
          this.L2Provider
        ).name()
      }
      // Add new LPs here
      // else if (tokenAddress === __________) {
      //   tokenBalance = await this.___________.connect(this.L2Provider).balanceOf(this.L2LPAddress)
      // }
      else {
        console.log('No liquidity pool for this token:', tokenAddress)
      }

      const [poolTokenInfo, userTokenInfo] = await Promise.all([
        L2LPContract.poolInfo(tokenAddress),
        L2LPContract.userInfo(tokenAddress, this.account),
      ])

      poolInfo[tokenAddress] = {
        symbol: tokenSymbol,
        name: tokenName,
        l1TokenAddress: poolTokenInfo.l1TokenAddress,
        l2TokenAddress: poolTokenInfo.l2TokenAddress,
        accUserReward: poolTokenInfo.accUserReward.toString(),
        accUserRewardPerShare: poolTokenInfo.accUserRewardPerShare.toString(),
        userDepositAmount: poolTokenInfo.userDepositAmount.toString(),
        startTime: poolTokenInfo.startTime.toString(),
        APR:
          Number(poolTokenInfo.userDepositAmount.toString()) === 0
            ? 0
            : accMul(
                accDiv(
                  accDiv(
                    poolTokenInfo.accUserReward,
                    poolTokenInfo.userDepositAmount
                  ),
                  accDiv(
                    new Date().getTime() -
                      Number(poolTokenInfo.startTime) * 1000,
                    365 * 24 * 60 * 60 * 1000
                  )
                ),
                100
              ), // ( accUserReward - userDepositAmount ) / timeDuration
        tokenBalance: tokenBalance.toString()
      }
      userInfo[tokenAddress] = {
        l2TokenAddress: tokenAddress,
        amount: userTokenInfo.amount.toString(),
        pendingReward: userTokenInfo.pendingReward.toString(),
        rewardDebt: userTokenInfo.rewardDebt.toString()
      }
    }

    return { poolInfo, userInfo }
  }

  /***********************************************/
  /*****            Add Liquidity            *****/
  /***********************************************/
  async addLiquidity(currency, value, L1orL2Pool) {
    
    const decimals = 18 //should not assume?
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
  async withdrawLiquidity(currency, value, L1orL2Pool) {
    const decimals = 18 //bit dangerous?
    let withdrawAmount = powAmount(value, decimals)
    try {
      // Deposit
      const withdrawLiquidityTX = await await (L1orL2Pool === 'L1LP'
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
  /***** SWAP ON to OMGX by depositing funds to the L1LP *****/
  /***********************************************************/
  async depositL1LP(currency, value) {
    
    const decimals = 18 //bit dangerous?
    let depositAmount = powAmount(value, decimals)

    const depositTX = await this.L1LPContract.clientDepositL1(
      depositAmount.toString(),
      currency,
      currency === this.L1_ETH_Address ? { value: depositAmount } : {}
    )
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
  async L1LPBalance(tokenAddress) {
    let balance
    const decimals = 18
    let tokenAddressLC = tokenAddress.toLowerCase()
    if (
      tokenAddressLC === this.L2_ETH_Address ||
      tokenAddressLC === this.L1_ETH_Address
    ) {
      balance = await this.L1Provider.getBalance(this.L1LPAddress)
    } else if (
      tokenAddressLC === this.L2_TEST_Address.toLowerCase() ||
      tokenAddressLC === this.L1_TEST_Address.toLowerCase()
    ) {
      balance = await this.L1_TEST_Contract.connect(this.L1Provider).balanceOf(
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
  async L2LPBalance(tokenAddress) {
    let balance
    const decimals = 18
    let tokenAddressLC = tokenAddress.toLowerCase()
    if (
      tokenAddressLC === this.L2_ETH_Address ||
      tokenAddressLC === this.L1_ETH_Address
    ) {
      //We are dealing with ETH
      balance = await this.L2_ETH_Contract.connect(this.L2Provider).balanceOf(
        this.L2LPAddress
      )
    } else if (
      tokenAddressLC === this.L2_TEST_Address.toLowerCase() ||
      tokenAddressLC === this.L1_TEST_Address.toLowerCase()
    ) {
      //we are dealing with TEST
      balance = await this.L2_TEST_Contract.connect(this.L2Provider).balanceOf(
        this.L2LPAddress
      )
    }
    
    console.log("L2LPBalance:",typeof(balance))

    if(typeof(balance) === 'undefined') {
      return logAmount('0', decimals)
    } else {
      return logAmount(balance.toString(), decimals)
    }
    
  }

  /**************************************************************/
  /***** SWAP OFF from OMGX by depositing funds to the L2LP *****/
  /**************************************************************/
  async depositL2LP(currencyAddress, depositAmount_string) {

    const L2ERC20Contract = new ethers.Contract(
      currencyAddress,
      L2ERC20Json.abi,
      this.provider.getSigner()
    )

    let allowance_BN = await L2ERC20Contract.allowance(
      this.account,
      this.L2LPAddress
    )

    //const decimals = await L2ERC20Contract.decimals() 

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

  // async getPriorityTokens() {
  //   try {

  //     return priorityTokens.map((token) => {

  //       let L1 = ''
  //       let L2 = ''

  //       if (token.symbol === 'ETH') {
  //         L1 = this.L1_ETH_Address
  //         L2 = this.L2_ETH_Address
  //       } else {
  //         L1 = this.tokenAddresses[token.symbol].L1
  //         L2 = this.tokenAddresses[token.symbol].L2
  //       }

  //       return {
  //         symbol: token.symbol,
  //         icon: token.icon,
  //         name: token.name,
  //         L1,
  //         L2,
  //       }

  //     })

  //   } catch (error) {
  //     return error
  //   }
  // }

  // async getSwapTokens() {
  //   try {

  //     return swapTokens.map((token) => {

  //       let L1 = ''
  //       let L2 = ''

  //       if (token.symbol === 'ETH') {
  //         L1 = this.L1_ETH_Address
  //         L2 = this.L2_ETH_Address
  //       } else {
  //         L1 = this.tokenAddresses[token.symbol].L1
  //         L2 = this.tokenAddresses[token.symbol].L2
  //       }

  //       return {
  //         symbol: token.symbol,
  //         icon: token.icon,
  //         name: token.name,
  //         L1,
  //         L2,
  //       }
  //     })

  //   } catch (error) {
  //     return error
  //   }
  // }

  // async getDropdownTokens() {
  //   try {
      
  //     return dropdownTokens.map((token) => {

  //       let L1 = ''
  //       let L2 = ''

  //       if (token.symbol === 'ETH') {
  //         L1 = this.L1_ETH_Address
  //         L2 = this.L2_ETH_Address
  //       } else {
  //         L1 = this.tokenAddresses[token.symbol].L1
  //         L2 = this.tokenAddresses[token.symbol].L2
  //       }

  //       return {
  //         symbol: token.symbol,
  //         icon: token.icon,
  //         name: token.name,
  //         L1,
  //         L2,
  //       }
  //     })

  //   } catch (error) {
  //     return error
  //   }
  // }

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
}

const networkService = new NetworkService()
export default networkService
