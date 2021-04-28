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

import React, { useMemo, useCallback } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import { Send, MergeType, ArrowBack, ArrowForward } from '@material-ui/icons';
import { isEqual } from 'lodash';
import truncate from 'truncate-middle';

import { selectLoading } from 'selectors/loadingSelector';
import { selectIsSynced } from 'selectors/statusSelector';
import { selectChildchainBalance, selectRootchainBalance } from 'selectors/balanceSelector';
import { selectPendingExits } from 'selectors/exitSelector';

import { SELECT_NETWORK } from 'Settings';

import { openModal } from 'actions/uiAction';

import Copy from 'components/copy/Copy';
import Button from 'components/button/Button';
import { logAmount } from 'util/amountConvert';

import networkService from 'services/networkService';

import bunny_happy from 'images/bunny_happy.svg';
import bunny_sad from 'images/bunny_sad.svg';

import * as styles from './Account.module.scss';

function Account () {

  const dispatch = useDispatch();
  const isSynced = useSelector(selectIsSynced);
  const childBalance = useSelector(selectChildchainBalance, isEqual);
  const rootBalance = useSelector(selectRootchainBalance, isEqual);
  const criticalTransactionLoading = useSelector(selectLoading([ 'EXIT/CREATE' ]));
  
  const disabled = !childBalance.length || !isSynced ;

  const handleModalClick = useCallback(
    async (name, beginner = false) => {
      if (name === 'transferModal') {
        const networkStatus = await dispatch(networkService.checkNetwork('L2'));
        if (!networkStatus) return 
      }
      if (name === 'depositModal') {
        const networkStatus = await dispatch(networkService.checkNetwork('L1'));
        if (!networkStatus) return 
      }
      dispatch(openModal(name, beginner))
    }, [ dispatch ]
  );

  let balances = {
    OMG : {have: false, amount: 0, amountShort: '0'},
    WETH : {have: false, amount: 0, amountShort: '0'}
  }

  childBalance.reduce((acc, cur) => {
    if (cur.symbol === 'WETH' && cur.amount > 0 ) {
      acc['WETH']['have'] = true;
      acc['WETH']['amount'] = cur.amount;
      acc['WETH']['amountShort'] = logAmount(cur.amount, cur.decimals, 2);
    }
    return acc;
  }, balances)

  const wAddress = networkService.account ? truncate(networkService.account, 6, 4, '...') : '';

  const handleDepositETHL1 = useCallback(
    () => dispatch(networkService.depositETHL1()),
    [dispatch]
  );

  return (
    <div className={styles.Account}>

      <div className={styles.wallet}>
        <span className={styles.address}>{`Wallet Address : ${wAddress}`}</span>
        <Copy value={networkService.account} />
      </div>

      {balances['WETH']['have'] &&
        <h3 style={{marginBottom: '30px'}}>Status: Ready to use OMGX</h3> 
      }
      {!balances['WETH']['have'] &&
        <h3 style={{marginBottom: '30px'}}>Status: Bunny Cry. You do not have any wETH on OMGX</h3> 
      }

      {balances['WETH']['have'] &&
        <div className={styles.RabbitBox}>
          <img className={styles.bunny} src={bunny_happy} alt='Happy Bunny' />
          <div className={styles.RabbitRight}>
            <div
              className={styles.RabbitRightTop}
            >
              Child Chain<br/>Balance
            </div>
            <div 
              className={styles.RabbitRightMiddle.sad}
              style={{color: '#0ebf9a', fontSize: '4em'}}
            >
              <span>
              {balances['WETH']['amountShort']}
              </span>
            </div>
            <div className={styles.RabbitRightBottom}>
              WETH
            </div>
          </div>
        </div>
      }

      {!balances['WETH']['have'] &&
        <div className={styles.RabbitBox}>
          <img className={styles.bunny} src={bunny_sad} alt='Sad Bunny' />
          <div className={styles.RabbitRight}>
            <div
              className={styles.RabbitRightTop}
            >
              OMGX L2<br/>wETH Balance
            </div>
            <div className={styles.RabbitRightMiddle}>
              <span className={styles.sad}>
                0
              </span>
            </div>
            <div className={styles.RabbitRightBottom}>
            </div>
          </div>
        </div>
      }

      <div className={styles.balances} style={{marginTop: 30}}>

        <div className={styles.box}>
          <div className={styles.header}>
            <div className={styles.title}>
              <span>Balance on Childchain</span>
              <span>OMGX</span>
            </div>
              <div
                onClick={()=>handleModalClick('transferModal')}
                className={[styles.transfer, disabled ? styles.disabled : ''].join(' ')}
              >
                <Send />
                <span>TRANSFER L2->L2</span>
              </div>
          </div>
          {childBalance.map((i, index) => {
            return (
              <div key={index} className={styles.row}>
                <div className={styles.token}>
                  <span className={styles.symbol}>{i.symbol}</span>
                </div>
                <span>{logAmount(i.amount, i.decimals, 4)}</span>
              </div>
            );
          })}
        </div>

        <div className={styles.boxActions}>
          <div className={styles.buttons}>
            <Button
              onClick={() => handleModalClick('depositModal')}
              type='primary'
              disabled={!isSynced || criticalTransactionLoading}
              style={{maxWidth: 'none'}}
            >
              <ArrowBack/>
              FAST ONRAMP
            </Button>
          </div>
          <div className={styles.buttons}>
            <Button
              onClick={() => handleModalClick('depositModal')}
              type='primary'
              disabled={!isSynced || criticalTransactionLoading}
              style={{maxWidth: 'none'}}
            > 
              FAST EXIT
              <ArrowForward/>
            </Button>
          </div>

          <div className={styles.buttons}>
            <Button
              onClick={() => handleModalClick('depositModal')}
              type='primary'
              disabled={!isSynced || criticalTransactionLoading}
              style={{maxWidth: 'none'}}
            >
              <ArrowBack/>
              SLOW ONRAMP
            </Button>
          </div>
          <div className={styles.buttons}>
            <Button
              onClick={() => handleModalClick('exitModal')}
              type='primary'
              disabled={disabled || criticalTransactionLoading}
              style={{maxWidth: 'none'}}
            >
              SLOW EXIT
              <ArrowForward/>
            </Button>
          </div>
        </div>

        <div className={styles.box}>
          <div className={styles.header}>
            <div className={styles.title}>
              <span>Balance on Rootchain</span>
              <span>Ethereum Network</span>
            </div>
            {SELECT_NETWORK === 'local' &&
              <div
                onClick={()=>handleDepositETHL1()}
                className={[styles.transfer, !isSynced ? styles.disabled : ''].join(' ')}
              >
                <Send />
                <span>L1 ETH Fountain</span>
              </div>
            }
          </div>

          {rootBalance.map((i, index) => {
            return (
              <div key={index} className={styles.row}>
                <div className={styles.token}>
                  <span className={styles.symbol}>{i.symbol}</span>
                </div>
                <span>{logAmount(i.amount, i.decimals, 4)}</span>
              </div>
            );
          })}

        </div>
      </div>

    </div>
  );

}

export default React.memo(Account);
