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

import useInterval from 'util/useInterval';

import {
  checkWatcherStatus,
  fetchBalances,
  addTokenList,
  fetchNFTs,
  fetchExits,
  fetchDeposits,
  fetchEthStats,
  checkPendingDepositStatus,
  checkPendingExitStatus
} from 'actions/networkAction';

import { checkVersion } from 'actions/serviceAction';

import DepositModal from 'containers/modals/deposit/DepositModal';
import TransferModal from 'containers/modals/transfer/TransferModal';
import ExitModal from 'containers/modals/exit/ExitModal';

import LedgerConnect from 'containers/modals/ledger/LedgerConnect';
import AddTokenModal from 'containers/modals/addtoken/AddTokenModal';
import FarmDepositModal from 'containers/modals/farm/FarmDepositModal';
import FarmWithdrawModal from 'containers/modals/farm/FarmWithdrawModal';
import TransferDaoModal from 'containers/modals/dao/TransferDaoModal';
import DelegateDaoModal from 'containers/modals/dao/DelegateDaoModal';
import NewProposalModal from 'containers/modals/dao/NewProposalModal';


//Wallet Functions
import Status from 'containers/status/Status';
import Account from 'containers/account/Account';
import Transactions from 'containers/transactions/History';

//NFT Example Page
import NFT from 'containers/nft/Nft';
import MobileHeader from 'components/mobileheader/MobileHeader';
import MobileMenu from 'components/mobilemenu/MobileMenu';

// Farm
import Farm from 'containers/farm/Farm';

// DAO
import DAO from 'containers/dao/Dao';


// import logo from 'images/omgx.png';
import logo from 'images/logo-boba.svg';

import * as styles from './Home.module.scss';
import { fetchDaoBalance, fetchDaoVotes } from 'actions/daoAction';

const POLL_INTERVAL = 5000; //milliseconds

function Home () {

  const dispatch = useDispatch();

  const [ mobileMenuOpen, setMobileMenuOpen ] = useState(false)
  
  const [ pageDisplay, setPageDisplay ] = useState("AccountNow");
  
  const depositModalState = useSelector(selectModalState('depositModal'))
  const transferModalState = useSelector(selectModalState('transferModal'))
  const exitModalState = useSelector(selectModalState('exitModal'))
  
  const fast = useSelector(selectModalState('fast'))
  const token = useSelector(selectModalState('token'))

  const addTokenModalState = useSelector(selectModalState('addNewTokenModal'))
  const ledgerConnectModalState = useSelector(selectModalState('ledgerConnectModal'))

  const farmDepositModalState = useSelector(selectModalState('farmDepositModal'))
  const farmWithdrawModalState = useSelector(selectModalState('farmWithdrawModal'))

  // DAO modal
  const tranferBobaDaoModalState = useSelector(selectModalState('transferDaoModal'))
  const delegateBobaDaoModalState = useSelector(selectModalState('delegateDaoModal'))
  const proposalBobaDaoModalState = useSelector(selectModalState('newProposalModal'))

  const walletMethod = useSelector(selectWalletMethod())
  //const transactions = useSelector(selectlayer2Transactions, isEqual);
  
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
      dispatch(fetchExits());
    });
  }, POLL_INTERVAL * 2);

  //get all account balances
  useInterval(() => {
    dispatch(fetchBalances());
    dispatch(addTokenList());
    dispatch(fetchNFTs());

    // get Dao balance / Votes
    dispatch(fetchDaoBalance());
    dispatch(fetchDaoVotes());
  }, POLL_INTERVAL);

  useEffect(() => {
    checkVersion();
  }, [])
  
  const handleSetPage = async (page) => {
    setPageDisplay(page)
  }

  return (

    <>
      
      <DepositModal  open={depositModalState}  token={token} fast={fast} />
      <TransferModal open={transferModalState} token={token} fast={fast} />
      <ExitModal     open={exitModalState}     token={token} fast={fast} />
      
      <AddTokenModal open={addTokenModalState} />
      <FarmDepositModal open={farmDepositModalState} />
      <FarmWithdrawModal open={farmWithdrawModalState} />

      <TransferDaoModal open={tranferBobaDaoModalState} />
      <DelegateDaoModal open={delegateBobaDaoModalState} />
      <NewProposalModal open={proposalBobaDaoModalState} />

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
              className={pageDisplay === "History" ? styles.subtitletextActive : styles.subtitletext}
              onClick={()=>{handleSetPage("History")}}
            >  
              History
            </h2>
            <h2
              className={pageDisplay === "Farm" ? styles.subtitletextActive : styles.subtitletext}
              onClick={()=>{handleSetPage("Farm")}}
            >  
              Earn
            </h2>
            <h2
              className={pageDisplay === "NFT" ? styles.subtitletextActive : styles.subtitletext}
              onClick={()=>{handleSetPage("NFT")}}
            >  
              NFT
            </h2>
            <h2
              className={pageDisplay === "DAO" ? styles.subtitletextActive : styles.subtitletext}
              onClick={()=>{handleSetPage("DAO")}}
            >  
              DAO
            </h2>
          </div>
          {pageDisplay === "AccountNow" &&
          <>  
            <Account/>
          </>
          }
          {pageDisplay === "History" &&
          <>  
            <Transactions/>
          </>
          }
          {pageDisplay === "NFT" &&
            <NFT/>
          }
          {pageDisplay === "Farm" &&
            <Farm/>
          }
          {pageDisplay === "DAO" &&
            <DAO/>
          }
        </div>
      </div>
    </>
  );
}

export default React.memo(Home);