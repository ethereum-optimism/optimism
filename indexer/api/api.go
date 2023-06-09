package api

import (
	"encoding/json"
	"net/http"

	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum/go-ethereum/common"
	"github.com/go-chi/chi/v5"
)

type PaginationResponse struct {
	// TODO type this better
	Data        interface{} `json:"data"`
	Cursor      string      `json:"cursor"`
	HasNextPage bool        `json:"hasNextPage"`
}

func (a *Api) DepositsHandler(w http.ResponseWriter, r *http.Request) {
	bv := a.bridgeView

	address := common.HexToAddress(chi.URLParam(r, "address"))

	// limit := getIntFromQuery(r, "limit", 10)
	// cursor := r.URL.Query().Get("cursor")
	// sortDirection := r.URL.Query().Get("sortDirection")

	deposits, err := bv.DepositsByAddress(address)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// This is not the shape of the response we want!!!
	// will add in the individual features in future prs 1 by 1
	response := PaginationResponse{
		Data: deposits,
		// Cursor:      nextCursor,
		HasNextPage: false,
	}

	jsonResponse(w, response, http.StatusOK)
}

func (a *Api) WithdrawalsHandler(w http.ResponseWriter, r *http.Request) {
	bv := a.bridgeView

	address := common.HexToAddress(chi.URLParam(r, "address"))

	// limit := getIntFromQuery(r, "limit", 10)
	// cursor := r.URL.Query().Get("cursor")
	// sortDirection := r.URL.Query().Get("sortDirection")

	withdrawals, err := bv.WithdrawalsByAddress(address)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// This is not the shape of the response we want!!!
	// will add in the individual features in future prs 1 by 1
	response := PaginationResponse{
		Data: withdrawals,
		// Cursor:      nextCursor,
		HasNextPage: false,
	}

	jsonResponse(w, response, http.StatusOK)
}

func jsonResponse(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type Api struct {
	Router     *chi.Mux
	bridgeView database.BridgeView
}

func NewApi(bv database.BridgeView) *Api {
	r := chi.NewRouter()

	api := &Api{
		Router:     r,
		bridgeView: bv,
	}
	// these regex are .+ because I wasn't sure what they should be
	// don't want a regex for addresses because would prefer to validate the address
	// with go-ethereum and throw a friendly error message
	r.Get("/api/v0/deposits/{address:.+}", api.DepositsHandler)
	r.Get("/api/v0/withdrawals/{address:.+}", api.WithdrawalsHandler)

	return api

}

func (a *Api) Listen(port string) error {
	return http.ListenAndServe(port, a.Router)
}
