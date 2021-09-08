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
import { useDispatch } from 'react-redux'
import Modal from 'components/modal/Modal'
import { closeModal } from 'actions/uiAction'

import ArrowUpwardIcon from '@material-ui/icons/ArrowUpward'
import ExpandMoreIcon from '@material-ui/icons/ExpandMore'

import { Box, Typography, useMediaQuery } from '@material-ui/core'
import { ReactComponent as Fox } from './../../../images/icons/fox-icon.svg'
import { ReactComponent as Account } from './../../../images/icons/mm-account.svg'

import { getAllNetworks } from 'util/masterConfig'
import store from 'store'

import * as styles from './WrongNetworkModal.module.scss'
import { useTheme } from '@emotion/react'

function WrongNetworkModal ({ open, onClose }) {

  const dispatch = useDispatch()

  const nw = getAllNetworks()
  const masterConfig = store.getState().setup.masterConfig
  const textLabel = nw[masterConfig].MM_Label
  const iconLabel = nw[masterConfig].MM_Label

  const theme = useTheme()
  const isMobile = useMediaQuery(theme.breakpoints.down('md'))

  function handleClose () {
    onClose()
    dispatch(closeModal('wrongNetworkModal'))
  }

  return (
    <Modal
      open={open}
      onClose={handleClose}
      light
      maxWidth="sm"
    >
      <Typography variant="h2" gutterBottom>
        Wrong Network
      </Typography>

      <Typography variant="body1">
        MetaMask is set to the wrong network. Please switch MetaMask to "{textLabel}" to continue.
      </Typography>

      <Box display="flex" sx={{ flexDirection: 'column', alignItems: 'center', mt: 3 }}>
        <div className={styles.metamask}>
          <Fox width={isMobile ? 30 : 30} />
          <div className={styles.button}>
            {iconLabel}
            <ExpandMoreIcon/>
          </div>
          <Account width={isMobile ? 40 : 40} />
        </div>
        <ArrowUpwardIcon fontSize={'large'} color={'primary'}/>
      </Box>
    </Modal>
  );
}

export default React.memo(WrongNetworkModal);
