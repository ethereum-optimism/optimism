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

import { OmgUtil } from '@omisego/omg-js';
import { JsonRpcProvider, Web3Provider } from "@ethersproject/providers";
import { hexlify } from "@ethersproject/bytes";
import { parseUnits, parseEther } from "@ethersproject/units";
import { Watcher } from "@eth-optimism/watcher";
import { ethers } from "ethers";

import Web3Modal from "web3modal";

import '@metamask/legacy-web3'

import { orderBy } from 'lodash';
import BN from 'bn.js';
import Web3 from 'web3';

import { setNetwork } from 'actions/setupAction';
import { getToken } from 'actions/tokenAction';
import { openAlert, openError } from 'actions/uiAction';
import { openNotification } from 'actions/notificationAction';
import { WebWalletError } from 'services/errorService';

import VarnaPoolABI from "contracts/VarnaPool.abi";
import VarnaPoolAddress from "contracts/VarnaPool.address";
import VarnaSwapABI from "contracts/AtomicSwap.abi";
import VarnaSwapAddress from "contracts/AtomicSwap.address";

import L1LPJson from '../deployment/artifacts/contracts/L1LiquidityPool.sol/L1LiquidityPool.json'
import L2LPJson from '../deployment/artifacts-ovm/contracts/L2LiquidityPool.sol/L2LiquidityPool.json'
import L1ERC20Json from '../deployment/artifacts/contracts/ERC20.sol/ERC20.json'
import L2DepositedERC20Json from '../deployment/artifacts-ovm/contracts/L2DepositedERC20.sol/L2DepositedERC20.json'
import L1ERC20GatewayJson from '../deployment/artifacts/contracts/L1ERC20Gateway.sol/L1ERC20Gateway.json'

import { powAmount, logAmount } from 'util/amountConvert';

import { L1ETHGATEWAY, L2DEPOSITEDERC20, NETWORKS, SELECT_NETWORK } from "Settings";

//All the current addresses
const addresses = require('../deployment/addresses.json');
const ERC20Address = addresses.L1ERC20
const L1ERC20GatewayAddress = addresses.L1ERC20Gateway
const L2DepositedERC20Address = addresses.L2DepositedERC20
const L1LPAddress = addresses.L1LiquidityPool
const L2LPAddress = addresses.L2LiquidityPool

const web3Modal = new Web3Modal({
  cacheProvider: true, // optional
  providerOptions: {},
});

const configChain = {
  local: {
    l1Network: NETWORKS['localL1'],
    l2Network: NETWORKS['localL2'],
  },
  rinkeby: {
    l1Network: NETWORKS['rinkeby'],
    l2Network: NETWORKS['rinkebyL2'],
  }
}

const l1Network = configChain[SELECT_NETWORK].l1Network;
const l2Network = configChain[SELECT_NETWORK].l2Network;

const l1Provider = new JsonRpcProvider(l1Network.rpcUrl);
const l2Provider = new JsonRpcProvider(l2Network.rpcUrl);

const l1ETHGatewayAddress = addresses.l1ETHGatewayAddress;
const l2ETHGatewayAddress = l2Network.l2ETHGatewayAddress;

const l1MessengerAddress = addresses.l1MessengerAddress;
const l2MessengerAddress = l2Network.l2MessengerAddress;

const l1ChainID = l1Network.chainId;
const l2ChainID = l2Network.chainId;

const l1NetworkName = l1Network.name;
const l2NetworkName = l2Network.name;

class NetworkService {

  constructor () {
    this.web3 = null;
    // based on MetaMask
    this.web3Provider = null;

    this.l1Web3Provider = null;
    this.l2Web3Provider = null;

    this.provider = null;
    this.OmgUtil = OmgUtil;
    this.environment = null;
    
    this.L1ETHGatewayContract = null;
    this.OVM_L1ERC20Gateway = null;
    
    this.L2ETHGatewayContract = null;
    this.OVM_L2DepositedERC20 = null;

    // hardcoded - for balance
    this.ERC20L1Contract = null;
    this.ERC20L2Contract = null;

    // Varna
    this.VarnaPoolContract = null;
    this.VarnaSwapContract = null;
    this.VarnaPoolAddress = VarnaPoolAddress;
    this.VarnaSwapAddress = VarnaSwapAddress;

    // L1 or L2
    this.selectedNetwork = null;

    // Watcher
    this.watcher = null;

    // LP address
    this.L1LPAddress = L1LPAddress;
    this.L2LPAddress = L2LPAddress;
  }

