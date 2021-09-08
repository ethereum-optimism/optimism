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
import {
  Fade,
  Typography,
  Grid,
  Container,
  Box,
  useMediaQuery
} from '@material-ui/core';
import { ReactComponent as CloseIcon } from './../../images/icons/close-modal.svg';
// import CloseIcon  from '../icons/CloseIcon.js';
import * as S from "./Modal.styles"
import { useTheme } from '@emotion/react';

function _Modal ({
  children,
  open,
  onClose,
  light,
  title,
  transparent,
  maxWidth
}) {
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down('md'));

  return (
    <S.StyledModal
      aria-labelledby='transition-modal-title'
      aria-describedby='transition-modal-description'
      open={open}
      onClose={onClose}
      ismobile={isMobile ? 1 : 0}
      // closeAfterTransition
      BackdropComponent={S.Backdrop}
      disableAutoFocus={true}
    >
      <Fade in={open}>
        <Container maxWidth={maxWidth || "lg"} sx={{border: 'none', position: 'relative'}}>
          <Grid container>
            <Grid item xs={12} md={title ? 2 : 1}>
              <Box sx={{mr: 8}}>
                <Typography variant="h2" component="h3" sx={{ fontWeight: "700"}}>{title}</Typography>
              </Box>
            </Grid>

            <Grid item xs={12} md={title ? 10 : 9}>
              <S.Style isMobile={isMobile} transparent={transparent || isMobile}>
                {children}
              </S.Style>
            </Grid>

            <Grid item xs={12} md={1}>
              <S.IconButtonTag onClick={onClose}>
                <CloseIcon />
              </S.IconButtonTag>
            </Grid>

          </Grid>
        </Container>
      </Fade>
    </S.StyledModal>
  );
}

export default _Modal;
