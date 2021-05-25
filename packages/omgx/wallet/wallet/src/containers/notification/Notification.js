/*
  OmgX - A Privacy-Preserving Marketplace
  OmgX uses Fully Homomorphic Encryption to make markets fair. 
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
  along with this program. If not, see <https://www.gnu.org/licenses/>.
*/

import React from 'react';
import { connect } from 'react-redux';

import { closeNotification } from 'actions/notificationAction';

import Button from 'components/button/Button';

import * as styles from './Notification.module.scss';

class Notification extends React.Component {
  constructor(props) {
    super(props);

    const { 
      notificationText, 
      notificationButtonText, 
      notificationButtonAction,
      notificationStatus,
    } = this.props.notification;

    this.state = {
      notificationText, 
      notificationButtonText, 
      notificationButtonAction,
      notificationStatus,
    }
  }

  componentDidUpdate(prevState) {
    const { 
      notificationText, 
      notificationButtonText, 
      notificationButtonAction,
      notificationStatus,
    } = this.props.notification;

    if (prevState.notification !== this.props.notification) {
      this.setState({
        notificationText, 
        notificationButtonText, 
        notificationButtonAction,
        notificationStatus,
      })
    }

  }

  async handleButtonAction() {
    const { notificationButtonAction } = this.state;
    try {
      const result = await notificationButtonAction();
      if (result === null) {
        this.props.dispatch(closeNotification());
      }
    } catch (error) {
      console.log(error);
    }
  }

  handleIgnore() {
    this.props.dispatch(closeNotification());
  }

  render() {
    const {
      notificationText, 
      notificationButtonText, 
      notificationStatus,
    } = this.state;

    if (notificationStatus === 'open') {
      return (
        <div className={styles.container}>
          <div>{notificationText}</div>
          {notificationButtonText &&
            <Button
              type='primary'
              size='small'
              className={styles.button}
              onClick={()=>this.handleButtonAction()}
            >
              {notificationButtonText}
            </Button>
          }
          <Button
            type='primary'
            size='small'
            className={styles.button}
            onClick={()=>this.handleIgnore()}
          >
            Ignore
          </Button>
        </div>
      )
    }

    if (notificationStatus === 'close') {
      return <></>
    }
  }
}

const mapStateToProps = state => ({ 
  notification: state.notification,
});

export default connect(mapStateToProps)(Notification);