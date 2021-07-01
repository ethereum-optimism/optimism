/*
  Varna - A Privacy-Preserving Marketplace
  Copyright (C) 2021 Enya Inc. Palo Alto, CA

  This program is free software: you can redistribute it and/or modify
  it under the terms of the GNU General Public License as published by
  the Free Software Foundation, either version 3 of the License, or
  (at your option) any later version.

  This program is distributed in the hope that it will be useful,
  but WITHOUT ANY WARRANTY; without even the implied warranty of
  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
  GNU General Public License for more details.

  You should have received a copy of the GNU General Public License
  along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import React from 'react';
import { connect } from 'react-redux';
import { orderBy, isEqual } from 'lodash';

import {
  isBidOpenOrClosed,
  getBidBuyer,
  get_number_of_items_on_varna, 
  listBid,
  getBidMakingTasks,
  getBidAcceptData,
} from 'actions/buyAction';

import { 
  startBuyTask, 
} from 'actions/buyTaskAction';

import Button from 'components/button/Button';
import Pager from 'components/pager/Pager';
import BidHistoryBox from "components/history/BidHistoryBox";
import PriceTickerBox from 'components/history/PriceTickerBox';
import AutoComplete from 'components/autocomplete/AutoComplete';

import Submarine from 'components/submarine/Submarine';

import { openModal } from 'actions/uiAction';

import { bidsSlice } from 'util/transactionSort';
import { arrayToCoin } from 'util/coinConvert';
import { accMul } from 'util/calculation';

import * as styles from 'containers/Varna.module.scss';

const PER_PAGE = 4;
const POLL_INTERVAL = process.env.REACT_APP_POLL_INTERVAL;

class Buyer extends React.Component {

  constructor(props) {

    super(props);
    
    const { 
      transaction,
    } = this.props;
    
    const { 
      FHEseed 
    } = this.props.login;

    const { 
      masterBidList,
      numberOfItemsOnVarna,
      
      // encrypt bid for buyer
      encryptBidForBuyerLoad,
      generateOfferLoad, 

      // start transaction
      configureBidToPlasmaLoad, 

      // list your active tasks
      bidMakingTasks, 
      bidMakingTasksLoad,
      bidMakingTasksError,

      // close bid
      closeBidError,

      // seller accept swap
      bidAcceptData,

      // buyer approve swap
      buyerApproveSwap,
      buyerApproveSwapLoad,

    } = this.props.buy;

    const { 
      taskStatus, 
      taskStatusError 
    } = this.props.buyTask;

    const {
      swapMetaData,
      swappedBid,
    } = this.props.swap;

    const { 
      childchain
    } = this.props.balance;

    const {
      tokenList
    } = this.props;

    this.state = {

      masterBidList,
      FHEseed,
      
      // Item
      itemToSend: {},
      itemToSendAmount: '',
      itemToReceive: {},
      itemToReceiveAmount: '',
      buyerExchangeRate: '',

      // Balance
      childchain,

      numberOfItemsOnVarna,
      generateOfferLoad, 
      encryptBidForBuyerLoad,
      configureBidToPlasmaLoad, 

      // Transaction History
      transactions: orderBy(transaction, i => i.blockNumber, 'desc'),

      // Page
      page: 1,

      //Bidding task list
      bidMakingTasks, 
      bidMakingTasksLoad,
      bidMakingTasksError,

      //Bid preparation
      taskStatus,
      // upload bids to seller
      taskStatusError,

      //buyer status panel and progress indicator
      //local to this screen
      bidsMade: 0, 
      bidsActive: 0, 
      bidsToDo: 0,
      bidsTD: 0,
      bidsTDprogress: 0, 
      //TD = to download

      bidDecryptionCompleted: false, // status flag for UI coordination
      bidDisplayList: {},

      closeBidError,
      buyerApproveSwap,
      buyerApproveSwapLoad,
      
      // swap data
      swapMetaData,
      swappedBid,

      // Current list of tokens
      tokenList,

      // seller accept swap
      bidAcceptData,
    }
  }

  componentDidMount() {
    const { masterBidList } = this.props.buy;
    // quick start
    const openBidIDList = Object.keys(masterBidList);
    this.props.dispatch(getBidMakingTasks(/*rescanPlasma =*/ false));
    this.props.dispatch(getBidAcceptData(openBidIDList));

    // interval
    this.intervalService = setInterval(
      () => {
        this.intervalServiceAction();
      }, POLL_INTERVAL
    )

    this.props.dispatch(get_number_of_items_on_varna());
    //Update numberOfItemsOnVarna for status display purposes

    this.getBidsAndShowStats();

  }

  componentWillUnmount() {
    clearInterval(this.intervalService);
  }

  intervalServiceAction() {
    const { masterBidList } = this.props.buy;
    const openBidIDList = Object.keys(masterBidList);
    this.props.dispatch(isBidOpenOrClosed());
    this.props.dispatch(getBidMakingTasks(/*rescanPlasma =*/ false)); /*no need to do a full rescan in this moment*/
    this.props.dispatch(getBidAcceptData(openBidIDList));
  }

  componentDidUpdate(prevState) {

    const { 

      masterBidList,
      numberOfItemsOnVarna, 

      //offer generation
      generateOfferLoad, 
      encryptBidForBuyerLoad,
      uploadBuyerBidAndStatusLoad, 

      //bid generation
      configureBidToPlasmaLoad, 

      //list of your active tasks
      bidMakingTasks, 
      bidMakingTasksLoad,
      bidMakingTasksError,

      // close bid
      closeBidError,
      // buyer approve swap
      buyerApproveSwap,
      buyerApproveSwapLoad,
      // seller accept swap
      bidAcceptData,
    } = this.props.buy;
    
    //this all relates to the list of tasks - basically if there 
    //are any items that you have not yet bid on
    const { 
      taskStatus, 
      taskStatusBidID, 
      taskStatusError
    } = this.props.buyTask;

    //your Plasma transactions
    const { 
      tokenList,
    } = this.props;

    const {
      swapMetaData,
      swappedBid,
    } = this.props.swap;

    const { 
      childchain
    } = this.props.balance;

    if (prevState.balance.childchain !== childchain) {
      this.setState({ childchain });
    }

    if (prevState.tokenList !== tokenList) {
      this.setState({ tokenList });
      const addedTokenCurrency = Object.keys(tokenList).filter(
        i => !Object.keys(prevState.tokenList).includes(i)
      );
      if (tokenList[addedTokenCurrency].symbol !== 'Not found') {
        this.handleItemToReceive(tokenList[addedTokenCurrency].symbol);
      }
    }

    if (!isEqual(prevState.buy.masterBidList, masterBidList)) {
      this.setState({ masterBidList });
      this.getBidsAndShowStats();
    }

    //configure a bid, encrypt, and write to Plasma and AWS S3
    if (prevState.buy.configureBidToPlasmaLoad !== configureBidToPlasmaLoad) {
      this.setState({ configureBidToPlasmaLoad });
      if(prevState.buy.configureBidToPlasmaLoad && configureBidToPlasmaLoad === false) {
        //Nice, we just configured a bid
        //So we update everything and the task list
        this.handleScanNow(); 
      }
    }

    if (prevState.buyTask.taskStatus !== taskStatus) {
      if(prevState.buyTask.taskStatus === 'run' && taskStatus === 'stop') {
        //oh good - just completed a bid calculation and submission
        //update the local task manager
        let bidID = taskStatusBidID;
        let tL = bidMakingTasks;
        if(typeof tL[bidID] !== 'undefined' && tL[bidID]['waitingAskOffer'] > 0) {
          tL[bidID]['waitingAskOffer'] = tL[bidID]['waitingAskOffer'] - 1;
          tL[bidID]['sentAskOffer'] = tL[bidID]['sentAskOffer'] + 1;
        }
        this.setState({ bidMakingTasks: tL });
        //refresh the local progress numbers
        this.getBidsAndShowStats();
      }
      this.setState({ taskStatus });
    }

    if (prevState.buyTask.taskStatusError !== taskStatusError) {
      this.setState({ taskStatusError });
      if (taskStatusError[taskStatusBidID] !== prevState.buyTask.taskStatusError[taskStatusBidID] && 
          taskStatusError[taskStatusBidID] !== null
        ) {
            this.make_a_bid();
      }
    }

    if (prevState.buy.numberOfItemsOnVarna !== numberOfItemsOnVarna) {
      this.setState({ numberOfItemsOnVarna });
    }

    if (prevState.buy.generateOfferLoad !== generateOfferLoad) {
      this.setState({ generateOfferLoad });
    }

    if (prevState.buy.encryptBidForBuyerLoad !== encryptBidForBuyerLoad) {
      this.setState({ encryptBidForBuyerLoad });
    }

    if (prevState.buy.uploadBuyerBidAndStatusLoad !== uploadBuyerBidAndStatusLoad) {
      this.setState({ uploadBuyerBidAndStatusLoad });
    }

    if (!isEqual(prevState.buy.bidAcceptData, bidAcceptData)) {
      this.setState({ bidAcceptData });
    }

    if (prevState.buy.bidMakingTasksLoad !== bidMakingTasksLoad) {
      this.setState({ bidMakingTasksLoad });
    }

    if (prevState.buy.bidMakingTasksError !== bidMakingTasksError) {
      if (bidMakingTasksError === false) {
        if (taskStatus !== 'run') {
          this.make_a_bid();
        }
      }
    }

    if (prevState.buy.bidMakingTasks !== bidMakingTasks) {
      this.setState({ bidMakingTasks });
      this.getBidsAndShowStats(); //refresh the local progress numbers right now
    }

    if (prevState.buy.closeBidError !== closeBidError) {
      this.setState({ closeBidError });
      this.props.dispatch(isBidOpenOrClosed());
      this.props.dispatch(getBidMakingTasks(/*rescanPlasma =*/ false));
    }

    if (!isEqual(prevState.buy.buyerApproveSwapLoad, buyerApproveSwapLoad)) {
      this.setState({ buyerApproveSwapLoad });
      this.getBidsAndShowStats();
    }

    if (!isEqual(prevState.buy.buyerApproveSwap, buyerApproveSwap)) {
      this.setState({ buyerApproveSwap });
      this.getBidsAndShowStats();
    }

    if (!isEqual(prevState.swap.swapMetaData, swapMetaData)) {
      this.setState({ swapMetaData });
      this.getBidsAndShowStats();
    }

    if (!isEqual(prevState.swap.swappedBid, swappedBid)) {
      this.setState({ swappedBid });
    }
  }

  handleItemToSend(e) {
    const { tokenList } = this.props;
    Object.values(tokenList).forEach(tokenInfo => {
      if (tokenInfo.symbol === e) {
        this.setState({ 
          itemToSend: tokenInfo,
        });
      }
    })
  }

  handleItemToReceive(e) {
    const { tokenList } = this.props;
    Object.values(tokenList).forEach(tokenInfo => {
      if (tokenInfo.symbol === e) {
        this.setState({ 
          itemToReceive: tokenInfo,
        });
      }
    })
  }

  /**********************************************/
  /***** The largest number we can support ******/
  /**************** 14266335233 *****************/
  /**********************************************/
  handleAmountToSwap(event) {
    //for the buyer, this is the number of tokens they want to send
    let splitArray = event.target.value.split(".");
    if (splitArray.length === 2) {
      if (splitArray[1].length < 5) {
        this.setState({ itemToReceiveAmount: event.target.value });
      }
    } else {
      this.setState({ itemToReceiveAmount: event.target.value });
    }
  }

  /**********************************************/
  /***** The largest number we can support ******/
  /**************** 14266335233 *****************/
  /**********************************************/
  handleBuyerExchangeRate(event) {
    let splitArray = event.target.value.split(".");
    if (splitArray.length === 2) {
      if (splitArray[1].length < 5) {
        this.setState({ buyerExchangeRate: event.target.value });
      }
    } else {
      this.setState({ buyerExchangeRate: event.target.value });
    }
  }

  //list a new bid
  handleSubmit() {

  const { 
    itemToReceive, 
    itemToReceiveAmount,
    itemToSend,
    buyerExchangeRate,
    FHEseed, 
  } = this.state;

  const amountToSend = accMul(buyerExchangeRate, itemToReceiveAmount);
  console.log("need to send:", amountToSend)

  // get itemToReceiveAmount and itemToReceive / itemToSend 
  // it's easy for us to compare the exchange rate and determine the
  // final agree amount

  this.props.dispatch(listBid(
    itemToReceive, 
    itemToReceiveAmount, 
    itemToSend, 
    buyerExchangeRate, 
    FHEseed,
  ));

  }

  async getBidsAndShowStats() {

    const { 
      FHEseed
    } = this.state;

    const { 
      masterBidList,
      buyerApproveSwap,
      buyerApproveSwapLoad,
      bidMakingTasks,
      bidAcceptData,
    } = this.props.buy;

    let bidsToDo = 0;
    let bidsMade = 0;
    let bidsActive = 0;
    let bidsDownloaded = 0;

    let bdl = {};

    // no bids found
    if (Object.keys(masterBidList).length === 0) {
      this.setState({ bidDecryptionCompleted: true });
    }
    
    for (const [bidID] of Object.entries(masterBidList)) {

      let thisBid = masterBidList[bidID];
      if(thisBid.status === 'active') {
        if(typeof bidMakingTasks !== 'undefined') {
          if(typeof bidMakingTasks[bidID] !== 'undefined') {
            bidsToDo = bidsToDo + bidMakingTasks[bidID]['waitingAskOffer'];
            bidsMade = bidsMade + bidMakingTasks[bidID]['sentAskOffer'];
          }
        }

        //ok, this bid is open and can probably be decrypted
        //we only care about active bids

        // Add bid timestamp, if needed
        // Only need to do this once - it's the listing time so will not change
        if(!thisBid.bidTimestamp) {
          //add the timestamp
          thisBid.bidTimestamp = masterBidList[bidID].createdAt;
        }

        //console.log("thisBid:",thisBid)
        //count it for the numerical stats of active bids
        bidsActive = bidsActive + 1;

        //This is updated by
        //this.props.dispatch(updateBuyerAcceptBidMetaData(sortedData.buyerAcceptBidMetaRaw));
        const bidAcceptDetails = bidAcceptData[bidID];

        //Not sure what this is doing
        if(typeof bidAcceptDetails === 'undefined' ? true : bidAcceptDetails.length === 0) {
          //bid is open, not accepted
          thisBid.bidAcceptDetails = null;
          thisBid.bidAcceptStatusString = 'Open';
        } else {
          thisBid.bidAcceptDetails = orderBy(bidAcceptDetails, 'createdAt')[0]; //first to accept wins
          if (thisBid.bidAcceptDetails.swapStatus === 'Open') {
            thisBid.bidAcceptStatusString = 'Seller Accepted';
          }
          if (thisBid.bidAcceptDetails.swapStatus === 'Close') {
            thisBid.bidAcceptStatusString = 'Swapped';
          }
          if (thisBid.bidAcceptDetails.swapStatus === 'Abort') {
            thisBid.bidAcceptStatusString = 'Aborted By Seller';
          }
        }

        if ( masterBidList[bidID].downloaded && masterBidList[bidID].downloadError ) {

          thisBid.bidDetailsString = 'DECRYPTION FAILED - PASSWORD?';

        } else if (masterBidList[bidID].downloaded && !masterBidList[bidID].decrypted) {

          thisBid.bidDetailsString = 'Downloaded; Decrypting';
          //console.log("Decrypting bid:",bidID)

        } else if (masterBidList[bidID].downloaded && masterBidList[bidID].decrypted) {
          
          //ok, we are all set - we have the bidinfo cleartext
          const bIC = masterBidList[bidID].bidInfoClear;

          let itemToReceive       = arrayToCoin(bIC.slice(0, 4));
          let itemToSend          = arrayToCoin(bIC.slice(4, 8));
          let itemToReceiveAmount = bIC[8] / 100000;
          let buyerExchangeRate   = bIC[9] / 100000;
          let itemToSendAmount    = accMul(buyerExchangeRate, itemToReceiveAmount);
          let sellerExchangeRate  = 1 / buyerExchangeRate;

          //for display in the history box
          thisBid.bidConvRate = 
          `(${sellerExchangeRate.toFixed(5)} ${itemToReceive}/${itemToSend}; ${buyerExchangeRate.toFixed(5)} ${itemToSend}/${itemToReceive})`

          thisBid.bidDetailsString = 
            `Bid ${itemToSendAmount} ${itemToSend} for ${itemToReceiveAmount.toFixed(5)} ${itemToReceive}`

          //console.log("All done with bid:",bidID)

          bidsDownloaded = bidsDownloaded + 1;

          thisBid.itemToSend = itemToSend;
          thisBid.itemToReceive = itemToReceive;
        } else if (masterBidList[bidID].loading) {

          thisBid.bidDetailsString = 'Download in Progress';

        } else if (!masterBidList[bidID].loading && !masterBidList[bidID].downloaded) {
          // console.log(masterBidList[bidID]);

          // console.log("Downloading bid:",bidID)
          thisBid.bidDetailsString = 'Starting Download';
          this.props.dispatch(getBidBuyer(bidID, FHEseed))

        }

        let buyerApproveSwapStatus = false;
        buyerApproveSwapStatus = buyerApproveSwapLoad[bidID];

        bdl[bidID] = {
          bidID, 
          bidAcceptStatusString: thisBid.bidAcceptStatusString, 
          bidAcceptDetails: thisBid.bidAcceptDetails, 
          bidDetails: thisBid.bidDetailsString,
          bidConvRate: thisBid.bidConvRate,
          bidClearText: thisBid.bidInfoClear, 
          timestamp: thisBid.bidTimestamp,
          buyerApproveSwapStatus,
          recentApprovedSwap: buyerApproveSwap, 
          itemToSendSymbol: thisBid.itemToSend,
          itemToReceiveSymbol: thisBid.itemToReceive
        }

        // console.log(bidID, bdl);
      }
    }
    
    this.setState({
      bidsTD: bidsActive,
      bidsTDprogress: bidsDownloaded,
      bidDecryptionCompleted: bidsActive === bidsDownloaded ? true : false,
      bidsToDo, 
      bidsMade, 
      bidsActive, 
      bidDisplayList: bdl //show everything nicely
    })

  }

  /*********************************************/
  /* Update the bid todo list ******************/
  /*********************************************/
  handleScanNow() {
    //Update bidOpenOrClosed - which of your bids are open or closed?
    //a bid can be either active or deactive
    this.props.dispatch(isBidOpenOrClosed());

    this.props.dispatch(getBidMakingTasks(/*rescanPlasma =*/ true));
  }

  /*********************************************/
  /*************** Make a bid ******************/
  /*********************************************/

  make_a_bid() {

    const { 
      masterBidList,
      bidMakingTasks,
      bidAcceptData,
    } = this.props.buy;

    for (var anID of Object.keys(masterBidList)) {
      if (typeof bidMakingTasks[anID] === 'undefined') {
        //to be safe
        continue;
      }

      //this is an important early test of datastructure integrity
      if (bidMakingTasks[anID].hasOwnProperty("waitingAskOffer") !== true) {
        continue;
      }

      if (typeof masterBidList[anID] === 'undefined') {
        continue; //can't decrypt - continue
      }

      if (bidMakingTasks[anID].waitingAskOffer < 1) {
        continue; //no work to be done here
      }

      if(masterBidList[anID].decrypted) {
        const bidAcceptDetails = bidAcceptData[anID];
        if (!bidAcceptDetails) {
          //console.log("Yay - all good - compute");
          //console.log(masterBidList[anID].bidInfoClear);
          this.props.dispatch( startBuyTask( masterBidList[anID].bidInfoClear, anID ) );
        }
        break; // just do one for now
      }
    }
  }

  handleAddToken() {
    this.props.dispatch(openModal('addNewTokenModal', false))
  }

  render() {

    const { 
      itemToReceive, 
      itemToReceiveAmount,
      itemToSend, 
      buyerExchangeRate,
      page,
      numberOfItemsOnVarna,

      // Loading status
      generateOfferLoad, 
      encryptBidForBuyerLoad,
      uploadBuyerBidAndStatusLoad, 
      configureBidToPlasmaLoad,

      //buyer status panel and progress indicator
      bidsMade,
      bidsActive,
      bidsToDo,
      bidsTD,
      bidsTDprogress,

      //list of bids needed to be computed
      bidMakingTasksLoad,

      //run || stop from the FHE code
      bidDecryptionCompleted,
      bidDisplayList,

      tokenList,

      // transactions
      transactions,
    } = this.state;

    let buttonText = 'BID';
    if (generateOfferLoad || encryptBidForBuyerLoad) buttonText = 'ENCRYPTING';
    if (uploadBuyerBidAndStatusLoad) buttonText = 'UPLOADING';
    if (configureBidToPlasmaLoad) buttonText = 'TRANSFERRING';

    //needed for the total number of pages so we can display Page X of Y

    let totalNumberOfPages = Math.ceil(Object.keys(bidDisplayList).length / PER_PAGE);
    //console.log("totalNumberOfPages:",totalNumberOfPages)

    //if totalNumberOfPages === 0, set to one so we don't get the strange "page 1 of 0" display
    if (totalNumberOfPages === 0) totalNumberOfPages = 1;

    const paginatedBids = bidsSlice(page, PER_PAGE, bidDisplayList);
    // console.log("paginatedBids:",paginatedBids)

    function progressTextBuild(Text, display=true) {
      return (
        <span className={styles.statusGreen} style={{marginTop:5, marginBottom: 5, marginLeft: 0}}>
          {Text}
        </span>
      )
    }

    let progressText = progressTextBuild(`No tasks running`);

    if (bidDecryptionCompleted === false) {
      let pt = 'Downloading/decrypting ' + bidsTDprogress + ' of ' + bidsTD;
      progressText = progressTextBuild(pt);
    } else if (bidMakingTasksLoad) {
      progressText = progressTextBuild(`Scanning for items`);
    } else if (bidsToDo > 0) {
      progressText = progressTextBuild(`New items! Bidding now...`);
    }

    const transactionPending = transactions.some(i => i.status === 'Pending');

    return (

<div className={styles.Varna}>
<div className={styles.VarnaCube}>

  <div className={styles.VarnaCubeTopTwo}>

    <div className={styles.VarnaInput}>

    <div className={styles.Entry}>

    <div style={{flex: 2}}>
      <h5 style={{marginBottom: '2px', marginTop: '8px'}}>Which token would you like?</h5>
      <AutoComplete 
        placeholder="e.g. OMG" 
        selectionList={tokenList}
        excludeItem={itemToSend.symbol}
        updateValue={(e)=>{this.handleItemToReceive(e)}}
        passValue={itemToReceive}
      />
      <Button
        onClick={()=>{this.handleAddToken()}}
        style={{marginTop: '0px', justifyContent: 'center', maxWidth: 'none'}}
        type='primary'
        size='tiny'
      >
        ADD OTHER TOKEN
      </Button>
    </div>

    <div style={{flex: 1, paddingLeft: '10px'}}>
      <h5 style={{marginBottom: '2px', marginTop: '8px'}}>How many?</h5>
      <input
        className={styles.Input}
        value={itemToReceiveAmount}
        type="number"
        placeholder="e.g. 10"
        onChange={event => {this.handleAmountToSwap(event)}}
      />
    </div>
    </div>

      {itemToReceive.currency &&
        <div className={styles.Verify}>
          Token address: {itemToReceive.currency}<br/>
          PLEASE CHECK CAREFULLY!
        </div>
      }

      <h5 style={{marginBottom: 2, marginTop: 10}}>How will you pay?</h5>
      <AutoComplete 
        placeholder="e.g. ETH"
        selectionList={tokenList}
        excludeItem={itemToReceive.symbol}
        updateValue={(e)=>{this.handleItemToSend(e)}}
      />

      {itemToSend.currency &&
        <div className={styles.Verify}>
          Token address: {itemToSend.currency}<br/>
          PLEASE CHECK CAREFULLY!
        </div>
      }

      {(!itemToReceive.symbol || !itemToSend.symbol) &&
        <h5 style={{marginBottom: 3, marginTop: 10}}>Exchange rate</h5>
      }

      {itemToReceive.symbol && itemToSend.symbol &&
        <h5 style={{marginBottom: 3, marginTop: 10}}>Exchange rate ({itemToSend.symbol} per {itemToReceive.symbol})</h5>
      }

      <input
        type="number"
        className={styles.Input}
        value={buyerExchangeRate}
        placeholder="e.g. 1.12"
        onChange={event => {this.handleBuyerExchangeRate(event)}}
        disabled={
          uploadBuyerBidAndStatusLoad || 
          configureBidToPlasmaLoad || 
          itemToReceive === '' || 
          itemToSend === ''
        }
      />

      {itemToReceive.symbol && !itemToReceiveAmount && !itemToSend.symbol && !buyerExchangeRate &&
        <div className={styles.Summary} >
          Want {itemToReceive.symbol}<br/>
        </div>
      }

      {itemToReceive.symbol && itemToReceiveAmount && !itemToSend.symbol && !buyerExchangeRate &&
        <div className={styles.Summary} >
          Want {itemToReceiveAmount} {itemToReceive.symbol}<br/>
        </div>
      }

      {itemToReceive.symbol && itemToReceiveAmount && itemToSend.symbol && !buyerExchangeRate &&
        <div className={styles.Summary} >
          Want {itemToReceiveAmount} {itemToReceive.symbol}<br/>
          Pay in {itemToSend.symbol}
        </div>
      }

      {itemToReceive.symbol && itemToReceiveAmount && itemToSend.symbol && buyerExchangeRate &&
        <div className={styles.Summary} >
          Want {itemToReceiveAmount} {itemToReceive.symbol}<br/>
          Offering {buyerExchangeRate} {itemToSend.symbol} per {itemToReceive.symbol}<br/>
          Total cost of swap: {accMul(buyerExchangeRate, itemToReceiveAmount)} {itemToSend.symbol}
        </div>
      }

      <Button
        onClick={()=>{this.handleSubmit()}}
        style={{flex: 0, maxWidth: 500, marginTop: 10, height: 20}}
        size='small'
        type='primary'
        loading={
          uploadBuyerBidAndStatusLoad || 
          configureBidToPlasmaLoad || 
          encryptBidForBuyerLoad ||
          generateOfferLoad
        }
        disabled={ 
          !itemToReceive.symbol || 
          itemToReceiveAmount === '' || 
          !itemToSend.symbol ||
          buyerExchangeRate === '' || 
          transactionPending
        }
      >
        {buttonText}
      </Button>
    </div>

    <div className={styles.Submarine}>
      <Submarine
        progressText={progressText}
        bidDecryptionCompleted={bidDecryptionCompleted}
        numberOfItemsOnVarna={numberOfItemsOnVarna}
        bidsMade={bidsMade}
        bidsActive={bidsActive}
        bidsToDo={bidsToDo}
      />
    </div>

  </div>

  <div 
    className={styles.LongHistory}
  >
    <Pager
      currentPage={page}
      totalPages={totalNumberOfPages}
      label={'My Bids'}
      isLastPage={paginatedBids.length < PER_PAGE}
      onClickNext={()=>this.setState({page:page+1})}
      onClickBack={()=>this.setState({page:page-1})}
    />

    {!paginatedBids.length && (
      <div className={styles.Disclaimer}>No bid history.</div>
    )}

    {paginatedBids.map((data, index) => {
      return (
        <BidHistoryBox data={data} key={index} />
      )
    })}
  </div>

  </div>

  <div className={styles.PriceContainer} >
    <PriceTickerBox />
  </div>

</div>
    )
  }
}

const mapStateToProps = state => ({ 
  login: state.login,
  buy: state.buy,
  buyTask: state.buyTask,
  balance: state.balance,
  transaction: state.transaction,
  hashcast: state.hashcast,
  swap: state.swap,
  tokenList: state.tokenList,
});

export default connect(mapStateToProps)(Buyer);