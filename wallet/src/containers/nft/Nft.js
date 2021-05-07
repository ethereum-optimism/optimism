import React from 'react';
import { connect } from 'react-redux';
import { isEqual } from 'lodash';

import InputSelect from 'components/inputselect/InputSelect';
import Input from 'components/input/Input';
import Button from 'components/button/Button';
import { openError, openAlert } from 'actions/uiAction';
import { logAmount } from 'util/amountConvert';
import networkService from 'services/networkService';

import * as styles from './Nft.module.scss';

class Nft extends React.Component {

  constructor(props) {

    super(props);

    const { balance } = this.props;

    this.state = {
      balance,

      initialL1Value: '',
      initialL1Currency: '',
      L1Value: '',
      L1Currency: '',
      LPL1SearchToken: '',
      LPL1Balance: '',
      LPL1FeeSearchToken: '',
      LPL1FeeBalance: '',
      L1FeeWithdrawToken: '',
      L1FeeReceiverAddress: '',
      L1FeeWithdrawAmount: '',

      initialL2Value: '',
      initialL2Currency: '',
      L2Value: '',
      L2Currency: '',
      LPL2SearchToken: '',
      LPL2Balance: '',
      LPL2FeeSearchToken: '',
      LPL2FeeBalance: '',
      L2FeeWithdrawToken: '',
      L2FeeReceiverAddress: '',
      L2FeeWithdrawAmount: '',

      loading: false,

      receiverAddress: '',
      tokenID: 1,
      tokenURI: '' 
    }
  }

  componentDidMount() {
    const { balance } = this.props;
    if (balance.rootchain.length && balance.childchain.length) {
      const L1Token = balance.rootchain.filter(i => i.symbol !== 'WETH')[0];
      const L2Token = balance.childchain.filter(i => i.symbol !== 'ETH')[0];
      this.setState({
        initialL1Currency: L1Token.currency,
        initialL2Currency: L2Token.currency,
        L1Currency : L1Token.currency,
        L2Currency : L2Token.currency,
        LPL1SearchToken : L1Token,
        LPL2SearchToken : L2Token,
        LPL1FeeSearchToken : L1Token,
        LPL2FeeSearchToken : L2Token,
        L1FeeWithdrawToken : L1Token,
        L2FeeWithdrawToken : L2Token
      });
    }
  }

  componentDidUpdate(prevState) {

    const { balance } = this.props;
    const { 
      initialL1Currency, initialL2Currency,
      L1Currency, L2Currency, 
      LPL1SearchToken, LPL2SearchToken,
      LPL1FeeSearchToken, LPL2FeeSearchToken,
      L1FeeWithdrawToken, L2FeeWithdrawToken,
    } = this.state;

    const L1Token = balance.rootchain.filter(i => i.symbol !== 'WETH')[0];
    const L2Token = balance.childchain.filter(i => i.symbol !== 'ETH')[0];

    if (!isEqual(prevState.balance, balance)) {
      this.setState({ balance });
      if (!initialL1Currency) this.setState({initialL1Currency : L1Token.currency});
      if (!initialL2Currency) this.setState({initialL2Currency : L2Token.currency});
      if (!L1Currency) this.setState({L1Currency : L1Token.currency});
      if (!L2Currency) this.setState({L2Currency : L2Token.currency});
      if (!LPL1SearchToken) this.setState({LPL1SearchToken : L1Token});
      if (!LPL2SearchToken) this.setState({LPL2SearchToken : L2Token});
      if (!LPL1FeeSearchToken) this.setState({LPL1FeeSearchToken : L1Token});
      if (!LPL2FeeSearchToken) this.setState({LPL2FeeSearchToken : L2Token});
      if (!L1FeeWithdrawToken) this.setState({L1FeeWithdrawToken : L1Token});
      if (!L2FeeWithdrawToken) this.setState({L2FeeWithdrawToken : L2Token});
    }
  }

