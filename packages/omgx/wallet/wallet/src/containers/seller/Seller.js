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
import { orderBy, isEqual, findIndex } from 'lodash';

import networkService from 'services/networkService';

import Button from 'components/button/Button';
import ItemHistoryBox from "components/history/ItemHistoryBox";
import OfferHistoryBox from 'components/history/OfferHistoryBox';
import PriceTickerBox from 'components/history/PriceTickerBox';
import AutoComplete from 'components/autocomplete/AutoComplete';
import Pager from 'components/pager/Pager';

import { 
  listItem, 
  isItemOpenOrClosed, 
  loadItem, 
  findNextDecryptBiditemIDIndex,
  decryptItemCache,
  getSellerAcceptBidData,
} from 'actions/sellAction';

import { 
  getBidOffer, 
  updateBidDecryptionWaitingList,
  findBidDecryptWaitingList,
  decryptBidOfferCache,
} from 'actions/sellTaskAction';

import { itemsSlice } from 'util/transactionSort';
import { accMul } from 'util/calculation';
import { openError } from 'actions/uiAction';

import * as styles from 'containers/Varna.module.scss';

const PER_PAGE = 5;
const POLL_INTERVAL = process.env.REACT_APP_POLL_INTERVAL;

class Seller extends React.Component {

  constructor(props) {

    super(props);

    const { transaction } = this.props;

    const { FHEseed, AESKey } = this.props.login;

    const {
      generateKeyLoad, 
      uploadKeyLoad,
      configureItemToOMGXLoad, 
      configureItemToOMGXError,
      itemOpenList, 
      itemOpenOrClosed, 
      itemOpenOrClosedError,
      // download data from API 
      // use FHE to decrypt it
      currentDecryptItemID, 
      decryptedItem, 
      decryptedItemLoad, 
      decryptedItemError,
      // use AES to decrypt the cache data
      decryptedItemCache, // load the data when users log in
      decryptedItemCacheLoad,
      decryptedItemCacheError,
    } = this.props.sell;
    
    const { 
      currentDecryptBiditemID,
      currentDecryptBidBidID, 
      bidOfferData, 
      bidOfferDataError, 
      bidDecryptionWaitingList,
      // use AES to decrypt the cache data
      bidOfferDataCache,
      decryptBidOfferCacheError,
    } = this.props.sellTask;
    
    const { 
      rootchain, 
      childchain
    } = this.props.balance;

    const {
      tokenList
    } = this.props;
    

    this.state = {
      FHEseed, 
      AESKey,

      // item
      itemToSend: {},
      itemToSendAmount: '',
      itemToReceive: {},
      sellerExchangeRate: '',

      // generate keys
      generateKeyLoad,
      // upload keys
      uploadKeyLoad,
      // start transactions
      configureItemToOMGXLoad, 
      configureItemToOMGXError,
      // Balance
      rootchain, childchain,
      // Transaction History
      transactions: orderBy(transaction, i => i.blockNumber, 'desc'),
      // Page
      page: 1,
      // Valid buy request items
      itemOpenOrClosed, 
      itemOpenList, 
      itemOpenOrClosedError,
      // Download data from API 
      // Use FHE to decrypt it
      currentDecryptItemID, 
      decryptedItem, 
      decryptedItemLoad, 
      decryptedItemError,
      // Use AES to decrypt the cach data
      decryptedItemCache,
      decryptedItemCacheLoad,
      decryptedItemCacheError,
      // Bid data
      currentDecryptBiditemID,
      currentDecryptBidBidID, 
      bidOfferData, 
      bidOfferDataError, 
      bidDecryptionWaitingList,
      // Use AES to decrypt cach data
      bidOfferDataCache,
      decryptBidOfferCacheError,
      // token list
      tokenList,
    }

  }
  
