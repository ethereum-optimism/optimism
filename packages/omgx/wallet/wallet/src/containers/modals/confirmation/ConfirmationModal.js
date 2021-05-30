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

import React from 'react';
import { connect } from 'react-redux';
import BN from 'bignumber.js';

import { closeModal } from 'actions/uiAction';
import { acceptBid } from 'actions/sellAction';
import { acceptSellerSwap } from 'actions/buyAction';
import { approveErc20 } from 'actions/networkAction';
import { openAlert } from 'actions/uiAction';

import Button from 'components/button/Button';
import Modal from 'components/modal/Modal';

import { accDiv, accMul } from 'util/calculation';
import { powAmount } from 'util/amountConvert';

import networkService from 'services/networkService';

import * as styles from './ConfirmationModal.module.scss';

class ConfirmationModal extends React.Component {

  constructor(props) {
    super(props);
    
    const { childchain } = this.props.balance;

    this.state = {
      loading: this.props,
      open: this.props.open,
      cMD: this.props.ui.cMD,
      approvedAllowance: 0,
      neededAllowance: 0,
      balances: childchain.reduce((acc, cur) => {
        acc[cur.symbol] = cur.amount;
        return acc;
      }, {}),
    }
  }

  async componentDidUpdate(prevState) {
    const { childchain } = this.props.balance;
    const { open, loading } = this.props;
    const { cMD } = this.props.ui;

    if (prevState.open !== open) {
      this.setState({ open });
    }

    if (prevState.ui.cMD !== cMD) {
      this.setState({ cMD });
      await this.checkAllowance();
    }

    if (prevState.loading !== loading) {
      this.setState({ loading });
    }

    if (prevState.balance.childchain !== childchain) {
      this.setState({ 
        balances: childchain.reduce((acc, cur) => {
          acc[cur.symbol] = cur.amount;
          return acc;
        }, {}) 
      });
    }
  }

  async checkAllowance() {
    const { cMD } = this.props.ui;
    const { tokenList } = this.props;
    if (cMD.type === "sellerAccept") {
      const approvedAllowance = await networkService.checkAllowance(
        cMD.sellerItemToSend.currency, 
        networkService.AtomicSwapAddress
      );
      // It has the security issue! Need to fix!
      const decimals = tokenList[cMD.sellerItemToSend.currency.toLowerCase()].decimals;
      const neededAllowance = powAmount(cMD.sellerItemToSendAmount, decimals);
      this.setState({ approvedAllowance, neededAllowance });
    }

    if (cMD.type === "buyerAccept") {
      const approvedAllowance = await networkService.checkAllowance(
        cMD.buyerItemToSend.currency, 
        networkService.AtomicSwapAddress
      );
      const decimals = tokenList[cMD.buyerItemToSend.currency].decimals;
      const neededAllowance = powAmount(accMul(cMD.agreeAmount, cMD.agreeExchangeRate), decimals);
      this.setState({ approvedAllowance, neededAllowance });
    }

  }

  async handleApprove() {
    const { cMD } = this.props.ui;
    const { neededAllowance } = this.state;

    if (cMD.type === "sellerAccept") {
      const res = await this.props.dispatch(approveErc20(
        neededAllowance,
        cMD.sellerItemToSend.currency,
        networkService.AtomicSwapAddress,
      ));
      if (res) {
        this.props.dispatch(openAlert('ERC20 approval submitted.'));
        await this.checkAllowance();
      }
    }

    if (cMD.type === "buyerAccept") {
      const allowance = new BN(neededAllowance).times(10).toString();
      const res = await this.props.dispatch(approveErc20(
        allowance,
        cMD.buyerItemToSend.currency,
        networkService.AtomicSwapAddress,
      ));
      if (res) {
        this.props.dispatch(openAlert('ERC20 approval submitted.'));
        await this.checkAllowance();
      }
    }
  }

