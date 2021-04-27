import React from 'react';
import { connect } from 'react-redux';
import { orderBy, isEqual } from 'lodash';
import moment from 'moment';
import { Search } from '@material-ui/icons';
import HighlightOffIcon from '@material-ui/icons/HighlightOff';

import AutoComplete from 'components/autocomplete/AutoComplete';

import * as styles from './PriceTickerBox.module.scss';

class PriceTickerBox extends React.Component {

  constructor(props) {
    super(props);

    const { bidData, bidDataPair } = this.props.priceTicker;

    this.state = {
      bidData,
      bidDataPair,
      currentTime: moment(new Date()),
      selectedPrice: [],
    }
  }

  componentDidMount() {
    this.intervalId = setInterval(this.timer.bind(this), 1000);
  }

  componentWillUnmount(){
    clearInterval(this.intervalId);
  }

  timer() {
    this.setState({ currentTime: moment(new Date()) });
  }

  componentDidUpdate(prevState) {
    const { bidData, bidDataPair } = this.props.priceTicker;

    if (!isEqual(prevState.priceTicker.bidData, bidData)) {
      this.setState({ bidData });
    }

    if (!isEqual(prevState.priceTicker.bidDataPair, bidDataPair)) {
      this.setState({ bidDataPair });
    }
  }

  handleSelectedPrice(e) {
    const { selectedPrice } = this.state;  
    if (e && !selectedPrice.includes(e)) {
      this.setState({ selectedPrice: [...selectedPrice, e]});
    }
  }

  handleClosePriceTicker(e) {
    let { selectedPrice } = this.state;
    selectedPrice = selectedPrice.filter(i => i !== e);
    this.setState({ selectedPrice });
  }

  render() {
    const { bidData, bidDataPair, selectedPrice, currentTime } = this.state;

    const priceTickerData = orderBy(bidData, ['symbolA', 'createdAt', 'exchangeRate'], ['desc', 'desc', 'desc']);
    const bidDataPairList = bidDataPair.reduce((acc, cur) => {acc.push({symbol: `${cur.symbolA}/${cur.symbolB}`}); return acc}, []);
    
    return (
      <div className={styles.main}>
        <div className={styles.searchBoxContainer}>
          <div>Pricing Tickers</div>
          <div className={styles.searchRow}>
            <Search className={styles.icon} />
            <AutoComplete
              placeholder="Add Pricing Stream"
              selectionList={bidDataPairList}
              excludeList={selectedPrice}
              onBlurDisappear={true}
              openControlLength={1}
              updateValue={(e)=>{this.handleSelectedPrice(e)}}
            />
          </div>
        </div>

        <div className={styles.priceTickersContainer}>
          {selectedPrice.map((eachSelectedPrice, selectedPriceIndex) => {
            const [selectedSymbolA, selectedSymbolB] = eachSelectedPrice.split('/');
            const sortedPriceTickerData = priceTickerData.filter(i => i.symbolA === selectedSymbolA && i.symbolB === selectedSymbolB);
            return (
              <div className={styles.priceTickerContainer} key={selectedPriceIndex}>
                <HighlightOffIcon className={styles.icon} onClick={()=>{this.handleClosePriceTicker(eachSelectedPrice)}}/>
                <h3 style={{marginTop: '5px', marginBottom: '5px', padding: '0px'}}>{selectedSymbolA}/{selectedSymbolB}</h3>
                <div className={styles.RtableWB}>
                  <div className={styles.Rtable + ' ' + styles.RtableHeader}>
                    <div className={styles.Rtable_cell}>
                      Price<br/>({selectedSymbolB})
                    </div>
                    <div className={styles.Rtable_cell}>
                      Amount<br/>({selectedSymbolA})
                    </div>
                    <div className={styles.Rtable_cell}>
                    <br/>Total
                    </div>
                    <div className={styles.Rtable_cell}>
                    <br/>Created
                    </div>
                  </div>
                  
                  <div className={styles.Rtable_line}/>

                  {sortedPriceTickerData.map((priceData, priceDataIndex) => {
                    let timeDiff = Number(moment.duration(currentTime.diff(moment.unix(priceData.createdAt / 1000))).asMinutes());
                    let timeText = `${parseInt(timeDiff)}m ago`
                    if (timeDiff > 60) timeText = `${parseInt(timeDiff / 60)}h ago`
                    if (timeDiff > 60 * 24) timeText = `${parseInt(timeDiff / 60 / 24)}d ago`;
                    return (
                      <div className={styles.Rtable} key={priceDataIndex}>
                        <div className={styles.Rtable_cell}>
                          <div className={styles.Rtable_row_height}>{(priceData.exchangeRate).toFixed(5)}</div>
                        </div>
                        <div className={styles.Rtable_cell}>
                          <div className={styles.Rtable_row_height}>{(priceData.amountA).toFixed(5)}</div>
                        </div>
                        <div className={styles.Rtable_cell}>
                          <div className={styles.Rtable_row_height}>{(priceData.amountB).toFixed(5)} </div>
                        </div>
                        <div className={styles.Rtable_cell}>
                          <div className={styles.Rtable_row_height}>{timeText}</div>
                        </div>
                      </div>
                    )
                  })}

                </div>
              </div>
            )
          })}
        </div>
      </div>
    )
  }
}

const mapStateToProps = state => ({ 
  transaction: state.transaction,
  priceTicker: state.priceTicker,
});

export default connect(mapStateToProps)(PriceTickerBox);