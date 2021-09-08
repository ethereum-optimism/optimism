import { Box } from '@material-ui/core';
import { styled } from '@material-ui/system';

export const WrapperActions = styled(Box)`
  display: flex;
  align-items: center;
  gap: 10px;
  justify-content: flex-end;
  margin-top: 50px;
`;

// export const StyleCreateTransactions = styled(Box)`
//   display: flex;
//   flex-direction: column;
//   gap: 50px;
//   background: #0F1B2D;
//   box-shadow: -13px 15px 39px rgba(0, 0, 0, 0.16), inset 123px 116px 230px rgba(255, 255, 255, 0.03);
//   backdrop-filter: blur(66px);
//   padding: 40px;
//   border-radius: 12px;
// `;

// export const Balance = styled(Box)`
//   background: rgba(9, 22, 43, 0.5);
//   box-shadow: -13px 15px 39px rgba(0, 0, 0, 0.16), inset 53px 36px 120px rgba(255, 255, 255, 0.06);
//   border-radius: 12px;
//   width: 100%;
//   /* position: relative; */
// `;

// export const ContentBalance = styled(Box)`
//   display: flex;
//   justify-content: space-between;
//   align-items: center;
//   padding: 40px 25px;
// `;

// export const TransactionsButton = styled(Box)`
//   display: flex;
//   flex-direction: column;
//   align-items: center;
//   border-radius: 50%;
//   box-shadow: 0 0 0 15px #0f1929;
//   backdrop-filter: blur(66px);
//   margin-top: 5px;
//   width: 134px;
//   height: 134px;
//   overflow: hidden;

//   position: relative;
//   z-index: 1;
// `;

// export const BridgeButton = styled(Button)`
//   height: 65px;
//   width: 65px;
//   border-radius: 100%;
//   background: #5663CE;
//   color: #fff;

//   position: absolute;
//   transform: translateY(50%);
//   z-index: 5;
//   min-width: 0;
//   &:hover {
//     background: #5663CE;
//     > span {
//       font-weight: 700;
//     }
//   }
// `;

// export const FastButton = styled(Button)`
//   width: 100%;
//   height: 100%;
//   z-index: 2;
//   align-items: flex-start;
//   color: #fff;
//   opacity: ${props => props.active === "fast" ? 1.0 : 0.2};
//   font-weight: ${props => props.active === "fast" ? 700 : 400};
//   background-color: ${props => props.active === "fast" ? "#415B92" : "transparent"};
//   &:hover {
//     background-color: ${props => props.active === "fast" ? "#415B92" : "transparent"};
//     opacity: 1.0;
//   }
// `;

// export const SlowButton = styled(Button)`
//   width: 100%;
//   height: 100%;
//   z-index: 2;
//   align-items: flex-end;
//   color: #fff;
//   opacity: ${props => props.active === "slow" ? 1.0 : 0.2};
//   font-weight: ${props => props.active === "slow" ? 700 : 400};
//   background-color: ${props => props.active === "slow" ? "#415B92" : "transparent"};
//   &:hover {
//     background-color: ${props => props.active === "slow" ? "#415B92" : "transparent"};
//     opacity: 1.0;
//   }
// `;

// export const SwapCircle = styled(Box)`
//   display: flex;
//   align-items: center;
//   justify-content: center;
//   text-align: center;
//   margin: 0 auto;
//   width: 40px;
//   height: 40px;
//   border-radius: 50%;
//   background-color: rgba(255, 255, 255, 0.05);
//   border: 1px solid rgba(9,22,43,0);
//   margin-bottom: 10px;
// `;

// export const Line = styled(Box)`
//   height: 80%;
//   width: 1px;
//   margin: 0 auto;
//   background-color: #fff;
//   opacity: 10%;
// `;
