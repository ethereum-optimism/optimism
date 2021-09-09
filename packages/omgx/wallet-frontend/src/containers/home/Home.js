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

import { closeAlert, closeError } from 'actions/uiAction';
import { selectAlert, selectError } from 'selectors/uiSelector';

import DepositModal from 'containers/modals/deposit/DepositModal';
import TransferModal from 'containers/modals/transfer/TransferModal';
import ExitModal from 'containers/modals/exit/ExitModal';

import LedgerConnect from 'containers/modals/ledger/LedgerConnect';
import AddTokenModal from 'containers/modals/addtoken/AddTokenModal';

//Farm
import FarmDepositModal from 'containers/modals/farm/FarmDepositModal';
import FarmWithdrawModal from 'containers/modals/farm/FarmWithdrawModal';

//DAO
import DAO from 'containers/dao/Dao';
import TransferDaoModal from 'containers/modals/dao/TransferDaoModal';
import DelegateDaoModal from 'containers/modals/dao/DelegateDaoModal';
import NewProposalModal from 'containers/modals/dao/NewProposalModal';

import { fetchDaoBalance, fetchDaoVotes, fetchDaoProposals, getProposalThreshold } from 'actions/daoAction';

//Wallet Functions
import Account from 'containers/account/Account';
import Transactions from 'containers/transactions/History';

//NFT Example Page
import NFT from 'containers/nft/Nft';

import { useTheme } from '@material-ui/core/styles'
import { Box, Container, useMediaQuery } from '@material-ui/core'
import MainMenu from 'components/mainMenu/MainMenu'
import FarmWrapper from 'containers/farm/FarmWrapper'

import Alert from 'components/alert/Alert';

const POLL_INTERVAL = 5000 //milliseconds

function Home () {

  const dispatch = useDispatch();
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down('md'));

  const errorMessage = useSelector(selectError);
  const alertMessage = useSelector(selectAlert);

  

  const [ mobileMenuOpen, setMobileMenuOpen ] = useState(false)

  const pageDisplay = useSelector(selectModalState('page'))
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

  const handleErrorClose=()=>dispatch(closeError());
  const handleAlertClose=()=>dispatch(closeAlert());

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
    dispatch(fetchBalances())
    dispatch(addTokenList())
    dispatch(fetchNFTs())

    // get Dao balance / Votes
    dispatch(fetchDaoBalance())
    dispatch(fetchDaoVotes())
    dispatch(fetchDaoProposals())
    dispatch(getProposalThreshold())
  }, POLL_INTERVAL);

  useEffect(() => {
    checkVersion();
  }, [])

  return (
    <>
      <DepositModal  open={depositModalState}  token={token} fast={fast} />
      <TransferModal open={transferModalState} token={token} fast={fast} />
      <ExitModal     open={exitModalState}     token={token} fast={fast} />

      <AddTokenModal     open={addTokenModalState} />
      <FarmDepositModal  open={farmDepositModalState} />
      <FarmWithdrawModal open={farmWithdrawModalState} />

      <TransferDaoModal open={tranferBobaDaoModalState} />
      <DelegateDaoModal open={delegateBobaDaoModalState} />
      <NewProposalModal open={proposalBobaDaoModalState} />

      <Alert
        type='error'
        duration={0}
        open={!!errorMessage}
        onClose={handleErrorClose}
        position={50}
      >
        {errorMessage}
      </Alert>

      <Alert
        type='success'
        duration={0}
        open={!!alertMessage}
        onClose={handleAlertClose}
        position={0}
      >
        {alertMessage}
      </Alert>

      <LedgerConnect
        open={walletMethod === 'browser'
          ? ledgerConnectModalState
          : false
        }
      />

      <Box sx={{ display: 'flex', flexDirection: isMobile ? 'column' : 'row', width: '100%' }}>
          <MainMenu />
          {/* The Top SubMenu Bar, non-mobile */}

        <Container maxWidth="lg">
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
            <FarmWrapper/>
          }
          {pageDisplay === "DAO" &&
            <DAO/>
          }
        </Container>
      </Box>
    </>
  );
}

export default React.memo(Home);