  componentDidMount() {
    
    const { 
      // ask data
      itemOpenOrClosed,
      itemOpenList,
      // current running ask ID 
      currentDecryptItemID, 
      // ask data
      decryptedItem,
      decryptedItemCache,
      decryptedItemCacheError,
      // bid data
      bidOfferDataCache,
      decryptBidOfferCacheError,
      // Key
      FHEseed, 
      AESKey,
    } = this.state;

    // run this when the page is loaded
    this.props.dispatch(getSellerAcceptBidData(itemOpenList));

    // interval
    this.intervalService = setInterval(
      () => {
        this.intervalServiceAction();
      }, POLL_INTERVAL
    )

    // Continue the decrypted ask history work once users come back from anothe page
    // load Cache
    let nextDecryptBiditemIDIndex = findNextDecryptBiditemIDIndex(itemOpenList, currentDecryptItemID);
    if (itemOpenList.length > nextDecryptBiditemIDIndex && nextDecryptBiditemIDIndex >= 0) {
      this.props.dispatch(loadItem(itemOpenList[nextDecryptBiditemIDIndex], FHEseed, AESKey));
    }

    // Continue the decrypted bid history work once users come back from another page
    let bidDecryptionWaitingListOrgin = findBidDecryptWaitingList(itemOpenOrClosed, itemOpenList);
    if (bidDecryptionWaitingListOrgin.length !== 0) {
      this.props.dispatch(getBidOffer(bidDecryptionWaitingListOrgin, FHEseed, AESKey));
      this.props.dispatch(updateBidDecryptionWaitingList(bidDecryptionWaitingListOrgin));
    }

    // Continue to decrypt the item cache data
    if (Object.keys(decryptedItemCache).length) {
      this.props.dispatch(decryptItemCache(
        decryptedItem, 
        decryptedItemCache, 
        decryptedItemCacheError, 
        itemOpenList, 
        AESKey)
      );
    }

    // Continue to decrypte the bid's cache data
    if (Object.keys(bidOfferDataCache).length) {
      this.props.dispatch(decryptBidOfferCache(
        itemOpenOrClosed, 
        itemOpenList, 
        bidOfferDataCache, 
        decryptBidOfferCacheError, 
        AESKey
        )
      );
    }
  }

  componentWillUnmount() {
    clearInterval(this.intervalService);
  }

  intervalServiceAction() {
    const { itemOpenList } = this.props.sell;
    this.props.dispatch(isItemOpenOrClosed());
    this.props.dispatch(getSellerAcceptBidData(itemOpenList));
  }

