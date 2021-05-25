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

import DepositModal from 'containers/modals/deposit/DepositModal';
import TransferModal from 'containers/modals/transfer/TransferModal';
import ExitModal from 'containers/modals/exit/ExitModal';
import LedgerConnect from 'containers/modals/ledger/LedgerConnect';
import AddTokenModal from 'containers/modals/addtoken/AddTokenModal';

//Wallet Functions
import Status from 'containers/status/Status';
import Account from 'containers/account/Account';
import Transactions from 'containers/transactions/Transactions';

//NFT Example Page
import NFT from 'containers/nft/Nft';
import MobileHeader from 'components/mobileheader/MobileHeader';
import MobileMenu from 'components/mobilemenu/MobileMenu';

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

  const walletMethod = useSelector(selectWalletMethod())
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
    dispatch(fetchExits());
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
      dispatch(fetchTransactions());
    });
  }, POLL_INTERVAL * 10);

  useInterval(() => {
    dispatch(fetchBalances());
  }, POLL_INTERVAL);

  const handleSetPage = async (page) => {
    setPageDisplay(page);
  }

  return (

    <>
      <DepositModal open={depositModalState} omgOnly={beginner} fast={fast}/>
      <TransferModal open={transferModalState} />
      <ExitModal open={exitModalState} fast={fast}/>
      <AddTokenModal open={addTokenModalState} />

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
              className={pageDisplay === "NFT" ? styles.subtitletextActive : styles.subtitletext}
              onClick={()=>{handleSetPage("NFT")}}
            >  
              NFT
            </h2>
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
        </div>
      </div>
    </>
  );
}

export default React.memo(Home);