  async enableBrowserWallet() {
    try {
      // connect to the wallet
      this.provider = await web3Modal.connect();
      // can't get rid of it at this moment, there are 
      // other functions that use this 
      this.web3Provider = new Web3Provider(this.provider);
      
      this.l1Web3Provider = new Web3(new Web3.providers.HttpProvider(l1Network.rpcUrl));
      this.l2Web3Provider = new Web3(new Web3.providers.HttpProvider(l2Network.rpcUrl));

      this.L1ETHGatewayContract = new ethers.Contract(
        l1ETHGatewayAddress, 
        L1ETHGATEWAY, 
        this.web3Provider.getSigner(),
      );

      this.L2ETHGatewayContract = new ethers.Contract(
        l2ETHGatewayAddress,
        L2DEPOSITEDERC20,
        this.web3Provider.getSigner(),
      );

      this.VarnaPoolContract = new ethers.Contract(
        VarnaPoolAddress, 
        VarnaPoolABI, 
        this.web3Provider.getSigner(),
      );
      
      this.VarnaSwapContract = new ethers.Contract(
        VarnaSwapAddress,
        VarnaSwapABI,
        this.web3Provider.getSigner(),
      );

      this.OVM_L1ERC20Gateway = new ethers.Contract(
        L1ERC20GatewayAddress, 
        L1ERC20GatewayJson.abi, 
        this.web3Provider.getSigner(),
      );

      this.OVM_L2DepositedERC20 = new ethers.Contract(
        L2DepositedERC20Address, 
        L2DepositedERC20Json.abi, 
        this.web3Provider.getSigner(),
      );

      // For the balance
      this.ERC20L1Contract = new this.l1Web3Provider.eth.Contract(
        L1ERC20Json.abi,
        ERC20Address,
      );

      this.ERC20L2Contract = new this.l2Web3Provider.eth.Contract(
        L2DepositedERC20Json.abi,
        L2DepositedERC20Address,
      );

      // Liquidity pools
      this.L1LPContract = new ethers.Contract(
        L1LPAddress,
        L1LPJson.abi,
        this.web3Provider.getSigner(),
      );

      this.L2LPContract = new ethers.Contract(
        L2LPAddress,
        L2LPJson.abi,
        this.web3Provider.getSigner(),
      );

      //Fire up the new watcher
      //const addressManager = getAddressManager(bobl1Wallet)
      //const watcher = await initWatcher(l1Provider, l2Provider, addressManager)
      this.watcher = new Watcher({
        l1: {
          provider: l1Provider,
          messengerAddress: l1MessengerAddress
        },
        l2: {
          provider: l2Provider,
          messengerAddress: l2MessengerAddress
        }
      })
      this.bindProviderListeners();

      return true;
    } catch(error) {
      return false;
    }
  }

  bindProviderListeners() {
    this.provider.on("accountsChanged", () => {
      window.location.reload();
    })

    this.provider.on("chainChanged", () => {
      window.location.reload();
    })

    this.OVM_L2DepositedERC20.on("WithdrawalInitiated", (sender, to, amount) => {
      console.log({ sender, to, amount: amount.toString() });
    })
  }

  initializeAccounts = () => async (dispatch) => {
    try {

      this.account = await this.web3Provider.getSigner().getAddress();
      
      const networkStatus = await dispatch(this.checkNetwork('L1L2'));
      
      if (!networkStatus) return 'wrongnetwork'

      const network = await this.web3Provider.getNetwork();
      
      this.selectedNetwork = network.chainId === l1ChainID ? "L1" : "L2";

      dispatch(setNetwork({
        network: {
          name: 'Optimism',
          shortName: 'L2',
          watcher: NETWORKS.localL1.rpcUrl,
        }
      })); 
      return 'enabled';
    } catch (error) {
      return false;
    }
  }