  componentDidUpdate(prevState) {
    
    const { 
      FHEseed, 
      AESKey,
    } = this.props.login;
    
    const { 
      generateKeyLoad, 
      uploadKeyLoad,
      configureItemToOMGXLoad, 
      configureItemToOMGXError,
      itemOpenOrClosed, 
      itemOpenList, 
      itemOpenOrClosedError,
      // download data from API 
      // use FHE to decrypt it
      currentDecryptItemID, 
      decryptedItem, 
      decryptedItemLoad, 
      decryptedItemError,
    } = this.props.sell;

    const { 
      currentDecryptBiditemID ,currentDecryptBidBidID, bidOfferData, bidOfferDataError, 
      bidDecryptionWaitingList,
    } = this.props.sellTask;
    
    const { 
      rootchain, 
      childchain
    } = this.props.balance;
    
    const { 
      tokenList
    } = this.props;

    if (prevState.tokenList !== tokenList) {
      this.setState({ tokenList });
    }

    if (prevState.sell.generateKeyLoad !== generateKeyLoad) {
      this.setState({ generateKeyLoad });
    }

    if (prevState.sell.uploadKeyLoad !== uploadKeyLoad) {
      this.setState({ uploadKeyLoad });
    }

    if (prevState.sell.configureItemToOMGXLoad !== configureItemToOMGXLoad) {
      this.setState({ configureItemToOMGXLoad });
    }

    if (prevState.sell.configureItemToOMGXError !== configureItemToOMGXError) {
      this.setState({ configureItemToOMGXError });

      if (configureItemToOMGXError === false) {
        this.props.dispatch(isItemOpenOrClosed()).then(itemOpenOrClosedData => {
            if (Object.keys(itemOpenOrClosedData) !== 0) {
              this.props.dispatch(
                loadItem(Object.keys(itemOpenOrClosedData)[0], FHEseed, AESKey)
              );
            }
          }
        )
      }
    }

    if (prevState.sell.itemOpenOrClosed !== itemOpenOrClosed) {
      this.setState({ itemOpenOrClosed });

      /********************************/
      /** Load the new incoming bids **/
      /********************************/ 

      let newIncomeBids = [];
      if (Object.keys(prevState.sell.itemOpenOrClosed).length !== 0 && !isEqual(itemOpenOrClosed, prevState.sell.itemOpenOrClosed)) {
        let itemIDList = Object.keys(itemOpenOrClosed);

        for (let eachitemID of itemIDList) {
          // select the active itemID
          if (itemOpenOrClosed[eachitemID].status === 'active') {
            // sort bidID
            let sortedUpdatedBidOffer = orderBy(itemOpenOrClosed[eachitemID].bidOffer, 'timestamp', 'desc');
            for (let eachUpdatedBidOffer of sortedUpdatedBidOffer) {
              // only check the defined bidID
              if (typeof eachUpdatedBidOffer !== 'undefined') {
                const bidDecryptionWaitingObj = {itemID: eachitemID, bidOffer: [eachUpdatedBidOffer]};
                // if bidOffer has this itemID and bidID not in the waiting list
                if (bidOfferData[eachitemID]){
                  if (!bidOfferData[eachitemID][eachUpdatedBidOffer.bidID] && 
                    findIndex(bidDecryptionWaitingList, bidDecryptionWaitingObj) === -1
                    ) {
                    newIncomeBids.push(bidDecryptionWaitingObj);
                  }
                // bidOffer doesn't have this itemID and bidID is not in the waiting list
                } else if (findIndex(bidDecryptionWaitingList, bidDecryptionWaitingObj) === -1) {
                  newIncomeBids.push(bidDecryptionWaitingObj);
                }
              }
            }
          }
        }

        let bidDecryptionWaitingListTemp = JSON.parse(JSON.stringify(bidDecryptionWaitingList));
        bidDecryptionWaitingListTemp = [...newIncomeBids, ...bidDecryptionWaitingListTemp];

        if (bidDecryptionWaitingList.length === 0 && bidDecryptionWaitingListTemp.length !== 0) {
          this.props.dispatch(getBidOffer(bidDecryptionWaitingListTemp, FHEseed, AESKey));
        }

        this.props.dispatch(updateBidDecryptionWaitingList(bidDecryptionWaitingListTemp));
      }
    }

    if (prevState.sell.itemOpenList !== itemOpenList) {
      this.setState({ itemOpenList });
    }
    
    if (prevState.sell.itemOpenOrClosedError !== itemOpenOrClosedError) {
      this.setState({ itemOpenOrClosedError });
    }

    if (prevState.balance.rootchain !== rootchain) {
      this.setState({ rootchain });
    }

    if (prevState.balance.childchain !== childchain) {
      this.setState({ childchain });
    }

    /**************************************************************************/
    /*********************** FHE decrypt downloaded data **********************/
    /**************************************************************************/
    if (prevState.sell.currentDecryptItemID !== currentDecryptItemID) {
      this.setState({ currentDecryptItemID });
    }

    if (prevState.sell.decryptedItem !== decryptedItem) {
      this.setState({ decryptedItem });
    }

    if (prevState.sell.decryptedItemLoad !== decryptedItemLoad) {
      this.setState({ decryptedItemLoad });

      if (decryptedItemLoad[currentDecryptItemID] === false && 
        prevState.sell.decryptedItemLoad[currentDecryptItemID] !== false) {
        // load Cache
        let nextDecryptBiditemIDIndex = findNextDecryptBiditemIDIndex(itemOpenList, currentDecryptItemID);

        if (itemOpenList.length > nextDecryptBiditemIDIndex && nextDecryptBiditemIDIndex) {
          this.props.dispatch(loadItem(itemOpenList[nextDecryptBiditemIDIndex], FHEseed, AESKey));
        }
      }
    }

    if (prevState.sell.decryptedItemError !== decryptedItemError) {
      this.setState({ decryptedItemError });
    }
    /**************************************************************************/

    if (prevState.sellTask.currentDecryptBiditemID !== currentDecryptBiditemID) {
      this.setState({ currentDecryptBiditemID });
    }

    if (prevState.sellTask.currentDecryptBidBidID !== currentDecryptBidBidID) {
      this.setState({ currentDecryptBidBidID });
    }

    if (prevState.sellTask.bidOfferData !== bidOfferData) {
      this.setState({ bidOfferData });
    }

    if (prevState.sellTask.bidDecryptionWaitingList !== bidDecryptionWaitingList) {
      this.setState({ bidDecryptionWaitingList });
    }

    if (prevState.sellTask.bidOfferDataError !== bidOfferDataError) {
      this.setState({ bidOfferDataError });

      /*************************/
      /***** Remove task *******/
      /*************************/
      if (currentDecryptBiditemID !== null && currentDecryptBidBidID !== null && 
          bidOfferDataError[currentDecryptBiditemID][currentDecryptBidBidID] !== null &&
          typeof bidOfferDataError[currentDecryptBiditemID][currentDecryptBidBidID] !== 'undefined' &&
          bidDecryptionWaitingList.length !== 0
        ) {
          let bidDecryptionWaitingListTemp = JSON.parse(JSON.stringify(bidDecryptionWaitingList));
          let bidOffers = orderBy(bidDecryptionWaitingListTemp[0].bidOffer, 'timestamp', 'desc');
          
          bidOffers.shift();
          if (bidOffers.length === 0) {
            bidDecryptionWaitingListTemp.shift();
          } else {
            bidDecryptionWaitingListTemp[0].bidOffer = bidOffers;
          }

          if (bidDecryptionWaitingListTemp.length !== 0) {
            this.props.dispatch(getBidOffer(bidDecryptionWaitingListTemp, FHEseed, AESKey));
          }

          this.props.dispatch(updateBidDecryptionWaitingList(bidDecryptionWaitingListTemp));
      }
    }
  }


/**** 
 * Removed due to time
 * the point here is to only allow sellers to sell things they have on 
 * Child Chain 
****/

