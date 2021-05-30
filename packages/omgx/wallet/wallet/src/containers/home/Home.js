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

import React, { useEffect, useState } from 'react';
import { useDispatch, useSelector, batch } from 'react-redux';
import { selectWalletMethod } from 'selectors/setupSelector';
import { selectModalState } from 'selectors/uiSelector';
// import { selectChildchainTransactions } from 'selectors/transactionSelector';
import { selectLogin } from 'selectors/loginSelector';

import useInterval from 'util/useInterval';
// import { isEqual } from 'lodash';

import {
  checkWatcherStatus,
  fetchBalances,
  fetchTransactions,
  fetchExits,
  fetchDeposits,
  fetchEthStats,
  checkPendingDepositStatus,
  checkPendingExitStatus
} from 'actions/networkAction';

import { checkVersion } from 'actions/serviceAction';

import { openError } from 'actions/uiAction';

import DepositModal from 'containers/modals/deposit/DepositModal';
import TransferModal from 'containers/modals/transfer/TransferModal';
import ExitModal from 'containers/modals/exit/ExitModal';
import LedgerConnect from 'containers/modals/ledger/LedgerConnect';
import AddTokenModal from 'containers/modals/addtoken/AddTokenModal';
import ConfirmationModal from 'containers/modals/confirmation/ConfirmationModal';
import FarmDepositModal from 'containers/modals/farm/FarmDepositModal';
import FarmWithdrawModal from 'containers/modals/farm/FarmWithdrawModal';

//Wallet Functions
import Status from 'containers/status/Status';
import Account from 'containers/account/Account';
import Transactions from 'containers/transactions/Transactions';

//NFT Example Page
import NFT from 'containers/nft/Nft';
import MobileHeader from 'components/mobileheader/MobileHeader';
import MobileMenu from 'components/mobilemenu/MobileMenu';

//Varna
import Login from 'containers/login/Login';
import Seller from 'containers/seller/Seller';
import Buyer from 'containers/buyer/Buyer';

// Farm
import Farm from 'containers/farm/Farm';

import networkService from 'services/networkService';

import logo from 'images/omgx.png';

import * as styles from './Home.module.scss';

const POLL_INTERVAL = 1000; //in milliseconds?

