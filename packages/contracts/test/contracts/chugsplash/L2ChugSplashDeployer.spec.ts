import { expect } from '../../setup'

describe('L2ChugSplashDeployer', () => {
  describe('owner', () => {
    it('should have an owner', async () => {})
  })

  describe('setOwner', () => {
    it('should allow the current owner to change ownership', async () => {})

    it('should revert if caller is not the owner', async () => {})
  })

  describe('approveTransactionBundle', () => {
    it('should revert if caller is not the owner', async () => {})

    it('should allow the owner to approve a new transaction bundle', async () => {})

    it('should revert if trying to approve a bundle when another bundle is already active', async () => {})
  })

  describe('executeAction', () => {
    it('should revert if there is no active upgrade bundle', async () => {})

    it('should revert if the given action proof is invalid (1)', async () => {})

    it('should revert if the given action proof is invalid (2)', async () => {})

    it('should be able to trigger a SETCODE action', async () => {})

    it('should be able to trigger a SETSTORAGE action', async () => {})

    it('should change the upgrade status when the bundle is complete', async () => {})

    it('should allow the upgrader to submit a new bundle when the previous bundle is complete', async () => {})
  })

  describe('cancelTransactionBundle', () => {
    it('should revert if caller is not the owner', async () => {})

    it('should revert if there is no active bundle', async () => {})

    it('should allow the owner to cancel an active bundle immediately after creating it', async () => {})

    it('should allow the owner to cancel an active bundle after a few actions have been completed', async () => {})
  })
})
