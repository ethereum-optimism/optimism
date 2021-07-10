import { IChainStore } from "./../types"
import { model, Schema } from "mongoose"
const chainStoreSchema: Schema = new Schema(
  {
    chainId: {
      type: Number,
      required: true,
    },
    lastFinalizedTxHeight: {
      type: Number,
      required: true,
    }
  }
)
export default model<IChainStore>("ChainStore", chainStoreSchema)