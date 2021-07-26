import React from 'react';
import { connect } from 'react-redux';
import { logAmount } from 'util/amountConvert';
import { isEqual } from 'lodash';

import { openModal } from 'actions/uiAction';
import Button from 'components/button/Button';

import ExpandMoreIcon from '@material-ui/icons/ExpandMore';

import * as styles from './listAccount.module.scss';

class ListAccount extends React.Component {
  
  constructor(props) {
    
    super(props);
    
    const { token, chain, networkLayer, disabled } = this.props;

    this.state = {
      token,
      chain,
      dropDownBox: false,
      dropDownBoxInit: true,
      networkLayer,
      disabled
    }
  }
  
  componentDidUpdate(prevState) {

    const { token, chain, networkLayer, disabled } = this.props;

    if (!isEqual(prevState.token, token)) {
      this.setState({ token });
    }

    if (!isEqual(prevState.chain, chain)) {
      this.setState({ chain });
    }

    if (!isEqual(prevState.networkLayer, networkLayer)) {
      this.setState({ networkLayer });
    }

    if (!isEqual(prevState.disabled, disabled)) {
      this.setState({ disabled });
    }

  }

  handleModalClick(modalName, token, fast) {
    this.props.dispatch(openModal(modalName, token, fast))
  }

  render() {

    const { 
      token, 
      chain,
      dropDownBox, 
      dropDownBoxInit,
      networkLayer,
      disabled
    } = this.state;

    const enabled = (networkLayer === chain) ? true : false

    return (
      <div className={styles.ListAccount}>
        
        <div 
          className={styles.topContainer}
          disabled={true} 
          onClick={()=>{
              this.setState({ 
                dropDownBox: !dropDownBox, 
                dropDownBoxInit: false 
              })
          }}
        >
          <div className={styles.Table1}>
            <div className={styles.BasicText}>{token.symbol}</div>
          </div>
          <div className={styles.Table2}>
            <div className={styles.BasicLightText}>{`${logAmount(token.balance, 18, 2)}`}</div>
          </div>
          <div className={styles.Table3}>
            {chain === 'L1' && 
              <div className={enabled ? styles.LinkText : styles.LinkTextOff}>Deposit</div>
            }
            {chain === 'L2' && 
              <div className={enabled ? styles.LinkText : styles.LinkTextOff}>Transact</div>
            }
            <ExpandMoreIcon 
              className={enabled ? styles.LinkButton : styles.LinkButtonOff} 
            />
          </div>
        </div>

        {/*********************************************/
        /**************  Drop Down Box ****************/
        /**********************************************/
        }
        <div 
          className={dropDownBox ? 
            styles.dropDownContainer: dropDownBoxInit ? styles.dropDownInit : styles.closeDropDown}
        >
          
          {!enabled && chain === 'L1' && 
            <div className={styles.boxContainer}>
              <div className={styles.underRed}>MetaMask is set to L2. To transact on L1, please change the chain in MetaMask to L1.</div>
            </div>
          }

          {!enabled && chain === 'L2' && 
            <div className={styles.boxContainer}>
              <div className={styles.underRed}>MetaMask is set to L1. To transact on L2, please change the chain in MetaMask to L2.</div>
            </div>
          }

          {enabled && chain === 'L1' && 
          <>
            <div className={styles.boxContainer}>
              <Button
                onClick={()=>{this.handleModalClick('depositModal', token, false)}}
                type='secondary'
                disabled={disabled}
                style={{width: '100px', padding: '6px', borderRadius: '5px'}}
              >
                DEPOSIT
              </Button>
            </div>
            <div className={styles.boxContainer}>
              <Button
                onClick={()=>{this.handleModalClick('depositModal', token, true)}}
                type='primary'
                disabled={disabled}
                style={{width: '120px', padding: '6px', borderRadius: '5px'}}
              >
                FAST DEPOSIT
              </Button>
            </div>
          </>
          }

          {enabled && chain === 'L2' && 
          <>
            <div className={styles.boxContainer}>
              <Button
                onClick={()=>{this.handleModalClick('transferModal', token, false)}}
                type='primary'
                disabled={disabled}
                style={{width: '90px', padding: '6px',borderRadius: '5px'}}
              >
                TRANSFER
              </Button>
            </div>
            <div className={styles.boxContainer}>
              <Button
                onClick={()=>{this.handleModalClick('exitModal', token, false)}}
                type='secondary'
                disabled={disabled}
                style={{width: '100px', padding: '6px',borderRadius: '5px'}}
              >
                7 DAY EXIT
              </Button>
            </div>
            <div className={styles.boxContainer}>
              <Button
                onClick={()=>{this.handleModalClick('exitModal', token, true)}}
                type='primary'
                disabled={disabled}
                style={{width: '90px', padding: '6px',borderRadius: '5px'}}
              >
                FAST EXIT
              </Button>
            </div>
          </>
          }

        </div>

      </div>
    )
  }
}

const mapStateToProps = state => ({ 
  //login: state.login,
  //sell: state.sell,
  //sellTask: state.sellTask,
  //buy: state.buy,
});

export default connect(mapStateToProps)(ListAccount);