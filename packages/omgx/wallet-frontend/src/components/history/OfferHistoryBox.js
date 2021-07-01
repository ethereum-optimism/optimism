import React from 'react';
import { connect } from 'react-redux';
import { orderBy, isEqual } from 'lodash';
import moment from 'moment';

import Pager from 'components/pager/Pager';
import Button from 'components/button/Button';
import { accDiv } from 'util/calculation';

import { arrayToCoin } from 'util/coinConvert';
import { openError, openModal, setModalData } from 'actions/uiAction'

import * as styles from './OfferHistoryBox.module.scss';

const PER_PAGE = 7;

class BidOfferHistoryBox extends React.Component {

  constructor(props) {
    super(props);

    const { transaction } = this.props;
    
    const { FHEseed, AESKey } = this.props.login;
    const { 
      itemOpenOrClosed, 
      decryptedItem, 
      acceptBidLoad, 
      acceptBidError, 
      sellerAcceptBidData,
    } = this.props.sell;

    const { bidOfferData, bidDecryptionWaitingList } = this.props.sellTask;

    this.state = {
      FHEseed, AESKey,
      itemOpenOrClosed,
      decryptedItem,
      bidOfferData, 
      bidDecryptionWaitingList,
      acceptBidLoad,
      acceptBidError,
      // load hashcast data
      loadHashcastData: {},
      page: 1,
      // transaction data
      transaction,
      // seller accepts bid status data
      sellerAcceptBidData,
    }
  }

  componentDidUpdate(prevState) {
    const { transaction } = this.props;

    const { FHEseed, AESKey } = this.props.login;
    const { 
      itemOpenOrClosed, 
      decryptedItem, 
      acceptBidLoad, 
      acceptBidError, 
      sellerAcceptBidData,
    } = this.props.sell;

    const { bidOfferData, bidDecryptionWaitingList } = this.props.sellTask;

    if (prevState.transaction !== transaction) {
      this.setState({ transaction });
    }

    if (prevState.login.FHEseed !== FHEseed) {
      this.setState({ FHEseed });
    }

    if (prevState.login.AESKey !== AESKey) {
      this.setState({ AESKey });
    }

    if (prevState.sell.itemOpenOrClosed !== itemOpenOrClosed) {
      this.setState({ itemOpenOrClosed });
    }

    if (prevState.sell.decryptedItem !== decryptedItem) {
      this.setState({ decryptedItem });
    }

    if (prevState.sell.acceptBidLoad !== acceptBidLoad) {
      this.setState({ acceptBidLoad });
    }

    if (prevState.sell.acceptBidError !== acceptBidError) {
      this.setState({ acceptBidError });
    }

    if (!isEqual(prevState.sell.sellerAcceptBidData, sellerAcceptBidData)) {
      this.setState({ sellerAcceptBidData });
    }

    if (prevState.sellTask.bidOfferData !== bidOfferData) {
      this.setState({ bidOfferData });
    }

    if (prevState.sellTask.bidDecryptionWaitingList !== bidDecryptionWaitingList) {
      this.setState({ bidDecryptionWaitingList });
    }
  }

  handleAcceptBid(itemID, bidID, address) {

    const { decryptedItem, bidOfferData, FHEseed, AESKey } = this.state;
    const { childchain } = this.props.balance;

    // The buyers are willing to pay more than the sellers' minimum
    // buyer data
    const buyerItemToReceiveAmount = bidOfferData[itemID][bidID].buyerItemToReceiveAmount;
    const buyerExchangeRate = bidOfferData[itemID][bidID].buyerExchangeRate;

    // get the seller's data
    const sellerItemToSendAmount = accDiv(decryptedItem[itemID][8], Math.pow(10, 5));
    const sellerExchangeRate = accDiv(decryptedItem[itemID][9], Math.pow(10, 5));
    // Security!!!!
    // It always returns the lower case
    const sellerItemToSendSymbol = arrayToCoin(decryptedItem[itemID].slice(0, 4));
    const sellerItemToReceiveSymbol = arrayToCoin(decryptedItem[itemID].slice(4, 8));
    const sellerItemToSendAmountRemain = accDiv(decryptedItem[itemID][10], Math.pow(10, 5));

    let sellerItemToSend = '', sellerItemToReceive = '';
    childchain.forEach(element => {
      if (element.symbol.toLowerCase() === sellerItemToSendSymbol.toLowerCase()) {
        sellerItemToSend = element;
      }
      if (element.symbol.toLowerCase() === sellerItemToReceiveSymbol.toLowerCase()) {
        sellerItemToReceive = element;
      }
    })

    if (buyerExchangeRate < sellerExchangeRate) {
      this.props.dispatch(openError("Exchange rate error! Please pick another one."));
      return;
    }

    let agreeAmount = 0;
    if (buyerItemToReceiveAmount > sellerItemToSendAmountRemain) {
      agreeAmount = sellerItemToSendAmountRemain;
    }  else {
      agreeAmount = buyerItemToReceiveAmount;
    }
    let agreeExchangeRate = buyerExchangeRate;

    //get ready to trigger swap
    const cMD = {
      itemID, 
      bidID,
      address, 
      sellerItemToSend,
      sellerItemToReceive,
      sellerItemToSendAmount, 
      sellerItemToSendAmountRemain,
      sellerExchangeRate,
      agreeAmount, 
      agreeExchangeRate, 
      FHEseed, 
      AESKey,
      type: 'sellerAccept',
    }

    this.props.dispatch(setModalData('confirmationModal', cMD));
    this.props.dispatch(openModal('confirmationModal'));
  }

