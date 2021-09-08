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

import { Box, CardActionArea, Grid, Typography } from '@material-ui/core';
import React, { useEffect } from 'react';
import { useSelector } from 'react-redux';

import { selectGas } from 'selectors/gasSelector';
import * as S from './GasPicker.styles'

function GasPicker ({ selectedSpeed, setSelectedSpeed, setGasPrice }) {
  const gas = useSelector(selectGas);

  useEffect(() => {
    setGasPrice(gas[selectedSpeed]);
  }, [ selectedSpeed, gas, setGasPrice ]);

  return (
    <Box sx={{ my: 3 }}>
      <Typography variant="h4" gutterBottom>
        Gas Fee
      </Typography>

      <Grid container spacing={1}>
        <Grid item xs={4}>
          <S.CardTag
            selected={selectedSpeed === 'slow'}
          >
            <CardActionArea onClick={() => setSelectedSpeed('slow')}>
              <S.WrapperItem>
                <Typography variant="body2" sx={{fontWeight: 700}}>Slow</Typography>
                <Typography variant="caption" sx={{fontWeight: 700}}>{gas.slow / 1000000000} gwei</Typography>
              </S.WrapperItem>
            </CardActionArea>
          </S.CardTag>
        </Grid>

        <Grid item xs={4}>
          <S.CardTag selected={selectedSpeed === 'normal'}
          >
            <CardActionArea onClick={() => setSelectedSpeed('normal')}>
              <S.WrapperItem>
                <Typography variant="body2" sx={{fontWeight: 700}}>Normal</Typography>
                <Typography variant="caption" sx={{fontWeight: 700}}>{gas.normal / 1000000000} gwei</Typography>
              </S.WrapperItem>
            </CardActionArea>
          </S.CardTag>
        </Grid>

        <Grid item xs={4}>
          <S.CardTag
            selected={selectedSpeed === 'fast'}
          >
            <CardActionArea onClick={() => setSelectedSpeed('fast')}>
              <S.WrapperItem>
                <Typography variant="body2" sx={{fontWeight: 700}}>Fast</Typography>
                <Typography variant="caption" sx={{fontWeight: 700}}>{gas.fast / 1000000000} gwei</Typography>
              </S.WrapperItem>
            </CardActionArea>
          </S.CardTag>
        </Grid>

      </Grid>
    </Box>
  );
}

export default React.memo(GasPicker);
