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
import Snackbar from '@material-ui/core/Snackbar';
import MuiAlert from '@material-ui/core/Alert';

function _Alert ({ children, open, onClose, type = 'success', duration = 3000, position = 0 }) {

  const alertStyle = {
    marginTop: position,
  };

  const Alert = React.forwardRef(function Alert(props, ref) {
    return <MuiAlert elevation={6} ref={ref} variant="filled" {...props} />;
  });

  let autohide = 0;
  if(type === 'success') {
    autohide = 2000; //autohide all the green alerts
  } else {
    autohide = duration;
  }

  return (
    <Snackbar
      open={open}
      autoHideDuration={autohide ? autohide : undefined}
      onClose={onClose}
      anchorOrigin={{
        vertical: 'top',
        horizontal: 'center'
      }}
      style={alertStyle}
    >
      <Alert
        onClose={onClose}
        severity={type}
      >
        {children}
      </Alert>
    </Snackbar>
  );
}

export default _Alert;
