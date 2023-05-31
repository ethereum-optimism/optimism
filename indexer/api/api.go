package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi"
)

// TODO in this pr most of these types should be coming from the ORM instead

// DepositsDAO represents the Database Access Object for deposits
type DepositDAO interface {
	GetDeposits(limit int, cursor string, sortDirection string) ([]Deposit, string, bool, error)
}

// WithdrawalDAO represents the Database Access Object for deposits
type WithdrawalDAO interface {
	GetWithdrawals(limit int, cursor string, sortDirection string, sortBy string) ([]Withdrawal, string, bool, error)
}

// Deposit data structure
type Deposit struct {
	Guid            string        `json:"guid"`
	Amount          string        `json:"amount"`
	BlockNumber     int           `json:"blockNumber"`
	BlockTimestamp  time.Time     `json:"blockTimestamp"`
	From            string        `json:"from"`
	To              string        `json:"to"`
	TransactionHash string        `json:"transactionHash"`
	L1Token         TokenListItem `json:"l1Token"`
	L2Token         TokenListItem `json:"l2Token"`
}

// Withdrawal data structure
type Withdrawal struct {
	Guid            string        `json:"guid"`
	Amount          string        `json:"amount"`
	BlockNumber     int           `json:"blockNumber"`
	BlockTimestamp  time.Time     `json:"blockTimestamp"`
	From            string        `json:"from"`
	To              string        `json:"to"`
	TransactionHash string        `json:"transactionHash"`
	WithdrawalState string        `json:"withdrawalState"`
	Proof           *ProofClaim   `json:"proof"`
	Claim           *ProofClaim   `json:"claim"`
	L1Token         TokenListItem `json:"l1Token"`
	L2Token         TokenListItem `json:"l2Token"`
}

// TokenListItem data structure
type TokenListItem struct {
	ChainId    int        `json:"chainId"`
	Address    string     `json:"address"`
	Name       string     `json:"name"`
	Symbol     string     `json:"symbol"`
	Decimals   int        `json:"decimals"`
	LogoURI    string     `json:"logoURI"`
	Extensions Extensions `json:"extensions"`
}

// Extensions data structure
type Extensions struct {
	OptimismBridgeAddress string `json:"optimismBridgeAddress"`
	BridgeType            string `json:"bridgeType"`
}

// ProofClaim data structure
type ProofClaim struct {
	TransactionHash string    `json:"transactionHash"`
	BlockTimestamp  time.Time `json:"blockTimestamp"`
	BlockNumber     int       `json:"blockNumber"`
}

// PaginationResponse for paginated responses
type PaginationResponse struct {
	// TODO type this better
	Data        interface{} `json:"data"`
	Cursor      string      `json:"cursor"`
	HasNextPage bool        `json:"hasNextPage"`
}

func (a *Api) DepositsHandler(w http.ResponseWriter, r *http.Request) {

	limit := getIntFromQuery(r, "limit", 10)
	cursor := r.URL.Query().Get("cursor")
	sortDirection := r.URL.Query().Get("sortDirection")

	deposits, nextCursor, hasNextPage, err := a.DepositDAO.GetDeposits(limit, cursor, sortDirection)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := PaginationResponse{
		Data:        deposits,
		Cursor:      nextCursor,
		HasNextPage: hasNextPage,
	}

	jsonResponse(w, response, http.StatusOK)
}

func (a *Api) WithdrawalsHandler(w http.ResponseWriter, r *http.Request) {
	limit := getIntFromQuery(r, "limit", 10)
	cursor := r.URL.Query().Get("cursor")
	sortDirection := r.URL.Query().Get("sortDirection")
	sortBy := r.URL.Query().Get("sortBy")

	withdrawals, nextCursor, hasNextPage, err := a.WithdrawalDAO.GetWithdrawals(limit, cursor, sortDirection, sortBy)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := PaginationResponse{
		Data:        withdrawals,
		Cursor:      nextCursor,
		HasNextPage: hasNextPage,
	}

	jsonResponse(w, response, http.StatusOK)
}

func getIntFromQuery(r *http.Request, key string, defaultValue int) int {
	valueStr := r.URL.Query().Get(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func jsonResponse(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

type Api struct {
	Router        *chi.Mux
	DepositDAO    DepositDAO
	WithdrawalDAO WithdrawalDAO
}

func NewApi(depositDAO DepositDAO, withdrawalDAO WithdrawalDAO) *Api {
	r := chi.NewRouter()

	api := &Api{
		Router:        r,
		DepositDAO:    depositDAO,
		WithdrawalDAO: withdrawalDAO,
	}

	r.Get("/api/v0/deposits", api.DepositsHandler)
	r.Get("/api/v0/withdrawals", api.WithdrawalsHandler)

	return api

}

func (a *Api) Listen(port string) {
	http.ListenAndServe(port, a.Router)
}
