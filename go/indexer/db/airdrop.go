package db

type Airdrop struct {
	Address              string `json:"address"`
	VoterAmount          string `json:"voterAmount"`
	MultisigSignerAmount string `json:"multisigSignerAmount"`
	GitcoinAmount        string `json:"gitcoinAmount"`
	ActiveBridgedAmount  string `json:"activeBridgedAmount"`
	OpUserAmount         string `json:"opUserAmount"`
	OpRepeatUserAmount   string `json:"opRepeatUserAmount"`
	BonusAmount          string `json:"bonusAmount"`
	TotalAmount          string `json:"totalAmount"`
}