  async checkStatus () {
    return {
      connection: true,
      byzantine: false,
      watcherSynced: true,
      lastSeenBlock: 0,
    };
  }

  async getBalances () {
    try {
      const rootChainBalance = await l1Provider.getBalance(this.account);
      const ERC20L1Balance = await this.ERC20L1Contract.methods.balances(this.account).call({from: this.account});

      const childChainBalance = await l2Provider.getBalance(this.account);
      const ERC20L2Balance = await this.ERC20L2Contract.methods.balanceOf(this.account).call({from: this.account});

      const ethToken = await getToken(OmgUtil.transaction.ETH_CURRENCY);
      let testToken = null;
      if (networkService.selectedNetwork === 'L1') {
        testToken = await getToken(ERC20Address);
      } else {
        testToken = await getToken(L2DepositedERC20Address);
      }

      const rootchainEthBalance = [
        {
          ...ethToken,
          amount: new BN(rootChainBalance.toString()),
        },
        {
          ...testToken,
          currency: ERC20Address,
          amount: new BN(ERC20L1Balance.toString()),
        }
      ];

      const childchainEthBalance = [
        {
          ...ethToken,
          currency: l2ETHGatewayAddress,
          symbol: 'WETH',
          amount: new BN(childChainBalance.toString()),
        },
        {
          ...testToken,
          currency: L2DepositedERC20Address,
          amount: new BN(ERC20L2Balance.toString()),
        },
      ]

      return {
        rootchain: orderBy(rootchainEthBalance, i => i.currency),
        childchain: orderBy(childchainEthBalance, i => i.currency)
      };
    } catch (error) {
      throw new WebWalletError({
        originalError: error,
        reportToSentry: false,
        reportToUi: false
      });
    }
  }

  depositETHL1 = () => async(dispatch) => {

    const networkStatus = await dispatch(this.checkNetwork('L1'));
    
    if (!networkStatus) return 

    try {
      const l1ProviderRPC = new JsonRpcProvider(l1Network.rpcUrl);
      const signer = l1ProviderRPC.getSigner();
      
      // Send 1 ETH
      const txOption = {
        to: this.account,
        value: parseEther('1'), 
        gasPrice: parseUnits("4.1", "gwei"),
        gasLimit: hexlify(120000),
      } 

      const tx = await signer.sendTransaction(txOption);
      await tx.wait();

      console.log(tx);

      dispatch(openAlert("Deposited ETH to L1"));

      // break;
    } catch (error) {
      dispatch(openError("Failed to deposit ETH to L1"));
    }
  }

  depositETHL2 = async (value='1') => {
    try {
      const depositTxStatus = await this.L1ETHGatewayContract.deposit({
        value: parseEther(value),
      });
      await depositTxStatus.wait();

      const [l1ToL2msgHash] = await this.watcher.getMessageHashesFromL1Tx(depositTxStatus.hash);
      console.log(' got L1->L2 message hash', l1ToL2msgHash);

      const l2Receipt = await this.watcher.getL2TransactionReceipt(l1ToL2msgHash);
      console.log(' completed Deposit! L2 tx hash:', l2Receipt.transactionHash);
      
      this.getBalances();
      return l2Receipt;

    } catch {
      return false;
    }
  }

  async transfer(address, value, currency) {
    if (currency === '0x4200000000000000000000000000000000000006') {
      const txStatus = await this.L2ETHGatewayContract.transfer(
        address,
        parseEther(value.toString()), 
      );
      const txRes = await txStatus.wait();
      console.log(txRes);
      return txRes;
    }
    if (currency.toLowerCase() === L2DepositedERC20Address.toLowerCase()) {
      const txStatus = await this.OVM_L2DepositedERC20.transfer(
        address,
        parseEther(value.toString()), 
      );
      const txRes = await txStatus.wait();
      console.log(txRes);
      return txRes;
    }
  }

  switchChainParam(chain) {
    return {
      chainId: '0x'+chain.chainID.toString(16),
      chainName: chain.chainName,
      nativeCurrency: {
        name: chain.currencyName || 'ETH',
        symbol: chain.currencySymbol || 'ETH', // 2-6 characters long
        decimals: chain.currencyDecimals || 18,
      },
      rpcUrls: [chain.rpcUrl]
    }
  }