  handleItemToSend(e) {
    const { tokenList } = this.props;
    const itemToSend = Object.values(tokenList).filter(i => i.symbol === e)[0];
    this.setState({ itemToSend });
  }

  handleItemToReceive(e) {
    const { tokenList } = this.props;
    const itemToReceive = Object.values(tokenList).filter(i => i.symbol === e)[0];
    this.setState({ itemToReceive });
  }

  /**********************************************/
  /***** The largest number we can support ******/
  /*************** is 14266335233 ***************/
  /**********************************************/
  handleItemToSellAmount(event) {
    let splitArray = event.target.value.split(".");
    if (splitArray.length === 2) {
      if (splitArray[1].length < 5) {
        this.setState({ itemToSendAmount: event.target.value });
      }
    } else {
      this.setState({ itemToSendAmount: event.target.value });
    }
  }

  /**********************************************/
  /***** The largest number we can support ******/
  /*************** is 14266335233 ***************/
  /**********************************************/
  handleSellerExchangeRate(event) {
    let splitArray = event.target.value.split(".");
    if (splitArray.length === 2) {
      if (splitArray[1].length < 5) {
        this.setState({ sellerExchangeRate: event.target.value });
      }
    } else {
      this.setState({ sellerExchangeRate: event.target.value });
    }
  }

  async handleListItem() {

    const { 
      itemToSend, 
      itemToSendAmount, 
      itemToReceive, 
      sellerExchangeRate, 
      FHEseed, 
    } = this.state;

    const networkStatus = await networkService.confirmLayer('L2');
    if (networkStatus) {
      this.props.dispatch(listItem(
        itemToSend, 
        itemToSendAmount, 
        itemToReceive, 
        sellerExchangeRate, 
        FHEseed, 
      ));
    } else {
      this.props.dispatch(openError("Network Error! Please change the network to Layer 2."))
    }
  }

