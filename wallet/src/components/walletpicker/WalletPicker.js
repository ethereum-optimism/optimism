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
import { getAllNetworks } from 'util/networkName';

import logo from 'images/omg_labs.svg';
import vp1 from 'images/vp_1.svg';
import vp2 from 'images/vp_2.svg';
import vp3 from 'images/vp_3.svg';
import chevron from 'images/chevron.svg';

import * as styles from './WalletPicker.module.scss';

function WalletPicker ({ onEnable }) {

  const dispatch = useDispatch();
  const dropdownNode = useRef(null);

  const [ walletEnabled, setWalletEnabled ] = useState(false);
  const [ accountsEnabled, setAccountsEnabled ] = useState(false);
  const [ wrongNetwork, setWrongNetwork ] = useState(false);
  const [ showAllNetworks, setShowAllNetworks ] = useState(false);

  const walletMethod = useSelector(selectWalletMethod());

  const networkName = useSelector(selectNetwork());
  const wrongNetworkModalState = useSelector(selectModalState('wrongNetworkModal'));

  const dispatchSetWalletMethod = useCallback((methodName) => {
    dispatch(setWalletMethod(methodName));
  }, [ dispatch ])

  const dispatchSetNetwork = useCallback((network) => {
    console.log(network)
    setShowAllNetworks(false);
    dispatch(setNetwork(network));
  }, [ dispatch ])

  useEffect(() => {

    async function enableBrowserWallet () {
      
      console.log("enableBrowserWallet for",networkName)

      const selectedNetwork = networkName ? networkName : "local";

      const walletEnabled = await networkService.enableBrowserWallet(selectedNetwork);
      
      return walletEnabled
        ? setWalletEnabled(true)
        : dispatchSetWalletMethod(null);
    }

    if (walletMethod === 'browser') {
      enableBrowserWallet();
    }

  }, [ dispatchSetWalletMethod, walletMethod, networkName ]);

  useEffect(() => {

    async function initializeAccounts () {

      console.log("initializeAccounts for",networkName)

      const initialized = await networkService.initializeAccounts( 
        networkName
      );

      if (!initialized) {
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
  }, [ dispatchSetNetwork, walletEnabled, networkName ]);

  useEffect(() => {
    if (accountsEnabled) {
      onEnable(true);
    }
  }, [ onEnable, accountsEnabled ]);

  useEffect(() => {
    if (walletEnabled && wrongNetwork) {
      dispatch(openModal('wrongNetworkModal'));
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
                OMGX&nbsp;{networkName}
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
                  style={{background: '#2A308E', color: 'white', marginTop: 5, padding: 5, borderRadius: 3}}
                  key={index}
                  onClick={()=>dispatchSetNetwork({network})}
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
          Privacy-preserving decentralized exchange.<br/>
          No front-running.<br/>
          Zero MEV.<br/>
        </div>
        <div 
          className={styles.MainRight}
          onClick={()=>dispatchSetWalletMethod('browser')}
        >
          <div
            className={[styles.MainButton, !browserEnabled ? styles.disabled : ''].join(' ')}
          >
            <span>Connect to MetaMask</span>
            {!browserEnabled && 
              <div className={styles.disabledMM}>Your browser does not have a web3 provider.</div>
            }
          </div>
        </div>
      </div>

      <div className={styles.VPBar}>

        <div className={styles.VPTile}>
          <img src={vp1} className={styles.VPImage} alt='no front running' />
          <div className={styles.VPText}>
            Bids are encrypted - others cannot see or front-run them. 
          </div>
        </div>

        <div className={styles.VPTile}>
          <img src={vp2} className={styles.VPImage} alt='privacy' />
          <div className={styles.VPText}>
            Only the sellers can see your bids. 
          </div>
        </div>

        <div className={styles.VPTile}>
          <img src={vp3} className={styles.VPImage} alt='no mev' />
          <div className={styles.VPText}>
            By running on OMG Plasma, MEV is always zero. 
          </div>
        </div>

      </div>

      <div className={styles.WalletPicker}>

        <div className={styles.directive}>

          <div className={styles.Title}>
            <span className={styles.B}>Sellers</span>.{' '}Your listings are protected by Lattice Cryptography. No one knows how much you are selling.<br/><br/> 
            <span className={styles.B}>Buyers</span>.{' '}Your bids are encrypted and visible only to sellers; no front-running.<br/><br/> 
            <span className={styles.B}>Zero miner extractable value (MEV)</span>.{' '} Since Varna is built on OMG Plasma, there are no miners who could reorder transactions.<br/><br/>
            <span className={styles.B}>Direct settlement</span>{' '} on OMG Plasma through atomic swaps.
            <br/>
            <br/>
          </div>

          <div className={styles.Note}>
            <span className={styles.B}>Requirements</span>. You will need Metamask and 
            some OMG on Plasma. To buy or sell ERC20 tokens, they must be on the 
            Child Chain. Go to {' '}<span className={styles.B}>Wallet&gt;Deposit</span>.
          </div>

        </div>

      </div>
    </>
  );
}

export default React.memo(WalletPicker);