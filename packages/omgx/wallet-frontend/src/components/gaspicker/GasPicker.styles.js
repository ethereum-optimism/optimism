import { styled } from '@material-ui/system';
import { Box, Card } from '@material-ui/core';

export const CardTag = styled(Card)`
  border: ${(props) => props.selected ? '2px solid #F0A000' : '2px solid #3A3F51'};
  opacity: ${(props) => props.selected ? 1.0 : 0.4}
`;

export const WrapperItem = styled(Box)`
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 18px 16px;
`;
