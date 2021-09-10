import { styled } from '@material-ui/core/styles'
import { Box, Typography } from '@material-ui/core';

export const Content = styled(Box)`
  display: flex;
  flex-direction: column;
  gap: 20px;
  margin-bottom: 5px;
  background-color: rgba(255, 255, 255, 0.04);
  padding: 10px;
  border-radius: 6px;
`;

export const DropdownWrapper = styled(Box)`
  display: flex;
  justify-content: center;
  align-items: center;
  flex-direction: row;
  gap: 10px;
`;

export const TableCell = styled(Box)`
  display: flex;
  align-items: center;
  justify-content: center;
  width: 33.33%;
`;

export const TextTableCell = styled(Typography)`
  opacity: ${(props) => !props.enabled ? "0.4" : "1.0"};
  font-weight: 700;
`;

export const TableBody = styled(Box)`
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 5px;
  text-align: center;
`;

export const AccountAlertBox = styled(Box)`
  display: flex;
  justify-content: space-around;
  align-items: center;
  gap: 30px;
`
