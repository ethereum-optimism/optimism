import React from 'react';
import { connect } from 'react-redux';
import { isEqual} from 'lodash';
import Button from 'components/button/Button';
import { closeBid } from 'actions/buyAction';
import { openModal, setModalData } from 'actions/uiAction';
import { getToken } from 'actions/tokenAction';

import moment from 'moment';

import * as styles from './HistoryBox.module.scss';
import { accMul } from 'util/calculation';

class BidHistoryBox extends React.Component {

  constructor(props) {

    super(props);

    this.state = {
      data: this.props.data,
      bidID: this.props.data.bidID,
    }

  }

  componentDidUpdate(prevState) {
    const { data } = this.props;
    if (!isEqual(prevState.data, data)) {
      this.setState({ data, bidID: data.bidID });
    }
  }

  handleCloseBid() {
    const { bidID } = this.state;
    this.props.dispatch(closeBid(bidID));
  }

  async handleAcceptSeller () {
    
    const { data } = this.state;
    const { childchain } = this.props.balance;

      // agree amount is how many tokens buyer will receive
      const agreeAmount = data.bidAcceptDetails.agreeAmount;
      // exchangeRate = buyer's itemToReceive / itemToSend
      const agreeExchangeRate = data.bidAcceptDetails.agreeExchangeRate;
      // buyer receives Token === seller sends token
      const buyerItemToReceiveCurrency = data.bidAcceptDetails.currencyA;
      // buyer sends Token ===  seller receives token
      const buyerItemToSendCurrency = data.bidAcceptDetails.currencyB;
      // buyer item Object
      let buyerItemToSend = '';
      childchain.forEach(element => {
        if (element.currency === buyerItemToSendCurrency) {
          buyerItemToSend = element;
        }
      })

      const buyerItemToReceive = await getToken(buyerItemToReceiveCurrency);

      //get ready to trigger swap
      const cMD = {
        UUID: data.bidAcceptDetails.UUID,
        agreeAmount,
        agreeExchangeRate,
        buyerItemToReceive,
        buyerItemToSend,
        sellerItemToSend: buyerItemToReceive,
        sellerItemToReceive: buyerItemToSend,
        bidID: data.bidID,
        itemID: data.bidAcceptDetails.itemID,
        address: data.bidAcceptDetails.sellerAddress,
        type: 'buyerAccept'
      }

      this.props.dispatch(setModalData('confirmationModal', cMD));
      this.props.dispatch(openModal('confirmationModal'));
  }

  render() {

    const { 
      data,
      bidID,
    } = this.state;

    let sellerAddress = '';
    let bidOrderStatus = <span className={styles.statusGreen}>{data.bidAcceptStatusString}</span>;

    let bottomInfoBar = <div className={styles.bottomGreenInfoContainer}>{data.bidDetails} {data.bidConvRate}</div>;

    if (data.bidAcceptStatusString !== 'Open') {
      // seller address
      let sellerWallet = data.bidAcceptDetails.sellerAddress;
      sellerAddress = <div className={styles.title + " " + styles.bottomGreenInfoContainer}>SELLER:{sellerWallet}</div>;
    }

    let buyerSettlement = <div></div>
    if (data.bidAcceptDetails && data.bidAcceptDetails.swapStatus === 'Close') {
      buyerSettlement = 
      <div className={styles.bottomGreenInfoContainer}>
        SETTLEMENT: Sent {accMul(data.bidAcceptDetails.agreeAmount, data.bidAcceptDetails.agreeExchangeRate)} {data.itemToSendSymbol}. 
        Received {data.bidAcceptDetails.agreeAmount} {data.itemToReceiveSymbol}
      </div>
    }

    return (
    <div className={styles.mainContainer}>
      <div 
          className={[
            styles.container,
            (data.bidAcceptStatusString === 'Seller Accepted' && 
            !data.recentApprovedSwap.includes(bidID)) ? styles.pulsate : '',
          ].join(' ')}
      >
        <div 
          className={[
            styles.topInfoContainer,
            (data.bidAcceptStatusString === 'Seller Accepted' && 
            !data.recentApprovedSwap.includes(bidID))? styles.pulsate : '',
          ].join(' ')}
        >

          {/* Left part */}
          <div className={styles.topLeftContainer}>
            <div className={styles.line}>{bidID} STATUS:{' '}{bidOrderStatus}</div>
            <div className={styles.line}>{moment.unix(data.timestamp).format('lll')}</div>
          </div>

          {/* Right part - the close button */}
          <div className={styles.topRightContainer}>
          {data.bidAcceptStatusString === 'Seller Accepted' && 
          !data.recentApprovedSwap.includes(bidID) &&
            <div className={styles.buttonContainer}>
              <Button 
                type='primary'
                size='small'
                disabled={data.buyerApproveSwapStatus === true}
                loading={data.buyerApproveSwapStatus === true}
                onClick={()=>{this.handleAcceptSeller()}}
              >
                ACCEPT
              </Button>
            </div>
          }
          {["Open", "Seller Accepted"].includes(data.bidAcceptStatusString) &&
            <div className={styles.buttonContainer}>
              <Button 
                type='primary'
                size='small'
                disabled={data.recentApprovedSwap.includes(bidID)}
                onClick={()=>{this.handleCloseBid()}}
              >
                CLOSE BID
              </Button>
            </div>
          }
          </div>

        </div>

        {buyerSettlement}
        {bottomInfoBar}
        {sellerAddress}
      </div>
    </div>
    )
  }
}

const mapStateToProps = state => ({
  balance: state.balance,
});

export default connect(mapStateToProps)(BidHistoryBox);