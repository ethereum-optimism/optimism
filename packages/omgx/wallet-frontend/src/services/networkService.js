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

import { hexlify } from '@ethersproject/bytes'
import { parseUnits, parseEther } from '@ethersproject/units'
import { Watcher } from '@eth-optimism/watcher'
import { ethers, BigNumber, utils } from 'ethers'

import { orderBy } from 'lodash'
import BN from 'bn.js'

import { getToken } from 'actions/tokenAction'
import { getNFTs, addNFT } from 'actions/nftAction'
import { setMinter } from 'actions/setupAction'

import { openAlert, openError } from 'actions/uiAction'
import { WebWalletError } from 'services/errorService'

//Base contracts
import L1StandardBridgeJson from '../deployment/artifacts/optimistic-ethereum/OVM/bridge/tokens/OVM_L1StandardBridge.sol/OVM_L1StandardBridge.json'
import L2StandardBridgeJson from '../deployment/artifacts-ovm/optimistic-ethereum/OVM/bridge/tokens/OVM_L2StandardBridge.sol/OVM_L2StandardBridge.json'
import L2ERC20Json          from '../deployment/artifacts-ovm/optimistic-ethereum/libraries/standards/L2StandardERC20.sol/L2StandardERC20.json'

//OMGX L1 Contracts
import L1LPJson             from '../deployment/artifacts/contracts/LP/L1LiquidityPool.sol/L1LiquidityPool.json'
import L1ERC20Json          from '../deployment/artifacts/contracts/L1ERC20.sol/L1ERC20.json'

//OMGX L2 Contracts
import L2LPJson             from '../deployment/artifacts-ovm/contracts/LP/L2LiquidityPool.sol/L2LiquidityPool.json'
import ERC721Json           from '../deployment/artifacts-ovm/contracts/ERC721Mock.sol/ERC721Mock.json'
import L2TokenPoolJson      from '../deployment/artifacts-ovm/contracts/TokenPool.sol/TokenPool.json'
import AtomicSwapJson       from '../deployment/artifacts-ovm/contracts/AtomicSwap.sol/AtomicSwap.json'

import { powAmount, logAmount } from 'util/amountConvert'
import { accDiv, accMul } from 'util/calculation'

import { getAllNetworks } from 'util/masterConfig'

import etherScanInstance from 'api/etherScanAxios'
import omgxWatcherAxiosInstance from 'api/omgxWatcherAxios'
import addressAxiosInstance from 'api/addressAxios'
import addressOMGXAxiosInstance from 'api/addressOMGXAxios'

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

    this.L2ETHContract = null

    this.L1ERC20Contract = null
    this.L2ERC20Contract = null

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

    this.ERC721Address = null
    this.L1ERC20Address = null
    this.L2ERC20Address = null
    this.L1MessengerAddress = null
    this.L1LPAddress = null
    this.L2LPAddress = null

    this.L2StandardBridgeAddress = '0x4200000000000000000000000000000000000010'
    this.L1ETHAddress = '0x0000000000000000000000000000000000000000'
    this.L2ETHAddress = '0x4200000000000000000000000000000000000006'
    this.L2MessengerAddress = '0x4200000000000000000000000000000000000007'

    // chain ID
    this.chainID = null

    // gas
    this.L1GasLimit = 9999999
    this.L2GasLimit = 9999999
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

  async mintAndSendNFT(receiverAddress, ownerName, tokenURI) {
    try {
      let meta = ownerName + '#' + Date.now().toString() + '#' + tokenURI

      console.log('meta:', meta)
      console.log('receiverAddress:', receiverAddress)

      let nft = await this.ERC721Contract.connect(
        this.provider.getSigner()
      ).mintNFT(receiverAddress, meta, { gasPrice: 0 })

      await nft.wait()
      console.log('New ERC721:', nft)
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
        }
        catch (error) {
          console.log(error)
        }

        try {
          resBase = await addressAxiosInstance('local').get()
        }
        catch (error) {
          console.log(error)
        }

        if ( resOMGX !== null && resBase !== null ) {
          addresses = {...resBase.data, ...resOMGX.data }
        } else {
          addresses = localAddresses //emergency fallback
        }

        console.log("Final Local Addresses:",addresses)
      } 
      else if (masterSystemConfig === 'rinkeby') {
        
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

        console.log("Final Rinkeby Addresses:",addresses)
      } 

      //at this point, the wallet should be connected
      this.account = await this.provider.getSigner().getAddress()
      console.log('this.account', this.account)
      const network = await this.provider.getNetwork()

      this.chainID = network.chainId
      this.masterSystemConfig = masterSystemConfig

      console.log('NS: masterConfig:', this.masterSystemConfig)
      console.log('NS: this.chainID:', this.chainID)

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
/*
{
"AddressManager": "0x5FbDB2315678afecb367f032d93F642f64180aa3",
"AtomicSwap": "0x2279B7A0a67DB372996a5FaB50D91eAA73d2eBe6",
"ERC721": "0xA51c1fc2f0D1a1b8494Ed1FE312d7C3a78Ed91C0",
"L1ERC20": "0x4c5859f0F772848b2D91F1D83E2Fe57935348029",
"L1LiquidityPool": "0x5eb3Bc0a489C5A8288765d2336659EbCA68FCd00",
"L1Message": "0x1291Be112d480055DaFd8a610b7d1e203891C274",
"L2ERC20": "0xCf7Ed3AccA5a467e9e704C703E8D87F634fB0Fc9",
"L2LiquidityPool": "0x5FbDB2315678afecb367f032d93F642f64180aa3",
"L2Message": "0x0165878A594ca255338adfa4d48449f69242Eb8F",
"L2TokenPool": "0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9",
"OVM_L1CrossDomainMessenger": "0x68B1D87F95878fE05B998F19b66F4baba5De1aed",
"OVM_L1CrossDomainMessengerFast": "0x0E801D84Fa97b50751Dbf25036d067dCf18858bF",
"Proxy__OVM_L1CrossDomainMessenger": "0x59b670e9fA9D0A427751Af201D676719a970857b",
"Proxy__OVM_L1StandardBridge": "0x851356ae760d987E095750cCeb3bC6014560891C"
}
*/

