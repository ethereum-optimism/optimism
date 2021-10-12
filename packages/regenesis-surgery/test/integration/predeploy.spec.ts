/* eslint-disable @typescript-eslint/no-empty-function */
describe.skip('predeploys', () => {
  describe('new predeploys that are not ETH', () => {
    it('should have the exact state specified in the base genesis file', async () => {})
  })

  describe('predeploys where the old state should be wiped', () => {
    it('should have the code and storage of the base genesis file', async () => {})

    it('should have the same nonce and balance as before', async () => {})
  })

  describe('predeploys where the old state should be preserved', () => {
    it('should have the code of the base genesis file', async () => {})

    it('should have the combined storage of the old and new state', async () => {})

    it('should have the same nonce and balance as before', async () => {})
  })

  describe('OVM_ETH', () => {
    it('should have disabled ERC20 features', async () => {})

    it('should no recorded balance for the contracts that move to WETH9', async () => {})

    it('should have a new balance for WETH9 equal to the sum of the moved contract balances', async () => {})
  })

  describe('WETH9', () => {
    it('should have balances for each contract that should move', async () => {})

    it('should have a balance equal to the sum of all moved balances', async () => {})
  })
})
