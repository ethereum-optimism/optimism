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

import React, { useCallback } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import { Send, ArrowBack, ArrowForward } from '@material-ui/icons';
import { isEqual } from 'lodash';
import truncate from 'truncate-middle';

import { selectLoading } from 'selectors/loadingSelector';
import { selectIsSynced } from 'selectors/statusSelector';
import { selectChildchainBalance, selectRootchainBalance } from 'selectors/balanceSelector';

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
    async (name, fast = false, beginner = false) => {
      if (name === 'transferModal' || name === 'exitModal') {
        const correctLayer = await dispatch(networkService.confirmLayer('L2'));
        if (!correctLayer) return 
      }
      if (name === 'depositModal') {
        const correctLayer = await dispatch(networkService.confirmLayer('L1'));
        if (!correctLayer) return 
      }
      dispatch(openModal(name, beginner, fast))
    }, [ dispatch ]
  );

  let balances = {
    /*____ : {have: false, amount: 0, amountShort: '0'},*/
    oETH : {have: false, amount: 0, amountShort: '0'}
  }

  childBalance.reduce((acc, cur) => {
    if (cur.symbol === 'oETH' && cur.amount > 0 ) {
      acc['oETH']['have'] = true;
      acc['oETH']['amount'] = cur.amount;
      acc['oETH']['amountShort'] = logAmount(cur.amount, cur.decimals, 2);
    }
    return acc;
  }, balances)

  const wAddress = networkService.account ? truncate(networkService.account, 6, 4, '...') : '';
  const networkLayer = networkService.L1orL2 === 'L1' ? 'L1' : 'L2';

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

      <div className={styles.wallet}>
        <span className={styles.address}>{`NetworkLayer : ${networkLayer}`}</span>
      </div>

      <div className={styles.wallet}>
        {networkLayer === 'L1' && 
          <span>Since you are on Mainnet (L1), you can only perform L1 functions, such as sending funds from L1 to OMGX. To do things on OMGX (L2), please switch to L2 in your wallet.</span>
        }
        {networkLayer === 'L2' && 
          <span>Since you are on OMGX (L2), you can only perform L2 functions, such as trading and sending funds from OMGX to L1. To do things on Mainchain (L1), please switch to L1 in your wallet.</span>
        }
      </div>

      {balances['oETH']['have'] &&
        <h3 style={{marginBottom: '30px'}}>Status: Ready to use OMGX</h3> 
      }
      {!balances['oETH']['have'] &&
        <h3 style={{marginBottom: '30px'}}>Status: Bunny Cry. You do not have any oETH on OMGX</h3> 
      }

      {balances['oETH']['have'] &&
        <div className={styles.RabbitBox}>
          <img className={styles.bunny} src={bunny_happy} alt='Happy Bunny' />
          <div className={styles.RabbitRight}>
            <div
              className={styles.RabbitRightTop}
            >
              OMGX Balance
            </div>
            <div 
              className={styles.RabbitRightMiddle.sad}
              style={{color: '#0ebf9a', fontSize: '4em'}}
            >
              <span>
              {balances['oETH']['amountShort']}
              </span>
            </div>
            <div className={styles.RabbitRightBottom}>
              oETH
            </div>
          </div>
        </div>
      }

      {!balances['oETH']['have'] &&
        <div className={styles.RabbitBox}>
          <img className={styles.bunny} src={bunny_sad} alt='Sad Bunny' />
          <div className={styles.RabbitRight}>
            <div
              className={styles.RabbitRightTop}
            >
              OMGX oETH Balance
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

        <div className={[styles.box, networkLayer === 'L2' ? styles.dim : styles.active].join(' ')}>
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

        <div className={styles.boxActions}>
          <div className={styles.buttons}>
            <Button
              onClick={() => handleModalClick('depositModal', true)}
              type='primary'
              disabled={!isSynced || criticalTransactionLoading || networkLayer === 'L2'}
              style={{maxWidth: 'none'}}
            >
              FAST ONRAMP<ArrowForward/>
            </Button>
          </div>
          <div className={styles.buttons}>
            <Button
              onClick={() => handleModalClick('exitModal', true)}
              type='primary'
              disabled={!isSynced || criticalTransactionLoading || networkLayer === 'L1'}
              style={{maxWidth: 'none'}}
            > 
            <ArrowBack/>FAST EXIT
            </Button>
          </div>

          <div className={styles.buttons}>
            <Button
              onClick={() => handleModalClick('depositModal')}
              type='primary'
              disabled={!isSynced || criticalTransactionLoading || networkLayer === 'L2'}
              style={{maxWidth: 'none'}}
            >
              SLOW ONRAMP<ArrowForward/>
            </Button>
          </div>
          <div className={styles.buttons}>
            <Button
              onClick={() => handleModalClick('exitModal')}
              type='primary'
              disabled={disabled || criticalTransactionLoading || networkLayer === 'L1'}
              style={{maxWidth: 'none'}}
            >
              <ArrowBack/>SLOW EXIT
            </Button>
          </div>
        </div>

        <div className={[styles.box, networkLayer === 'L1' ? styles.dim : styles.active].join(' ')}>
          <div className={styles.header}>
            <div className={styles.title}>
              <span>Balance on Childchain</span>
              <span>OMGX</span>
            </div>
              <div
                onClick={()=>handleModalClick('transferModal')}
                className={[styles.transfer, networkLayer === 'L1' ? styles.disabled : ''].join(' ')}
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
       
      </div>

    </div>
  );

}

export default React.memo(Account);
