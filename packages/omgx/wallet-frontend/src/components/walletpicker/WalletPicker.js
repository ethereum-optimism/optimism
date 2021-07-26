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

import React, { useCallback, useState, useEffect, useRef } from 'react';
import { useSelector, useDispatch } from 'react-redux';

import WrongNetworkModal from 'containers/modals/wrongnetwork/WrongNetworkModal';
import networkService from 'services/networkService';

import { selectModalState } from 'selectors/uiSelector';

import {
  selectWalletMethod,
  selectNetwork,
} from 'selectors/setupSelector';

import { openModal } from 'actions/uiAction';
import { setWalletMethod, setNetwork } from 'actions/setupAction';
import { getAllNetworks } from 'util/masterConfig';

import logo from 'images/omgx.png';
import chevron from 'images/chevron.svg';

import * as styles from './WalletPicker.module.scss';
import { isChangingChain } from 'util/changeChain';
import Button from 'components/button/Button';

function WalletPicker ({ onEnable, enabled }) {
  const dispatch = useDispatch();
  const dropdownNode = useRef(null);

  const [ walletEnabled, setWalletEnabled ] = useState(false);
  const [ accountsEnabled, setAccountsEnabled ] = useState(false);
  const [ wrongNetwork, setWrongNetwork ] = useState(false);
  const [ showAllNetworks, setShowAllNetworks ] = useState(false);

  const walletMethod = useSelector(selectWalletMethod())
  const masterConfig = useSelector(selectNetwork())

  const wrongNetworkModalState = useSelector(selectModalState('wrongNetworkModal'));

  const dispatchSetWalletMethod = useCallback((methodName) => {
    dispatch(setWalletMethod(methodName));
  }, [ dispatch ])

  const dispatchSetNetwork = useCallback((network) => {
    //console.log("dispatchSetNetwork:",network)
    setShowAllNetworks(false);
    dispatch(setNetwork(network));
  }, [ dispatch ])

  useEffect(() => {

    if (walletMethod === 'browser') {
      enableBrowserWallet();
    }

    async function enableBrowserWallet () {
      //console.log("enableBrowserWallet() for",masterConfig)
      const selectedNetwork = masterConfig ? masterConfig : "local";
      const walletEnabled = await networkService.enableBrowserWallet(selectedNetwork);
      //console.log("walletEnabled:",walletEnabled)
      return walletEnabled
        ? setWalletEnabled(true)
        : dispatchSetWalletMethod(null);
    }

  }, [ dispatchSetWalletMethod, walletMethod, masterConfig ]);

  useEffect(() => {

    async function initializeAccounts () {

      //console.log("initializeAccounts() for:",masterConfig)

      const initialized = await networkService.initializeAccounts(masterConfig);

      if (!initialized) {
        console.log("Error !initialized for:",masterConfig)
        return setAccountsEnabled(false);
      }

      if (initialized === 'wrongnetwork') {
        setAccountsEnabled(false);
        return setWrongNetwork(true);
      }

      if (initialized === 'enabled') {
        return setAccountsEnabled(true);
      }

    }
    if (walletEnabled) {
      initializeAccounts();
    }
  }, [ walletEnabled, masterConfig ]);

  useEffect(() => {
    if (accountsEnabled) {
      onEnable(true);
    }
  }, [ onEnable, accountsEnabled ]);

  useEffect(() => {
    if (walletEnabled && wrongNetwork) {
      dispatch(openModal('wrongNetworkModal'));
      localStorage.setItem('changeChain', false);
    }
  }, [ dispatch, walletEnabled, wrongNetwork ]);

  function resetSelection () {
    dispatchSetWalletMethod(null);
    setWalletEnabled(false);
    setAccountsEnabled(false);
  }

  const browserEnabled = !!window.ethereum;

  // defines the set of possible networks
  const networks = getAllNetworks();

  let allNetworks = [];
  for (var prop in networks) allNetworks.push(prop)

  if (!wrongNetwork && !enabled && isChangingChain) {
    return <div className={styles.loading}>Switching Chain...</div>
  }

  return (
    <>

      <WrongNetworkModal
        open={wrongNetworkModalState}
        onClose={resetSelection}
      />

      <div className={styles.WalletPicker}>
        <div className={styles.title}>
          <img src={logo} alt='logo' />
          <div className={styles.menu}>

            <div
              onClick={()=>setShowAllNetworks(prev => !prev)}
              className={styles.network}
            >
              <div className={styles.indicator} />
              <div>
                OMGX {masterConfig}
              </div>
              {!!allNetworks.length && (
                <img
                  src={chevron}
                  alt='chevron'
                  className={[
                    styles.chevron,
                    showAllNetworks ? styles.open : ''
                  ].join(' ')}
                />
              )}
            </div>

            <div
              ref={dropdownNode}
              className={styles.dropdown}
            >
              {!!allNetworks.length && showAllNetworks && allNetworks.map((network, index) => (
                <div
                  style={{background: '#2A308E', color: 'white', marginTop: 5, padding: 5, borderRadius: 3, cursor: 'pointer'}}
                  key={index}
                  onClick={()=>dispatchSetNetwork(network)}
                >
                  {network}
                </div>))
              }
            </div>

          </div>
        </div>
      </div>

      <div className={styles.MainBar} >
        <div className={styles.MainLeft}>
          OMGX Gateway<br/>
          90 Second Swap-On and Swap-Off<br/>
          Traditional Deposits and 7 Day Exits<br/>
        </div>
        <div className={styles.MainRightContainer}>
          <Button
            type="primary"
            disabled={!browserEnabled}
            pulsate={true}
            className={styles.ButtonConnect}
            onClick={() => dispatchSetWalletMethod('browser')}
          >
            Connect to MetaMask
          </Button>
          {!browserEnabled &&
            <div className={styles.disabledMM}>Your browser does not have a web3 provider.</div>
          }

          <Button
            type="primary"
            className={styles.ButtonAdd}
            onClick={() => networkService.addL2Network()}
          >
            Add OMGX L2 Provider
          </Button>
        </div>
      </div>

      <div className={styles.WalletPicker}>

        <div className={styles.directive}>

          <div className={styles.Title}>
            <span className={styles.B}>Demo of Traditional Deposit and Exit.</span>{' '}Note - for testing, we have turned off the 7 day exit delay.<br/><br/>
            <span className={styles.B}>NEW.</span>{' '}Fast (90 second) Swap-On and Swap-Off, from L1 to L2, and back from L2 to L1. Depositing ETH on L1
            transfers oETH to you on the L2, and vice versa. No more waiting to exit.<br/><br/>
            <span className={styles.B}>Staking and Community-provided Liquidity.</span>{' '}This fast on/off capability is
            based on paired Liquidity Pools on L1 and L2 provided by the operator and the broader community,
            who can earn rewards for providing liquidity.<br/><br/>
            <span className={styles.B}>Easy to customize.</span>{' '}We have tried to keep the code simple to make it easy to customize and modify.<br/><br/>
            <span className={styles.B}>Requirements.</span>{' '}You will need Metamask and,
            if you want to test on the Rinkeby testnet, some Rinkeby ETH.<br/><br/>
            <span className={styles.B}>MetaMask L2 Setup.</span>{' '}Click 'Add OMGX L2 Provider', or, if want to add it manually, go to <span className={styles.B}>MetaMask&#62;Settings&#62;Networks&#62;Add Network</span>.{' '}Specify `https://rinkeby.omgx.network` as the New RPC URL.<br/><br/>
            <br/>
            <br/>
          </div>

        </div>

      </div>
    </>
  );
}

export default React.memo(WalletPicker);