  async handleInitialDepositL1() {
    const { initialL1Value, initialL1Currency } = this.state;

    const networkStatus = await this.props.dispatch(networkService.checkNetwork('L1'));
    if (!networkStatus) {
      this.props.dispatch(openError('Please use L1 network.'));
      return;
    }

    this.setState({ loading: true });
    const depositTX = await networkService.initialDepositL1LP(initialL1Currency, initialL1Value);
    this.setState({ loading: false });
    if (depositTX) {
      this.props.dispatch(openAlert(`Deposited ${initialL1Value} ${initialL1Currency} into L1 Liquidity Pool.`))
    } else {
      this.props.dispatch(openError('Unknown error'));
    }
  }

  async handleInitialDepositL2() {
    const { initialL2Value, initialL2Currency } = this.state;

    const networkStatus = await this.props.dispatch(networkService.checkNetwork('L2'));
    if (!networkStatus) {
      this.props.dispatch(openError('Please use L2 network.'));
      return;
    }

    this.setState({ loading: true });
    const depositTX = await networkService.initialDepositL2LP(initialL2Currency, initialL2Value);
    this.setState({ loading: false });
    if (depositTX) {
      this.props.dispatch(openAlert(`Deposited ${initialL2Value} ${initialL2Currency} into L2 Liquidity Pool.`))
    } else {
      this.props.dispatch(openError('Unknown error'));
    }
  }

  async handleDepositL1() {
    const { L1Value, L1Currency } = this.state;

    const networkStatus = await this.props.dispatch(networkService.checkNetwork('L1'));
    if (!networkStatus) {
      this.props.dispatch(openError('Please use L1 network.'));
      return;
    }

    this.setState({ loading: true });
    const depositTX = await networkService.depositL1LP(L1Currency, L1Value);
    this.setState({ loading: false });
    if (depositTX) {
      this.props.dispatch(openAlert(`Deposited ${L1Value} ${L1Currency} into L1 Liquidity Pool.`))
    } else {
      this.props.dispatch(openError('L2 Liquidity Pool doesn\'t have enough balance to cover this.'));
    }
  }

  async handleDepositL2() {
    const { L2Value, L2Currency } = this.state;

    const networkStatus = await this.props.dispatch(networkService.checkNetwork('L2'));
    if (!networkStatus) {
      this.props.dispatch(openError('Please use L2 network.'));
      return;
    }

    this.setState({ loading: true });
    const depositTX = await networkService.depositL2LP(L2Currency, L2Value);
    if (depositTX) {
      this.setState({ loading: false });
    }
  }

  async handleBalanceL1() {
    const { LPL1SearchToken } = this.state;
    const balanceTX = await networkService.L1LPBalance(LPL1SearchToken.currency);
    if (balanceTX) {
      this.setState({ LPL1Balance: balanceTX });
    }
  }

  async handleBalanceL2() {
    const { LPL2SearchToken } = this.state;
    const balanceTX = await networkService.L2LPBalance(LPL2SearchToken.currency);
    console.log({ balanceTX });
    if (balanceTX) {
      this.setState({ LPL2Balance: balanceTX });
    }
  }

  async handleFeeBalanceL1() {
    const { LPL1FeeSearchToken } = this.state;
    const balanceTX = await networkService.L1LPFeeBalance(LPL1FeeSearchToken.currency);
    if (balanceTX) {
      this.setState({ LPL1FeeBalance: balanceTX });
    }
  }

  async handleFeeBalanceL2() {
    const { LPL2FeeSearchToken } = this.state;
    const balanceTX = await networkService.L2LPFeeBalance(LPL2FeeSearchToken.currency);
    if (balanceTX) {
      this.setState({ LPL2FeeBalance: balanceTX });
    }
  }

  async handleWithdrawFeeL1() {
    const { L1FeeWithdrawToken, L1FeeReceiverAddress, L1FeeWithdrawAmount } = this.state;

    const networkStatus = await this.props.dispatch(networkService.checkNetwork('L1'));
    if (!networkStatus) {
      this.props.dispatch(openError('Please use L1 network.'));
      return;
    }

    this.setState({ loading: true });
    const withdrawTX = await networkService.L1LPWithdrawFee(
      L1FeeWithdrawToken.currency, 
      L1FeeReceiverAddress, 
      L1FeeWithdrawAmount
    );
    
    if (withdrawTX) {
      this.props.dispatch(openAlert(`You withdraw ${L1FeeWithdrawAmount} ${L1FeeWithdrawToken.symbol}.`));
    } else {
      this.props.dispatch(openError('You don\'t have enough fee amount to be withdrew'));
    }
    this.setState({ loading: false })
  }

