export interface CommitmentContract {
  submitBlock(root: Buffer): Promise<void>
}