  checkNetwork = (networkCase) => async (dispatch) =>{
    const network = await this.web3Provider.getNetwork();

    let networkCorrect = false;
    switch(networkCase) {
      case 'L1':
        networkCorrect = network.chainId === l1ChainID;
        break;
      case 'L2':
        networkCorrect = network.chainId === l2ChainID;
        break;
      case 'L1L2':
        networkCorrect = network.chainId === l1ChainID || network.chainId === l2ChainID;
        break;
      default:
        break;
    }

    const chainParam = this.switchChainParam({
      chainID: l2ChainID,
      chainName: l2NetworkName,
      rpcUrl: l2Network.rpcUrl,
      currencyName: 'oWETH',
      currencySymbol: 'oWETH',
    });

    if (networkCase === 'L1') {
      if (!networkCorrect) {
        dispatch(openNotification({
          notificationText: `Wrong Network. Please use ${l1NetworkName}.`,
        }))
        return false
      }
    }

    if (networkCase === 'L2') {
      if (!networkCorrect) {
        dispatch(openNotification({
          notificationText: `Wrong Network. Please use ${l2NetworkName}.`,
          notificationButtonText: `Switch`,
          notificationButtonAction: () => this.web3Provider.jsonRpcFetchFunc(
            'wallet_addEthereumChain',
            [chainParam, this.account],
          )
        }))
        return  false
      }
    }

    // L1 or L2 network
    if (networkCase === 'L1L2') {
      if (!networkCorrect) {
        dispatch(openNotification({
          notificationText: `Wrong Network. Please use ${l1NetworkName} or ${l2NetworkName}.`,
          notificationButtonText: `Switch`,
          notificationButtonAction: () => this.web3Provider.jsonRpcFetchFunc(
            'wallet_addEthereumChain',
            [chainParam, this.account],
          )
        }))
        return  false
      }
    }

    return true
  }

  async getAllTransactions () {
    let transactionHistory = {};
    const latest = await l2Provider.eth.getBlockNumber();
    const blockNumbers = Array.from(Array(latest).keys());
    
    for (let blockNumber of blockNumbers) {
      const blockData = await l2Provider.eth.getBlock(blockNumber);
      const transactionsArray = blockData.transactions;

      if (transactionsArray.length === 0) {
        transactionHistory.push({

        })
      }
    }
  }

  async checkAllowance (currency, targetContract=L1ERC20GatewayAddress) {
    try {
      const ERC20Contract = new ethers.Contract(
        currency, 
        L1ERC20Json.abi, 
        this.web3Provider.getSigner(),
      );
      const allowance = await ERC20Contract.allowance(this.account, targetContract);
      return allowance.toString();
    } catch (error) {
      throw new WebWalletError({
        originalError: error,
        customErrorMessage: 'Could not check deposit allowance for ERC20.',
        reportToSentry: false,
        reportToUi: true
      });
    }
  }

  async approveErc20 (value, currency, approveContractAddress=L1ERC20GatewayAddress, contractABI= L1ERC20Json.abi) {
    try {
      const ERC20Contract = new ethers.Contract(
        currency, 
        contractABI, 
        this.web3Provider.getSigner(),
      );

      const approveStatus = await ERC20Contract.approve(
        approveContractAddress,
        value,
      );
      await approveStatus.wait();

      return true;
    } catch (error) {
      throw new WebWalletError({
        originalError: error,
        customErrorMessage: 'Could not approve ERC20 for deposit.',
        reportToSentry: false,
        reportToUi: true
      });
    }
  }

  async resetApprove (value, currency, approveContractAddress=L1ERC20GatewayAddress, contractABI= L1ERC20Json.abi) {
    try {
      const ERC20Contract = new ethers.Contract(
        currency, 
        contractABI, 
        this.web3Provider.getSigner(),
      );

      const resetApproveStatus = await ERC20Contract.approve(
        approveContractAddress,
        0,
      );
      await resetApproveStatus.wait();

      const approveStatus = await ERC20Contract.approve(
        approveContractAddress,
        value,
      );
      await approveStatus.wait();
      return true;
    } catch (error) {
      throw new WebWalletError({
        originalError: error,
        customErrorMessage: 'Could not reset approval allowance for ERC20.',
        reportToSentry: false,
        reportToUi: true
      });
    }
  }

