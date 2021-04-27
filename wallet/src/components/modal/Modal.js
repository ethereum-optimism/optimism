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
import { makeStyles } from '@material-ui/core/styles';
import {
  Modal,
  Backdrop,
  Fade
} from '@material-ui/core';

import { gray2 } from 'index.scss';

const useStyles = makeStyles({
  modal: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    outline: 'none'
  },
  paper: {
    backgroundColor: 'white',
    color: gray2,
    padding: '20px',
    border: 'none',
    outline: 'none',
    width: '500px',
    boxSizing: 'border-box',
    maxWidth: '100%',
    borderRadius: '4px'
  },
});

function _Modal ({
  children,
  open,
  onClose,
  light
}) {
  const classes = useStyles();

  return (
    <Modal
      aria-labelledby='transition-modal-title'
      aria-describedby='transition-modal-description'
      className={classes.modal}
      open={open}
      onClose={onClose}
      closeAfterTransition
      BackdropComponent={Backdrop}
      BackdropProps={{
        timeout: 500
      }}
    >
      <Fade in={open}>
        <div
          className={classes.paper}
        >
          {children}
        </div>
      </Fade>
    </Modal>
  );
}

export default _Modal;
