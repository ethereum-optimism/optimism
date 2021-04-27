import React from 'react';
import { connect } from 'react-redux';
import moment from 'moment';
import { CircularProgress } from '@material-ui/core';

import Button from 'components/button/Button';
import Tooltip from 'components/tooltip/Tooltip';

import { isItemOpenOrClosed, deleteItem  } from 'actions/sellAction';

import { arrayToCoin } from 'util/coinConvert';

import * as styles from './HistoryBox.module.scss';

class ItemHistoryBox extends React.Component {
  constructor(props) {
    super(props);

    const { password } = this.props.login;
    const { 
      itemOpenOrClosed, itemOpenList, itemOpenOrClosedLoadIndicator,
      downloadItemCiphertext, downloadItemCiphertextLoad, downloadItemCiphertextError,
      decryptedItem, decryptedItemLoad, decryptedItemError,
      decryptedItemCacheError,
      deleteItemLoad, deleteItemError,
    } = this.props.sell;

    this.state = {
      password,
      itemID: this.props.itemID,
      itemOpenOrClosed, itemOpenList, itemOpenOrClosedLoadIndicator,
      // Download data
      downloadItemCiphertext, 
      downloadItemCiphertextLoad, 
      downloadItemCiphertextError,
      // Decrypt data using FHE
      decryptedItem, 
      decryptedItemLoad, 
      decryptedItemError,
      // Decrypt data using AES
      decryptedItemCacheError,
      // Cancel order
      deleteItemLoad, 
      deleteItemError,
      // load hashcast data
      loadHashcastData: false,
    }
  }

  componentDidUpdate(prevState) {
    const { password } = this.props.login;
    const { itemID } = this.props;
    const { 
      itemOpenOrClosed, itemOpenList, itemOpenOrClosedLoadIndicator,
      downloadItemCiphertext, downloadItemCiphertextLoad, downloadItemCiphertextError,
      decryptedItem, decryptedItemLoad, decryptedItemError,
      decryptedItemCacheError,
      deleteItemLoad, deleteItemError,
    } = this.props.sell;

    if (prevState.login.password !== password) {
      this.setState({ password });
    }

    if (prevState.itemID !== itemID) {
      this.setState({ itemID });
    }

    if (prevState.sell.itemOpenOrClosed !== itemOpenOrClosed) {
      this.setState({ itemOpenOrClosed });
    }

    if (prevState.sell.itemOpenList !== itemOpenList) {
      this.setState({ itemOpenList });
    }

    if (prevState.sell.itemOpenOrClosedLoadIndicator !== itemOpenOrClosedLoadIndicator) {
      this.setState({ itemOpenOrClosedLoadIndicator });
    }

    if (prevState.sell.downloadItemCiphertext !== downloadItemCiphertext) {
      this.setState({ downloadItemCiphertext });
    }

    if (prevState.sell.downloadItemCiphertextLoad !== downloadItemCiphertextLoad) {
      this.setState({ downloadItemCiphertextLoad });
    }

    if (prevState.sell.downloadItemCiphertextError !== downloadItemCiphertextError) {
      this.setState({ downloadItemCiphertextError });
    }

    /**************************************************************************/
    /*********************** FHE decrypt downloaded data **********************/
    /**************************************************************************/
    if (prevState.sell.decryptedItem !== decryptedItem) {
      this.setState({ decryptedItem });
    }

    if (prevState.sell.decryptedItemLoad !== decryptedItemLoad) {
      this.setState({ decryptedItemLoad });
    }

    if (prevState.sell.decryptedItemError !== decryptedItemError) {
      this.setState({ decryptedItemError });
    }
    /**************************************************************************/

    /**************************************************************************/
    /************************* AES decrypt cache data *************************/
    /**************************************************************************/
    if (prevState.sell.decryptedItemCacheError !== decryptedItemCacheError) {
      this.setState({ decryptedItemCacheError });
    }
    /**************************************************************************/

    if (prevState.sell.deleteItemLoad !== deleteItemLoad) {
      this.setState({ deleteItemLoad });
    }

    if (prevState.sell.deleteItemError !== deleteItemError) {
      this.setState({ deleteItemError });

      if (deleteItemError[itemID] === false) {
        // Refresh the status
        this.props.dispatch(isItemOpenOrClosed());
      }
    }
  }

  handleCancelOrder() {
    const { itemID } = this.state;
    this.props.dispatch(deleteItem(itemID));
  }

