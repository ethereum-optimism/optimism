/* eslint-disable quotes */
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

import React from 'react'
import { Typography } from '@material-ui/core'

import truncate from 'truncate-middle'
import { connect } from 'react-redux'
import { isEqual } from 'lodash'
import * as styles from './listNFT.module.scss'
import Copy from 'components/copy/Copy'
class listNFT extends React.Component {

  constructor(props) {

    super(props);

    const {
      name,
      symbol,
      address,
      UUID,
      time,
      URL,
      attributes
    } = this.props;


    this.state = {
      name,
      symbol,
      address,
      UUID,
      time,
      URL,
      attributes
    }
  }

  componentDidUpdate(prevState) {

    const {
      name, symbol, address,
      UUID, time, URL, attributes
    } = this.props;

    if (!isEqual(prevState.name, name)) {
      this.setState({ name })
    }

    if (!isEqual(prevState.symbol, symbol)) {
      this.setState({ symbol })
    }

    if (!isEqual(prevState.address, address)) {
      this.setState({ address })
    }

    if (!isEqual(prevState.UUID, UUID)) {
      this.setState({ UUID })
    }

    if (!isEqual(prevState.time, time)) {
      this.setState({ time })
    }

    if (!isEqual(prevState.URL, URL)) {
      this.setState({ URL })
    }

    if (!isEqual(prevState.attributes, attributes)) {
      this.setState({ attributes })
    }

  }

  render() {

    const {
      name,
      symbol,
      address,
      UUID,
      time,
      URL,
      attributes
    } = this.state;

    return (
      <div className={styles.ListNFT}>

        <img
          src={URL}
          alt="NFT URI"
          width={'100%'}
        />

        <div className={styles.topContainer}>
          <div className={styles.Table2}>
            <Typography variant="h4">{name}
              ({symbol})
            </Typography>
            {(attributes || []).map((attr, index) => {
              return (<Typography variant="body2" key={index}>{attr.trait_type}: 
                <Typography variant="body2" component="span" className={styles.muted}>
                  {attr.value}
                </Typography>
              </Typography>)
            })}
            <Typography variant="body2">UUID:
              <Typography variant="body2" component="span" className={styles.muted}>
                {UUID}
              </Typography>
            </Typography>
            <Typography variant="body2">Address:
              <Typography variant="body2" component="span" className={styles.muted}>
                {truncate(address, 8, 6, '...')} <Copy value={address} light={true} />
              </Typography>
            </Typography>
            <Typography variant="body2">URI:
              <Typography variant="body2" component="span" className={styles.muted}>
                {truncate(URL, 8, 6, '...')} <Copy value={URL} light={true} />
              </Typography>
            </Typography>
            <Typography variant="body2">Time minted:
              <Typography variant="body2" component="span" className={styles.muted}>
                {time}
              </Typography>
            </Typography>
          </div>
        </div>

      </div>
    )
  }
}

const mapStateToProps = state => ({
  nft: state.nft
})

export default connect(mapStateToProps)(listNFT)