  render () {
    const { 
      page, 
      itemOpenOrClosed, 
      bidOfferData, 
      acceptBidLoad, 
      acceptBidError, 
      decryptedItem,
      // transaction
      transaction,
      // seller accepts the bid
      sellerAcceptBidData
    } = this.state;

    let activeitemIDStatus = [];
    let timestampDictionary = {};
    
    Object.keys(itemOpenOrClosed).forEach(eachitemID => {
      if (itemOpenOrClosed[eachitemID].status === "active") {
        activeitemIDStatus.push({...itemOpenOrClosed[eachitemID], itemID: eachitemID});
        itemOpenOrClosed[eachitemID].bidOffer.forEach(bidData => {
          timestampDictionary[bidData.bidID] = bidData.timestamp;
        })
      }
    })

    const totalNumberOfPages = Math.ceil(activeitemIDStatus.length / PER_PAGE);

    const startingIndex = page === 1 ? 0 : ((page - 1) * PER_PAGE);
    const endingIndex = page * PER_PAGE;
    const paginatedActiveitemIDStatus = activeitemIDStatus.slice(startingIndex, endingIndex);

    const transactions = orderBy(transaction, i => i.blockNumber, 'desc');
    const transferPending = transactions.some(i => i.status === 'Pending');
    const transferPendingMeta = transactions.reduce((acc, cur) => {
      if (cur.status === 'Pending') acc.push(cur.metadata);
      return acc;
    } ,[]);

    return (
      <div style={{margin: 0, padding: 0}} >

      <Pager
        label={'Incoming Offers'}
        currentPage={page}
        totalPages={totalNumberOfPages}
        isLastPage={paginatedActiveitemIDStatus.length < PER_PAGE}
        onClickNext={()=>this.setState({page:page+1})}
        onClickBack={()=>this.setState({page:page-1})}
      />

      <div 
        className={styles.RtableWB} 
        style={{padding: 0, margin: 0}}
      >
        <div 
          className={styles.Rtable + ' ' + styles.RtableHeader}
          style={{marginTop: 0, paddingTop: 0}}
        >
          <div 
            className={styles.Rtable_cell}
            style={{width: '50px'}}
          >
            Item
          </div>
          <div 
            className={styles.Rtable_cell}
            style={{width: '50px', background: '#ff8080'}}
          >
            Bid
          </div>
          <div 
            className={styles.Rtable_cell}
            style={{width: '60px'}}
          >
            Bid Amount
          </div>
          <div 
            className={styles.Rtable_cell}
            style={{width: '60px', background: '#ff8080'}}
          >
            Settle Amount
          </div>
          <div 
            className={styles.Rtable_cell}
            style={{width: '120px'}}
          >
            Exchange Rate
          </div>
          <div 
            className={styles.Rtable_cell}
            style={{width: '100px', background: '#ff8080'}}
          >
            Time
          </div>
          <div className={styles.Rtable_cell}
          >
            Status/Action 
          </div>
        </div>

        {activeitemIDStatus !== {} && (
          paginatedActiveitemIDStatus.map((v,i)=>{
            if (v.status === "active") {
              let validBidData = [];
              if (Object.keys(bidOfferData).length !== 0) {
                if (bidOfferData[v.itemID]) {
                  Object.keys(bidOfferData[v.itemID]).forEach((bidID) => {
                    // Check whether ask was decrypted
                    if (decryptedItem[v.itemID]) {
                      // Remaining amount is larger than 0
                      if (decryptedItem[v.itemID][10] > 0) {
                        if (bidOfferData[v.itemID][bidID]) {
                          validBidData.push({ ...bidOfferData[v.itemID][bidID], bidID });
                        }
                      } else {
                        // Remaining amount is zero
                        // only display bids accepted by the seller
                        if (sellerAcceptBidData[v.itemID]) {
                          if (sellerAcceptBidData[v.itemID].filter(i=>i.bidID === bidID).length) {
                            if (bidOfferData[v.itemID][bidID]) {
                              validBidData.push({ ...bidOfferData[v.itemID][bidID], bidID });
                            }
                          }
                        }
                      }
                    }
                  })
                }
              }
              

              // sort the data
              const sellerExchangeRate = decryptedItem[v.itemID] ? accDiv(decryptedItem[v.itemID][9], Math.pow(10, 5)) : 0;
              //show bid id ER > or = 
              const filteredValidBidData = validBidData.filter(i => i.buyerExchangeRate >= sellerExchangeRate);
              const orderedValidBidData = orderBy(filteredValidBidData, 'buyerExchangeRate');

              return (

                <div 
                  className={styles.Rtable}
                  style={{marginTop: '2px'}}
                  key={i}
                >

                  {/*************************
                   ********** itemID *********
                   **************************/}
                  <div 
                    className={styles.Rtable_cell}
                    style={{width: '50px'}}
                  >
                    <div className={styles.Rtable_row_height}>{`${v.itemID.slice(0,6)}`}</div>
                  </div>

                  {/*************************
                   ********** bidID *********
                   **************************/}
                  <div 
                    className={styles.Rtable_cell}
                    style={{width: '50px', background: '#ff8080'}}
                  >
                    {bidOfferData[v.itemID] && orderedValidBidData.length !== 0 && (
                      orderedValidBidData.map((data,i) => {
                        return <div key={i} className={styles.Rtable_row_height}>{`${data.bidID.slice(0,6)}`}</div>
                      })
                    )}
                  </div>
                  
                  {/*************************
                   ********** Amount ********
                   **************************/}
                  <div 
                    className={styles.Rtable_cell}
                    style={{width: '60px'}}
                  >
                    {bidOfferData[v.itemID] && orderedValidBidData.length !== 0 && (
                      orderedValidBidData.map((data,i) => {
                        return (
                          <div key={i} className={styles.Rtable_row_height}>
                            {`${data.buyerItemToReceiveAmount}`}
                          </div>
                        )
                      })
                    )}
                  </div>
                  {/**************************************
                   ********** SETTLEMENT Amount **********
                   **************************************/}
                  <div 
                    className={styles.Rtable_cell}
                    style={{width: '60px', background: '#ff8080'}}
                  >
                    {bidOfferData[v.itemID] && orderedValidBidData.length !== 0 && (
                      orderedValidBidData.map((data,i) => {
                        // if (typeof swapMetaData[v.itemID.slice(0, 13)] !== 'undefined' && 
                        //   swapMetaData[v.itemID.slice(0, 13)].includes(data.bidID.slice(0, 13))
                        // ) {
                        //   return (
                        //     <div key={i} className={styles.Rtable_row_height}>
                        //        {`${Math.min(data.buyerItemToReceiveAmount, decryptedItem[v.itemID][8])}`}
                        //     </div>
                        //   )
                        // } else {
                          return <div key={i} className={styles.Rtable_row_height} />
                        // }
                      })
                    )}
                  </div>
                  {/**********************************
                   ********** Exchange Rate **********
                   ***********************************/}
                  <div 
                    className={styles.Rtable_cell}
                    style={{width: '120px'}}
                  >
                    {bidOfferData[v.itemID] && orderedValidBidData.length !== 0 && (
                      orderedValidBidData.map((data,i) => {
                        return (
                          <div 
                            key={i} 
                            className={styles.Rtable_row_height}
                          >
                            {`${data.buyerExchangeRate} ${arrayToCoin(decryptedItem[v.itemID].slice(4, 8))}/${arrayToCoin(decryptedItem[v.itemID].slice(0, 4))}`}
                          </div>
                        )
                      })
                    )}
                  </div>

                  {/*****************************
                   ********** TimeStamp *********
                   ******************************/}
                  <div 
                    className={styles.Rtable_cell}
                    style={{width: '100px', background: '#ff8080'}}
                  >
                  {bidOfferData[v.itemID] && orderedValidBidData.length !== 0 && (
                      orderedValidBidData.map((data,i) => {
                        const now = moment(new Date());
                        let timeDiff = Number(moment.duration(now.diff(moment(timestampDictionary[data.bidID]))).asMinutes());
                        let timeText = `${parseInt(timeDiff)}m ago`
                        if (timeDiff > 60) timeText = `${parseInt(timeDiff / 60)}h ago`
                        if (timeDiff > 60 * 24) timeText = `${parseInt(timeDiff / 60 / 24)}d ago`;
                        return <div key={i} className={styles.Rtable_row_height}>{timeText}</div>
                      })
                    )}
                  </div>

                  {/******************************
                   ************ Button ***********
                   ******************************/}
                  <div 
                    className={styles.Rtable_cell}
                  >
                    {bidOfferData[v.itemID] && orderedValidBidData.length !== 0 && (
                      orderedValidBidData.map((data,i) => {
                        // Successfully split the UTXO
                        let acceptBidButtonText = "Accept Bid";

                        // Loading status
                        if (acceptBidLoad[v.itemID]) {
                          if (acceptBidLoad[v.itemID][data.bidID]) {
                            //this is the wait message after step 2
                            acceptBidButtonText = 'Loading Swap Body';
                          }
                          if (acceptBidLoad[v.itemID][data.bidID] === false && 
                            acceptBidError[v.itemID][data.bidID] === false
                          ) {
                            //i've never seen this message come up, ever - can this code even be reached?
                            //no idea
                            acceptBidButtonText = 'Please be patient';
                          }
                        }

                        let sellerOpenCase = false, sellerAbortCase = false, buyerCloseCase = false;
                        let filteredSellerAgreement = sellerAcceptBidData[v.itemID] ? 
                          sellerAcceptBidData[v.itemID].filter(i => i.bidID === data.bidID) : [];
                        if (filteredSellerAgreement.length) {
                          if (filteredSellerAgreement[0].swapStatus === 'Open') {
                            sellerOpenCase = true;
                          }
                          if (filteredSellerAgreement[0].swapStatus === 'Abort') {
                            sellerAbortCase = true;
                          }
                          if (filteredSellerAgreement[0].swapStatus === 'Close') {
                            buyerCloseCase = true;
                          }
                        }

                        if (sellerOpenCase || buyerCloseCase || sellerAbortCase) {
                          /***********************************/
                          /********** Swap Finished **********/
                          /***********************************/
                          if (buyerCloseCase) {
                            return (
                              <div 
                                key={i} 
                                className={styles.Rtable_row_height}
                              >
                                All done: Swap Completed
                              </div>
                            )

                          /****************************************/
                          /********** Waiting the Buyer ***********/
                          /****************************************/
                          } else if (sellerOpenCase) {
                            return (
                              <div key={i} className={styles.cancelContainer}>
                                <div 
                                  className={styles.Rtable_row_height} 
                                >
                                  Waiting for Buyer
                                </div>
                                {/* <Button
                                  type='primary'
                                  size='tiny'
                                  className={styles.cancelButton}
                                  onClick={()=>{this.handleCancelOffer(v.itemID, data.bidID)}}
                                >
                                  Cancel
                                </Button> */}
                              </div>
                            )
                          /****************************************/
                          /*********** Abort the case *************/
                          /****************************************/
                          } else {
                            return (
                              <div 
                                key={i} 
                                className={styles.Rtable_row_height} 
                              >
                                Cancelled
                              </div>
                            )
                          }
    
                        // Bids are waiting to be accepted
                        } else if (data.address) {
                          // Good bids
                          return (
                            <div key={i} className={styles.Rtable_row_height}>
                              <Button
                                type='primary'
                                size='tiny'
                                onClick={()=>{this.handleAcceptBid(v.itemID, data.bidID, data.address)}}
                                disabled={
                                  (acceptBidLoad[v.itemID] ? acceptBidLoad[v.itemID][data.bidID] : false) || 
                                  transferPending
                                }
                                loading={
                                  (acceptBidLoad[v.itemID] ? acceptBidLoad[v.itemID][data.bidID] : false) || 
                                  transferPendingMeta.includes(`${v.itemID.slice(0, 12)}-SPLIT-${data.bidID.slice(0, 12)}`)
                                }
                                pulsate={true}
                                triggerTime={new Date()}
                              >
                                {acceptBidButtonText}
                              </Button>
                            </div>
                          )
                        // address is not found (can't do anything)
                        } else {
                          return (
                            <div key={i} className={styles.Rtable_row_height + styles.statusRed}>
                              Address not found
                            </div>
                          )
                        }
                      })
                    )}
                  </div>

                </div>
              )
            } else {
              return <div key={i}></div>
            }
          })
        )}
      </div>
      </div>
    )
  }
}

const mapStateToProps = state => ({ 
  login: state.login,
  sell: state.sell,
  sellTask: state.sellTask,
  balance: state.balance,
  transaction: state.transaction,
  ui: state.ui,
  swap: state.swap,
  hashcast: state.hashcast,
  rUTXOs: state.rUTXOs,
});

export default connect(mapStateToProps)(BidOfferHistoryBox);