  async handleConfirm() {
    const { cMD } = this.props.ui;

    if (cMD.type === "sellerAccept") {
      this.props.dispatch(acceptBid(cMD))
    }

    if (cMD.type === "buyerAccept") {
      this.props.dispatch(acceptSellerSwap(cMD))
    }

    this.props.dispatch(closeModal('confirmationModal'));
  }

  handleClose() {
    this.props.dispatch(closeModal('confirmationModal'));
  }

  render() {
    const { 
      open, cMD, 
      approvedAllowance, neededAllowance, 
      loading,
      balances,
    } = this.state;

    if (typeof cMD === 'undefined') return null;

    const exchangeRate = 'This is an exchange rate of ' + accDiv(1, cMD.agreeExchangeRate).toFixed(2) + ' ' + cMD.sellerItemToSend.symbol + ' per ' + cMD.sellerItemToReceive.symbol;

    let enoughBalance = false;
    if (cMD.type === "sellerAccept") {
      let balanceSendToken = balances[cMD.sellerItemToSend.symbol].toString();
      if (new BN(balanceSendToken).gte(new BN(neededAllowance))) enoughBalance = true;
    }
    if (cMD.type === "buyerAccept") {
      let balanceSendToken = balances[cMD.buyerItemToSend.symbol].toString();
      if (new BN(balanceSendToken).gte(new BN(neededAllowance))) enoughBalance = true;
    }

    return (
      <Modal open={open}>
        
        <h2>Notice</h2>

        {cMD.type === "sellerAccept" &&
          <div className={styles.disclaimer}>
            You propose to send {cMD.agreeAmount} {cMD.sellerItemToSend.symbol}.<br/>
            You will receive {accMul(cMD.agreeAmount, cMD.agreeExchangeRate)} {cMD.sellerItemToReceive.symbol}.<br/>
            {exchangeRate}.<br/>
            The {cMD.sellerItemToSend.symbol} will be sent to:<br/>{cMD.address}
          </div>
        }

        {cMD.type === "buyerAccept" &&
          <div className={styles.disclaimer}>
            You are about to send {accMul(cMD.agreeAmount, cMD.agreeExchangeRate)} {cMD.buyerItemToSend.symbol}.<br/> 
            You will receive {cMD.agreeAmount} {cMD.buyerItemToReceive.symbol}.<br/> 
            {exchangeRate}.<br/>
            The {cMD.buyerItemToSend.symbol} will be sent to:<br/>{cMD.address}
          </div>
        }
        
        {!enoughBalance &&
          <div className={styles.disclaimer}>
            You don't have enough balance to cover the swap. Please add more tokens.
          </div>
        }

        {new BN(approvedAllowance).gte(new BN(neededAllowance)) &&
          <div className={styles.buttons}>
            <Button
              onClick={()=>{this.handleClose()}}
              type='outline'
              className={styles.button}
            >
              CANCEL
            </Button>
            <Button
              onClick={()=>{this.handleConfirm()}}
              type='primary'
              className={styles.button}
              disabled={!enoughBalance}
            >
              CONFIRM
            </Button>
          </div>        
        }

        {new BN(approvedAllowance).lt(new BN(neededAllowance)) &&
          <>
            <div className={styles.disclaimer}>
              To swap {cMD.agreeAmount} {cMD.sellerItemToSend.symbol}, 
              you first need to approve this. 
              Click below to submit an approval transaction.
            </div>
            <div className={styles.buttons}>
              <Button
                onClick={()=>{this.handleClose()}}
                type='outline'
                className={styles.button}
              >
                CANCEL
              </Button>
              <Button
                onClick={()=>{this.handleApprove()}}
                type='primary'
                className={styles.button}
                disabled={loading['APPROVE/CREATE'] || !enoughBalance}
              >
                Approve
              </Button>
            </div>  
          </>      
        }

      </Modal>
    )

  }
}

const mapStateToProps = state => ({
  ui: state.ui,
  tokenList: state.tokenList,
  loading: state.loading,
  balance: state.balance,
});

export default connect(mapStateToProps)(ConfirmationModal);