  async handleWithdrawFeeL2() {
    const { L2FeeWithdrawToken, L2FeeReceiverAddress, L2FeeWithdrawAmount } = this.state;

    const networkStatus = await this.props.dispatch(networkService.checkNetwork('L2'));
    if (!networkStatus) {
      this.props.dispatch(openError('Please use L2 network.'));
      return;
    }

    this.setState({ loading: true });
    const withdrawTX = await networkService.L2LPWithdrawFee(
      L2FeeWithdrawToken.currency, 
      L2FeeReceiverAddress, 
      L2FeeWithdrawAmount
    );
    
    if (withdrawTX) {
      this.props.dispatch(openAlert(`You withdraw ${L2FeeWithdrawAmount} ${L2FeeWithdrawToken.symbol}.`));
    } else {
      this.props.dispatch(openError('You don\'t have enough fee amount to be withdrew'));
    }
    this.setState({ loading: false })
  }

  async handleMintAndSend() {

    const { receiverAddress, tokenID, tokenURI } = this.state;

    //we are doing this on L2

    const networkStatus = await this.props.dispatch(networkService.checkNetwork('L2'));
    if (!networkStatus) {
      this.props.dispatch(openError('Please use L2 network.'));
      return;
    }

    this.setState({ loading: true });

    const mintTX = await networkService.mintAndSendNFT(
      receiverAddress, 
      tokenID, 
      tokenURI
    );
    
    if (mintTX) {
      this.props.dispatch(openAlert(`You minted a new NFT for ${receiverAddress}`));
    } else {
      this.props.dispatch(openError('NFT minting error'));
    }

    this.setState({ loading: false })
  }

