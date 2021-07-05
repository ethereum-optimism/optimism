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

import React from 'react';
import { useSelector, useDispatch } from 'react-redux';
import Modal from 'components/modal/Modal';
import { closeModal } from 'actions/uiAction';
import close from 'images/close.png';
import arrow from 'images/arrow.png';

import * as styles from './WrongNetworkModal.module.scss';

import { selectNetwork } from 'selectors/setupSelector';

function WrongNetworkModal ({ open, onClose }) {

  const masterConfig = useSelector(selectNetwork());
  const dispatch = useDispatch();

  function handleClose () {
    onClose();
    dispatch(closeModal('wrongNetworkModal'));
  }

  return (
    <Modal
      open={open}
      onClose={handleClose}
      light
    >
      <div className={styles.WrongNetworkModal}>
        <img
          className={styles.close}
          onClick={handleClose}
          src={close}
          alt='close'
        />
        <h2>Wrong Network</h2>

        <div className={styles.content}>
          <div className={styles.description}>
            Metamask is set to the wrong network. Please switch Metamask to {masterConfig} to continue.
          </div>

          <div className={styles.currentNetwork}>
            <div
              className={[
                styles.indicator,
                styles.active
              ].join(' ')}
            />
            <span>{masterConfig}</span>
          </div>

          <img
            className={styles.arrow}
            src={arrow}
            alt='arrow'
          />
          
        </div>
      </div>
    </Modal>
  );
}

export default React.memo(WrongNetworkModal);
