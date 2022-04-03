package db

// PaginationParam holds the pagination fields passed through by the REST
// middleware and queried by the database to page through deposits and
// withdrawals.
type PaginationParam struct {
	Limit  uint64 `json:"limit"`
	Offset uint64 `json:"offset"`
	Total  uint64 `json:"total"`
}

type PaginatedDeposits struct {
	Param    *PaginationParam `json:"pagination"`
	Deposits []DepositJSON    `json:"items"`
}

type PaginatedWithdrawals struct {
	Param       *PaginationParam `json:"pagination"`
	Withdrawals []WithdrawalJSON `json:"items"`
}
