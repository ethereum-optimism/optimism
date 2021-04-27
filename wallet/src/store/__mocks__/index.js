import configureMockStore from 'redux-mock-store';
import thunk from 'redux-thunk';

const middlewares = [ thunk ];
const mockStore = configureMockStore(middlewares);

const store = mockStore({
  fees: {},
  token: {
    '0x0000000000000000000000000000000000000000': {
      currency: '0x0000000000000000000000000000000000000000',
      decimals: 18,
      name: 'ETH'
    }
  }
});

export default store;
