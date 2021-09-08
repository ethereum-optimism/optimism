import styled from '@emotion/styled';
import { Box } from '@material-ui/core';

export const ThemeSwitcherTag = styled.div`
  margin-left: -15px;
  margin-top: 100px;
  display: flex;
  position: relative;
`;

export const Button = styled.button`
  border: 0;
  padding: 10px;
  border-radius: 16px;
  background-color: ${(props) => props.selected ? props.theme.palette.action.disabledBackground : 'transparent'};
  cursor: pointer;
  transition: all .2s ease-in-out;
  z-index: 5;
`;

export const Shadow = styled(Box)`
  position: absolute;
  bottom: -40px;
  left: -290px;
  z-index: -1;
`;