  async depositErc20 (value, currency, gasPrice) {
    try {
      const ERC20Contract = new ethers.Contract(
        currency, 
        L1ERC20Json.abi, 
        this.web3Provider.getSigner(),
      );
      const allowance = await ERC20Contract.allowance(this.account, L1ERC20GatewayAddress);
      
      console.log({allowance:  allowance.toString(), value});

      const depositTxStatus = await this.OVM_L1ERC20Gateway.deposit(
        value,
        {gasLimit: 1000000},
      );
      await depositTxStatus.wait();

      const [l1ToL2msgHash] = await this.watcher.getMessageHashesFromL1Tx(depositTxStatus.hash);
      console.log(' got L1->L2 message hash', l1ToL2msgHash);

      const l2Receipt = await this.watcher.getL2TransactionReceipt(l1ToL2msgHash);
      console.log(' completed Deposit! L2 tx hash:', l2Receipt.transactionHash);
      
      this.getBalances();

      return l2Receipt;
    } catch (error) {
      throw new WebWalletError({
        originalError: error,
        customErrorMessage: 'Could not deposit ERC20. Please check to make sure you have enough in your wallet to cover both the amount you want to deposit and the associated gas fees.',
        reportToSentry: false,
        reportToUi: true
      });
    }
  }

  async exitOptimism(currency, value) {
    if (currency === '0x4200000000000000000000000000000000000006') {
      const tx = await this.L2ETHGatewayContract.withdraw(
        parseEther(value.toString()), 
        {gasLimit: 5000000}, 
      );
      await tx.wait();

      const [l2ToL1msgHash] = await this.watcher.getMessageHashesFromL2Tx(tx.hash)
      console.log(' got L2->L1 message hash', l2ToL1msgHash)
      
      const l1Receipt = await this.watcher.getL1TransactionReceipt(l2ToL1msgHash)
      console.log(' completed Deposit! L1 tx hash:', l1Receipt.transactionHash)
    
      return tx
    }
    if (currency === L2DepositedERC20Address) {
      const tx = await this.OVM_L2DepositedERC20.withdraw(
        parseEther(value.toString()), 
        {gasLimit: 5000000}, 
      );
      await tx.wait();

      const [l2ToL1msgHash] = await this.watcher.getMessageHashesFromL2Tx(tx.hash)
      console.log(' got L2->L1 message hash', l2ToL1msgHash)
      const l1Receipt = await this.watcher.getL1TransactionReceipt(l2ToL1msgHash)
      console.log(' completed Deposit! L1 tx hash:', l1Receipt.transactionHash)
      
      return tx
    }
    
  }

  async initialDepositL1LP(currency, value) {

    const decimals = 18;
    let depositAmount = powAmount(value, decimals);
    depositAmount = new BN(depositAmount);

    if (currency === ERC20Address) {
      // L2 LP has enough tokens
      const ERC20Contract = new ethers.Contract(
        currency, 
        L1ERC20Json.abi, 
        this.web3Provider.getSigner(),
      );
      
      // Check if the allowance is large enough
      let allowance = await ERC20Contract.allowance(this.account, L1LPAddress);
      allowance = new BN(allowance.toString());

      if (depositAmount.gt(allowance)) {
        const approveStatus = await ERC20Contract.approve(
          L1LPAddress,
          depositAmount.toString(),
        );
        await approveStatus.wait();
      }

      // Deposit
      const depositTX = await this.L1LPContract.initiateDepositTo(
        depositAmount.toString(),
        currency,
      );
      await depositTX.wait();

      return depositTX
    } else {
      const web3 = new Web3(this.provider);
      const depositTX = await web3.eth.sendTransaction({
        from: this.account,
        to: L1LPAddress,
        value: depositAmount,
      });
      return depositTX
    }
  }

