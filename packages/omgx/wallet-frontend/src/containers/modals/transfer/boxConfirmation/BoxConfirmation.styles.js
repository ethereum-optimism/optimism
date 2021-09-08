import { Box } from '@material-ui/core';
import { styled } from '@material-ui/system';

export const StyleStages = styled(Box)`
  box-shadow: 10px -6px 234px rgba(1, 0, 74, 0.55), inset 33px 16px 80px rgba(255, 255, 255, 0.06);
  background: rgba(9, 22, 43, 0.5);
  border: 1px solid rgba(255, 255, 255, 0.2);
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-radius: 12px;
  padding: 20px 40px;
  margin-top: 30px;
`;

export const ContentCircles = styled(Box)`
  display: flex;
  justify-content: center;
  align-items: center;
  position: absolute;
  bottom: 30px;
  right: 70px;
  gap: 5px;
`;

export const Circle = styled(Box)`
  height: 7px;
  width: 7px;
  border-radius: 50%;
  background-color: #fff;
  opacity: ${props => props.active ? 0.8 : 0.2};
`;