  render() {
    const { 
      itemID, 
      //
      itemOpenOrClosed, itemOpenOrClosedLoadIndicator,
      // Load status
      downloadItemCiphertextLoad, decryptedItemLoad, deleteItemLoad,
      // Error
      downloadItemCiphertextError, decryptedItemError,
      // Decrypt data
      decryptedItem, 
      // AES decrypt status
      decryptedItemCacheError,
    } = this.state;

    let j_itemOpenOrClose = true;
    let itemToSendAmountRemain = 0;

    let swapSuccessDisplay = false;

    let bottomInfoBar = <div></div>;

    if (j_itemOpenOrClose) {
      if (
        decryptedItem[itemID] || 
        decryptedItemError[itemID] === false ||
        downloadItemCiphertextError[itemID] === false
      ) {
        if (downloadItemCiphertextLoad[itemID]) {
          bottomInfoBar = <div className={styles.bottomGreenInfoContainer}>Downloading</div>;
        }
        if (decryptedItemLoad[itemID]) {
          bottomInfoBar = <div className={styles.bottomGreenInfoContainer}>Decrypting</div>;
        }
        if (decryptedItemError[itemID] === 404) {
          bottomInfoBar = <div className={styles.bottomRedInfoContainer}>We can't access your item/token information - wrong password?</div>
        }
        if (decryptedItem[itemID]) {
          const itemToSend = arrayToCoin(decryptedItem[itemID].slice(0, 4));
          const itemToReceive = arrayToCoin(decryptedItem[itemID].slice(4, 8));
          const itemToSendAmount = decryptedItem[itemID][8] / 100000;
          const sellerExchangeRate = decryptedItem[itemID][9] / 100000;
          itemToSendAmountRemain = decryptedItem[itemID][10] / 100000;
          const itemToReceiveAmount = (itemToSendAmount * sellerExchangeRate).toFixed(5);

          bottomInfoBar = <div className={styles.bottomGreenInfoContainer}>{`Send ${itemToSendAmount} ${itemToSend}. Receive ${itemToReceiveAmount} ${itemToReceive}. Remaining: ${itemToSendAmountRemain} ${itemToSend}.`}</div>
        }
      }

      if (decryptedItemCacheError[itemID]) {
        bottomInfoBar = <div className={styles.bottomGreenInfoContainer}>DECRYPTION FAILED - PASSWORD?</div>
      }
    }

    let itemStatus = <span className={styles.statusGreen}></span>;

    if (itemOpenOrClosedLoadIndicator) { 
      itemStatus =
        <span className={styles.statusGreen}>
          Loading
          <Tooltip >
            <div className={styles.loading}>
              <CircularProgress size={14} color='inherit' />
            </div>
          </Tooltip>
        </span>;
    } else if (!j_itemOpenOrClose) { 
      itemStatus = <span className={styles.statusRed}>CLOSED</span>;
    } else if (swapSuccessDisplay && itemToSendAmountRemain > 0) {
      itemStatus = <span className={styles.statusGreen}>PARTIALLY FILLED</span>;
    } else if (swapSuccessDisplay && itemToSendAmountRemain === 0) {
      itemStatus = <span className={styles.statusGreen}>FILLED</span>;
    } else { 
      itemStatus = <span className={styles.statusGreen}>OPEN</span>;
    }

    let cancelButtonStatus = false;
    if (deleteItemLoad[itemID]) 
      cancelButtonStatus = true;
    else 
      cancelButtonStatus = false;

    return (
      <div className={styles.container}>

        <div className={styles.topInfoContainer}>

          {/* Left part */}
          <div className={styles.topLeftContainer}>
            <div className={styles.line}>ID: {itemID}</div>
            <div className={styles.line}>STATUS: {itemStatus}</div>
            <div className={styles.line}>{moment.unix(itemOpenOrClosed[itemID].createdAt / 1000).format('D MMM YYYY LTS')}</div>
          </div>

          {/* Right part */}

          <div className={styles.topRightContainer}>
            {/* Close ask button */}
            {!itemOpenOrClosedLoadIndicator && j_itemOpenOrClose && 
              <div>
                <Button 
                  type='primary'
                  size='small'
                  onClick={()=>{this.handleCancelOrder()}}
                  disabled={cancelButtonStatus}
                  style={cancelButtonStatus ? {width: 100}:{}}
                >
                  {cancelButtonStatus ? "UNLISTING..." : "UNLIST ITEM"}
                </Button>
              </div>
            }
          </div>
        </div>

        {bottomInfoBar}

      </div>
    )
  }
}

const mapStateToProps = state => ({ 
  login: state.login,
  sell: state.sell,
  sellTask: state.sellTask,
  balance: state.balance,
  ui: state.ui,
  hashcast: state.hashcast,
  swap: state.swap,
});

export default connect(mapStateToProps)(ItemHistoryBox);