  async depositL1LP(currency, value) {

    const decimals = 18;
    let depositAmount = powAmount(value, decimals);
    depositAmount = new BN(depositAmount);

    let l2TokenCurrency = null;

    if (currency.toLowerCase() === ERC20Address.toLowerCase()) {
      
      l2TokenCurrency = L2DepositedERC20Address;

      const ERC20Contract = new ethers.Contract(
        currency, 
        L1ERC20Json.abi, 
        this.web3Provider.getSigner(),
      );
      
      // Check if the allowance is large enough
      let allowance = await ERC20Contract.allowance(this.account, L1LPAddress);
      allowance = new BN(allowance.toString());

      if (depositAmount.gt(allowance)) {
        const approveStatus = await ERC20Contract.approve(
          L1LPAddress,
          depositAmount.toString(),
        );
        await approveStatus.wait();
      }

      const depositTX = await this.L1LPContract.clientDepositL1(
        depositAmount.toString(),
        currency,
        l2TokenCurrency,
      );
      await depositTX.wait();

      // Waiting the response from L2
      const [l1ToL2msgHash] = await this.watcher.getMessageHashesFromL1Tx(depositTX.hash)
      console.log(' got L1->L2 message hash', l1ToL2msgHash)
      const l2Receipt = await this.watcher.getL2TransactionReceipt(l1ToL2msgHash)
      console.log(' completed Deposit! L2 tx hash:', l2Receipt.transactionHash)

      return l2Receipt
    } else {
      const web3 = new Web3(this.provider);
      const depositTX = await web3.eth.sendTransaction({
        from: this.account,
        to: L1LPAddress,
        value: depositAmount,
      })
      await depositTX.wait();
      console.log(depositTX);
      const [l1ToL2msgHash] = await this.watcher.getMessageHashesFromL1Tx(depositTX.transactionHash)
      console.log(' got L1->L2 message hash', l1ToL2msgHash)
      const l2Receipt = await this.watcher.getL2TransactionReceipt(l1ToL2msgHash)
      console.log(' completed Deposit! L2 tx hash:', l2Receipt.transactionHash)

      return l2Receipt
    }

  }

  async L1LPBalance(currency) {
    if (currency === l2ETHGatewayAddress) {
      currency = '0x0000000000000000000000000000000000000000';
    }
    if (currency === L2DepositedERC20Address) {
      currency = ERC20Address;
    }
    const L1LPContract = new this.l1Web3Provider.eth.Contract(
      L1LPJson.abi,
      L1LPAddress,
    );
    const balance = await L1LPContract.methods.balanceOf(
      currency,
    ).call({ from: this.account });
    // Demo purpose
    const decimals = 18;
    return logAmount(balance.toString(), decimals);
  }

  async L1LPFeeBalance(currency) {
    const L1LPContract = new this.l1Web3Provider.eth.Contract(
      L1LPJson.abi,
      L1LPAddress,
    );
    const balance = await L1LPContract.methods.feeBalanceOf(
      currency,
    ).call({ from: this.account });
    // Demo purpose
    const decimals = 18;
    return logAmount(balance.toString(), decimals);
  }

  async L1LPWithdrawFee(currency, receiver, amount) {
    
    const L1LPFeeBalance = await this.L1LPContract.feeBalanceOf(currency);
    
    let L1LPBalance = 0;

    if (currency !== '0x0000000000000000000000000000000000000000') {
      const ERC20Contract = new this.l1Web3Provider.eth.Contract(
        L1ERC20Json.abi,
        currency,
      )
      L1LPBalance = await ERC20Contract.methods.balanceOf(L1LPAddress).call({from: this.account});
    } else {
      L1LPBalance = L1LPFeeBalance;
    }

    const decimals = 18;
    const sendAmount = powAmount(amount, decimals);

    if (new BN(sendAmount).lte(new BN(L1LPBalance.toString())) && new BN(sendAmount).lte(new BN(L1LPFeeBalance.toString()))) {
      const withdrawTX = await this.L1LPContract.withdrawFee(sendAmount, currency, receiver);
      await withdrawTX.wait();
      return withdrawTX;
    } else {
      return false;
    }

  }