function Home () {

  const dispatch = useDispatch();

  const [ mobileMenuOpen, setMobileMenuOpen ] = useState(false)
  
  const [ pageDisplay, setPageDisplay ] = useState("AccountNow");
  
  const depositModalState = useSelector(selectModalState('depositModal'))
  const beginner = useSelector(selectModalState('beginner'))
  const fast = useSelector(selectModalState('fast'))
  const transferModalState = useSelector(selectModalState('transferModal'))
  const exitModalState = useSelector(selectModalState('exitModal'))
  const addTokenModalState = useSelector(selectModalState('addNewTokenModal'))
  const ledgerConnectModalState = useSelector(selectModalState('ledgerConnectModal'))
  const confirmationModalState = useSelector(selectModalState('confirmationModal'))
  const farmDepositModalState = useSelector(selectModalState('farmDepositModal'))
  const farmWithdrawModalState = useSelector(selectModalState('farmWithdrawModal'))

  const walletMethod = useSelector(selectWalletMethod())
  const loggedIn = useSelector(selectLogin());
  // const transactions = useSelector(selectChildchainTransactions, isEqual);
  
  useEffect(() => {
    const body = document.getElementsByTagName('body')[0];
    mobileMenuOpen
      ? body.style.overflow = 'hidden'
      : body.style.overflow = 'auto';
  }, [ mobileMenuOpen ]);

  // calls only on boot
  useEffect(() => {
    window.scrollTo(0, 0);
    dispatch(fetchDeposits());
    setPageDisplay("AccountNow");
  }, [ dispatch ]);

  useInterval(() => {
    batch(() => {
      // infura call
      dispatch(fetchEthStats());
      dispatch(checkPendingDepositStatus());
      dispatch(checkPendingExitStatus());

      // watcher only calls
      dispatch(checkWatcherStatus());
      dispatch(fetchBalances());
      dispatch(fetchExits());
      dispatch(fetchTransactions());
    });
  }, POLL_INTERVAL * 5);

  useInterval(() => {
    dispatch(fetchBalances());
  }, POLL_INTERVAL);

  useEffect(() => {
    checkVersion()
  }, [])

  useEffect(() => {
    if (!loggedIn) {
      setPageDisplay("AccountNow");
    } else {
      setPageDisplay("VarnaSell");
    }
  },[loggedIn]);
  
  const handleSetPage = async (page) => {
    if (page === 'VarnaLogin') {
      if (!(networkService.L1orL2 === 'L2')) {
        dispatch(openError('Wrong network! Please switch to L2 network to use Varna.'));
        return
      }
    }
    setPageDisplay(page);
  }

  return (

    <>
      <DepositModal open={depositModalState} omgOnly={beginner} fast={fast}/>
      <TransferModal open={transferModalState} />
      <ExitModal open={exitModalState} fast={fast}/>
      <AddTokenModal open={addTokenModalState} />
      <ConfirmationModal open={confirmationModalState} />
      <FarmDepositModal open={farmDepositModalState} />
      <FarmWithdrawModal open={farmWithdrawModalState} />

      <LedgerConnect
        open={walletMethod === 'browser'
          ? ledgerConnectModalState
          : false
        }
      />

      <div className={styles.Home}>
        <div className={styles.sidebar}>
          <img className={styles.logo} src={logo} alt='omgx' />
          <Status />
        </div>
        <div className={styles.main}>
          <MobileHeader
            mobileMenuOpen={mobileMenuOpen}
            onHamburgerClick={()=>setMobileMenuOpen(open=>!open)}
          />
          <MobileMenu 
            mobileMenuOpen={mobileMenuOpen}
          />

          {/* The Top SubMenu Bar, non-mobile */}

          <div className={styles.secondtab}>
            <h2
              className={pageDisplay === "AccountNow" ? styles.subtitletextActive : styles.subtitletext}
              onClick={()=>{handleSetPage("AccountNow")}}
            >  
              Wallet
            </h2>
            <h2
              className={pageDisplay === "Farm" ? styles.subtitletextActive : styles.subtitletext}
              onClick={()=>{handleSetPage("Farm")}}
            >  
              Farm
            </h2>
            <h2
              className={pageDisplay === "NFT" ? styles.subtitletextActive : styles.subtitletext}
              onClick={()=>{handleSetPage("NFT")}}
            >  
              NFT
            </h2>
            {!loggedIn ?
              <h2
                className={pageDisplay === "VarnaLogin" ? styles.subtitletextActive : styles.subtitletext}
                onClick={()=>{handleSetPage("VarnaLogin")}}
              >  
                Login
              </h2>:
              <>
                <h2 
                  className={pageDisplay === "VarnaBuy" ? styles.subtitletextActive : styles.subtitletext}
                  onClick={()=>{handleSetPage("VarnaBuy")}}
                  style={{position: 'relative'}}
                >  
                  Buy
                </h2>
                <h2
                  className={pageDisplay === "VarnaSell" ? styles.subtitletextActive : styles.subtitletext}
                  onClick={()=>{handleSetPage("VarnaSell")}}
                >  
                  Sell
                </h2>
              </>
            }
          </div>
          {pageDisplay === "AccountNow" &&
          <>  
            <Account/>
            <Transactions/>
          </>
          }
          {pageDisplay === "NFT" &&
            <NFT/>
          }
          {pageDisplay === "VarnaLogin" &&
            <Login/>
          }
          {pageDisplay === "VarnaSell" &&
            <Seller/>
          }
          {pageDisplay === "VarnaBuy" &&
            <Buyer/>
          }
          {pageDisplay === "Farm" &&
            <Farm/>
          }
        </div>
      </div>
    </>
  );
}

export default React.memo(Home);