/*
AtomicSwap: "0xfCc9525fDDbafbD3393821F7dAf96F68Bba93294"
L1ERC20: "0xaB1e3377dEED7811beADf4b5773B59fB267089fb"
L1FastMessengerAddress: "0xF296F4ca6A5725F55EdF1C67F80204871E65F87d"
L1LiquidityPool: "0x2C12649A5A4FC61F146E0a3409f3e4c7FbeD15Dc"
L1Message: "0x1E7C2Ed00FaaFeD62afC9DD630ACB8C8c6C16D52"
L1MessengerAddress: "0xF10EEfC14eB5b7885Ea9F7A631a21c7a82cf5D76"
L1StandardBridge: "0xDe085C82536A06b40D20654c2AbA342F2abD7077"
L2ERC20: "0x0e52DEfc53ec6dCc52d630af949a9b6313455aDF"
L2ERC721: "0xB08e122b98889321040AB251316887E59ee1d3Df"
L2LiquidityPool: "0xEd3417AE90fA0BfE15Fab8b949a9a459F9f4ef26"
L2Message: "0x8165E68dD175B4D6e913D1DF5aF456d8C04cA01e"
L2StandardBridge: "0x4200000000000000000000000000000000000010"
L2TokenPool: "0x82B178EE692572e21D73d5F1ebC1c7c438Fc52DD"
*/

      // addresses

      //this.L1MessengerAddress = addresses.Proxy__OVM_L1CrossDomainMessenger
      //backwards compat
      if( addresses.hasOwnProperty('Proxy__OVM_L1CrossDomainMessenger') )
        this.L1MessengerAddress = addresses.Proxy__OVM_L1CrossDomainMessenger
      else 
        this.L1MessengerAddress = addresses.L1MessengerAddress

      //this.L1FastMessengerAddress = addresses.OVM_L1CrossDomainMessengerFast
      //backwards compat
      if( addresses.hasOwnProperty('OVM_L1CrossDomainMessengerFast') )
        this.L1FastMessengerAddress = addresses.OVM_L1CrossDomainMessengerFast
      else 
        this.L1FastMessengerAddress = addresses.L1FastMessengerAddress

      //backwards compat
      if( addresses.hasOwnProperty('Proxy__OVM_L1StandardBridge') )
        this.L1StandardBridgeAddress = addresses.Proxy__OVM_L1StandardBridge
      else 
        this.L1StandardBridgeAddress = addresses.L1StandardBridge

      this.L1ERC20Address = addresses.L1ERC20
      this.L2ERC20Address = addresses.L2ERC20

      this.L1LPAddress = addresses.L1LiquidityPool
      this.L2LPAddress = addresses.L2LiquidityPool

      //backwards compat
      if( addresses.hasOwnProperty('L2ERC721') )
        this.ERC721Address = addresses.L2ERC721
      else 
        this.ERC721Address = addresses.ERC721

      this.L2TokenPoolAddress = addresses.L2TokenPool
      this.AtomicSwapAddress = addresses.AtomicSwap

      this.L1StandardBridgeContract = new ethers.Contract(
        this.L1StandardBridgeAddress,
        L1StandardBridgeJson.abi,
        this.provider.getSigner()
      )

      this.L2StandardBridgeContract = new ethers.Contract(
        this.L2StandardBridgeAddress,
        L2StandardBridgeJson.abi,
        this.provider.getSigner()
      )

      this.L2ETHContract = new ethers.Contract(
        this.L2ETHAddress,
        L2ERC20Json.abi,
        this.provider.getSigner()
      )

      this.L1ERC20Contract = new ethers.Contract(
        this.L1ERC20Address,
        L1ERC20Json.abi,
        this.provider.getSigner()
      )

      this.L2ERC20Contract = new ethers.Contract(
        this.L2ERC20Address,
        L2ERC20Json.abi,
        this.provider.getSigner()
      )

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
        ERC721Json.abi,
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

      const ERC721Owner = await this.ERC721Contract.owner()

      if (this.account === ERC721Owner) {
        //console.log("Great, you are the NFT owner")
        setMinter(true)
      } else {
        //console.log("Sorry, not the NFT owner")
        setMinter(false)
      }
      
      //Fire up the new watcher
      //const addressManager = getAddressManager(bobl1Wallet)
      //const watcher = await initWatcher(L1Provider, this.L2Provider, addressManager)

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
    
    //rinkeby L1
    if (this.masterSystemConfig === 'rinkeby' && this.chainID === 4) { 
      const response = await etherScanInstance(this.masterSystemConfig, this.L1orL2).get(`&address=${this.account}`)
      if (response.status === 200) {
        const transactions = await response.data
        if (transactions.status === '1') {
          return transactions.result
        }
      }
    }
    //rinkeby L2
    if (this.masterSystemConfig === 'rinkeby' && this.chainID === 28) {
      const response = await omgxWatcherAxiosInstance(this.masterSystemConfig).post('get.transaction', {
        address: this.account,
        fromRange: 0,
        toRange: 100,
      })
      if (response.status === 201) {
        return response.data
      }
    }
  }

  async getExits() {

    //this is NOT SUPPORTED on LOCAL
    
    if (this.masterSystemConfig === 'rinkeby') {
      const response = await omgxWatcherAxiosInstance(this.masterSystemConfig).post('get.transaction', {
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
              this.L2ERC20Address.toLowerCase(),
              this.L2ETHAddress.toLowerCase(),
            ].includes(i.to ? i.to.toLowerCase() : null) && i.crossDomainMessage
        )
        return { exited: filteredTransactions }
      }
    }
  }

  async getBalances() {
    try {
      const rootChainBalance = await this.L1Provider.getBalance(this.account)
      const ERC20L1Balance = await this.L1ERC20Contract.connect(
        this.L1Provider
      ).balanceOf(this.account)

      const childChainBalance = await this.L2Provider.getBalance(this.account)
      const ERC20L2Balance = await this.L2ERC20Contract.connect(
        this.L2Provider
      ).balanceOf(this.account)

      // //how many NFTs do I own?
      const ERC721L2Balance = await this.ERC721Contract.connect(
        this.L2Provider
      ).balanceOf(this.account)
      // console.log("ERC721L2Balance",ERC721L2Balance)
      // console.log("this.account",this.account)
      // console.log(this.ERC721Contract)

      //let see if we already know about them
      const myNFTS = getNFTs()
      const numberOfNFTS = Object.keys(myNFTS).length

      if (Number(ERC721L2Balance.toString()) !== numberOfNFTS) {
        //oh - something just changed - either got one, or sent one
        console.log('NFT change detected!')

        //we need to do something
        //get the first one

        let tokenID = null
        let nftTokenIDs = null
        let nftMeta = null
        let meta = null

        //always the same, no need to have in the loop
        let nftName = await this.ERC721Contract.getName()
        let nftSymbol = await this.ERC721Contract.getSymbol()

        for (var i = 0; i < Number(ERC721L2Balance.toString()); i++) {
          tokenID = BigNumber.from(i)
          nftTokenIDs = await this.ERC721Contract.tokenOfOwnerByIndex(
            this.account,
            tokenID
          )
          nftMeta = await this.ERC721Contract.getTokenURI(tokenID)
          meta = nftMeta.split('#')

          const time = new Date(parseInt(meta[1]))

          addNFT({
            UUID:
              this.ERC721Address.substring(1, 6) +
              '_' +
              nftTokenIDs.toString() +
              '_' +
              this.account.substring(1, 6),
            owner: meta[0],
            mintedTime: String(
              time.toLocaleString('en-US', {
                day: '2-digit',
                month: '2-digit',
                year: 'numeric',
                hour: 'numeric',
                minute: 'numeric',
                hour12: true,
              })
            ),
            url: meta[2],
            tokenID: tokenID,
            name: nftName,
            symbol: nftSymbol,
          })
        }
      } else {
        // console.log("No NFT changes")
        //all set - do nothing
      }

      const ethToken = await getToken(this.L1ETHAddress)
      let testToken = null

      //For testing - we always provide a test token
      if (this.L1orL2 === 'L1') {
        testToken = await getToken(this.L1ERC20Address)
      } else {
        testToken = await getToken(this.L2ERC20Address)
      }

      const rootchainEthBalance = [
        {
          ...ethToken,
          amount: new BN(rootChainBalance.toString()),
        },
        {
          ...testToken,
          currency: this.L1ETHAddress,
          amount: new BN(ERC20L1Balance.toString()),
        },
      ]

      const childchainEthBalance = [
        {
          ...ethToken,
          currency: this.L2ETHAddress,
          symbol: 'oETH',
          amount: new BN(childChainBalance.toString()),
        },
        {
          ...testToken,
          currency: this.L2ERC20Address,
          amount: new BN(ERC20L2Balance.toString()),
        },
      ]

      return {
        rootchain: orderBy(rootchainEthBalance, (i) => i.currency),
        childchain: orderBy(childchainEthBalance, (i) => i.currency),
      }
    } catch (error) {
      throw new WebWalletError({
        originalError: error,
        reportToSentry: false,
        reportToUi: false,
      })
    }
  }

  depositETHL1 = () => async (dispatch) => {
    //for this to work, we have to be on the L1
    //otherwise makes no sense
    if (this.L1orL2 !== 'L1') return

    try {
      //const L1ProviderRPC = new JsonRpcProvider(l1Network.rpcUrl);
      const signer = this.L1Provider.getSigner()

      // Send 1 ETH
      const txOption = {
        to: this.account,
        value: parseEther('1'),
        gasPrice: parseUnits('4.1', 'gwei'),
        gasLimit: hexlify(120000),
      }

      const tx = await signer.sendTransaction(txOption)
      await tx.wait()

      console.log(tx)

      dispatch(openAlert('Deposited ETH to L1'))
    } catch (error) {
      dispatch(openError('Failed to deposit ETH to L1'))
    }
  }

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

      this.getBalances()

      return l2Receipt
    } catch {
      return false
    }
  }

  async transfer(address, value, currency) {
    try {
      const tx = await this.L2ERC20Contract.attach(currency).transfer(
        address,
        parseEther(value.toString()),
        { gasPrice: 0 }
      )
      await tx.wait()
      return tx
    } catch (error) {
      console.log(error)
    }
  }

  confirmLayer = (layerToConfirm) => async (dispatch) => {
    if (layerToConfirm === this.L1orL2) {
      return true
    } else {
      return false
    }
  }

  async getAllTransactions() {
    let transactionHistory = {}
    const latest = await this.L2Provider.eth.getBlockNumber()
    const blockNumbers = Array.from(Array(latest).keys())

    for (let blockNumber of blockNumbers) {
      const blockData = await this.L2Provider.eth.getBlock(blockNumber)
      const transactionsArray = blockData.transactions
      if (transactionsArray.length === 0) {
        transactionHistory.push({
          /*ToDo*/
        })
      }
    }
  }

  async checkAllowance(
    currency,
    targetContract = this.L1StandardBridgeAddress
  ) {
    try {
      const ERC20Contract = new ethers.Contract(
        currency,
        L1ERC20Json.abi,
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
        customErrorMessage: 'Could not check deposit allowance for ERC20.',
        reportToSentry: false,
        reportToUi: true,
      })
    }
  }

  async approveErc20(
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

      const approveStatus = await ERC20Contract.approve(
        approveContractAddress,
        value,
        this.L1orL2 === 'L1' ? {} : { gasPrice: 0 }
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
        customErrorMessage: 'Could not reset approval allowance for ERC20.',
        reportToSentry: false,
        reportToUi: true,
      })
    }
  }

  async depositErc20(value, currency, gasPrice) {
    try {
      const L1ERC20Contract = this.L1ERC20Contract.attach(currency)
      const allowance = await L1ERC20Contract.allowance(
        this.account,
        this.L1StandardBridgeAddress
      )

      console.log({ allowance: allowance.toString(), value })

      const depositTxStatus = await this.L1StandardBridgeContract.depositERC20(
        this.L1ERC20Address,
        this.L2ERC20Address,
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

  async exitOMGX(currency, value) {
    const allowance = await this.checkAllowance(
      currency,
      this.L2StandardBridgeAddress
    )
    // need the frontend updates
    if (BigNumber.from(allowance).lt(parseEther(value))) {
      const res = await this.approveErc20(
        parseEther(value),
        currency,
        this.L2StandardBridgeAddress
      )
      if (!res) return false
    }
    const tx = await this.L2StandardBridgeContract.withdraw(
      currency,
      parseEther(value),
      this.L1GasLimit,
      utils.formatBytes32String(new Date().getTime().toString()),
      { gasPrice: 0 }
    )
    await tx.wait()

    const [L2ToL1msgHash] = await this.watcher.getMessageHashesFromL2Tx(tx.hash)
    console.log(' got L2->L1 message hash', L2ToL1msgHash)

    return tx
  }

  /***********************************************/
  /*****                  Fee                *****/
  /***********************************************/
  // Total exist fee
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
  /***********************************************/

  /***********************************************/
  /*****           Pool, User Info           *****/
  /***********************************************/
  async getL1LPInfo() {
    const tokenList = [this.L1ETHAddress, this.L1ERC20Address]
    const L1LPContract = new ethers.Contract(
      this.L1LPAddress,
      L1LPJson.abi,
      this.L1Provider
    )
    const poolInfo = {},
      userInfo = {}
    for (let token of tokenList) {
      const [poolTokenInfo, userTokenInfo, tokenBalance] = await Promise.all([
        L1LPContract.poolInfo(token),
        L1LPContract.userInfo(token, this.account),
        token === this.L1ETHAddress
          ? this.L1Provider.getBalance(this.L1LPAddress)
          : this.L1ERC20Contract.connect(this.L1Provider).balanceOf(
              this.L1LPAddress
            ),
      ])
      poolInfo[token] = {
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
        tokenBalance: tokenBalance.toString(),
      }
      userInfo[token] = {
        l1TokenAddress: token,
        amount: userTokenInfo.amount.toString(),
        pendingReward: userTokenInfo.pendingReward.toString(),
        rewardDebt: userTokenInfo.rewardDebt.toString(),
      }
    }
    return { poolInfo, userInfo }
  }

  async getL2LPInfo() {
    const tokenList = [this.L2ETHAddress, this.L2ERC20Address]
    const L2LPContract = new ethers.Contract(
      this.L2LPAddress,
      L2LPJson.abi,
      this.L2Provider
    )
    const poolInfo = {},
      userInfo = {}
    for (let token of tokenList) {
      const [poolTokenInfo, userTokenInfo, tokenBalance] = await Promise.all([
        L2LPContract.poolInfo(token),
        L2LPContract.userInfo(token, this.account),
        token === this.L2ETHAddress
          ? this.L2Provider.getBalance(this.L2LPAddress)
          : this.L2ERC20Contract.connect(this.L2Provider).balanceOf(
              this.L2LPAddress
            ),
      ])
      poolInfo[token] = {
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
        tokenBalance: tokenBalance.toString(),
      }
      userInfo[token] = {
        l2TokenAddress: token,
        amount: userTokenInfo.amount.toString(),
        pendingReward: userTokenInfo.pendingReward.toString(),
        rewardDebt: userTokenInfo.rewardDebt.toString(),
      }
    }
    return { poolInfo, userInfo }
  }

  /***********************************************/

  /***********************************************/
  /*****            Add Liquidity            *****/
  /***********************************************/
  async addLiquidity(currency, value, L1orL2Pool) {
    const decimals = 18
    let depositAmount = powAmount(value, decimals)

    try {
      // Deposit
      const addLiquidityTX = await (L1orL2Pool === 'L1LP'
        ? this.L1LPContract
        : this.L2LPContract
      ).addLiquidity(
        depositAmount,
        currency,
        // deposit ETH or not
        currency === this.L1ETHAddress
          ? { value: depositAmount }
          : L1orL2Pool === 'L1LP'
          ? {}
          : { gasPrice: 0 }
      )
      await addLiquidityTX.wait()
      return true
    } catch (err) {
      console.log(err)
      return false
    }
  }
  /***********************************************/

  /***********************************************/
  /*****              Get Reward             *****/
  /***********************************************/
  async getReward(currency, value) {
    try {
      // Deposit
      const withdrawRewardTX = await this.L2LPContract.withdrawReward(
        value,
        currency,
        this.account,
        { gasPrice: 0 }
      )
      await withdrawRewardTX.wait()

      return true
    } catch (err) {
      return false
    }
  }
  /***********************************************/

  /***********************************************/
  /*****          Withdraw Liquidity         *****/
  /***********************************************/
  async withdrawLiquidity(currency, value, L1orL2Pool) {
    const decimals = 18
    let withdrawAmount = powAmount(value, decimals)
    try {
      // Deposit
      const withdrawLiquidityTX = await await (L1orL2Pool === 'L1LP'
        ? this.L1LPContract
        : this.L2LPContract
      ).withdrawLiquidity(
        withdrawAmount,
        currency,
        this.account,
        L1orL2Pool === 'L1LP' ? {} : { gasPrice: 0 }
      )
      await withdrawLiquidityTX.wait()
      return true
    } catch (err) {
      return false
    }
  }
  /***********************************************/

  async depositL1LP(currency, value) {
    const decimals = 18
    let depositAmount = powAmount(value, decimals)

    const depositTX = await this.L1LPContract.clientDepositL1(
      depositAmount.toString(),
      currency,
      currency === this.L1ETHAddress ? { value: depositAmount } : {}
    )
    await depositTX.wait()

    // Waiting the response from L2
    const [l1ToL2msgHash] = await this.watcher.getMessageHashesFromL1Tx(
      depositTX.hash
    )
    console.log(' got L1->L2 message hash', l1ToL2msgHash)
    const l2Receipt = await this.watcher.getL2TransactionReceipt(l1ToL2msgHash)
    console.log(' completed Deposit! L2 tx hash:', l2Receipt.transactionHash)

    return l2Receipt
  }

  async L1LPBalance(currency) {
    let balance
    if (currency === this.L2ETHAddress || currency === this.L1ETHAddress) {
      balance = await this.L1Provider.getBalance(this.L1LPAddress)
    } else if (
      currency === this.L2ERC20Address ||
      currency === this.L1ERC20Address
    ) {
      balance = await this.L1ERC20Contract.connect(this.L1Provider).balanceOf(
        this.L1LPAddress
      )
    }
    const decimals = 18
    return logAmount(balance.toString(), decimals)
  }

  async depositL2LP(currency, value) {
    const L2ERC20Contract = this.L2ERC20Contract.attach(currency)

    let allowance = await L2ERC20Contract.allowance(
      this.account,
      this.L2LPAddress
    )
    allowance = new BN(allowance.toString())

    const token = await getToken(currency)
    const decimals = token.decimals
    let depositAmount = powAmount(value, decimals)
    depositAmount = new BN(depositAmount)

    if (depositAmount.gt(allowance)) {
      const approveStatus = await L2ERC20Contract.approve(
        this.L2LPAddress,
        depositAmount.toString(),
        { gasPrice: 0 }
      )
      await approveStatus.wait()
    }

    const depositTX = await this.L2LPContract.clientDepositL2(
      depositAmount.toString(),
      currency,
      { gasPrice: 0 }
    )

    await depositTX.wait()

    // Waiting the response from L1
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

  async L2LPBalance(currency) {
    let balance
    if (currency === this.L1ETHAddress) {
      balance = await this.L2ETHContract.connect(this.L2Provider).balanceOf(
        this.L2LPAddress
      )
    } else if (currency.toLowerCase() === this.L1ERC20Address.toLowerCase()) {
      balance = await this.L2ERC20Contract.balanceOf(this.L2LPAddress)
    } else {
      balance = await this.L2ERC20Contract.connect(this.L2Provider).balanceOf(
        this.L2LPAddress
      )
    }
    const decimals = 18
    return logAmount(balance.toString(), decimals)
  }

  async getTestToken() {
    try {
      const getTokenTX = await this.L2TokenPoolContract.requestToken({
        gasPrice: 0,
      })
      await getTokenTX.wait()
      //console.log(getTokenTX)
      return true
    } catch (error) {
      console.log(error)
      return false
    }
  }
}

const networkService = new NetworkService()
export default networkService