  async initialDepositL2LP(currency, value) {
    const ERC20Contract = new ethers.Contract(
      currency, 
      L2DepositedERC20Json.abi, 
      this.web3Provider.getSigner(),
    );

    let allowance = await ERC20Contract.allowance(this.account, L2LPAddress);
    allowance = new BN(allowance.toString());

    const token = await getToken(currency);
    const decimals = token.decimals;
    let depositAmount = powAmount(value, decimals);
    depositAmount = new BN(depositAmount);

    if (depositAmount.gt(allowance)) {
      const approveStatus = await ERC20Contract.approve(
        L2LPAddress,
        depositAmount.toString(),
      );
      await approveStatus.wait();
    }

    const depositTX = await this.L2LPContract.initiateDepositTo(
      depositAmount.toString(),
      currency,
    );

    await depositTX.wait();

    return depositTX
  }

  async depositL2LP(currency, value) {
    
    let l1TokenCurrency = null;
    
    if (currency === l2ETHGatewayAddress) {
      l1TokenCurrency = "0x0000000000000000000000000000000000000000";
    } else {
      l1TokenCurrency = ERC20Address;
    }

    const ERC20Contract = new ethers.Contract(
      currency, 
      L2DepositedERC20Json.abi, 
      this.web3Provider.getSigner(),
    );

    let allowance = await ERC20Contract.allowance(this.account, L2LPAddress);
    allowance = new BN(allowance.toString());

    const token = await getToken(currency);
    const decimals = token.decimals;
    let depositAmount = powAmount(value, decimals);
    depositAmount = new BN(depositAmount);

    if (depositAmount.gt(allowance)) {
      const approveStatus = await ERC20Contract.approve(
        L2LPAddress,
        depositAmount.toString(),
      );
      await approveStatus.wait();
    }

    const depositTX = await this.L2LPContract.clientDepositL2(
      depositAmount.toString(),
      currency,
      l1TokenCurrency,
    );

    await depositTX.wait();

    // Waiting the response from L1
    const [l2ToL1msgHash] = await this.watcher.getMessageHashesFromL2Tx(depositTX.hash)
    console.log(' got L2->L1 message hash', l2ToL1msgHash)
    const l1Receipt = await this.watcher.getL1TransactionReceipt(l2ToL1msgHash)
    console.log(' completed Deposit! L1 tx hash:', l1Receipt.transactionHash)

    return l1Receipt
  }

  async L2LPBalance(currency) {
    if (currency === '0x0000000000000000000000000000000000000000') {
      currency = l2ETHGatewayAddress;
    }
    if (currency.toLowerCase() === ERC20Address.toLowerCase()) {
      currency = L2DepositedERC20Address;
    }
    const L2LPContract = new this.l2Web3Provider.eth.Contract(
      L2LPJson.abi,
      L2LPAddress,
    );
    const balance = await L2LPContract.methods.balanceOf(
      currency,
    ).call({ from: this.account });
    // Demo purpose
    const decimals = 18;
    return logAmount(balance.toString(), decimals);
  }

  async L2LPFeeBalance(currency) {
    const L2LPContract = new this.l2Web3Provider.eth.Contract(
      L2LPJson.abi,
      L2LPAddress,
    );
    const balance = await L2LPContract.methods.feeBalanceOf(
      currency,
    ).call({ from: this.account });
    // Demo purpose
    const decimals = 18;
    return logAmount(balance.toString(), decimals);
  }

  async L2LPWithdrawFee(currency, receiver, amount) {
    const ERC20Contract = new this.l2Web3Provider.eth.Contract(
      L2DepositedERC20Json.abi,
      currency,
    );
    const L2LPBalance = await ERC20Contract.methods.balanceOf(L2LPAddress).call({from: this.account});
    const L2LPFeeBalance = await this.L2LPContract.feeBalanceOf(currency);

    const decimals = 18;
    const sendAmount = powAmount(amount, decimals);

    if (new BN(sendAmount).lt(new BN(L2LPBalance)) && new BN(sendAmount).lte(new BN(L2LPFeeBalance.toString()))) {
      const withdrawTX = await this.L2LPContract.withdrawFee(sendAmount, currency, receiver);
      await withdrawTX.wait();
      return withdrawTX;
    } else {
      return false;
    }

  }
}

const networkService = new NetworkService();
export default networkService;
