
export interface Invariant {
	held: boolean
  checker: (world: any) => Promise<void>;
}