  render() {

    const { 
      itemToSend, 
      itemToSendAmount, 
      itemToReceive, 
      sellerExchangeRate,
      // load status
      generateKeyLoad, 
      uploadKeyLoad, 
      configureItemToOMGXLoad,
      page,
      transactions,
      // item list
      itemOpenList,
      // token list
      tokenList,
    } = this.state;

    /*** 
    A simple check to figure out if someone has OMG on Child Chain, so they can transact. 
    Minimally, they will need OMG to pay fees.
    ***/

    let buttonText = 'LIST';

    if (generateKeyLoad) buttonText = "ENCRYPTING";

    if (uploadKeyLoad) buttonText = "UPLOADING";
    
    if (configureItemToOMGXLoad) buttonText = "TRANSFERRING"

    const _transactions = transactions.filter(i => itemOpenList.includes(i.metadata));
    const paginatedItems = itemsSlice(page, PER_PAGE, itemOpenList);

    //needed for the total number of pages so we can display Page X of Y
    let totalNumberOfPages = Math.ceil(_transactions.length / PER_PAGE);

    //if totalNumberOfPages === 0, set to one so we don't get the strange "page 1 of 0" display
    if (totalNumberOfPages === 0) totalNumberOfPages = 1;

    return (

      <div className={styles.Varna}>

      <div className={styles.VarnaCube}>

        <div className={styles.VarnaCubeTopTwo}>

        <div className={styles.VarnaInput}>

        <div className={styles.Entry}>
          <div style={{flex: 2}}>
            <h5 style={{marginBottom: 2, marginTop: '8px'}}>Tokens you are selling</h5>
            <AutoComplete 
              placeholder="e.g. UNI" 
              selectionList={tokenList}
              excludeItem={itemToReceive.symbol}
              updateValue={(e)=>{this.handleItemToSend(e)}}
            />
          </div>
          <div style={{flex: 1, paddingLeft: '10px'}}>
            <h5 style={{marginBottom: 2, marginTop: '8px'}}>Amount to sell</h5>
            <input
              type="number"
              className={styles.Input}
              value={itemToSendAmount}
              placeholder="0"
              onChange={event=>{this.handleItemToSellAmount(event)}}
              disabled={generateKeyLoad}
            />
          </div>
        </div>

        {itemToSend.currency &&
          <div className={styles.Verify}>
            Token address: {itemToSend.currency}<br/>
            PLEASE CHECK CAREFULLY!
          </div>
        }

        <h5 style={{marginBottom: 2, marginTop: 10}}>Desired payment</h5>
        <AutoComplete 
          placeholder="e.g. ETH"
          selectionList={tokenList}
          excludeItem={itemToSend.symbol}
          updateValue={(e)=>{this.handleItemToReceive(e)}}
        />

        {itemToReceive.currency &&
          <div className={styles.Verify}>
            Token address: {itemToReceive.currency}<br/>
            PLEASE CHECK CAREFULLY!
          </div>
        }

        <h5 style={{marginBottom: 2, marginTop: 10}}>Minimum exchange rate</h5>
        <input
          type="number"
          className={styles.Input}
          value={sellerExchangeRate}
          placeholder="e.g. 0.1"
          onChange={event => {this.handleSellerExchangeRate(event)}}
          disabled={generateKeyLoad || itemToSend === '' || itemToReceive === ''}
        />

        {itemToSend.symbol && !itemToSendAmount && !itemToReceive.symbol && !sellerExchangeRate &&
          <div className={styles.Summary} >
            Selling {itemToSend.symbol}<br/>
          </div>
        }

        {itemToSend.symbol && itemToSendAmount && !itemToReceive.symbol && !sellerExchangeRate &&
          <div className={styles.Summary} >
            Selling {itemToSendAmount} {itemToSend.symbol}<br/>
          </div>
        }

        {itemToSend.symbol && itemToSendAmount && itemToReceive.symbol && !sellerExchangeRate &&
          <div className={styles.Summary} >
            Selling {itemToSendAmount} {itemToSend.symbol}<br/>
            Receive {itemToReceive.symbol}
          </div>
        }

        {itemToSend.symbol && itemToSendAmount && itemToReceive.symbol && sellerExchangeRate &&
          <div className={styles.Summary} >
            Selling {itemToSendAmount} {itemToSend.symbol}<br/>
            Minimum bid to consider: {sellerExchangeRate} {itemToReceive.symbol} per {itemToSend.symbol}<br/>
            Total proceeds: {accMul(sellerExchangeRate, itemToSendAmount)} {itemToReceive.symbol} or more
          </div>
        }

        <Button
          onClick={()=>{this.handleListItem()}}
          style={{flex: 0, maxWidth: 500, marginTop: 10, height: 20}}
          size='small'
          type='primary'
          loading={generateKeyLoad || uploadKeyLoad || configureItemToOMGXLoad}
          disabled={
            itemToSend.symbol === "" || 
            itemToSendAmount === "" || 
            itemToReceive.symbol === "" || 
            sellerExchangeRate === "" ||
            generateKeyLoad || 
            uploadKeyLoad || 
            configureItemToOMGXLoad
          }
        >
          {buttonText}
        </Button>
      </div>
        
        <div className={styles.VarnaHistory}>
          <Pager
            label={'My Listings'}
            currentPage={page}
            totalPages={totalNumberOfPages}
            isLastPage={paginatedItems.length < PER_PAGE}
            onClickNext={()=>this.setState({page:page+1})}
            onClickBack={()=>this.setState({page:page-1})}
          />

          {!paginatedItems.length && (
            <div className={styles.Disclaimer}>No More Items.</div>
          )}

          {paginatedItems.map((v,i) => {
            return (
              <ItemHistoryBox 
                key={i} 
                itemID={v} 
              />
            )
          })}
        </div>

      </div>

    <div className={styles.LongHistory} >
      <OfferHistoryBox />
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
  sell: state.sell,
  sellTask: state.sellTask,
  balance: state.balance,
  transaction: state.transaction,
  tokenList: state.tokenList
});

export default connect(mapStateToProps)(Seller);