  render() {

    const { 
      balance,
      initialL1Value, initialL1Currency,
      L1Value, L1Currency, 
      LPL1SearchToken, LPL1Balance,
      LPL1FeeSearchToken, LPL1FeeBalance,
      L1FeeReceiverAddress, L1FeeWithdrawAmount,
      initialL2Value, initialL2Currency,
      L2Value, L2Currency, 
      LPL2SearchToken, LPL2Balance,
      LPL2FeeSearchToken, LPL2FeeBalance,
      L2FeeReceiverAddress, L2FeeWithdrawAmount,
      loading,

      receiverAddress,
      tokenID,
      tokenURI 
    } = this.state;

    const rootChainBalance = balance.rootchain;
    const selectL1Options = rootChainBalance.map(i => ({
      title: i.symbol,
      value: i.currency,
      subTitle: `Balance: ${logAmount(i.amount, i.decimals)}`
    })).filter(i => i.title !== 'WETH');

    const childChainBalance = balance.childchain;
    const selectL2Options = childChainBalance.map(i => ({
      title: i.symbol,
      value: i.currency,
      subTitle: `Balance: ${logAmount(i.amount, i.decimals)}`
    })).filter(i => i.title !== 'ETH');

    return (
      <div className={styles.container}>
        <div className={styles.boxContainer}>
          
          <h2>Minter/Owner Functions</h2>
          
          <h3>Mint and Send</h3>
          
          <Input
            placeholder="Receiver Address (e.g. Ox.....)"
            onChange={i=>{this.setState({receiverAddress: i.target.value})}}
          />
          <Input
            placeholder="NFT Unique ID (e.g. 7)"
            onChange={i=>{this.setState({tokenID: i.target.value})}}
          />
          <Input
            placeholder="NFT URL (e.g. https://jimb.stanford.edu)"
            onChange={i=>{this.setState({tokenURI: i.target.value})}}
          />
          <Button
            className={styles.button}
            onClick={() => {this.handleMintAndSend()}}
            type='primary'
            loading={loading}
            disabled={!receiverAddress || !tokenID || !tokenURI}
          >
            Mint and Send
          </Button>
          
          <Button
            className={styles.button}
            onClick={() => {this.handleInitialDepositL1();}}
            type='primary'
            loading={loading}
            tooltip='Your deposit transaction is still pending. Please wait for confirmation.'
            disabled={!initialL1Value || !initialL1Currency}
          >
            Initial Deposit L1
          </Button>

          <h3>Deposit L1</h3>
          <InputSelect
            label='Amount to deposit'
            placeholder={0}
            value={L1Value}
            onChange={i => {this.setState({L1Value: i.target.value})}}
            selectOptions={selectL1Options}
            onSelect={i => {this.setState({L1Currency: i.target.value})}}
            selectValue={L1Currency}
          />

          <Button
            className={styles.button}
            onClick={() => {this.handleDepositL1();}}
            type='primary'
            loading={loading}
            tooltip='Your deposit transaction is still pending. Please wait for confirmation.'
            disabled={!L1Value || !L1Currency}
          >
            Deposit L1
          </Button>

          <h3>L1 Liquidity Pool Balance</h3>
          <select 
            className={styles.select} 
            onChange={i => {
              const token = balance.rootchain.filter(j => j.currency === i.target.value)[0];
              this.setState({ LPL1SearchToken: token, LPL1Balance: '' });
            }}
          >
            {selectL1Options.map((option, i) => (
              <option key={i} value={option.value}>{option.title} - {option.value}</option>
            ))}
          </select>

          <Button
            className={styles.button}
            onClick={() => {this.handleBalanceL1();}}
            type='primary'
            loading={loading}
            tooltip='Your deposit transaction is still pending. Please wait for confirmation.'
          >
            L1 Balance
          </Button>

          {LPL1Balance &&
            <h3>
              {`The L1 liquidity pool has ${LPL1Balance} ${LPL1SearchToken.symbol}.`}
            </h3>
          }

          <h3>L1 Liquidity Pool Fee Balance</h3>
          <select 
            className={styles.select} 
            onChange={i => {
              const token = balance.rootchain.filter(j => j.currency === i.target.value)[0];
              this.setState({ LPL1FeeSearchToken: token, LPL1FeeBalance: '' });
            }}
          >
            {selectL1Options.map((option, i) => (
              <option key={i} value={option.value}>{option.title} - {option.value}</option>
            ))}
          </select>

          <Button
            className={styles.button}
            onClick={() => {this.handleFeeBalanceL1();}}
            type='primary'
            loading={loading}
            tooltip='Your deposit transaction is still pending. Please wait for confirmation.'
          >
            L1 Fee Balance
          </Button>

          {LPL1FeeBalance &&
            <h3>
              {`The L1 liquidity pool has ${LPL1FeeBalance} ${LPL1FeeSearchToken.symbol}.`}
            </h3>
          }

          <h3>Withdraw L1 Liquidity Pool Fee</h3>
          <h4 style={{marginTop: 5, marginBottom: 5}}>Fee</h4>
          <select 
            className={styles.select} 
            onChange={i => {
              const token = balance.rootchain.filter(j => j.currency === i.target.value)[0];
              this.setState({ L1FeeWithdrawToken: token });
            }}
          >
            {selectL1Options.map((option, i) => (
              <option key={i} value={option.value}>{option.title} - {option.value}</option>
            ))}
          </select>
          <h4 style={{marginTop: 5, marginBottom: 5}}>Receiver</h4>
          <Input
            placeholder="Receiver"
            onChange={i => {this.setState({ L1FeeReceiverAddress: i.target.value })}}
          />
          <h4 style={{marginTop: 5, marginBottom: 5}}>Amount</h4>
          <Input
            placeholder="Amount"
            onChange={i => {this.setState({ L1FeeWithdrawAmount: i.target.value })}}
          />
          <Button
            className={styles.button}
            onClick={() => {this.handleWithdrawFeeL1();}}
            type='primary'
            loading={loading}
            disabled={!L1FeeReceiverAddress || !L1FeeWithdrawAmount}
          >
            Withdraw Fee
          </Button>

        </div>

        <div className={styles.boxContainer}>
          <h2>Layer 2 Liquidity Pool</h2>
          <h3>Initial Deposit L2</h3>
          <InputSelect
            label='Amount to deposit (No fund on L1)'
            placeholder={0}
            value={initialL2Value}
            onChange={i => {this.setState({initialL2Value: i.target.value})}}
            selectOptions={selectL2Options}
            onSelect={i => {this.setState({initialL2Currency: i.target.value})}}
            selectValue={initialL2Currency}
          />
          <Button
            className={styles.button}
            onClick={() => {this.handleInitialDepositL2();}}
            type='primary'
            loading={loading}
            tooltip='Your deposit transaction is still pending. Please wait for confirmation.'
            disabled={!initialL2Value || !initialL2Currency}
          >
            Initial Deposit L2
          </Button>

          <h3>Deposit L2</h3>
          <InputSelect
            label='Amount to deposit'
            placeholder={0}
            value={L2Value}
            onChange={i => {this.setState({L2Value: i.target.value})}}
            selectOptions={selectL2Options}
            onSelect={i => {this.setState({L2Currency: i.target.value})}}
            selectValue={L2Currency}
          />

          <Button
            className={styles.button}
            onClick={() => {this.handleDepositL2();}}
            type='primary'
            loading={loading}
            tooltip='Your deposit transaction is still pending. Please wait for confirmation.'
            disabled={!L2Value || !L2Currency}
          >
            Deposit L2
          </Button>

          <h3>L2 Liquidity Pool Balance</h3>
          <select 
            className={styles.select} 
            onChange={i => {
              const token = balance.childchain.filter(j => j.currency === i.target.value)[0];
              this.setState({ LPL2SearchToken: token, LPL2Balance: '' });
            }}
          >
            {selectL2Options.map((option, i) => (
              <option key={i} value={option.value}>{option.title} - {option.value}</option>
            ))}
          </select>

          <Button
            className={styles.button}
            onClick={() => {this.handleBalanceL2();}}
            type='primary'
            loading={loading}
            tooltip='Your deposit transaction is still pending. Please wait for confirmation.'
          >
            L2 Balance
          </Button>

          {LPL2Balance &&
            <h3>
              {`The L2 liquidity pool has ${LPL2Balance} ${LPL2SearchToken.symbol}.`}
            </h3>
          }

          <h3>L2 Liquidity Pool Fee Balance</h3>
          <select 
            className={styles.select} 
            onChange={i => {
              const token = balance.childchain.filter(j => j.currency === i.target.value)[0];
              this.setState({ LPL2FeeSearchToken: token, LPL2FeeBalance: '' });
            }}
          >
            {selectL2Options.map((option, i) => (
              <option key={i} value={option.value}>{option.title} - {option.value}</option>
            ))}
          </select>

          <Button
            className={styles.button}
            onClick={() => {this.handleFeeBalanceL2();}}
            type='primary'
            loading={loading}
            tooltip='Your deposit transaction is still pending. Please wait for confirmation.'
          >
            L1 Fee Balance
          </Button>

          {LPL2FeeBalance &&
            <h3>
              {`The L2 liquidity pool has ${LPL2FeeBalance} ${LPL2FeeSearchToken.symbol}.`}
            </h3>
          }

          <h3>Withdraw L2 Liquidity Pool Fee</h3>
          <h4 style={{marginTop: 5, marginBottom: 5}}>Fee</h4>
          <select 
            className={styles.select} 
            onChange={i => {
              const token = balance.childchain.filter(j => j.currency === i.target.value)[0];
              this.setState({ L2FeeWithdrawToken: token });
            }}
          >
            {selectL2Options.map((option, i) => (
              <option key={i} value={option.value}>{option.title} - {option.value}</option>
            ))}
          </select>
          <h4 style={{marginTop: 5, marginBottom: 5}}>Receiver</h4>
          <Input
            placeholder="Receiver"
            onChange={i => {this.setState({ L2FeeReceiverAddress: i.target.value })}}
          />
          <h4 style={{marginTop: 5, marginBottom: 5}}>Amount</h4>
          <Input
            placeholder="Amount"
            onChange={i => {this.setState({ L2FeeWithdrawAmount: i.target.value })}}
          />
          <Button
            className={styles.button}
            onClick={() => {this.handleWithdrawFeeL2();}}
            type='primary'
            loading={loading}
            disabled={!L2FeeReceiverAddress || !L2FeeWithdrawAmount}
          >
            Withdraw Fee
          </Button>
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

export default connect(mapStateToProps)(Nft);