package db

// Token contains the token details of the ERC20 contract at the given address.
// NOTE: The Token address will almost definitely be different on L1 and L2, so
// we need to track it on both chains when handling transactions.
type Token struct {
	Address  string `json:"address"`
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Decimals uint8  `json:"decimals"`
}
