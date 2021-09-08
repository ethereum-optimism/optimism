import styled from '@emotion/styled';
import { Box } from '@material-ui/core'

export const DropdownWrapper = styled(Box)`
  display: flex;
  justify-content: center;
  align-items: center;
  flex-direction: row;
  gap: 10px;
`;

export const TableCell = styled(Box)`
  padding: 5px;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  justify-content: center;
  word-break: break-all
`;

export const TableBody = styled(Box)`
  display: flex;
  align-items: flex-start;
  justify-content: flex-start;
  gap: 5px;
  text-align: left;
`;
