
export interface Expectation {
  checker: (world: any) => Promise<void>;
}
