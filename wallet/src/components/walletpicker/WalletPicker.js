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

import React, { useCallback, useState, useEffect } from 'react';
import { useSelector, useDispatch } from 'react-redux';

import networkService from 'services/networkService';

import { selectWalletMethod } from 'selectors/setupSelector';

import { setWalletMethod } from 'actions/setupAction';

import logo from 'images/omg_labs.svg';
import vp1 from 'images/vp_1.svg';
import vp2 from 'images/vp_2.svg';
import vp3 from 'images/vp_3.svg';

import * as styles from './WalletPicker.module.scss';

function WalletPicker ({ onEnable }) {

  const dispatch = useDispatch();

  const [ walletEnabled, setWalletEnabled ] = useState(false);
  const [ accountsEnabled, setAccountsEnabled ] = useState(false);

  const walletMethod = useSelector(selectWalletMethod());

  const dispatchSetWalletMethod = useCallback((methodName) => {
    dispatch(setWalletMethod(methodName));
  }, [ dispatch ]);

  useEffect(() => {

    async function enableBrowserWallet () {
      
      const walletEnabled = await networkService.enableBrowserWallet();
      
      return walletEnabled
        ? setWalletEnabled(true)
        : dispatchSetWalletMethod(null);
    }

    if (walletMethod === 'browser') {
      enableBrowserWallet();
    }

  }, [ dispatchSetWalletMethod, walletMethod ]);

  useEffect(() => {

    async function initializeAccounts () {

      const initialized = await dispatch(networkService.initializeAccounts());

      if (!initialized) {
        return setAccountsEnabled(false);
      }
      
      if (initialized === 'enabled') {
        return setAccountsEnabled(true);
      }

    }
    if (walletEnabled) {
      initializeAccounts();
    }
  }, [ dispatch, walletEnabled ]);

  useEffect(() => {
    if (accountsEnabled) {
      onEnable(true);
    }
  }, [ onEnable, accountsEnabled ]);

  const browserEnabled = !!window.ethereum;

  return (
    <>
      <div className={styles.WalletPicker}>
        <div className={styles.title}>
          <img src={logo} alt='logo' />
          <div className={styles.menu}>
            <div className={styles.network}>
              <div className={styles.indicator} />
              <div>
                OMGX Local Chain
              </div>
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