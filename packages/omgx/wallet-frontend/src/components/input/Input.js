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
import BN from 'bignumber.js'
import * as S from './Input.styles'
import Button from 'components/button/Button'
import { Box, Typography } from '@material-ui/core'
import { useTheme } from '@emotion/react'
import { getCoinImage } from 'util/coinImage'

function Input({
  placeholder,
  label,
  type = 'text',
  disabled,
  icon,
  unit,
  value,
  onChange,
  sx,
  paste,
  // className,
  maxValue,
  // small,
  fullWidth,
  size,
  variant,
  newStyle = false,
}) {

  async function handlePaste() {
    try {
      const text = await navigator.clipboard.readText()
      if (text) {
        onChange({ target: { value: text } })
      }
    } catch (err) {
      // navigator clipboard api not supported in client browser
    }
  }

  function handleMaxClick() {
    onChange({ target: { value: maxValue } })
  }

  const overMax = new BN(value).gt(new BN(maxValue))

  const theme = useTheme();

  return (

    <S.Wrapper newstyle={newStyle ? 1 : 0}>
      {unit && (
        <S.UnitContent>
          <div>
            <Typography variant="body2" component="div">{unit}</Typography>
            <img src={getCoinImage(unit)} alt="logo" width={50} height={50} />
          </div>
        </S.UnitContent>
      )}

      <S.InputWrapper>
        {label && (
          <Typography variant="body2" component="div" sx={{opacity: 0.7, mb: 1, ml: '15px'}}>
            {label}
          </Typography>
        )}
        <S.TextFieldTag
          placeholder={placeholder}
          type={type}
          value={value}
          onChange={onChange}
          disabled={disabled}
          fullWidth={fullWidth}
          size={size}
          variant={variant}
          error={overMax}
          sx={sx}
          newstyle={newStyle ? 1 : 0}
        />
      </S.InputWrapper>

      {unit && (
        <S.ActionsWrapper>
          <Typography variant="body2" component="p" sx={{opacity: 0.7, textAlign: "end", mb: 2}}>
            Available: {Number(maxValue).toFixed(3)}
          </Typography>

          {maxValue && value !== maxValue && (
            <Box>
              <Button onClick={handleMaxClick} variant="small" >
                Use All
              </Button>
            </Box>
          )}
        </S.ActionsWrapper>
      )}
      {paste && (
        <Box onClick={handlePaste} sx={{color: theme.palette.secondary.main, opacity: 0.9, cursor: 'pointer', position: 'absolute', right: '70px', fontSize: '14px'}}>
          PASTE
        </Box>
      )}
    </S.Wrapper>
  )
}

export default React.memo(Input)
