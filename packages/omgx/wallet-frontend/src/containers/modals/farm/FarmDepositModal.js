import React from 'react';
import { connect } from 'react-redux';
import { isEqual } from 'lodash';
import BN from 'bignumber.js';

import { closeModal, openAlert, openError } from 'actions/uiAction';
import { getFarmInfo } from 'actions/farmAction';

import Button from 'components/button/Button';
import Modal from 'components/modal/Modal';
import Input from 'components/input/Input';
import { logAmount, powAmount } from 'util/amountConvert';

import networkService from 'services/networkService';

import * as styles from './Farm.module.scss';

class FarmDepositModal extends React.Component {
  constructor(props) {
    super(props);

    const { open } = this.props;
    const { stakeToken } = this.props.farm;

    this.state = {
      open,
      stakeToken,
      stakeValue: '',
      // allowance
      approvedAllowance: 0,
      // loading
      loading: false,
    }
  }

  async componentDidUpdate(prevState) {

    const { open } = this.props;
    const { stakeToken } = this.props.farm;

    if (prevState.open !== open) {
      this.setState({ open });
    }

    if (!isEqual(prevState.farm.stakeToken, stakeToken)) {
      let approvedAllowance = powAmount(10, 50);
      // There is no need to check allowance for depositing ETH
      if (stakeToken.currency !== networkService.L1_ETH_Address) {
        approvedAllowance = await networkService.checkAllowance(
          stakeToken.currency,
          stakeToken.LPAddress
        );
      }
      this.setState({ approvedAllowance, stakeToken });
    }

  }

  getMaxTransferValue () {
    const { stakeToken } = this.state;
    // const transferingBalanceObject = (stakeToken.L1orL2Pool === 'L1LP' ? layer1Balance : layer2Balance)
    //   .find(i => i.currency === stakeToken.currency);
    // if (!transferingBalanceObject) {
    //   return;
    // }
    return logAmount(stakeToken.balance, stakeToken.decimals);
  }

  handleClose() {
    this.props.dispatch(closeModal("farmDepositModal"));
  }

  async handleApprove() {
    
    const { stakeToken, stakeValue } = this.state;

    this.setState({ loading: true })

    let approveTX

    if (stakeToken.L1orL2Pool === 'L2LP') {
      approveTX = await networkService.approveERC20_L2LP(
        powAmount(stakeValue, 18),
        stakeToken.currency,
      )
    } else if (stakeToken.L1orL2Pool === 'L1LP') {
      approveTX = await networkService.approveERC20_L1LP(
        powAmount(stakeValue, 18),
        stakeToken.currency,
      )
    }

    console.log("stakeToken.LPAddress:",stakeToken.LPAddress)
    //0x2C12649A5A4FC61F146E0a3409f3e4c7FbeD15Dc
    //for trying to stake TST

    if (approveTX) {
      this.props.dispatch(openAlert("Amount was approved"))
      let approvedAllowance = powAmount(10, 50)
      // There is no need to check allowance for depositing ETH
      if (stakeToken.currency !== networkService.L1_ETH_Address) {
        approvedAllowance = await networkService.checkAllowance(
          stakeToken.currency,
          stakeToken.LPAddress
        )
      }
      console.log("approvedAllowance:",approvedAllowance)
      this.setState({ approvedAllowance, loading: false })
    } else {
      this.props.dispatch(openError("Failed to approve amount"))
      this.setState({ loading: false })
    }
  }

  async handleConfirm() {
    const { stakeToken, stakeValue } = this.state;

    this.setState({ loading: true });

    const addLiquidityTX = await networkService.addLiquidity(
      stakeToken.currency,
      stakeValue,
      stakeToken.L1orL2Pool
    );
    if (addLiquidityTX) {
      this.props.dispatch(openAlert("Your liquidity was added"));
      this.props.dispatch(getFarmInfo());
      this.setState({ loading: false, stakeValue: '' });
      this.props.dispatch(closeModal("farmDepositModal"));
    } else {
      this.props.dispatch(openError("Failed to add liquidity"));
      this.setState({ loading: false, stakeValue: '' });
    }
  }

  render() {
    
    const {
      open,
      stakeToken, 
      stakeValue,
      approvedAllowance,
      loading,
    } = this.state;

    return (

      <Modal open={open}>
        
        <h2>Stake {`${stakeToken.symbol}`}</h2>

        <Input
          placeholder={`Amount to stake`}
          value={stakeValue}
          type="number"
          onChange={i=>{this.setState({stakeValue: i.target.value})}}
          unit={stakeToken.symbol}
          maxValue={this.getMaxTransferValue()}
        />

        {Number(stakeValue) > Number(this.getMaxTransferValue()) &&
          <div className={styles.disclaimer}>
            You don't have enough {stakeToken.symbol} to stake.
          </div>
        }

        {(new BN(approvedAllowance).gte(powAmount(stakeValue, 18)) || stakeValue === '') &&
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
              disabled={Number(this.getMaxTransferValue()) < Number(stakeValue) || stakeValue === '' || !stakeValue}
              loading={loading}
            >
              STAKE!
            </Button>
          </div>
        }

        {new BN(approvedAllowance).lt(new BN(powAmount(stakeValue, 18))) &&
          <>
            <div className={styles.disclaimer}>
              To stake {stakeValue} {stakeToken.symbol},
              you first need to approve this amount.
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
                loading={loading}
                disabled={Number(this.getMaxTransferValue()) < Number(stakeValue)}
              >
                APPROVE AMOUNT
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
  farm: state.farm,
  balance: state.balance,
});

export default connect(mapStateToProps)(FarmDepositModal);